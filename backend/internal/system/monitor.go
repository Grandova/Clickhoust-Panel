package system

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type HostMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	CPUCount      int     `json:"cpu_count"`
	MemTotal      uint64  `json:"mem_total"`
	MemUsed       uint64  `json:"mem_used"`
	MemFree       uint64  `json:"mem_free"`
	MemPercent    float64 `json:"mem_percent"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskFree      uint64  `json:"disk_free"`
	DiskPercent   float64 `json:"disk_percent"`
	NetRxBytesSec uint64  `json:"net_rx_bytes_sec"`
	NetTxBytesSec uint64  `json:"net_tx_bytes_sec"`
	DataDirBytes  int64   `json:"data_dir_bytes"`
	LogDirBytes   int64   `json:"log_dir_bytes"`
	Timestamp     int64   `json:"timestamp"`
}

type MetricsCollector struct {
	mu          sync.Mutex
	lastNetRx   uint64
	lastNetTx   uint64
	lastNetTime time.Time
	lastMetrics HostMetrics
	collectedAt time.Time
	dirsAt      time.Time
	dirsLoading bool
}

var DefaultCollector = &MetricsCollector{
	lastNetTime: time.Now(),
}

func (m *MetricsCollector) Collect() HostMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	if time.Since(m.collectedAt) < 2*time.Second {
		return m.lastMetrics
	}

	var result HostMetrics
	result.Timestamp = time.Now().Unix()

	// CPU
	if percentages, err := cpu.Percent(0, false); err == nil && len(percentages) > 0 {
		result.CPUPercent = percentages[0]
	}
	if counts, err := cpu.Counts(true); err == nil {
		result.CPUCount = counts
	}

	// Memory
	if v, err := mem.VirtualMemory(); err == nil {
		result.MemTotal = v.Total
		result.MemUsed = v.Used
		result.MemFree = v.Available
		result.MemPercent = v.UsedPercent
	}

	// Root Disk or current path disk
	rootPath := "/"
	if !IsLinux() {
		rootPath = "C:\\"
	}
	if d, err := disk.Usage(rootPath); err == nil {
		result.DiskTotal = d.Total
		result.DiskUsed = d.Used
		result.DiskFree = d.Free
		result.DiskPercent = d.UsedPercent
	}

	// Network
	if ioCounters, err := net.IOCounters(false); err == nil && len(ioCounters) > 0 {
		currRx := ioCounters[0].BytesRecv
		currTx := ioCounters[0].BytesSent
		now := time.Now()

		durationSec := now.Sub(m.lastNetTime).Seconds()
		if durationSec > 0 && m.lastNetRx > 0 && currRx >= m.lastNetRx && currTx >= m.lastNetTx {
			result.NetRxBytesSec = uint64(float64(currRx-m.lastNetRx) / durationSec)
			result.NetTxBytesSec = uint64(float64(currTx-m.lastNetTx) / durationSec)
		}

		m.lastNetRx = currRx
		m.lastNetTx = currTx
		m.lastNetTime = now
	}

	result.DataDirBytes = m.lastMetrics.DataDirBytes
	result.LogDirBytes = m.lastMetrics.LogDirBytes
	// Directory walks must not block metrics requests on large ClickHouse installations.
	if !m.dirsLoading && time.Since(m.dirsAt) >= 5*time.Minute {
		m.dirsLoading = true
		go func() {
			dataSize := getDirSize("/var/lib/clickhouse")
			logSize := getDirSize("/var/log/clickhouse-server")
			m.mu.Lock()
			m.lastMetrics.DataDirBytes = dataSize
			m.lastMetrics.LogDirBytes = logSize
			m.dirsAt = time.Now()
			m.dirsLoading = false
			m.mu.Unlock()
		}()
	}
	m.lastMetrics = result
	m.collectedAt = time.Now()

	return result
}

func getDirSize(path string) int64 {
	var size int64
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return 0
	}

	_ = filepath.Walk(path, func(_ string, f os.FileInfo, err error) error {
		if err == nil && f != nil && !f.IsDir() {
			size += f.Size()
		}
		return nil
	})
	return size
}
