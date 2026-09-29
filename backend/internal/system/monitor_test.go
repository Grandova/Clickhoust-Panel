package system

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMetricsCacheConcurrentReaders(t *testing.T) {
	want := HostMetrics{CPUPercent: 42, DataDirBytes: 123, Timestamp: 12345}
	m := MetricsCollector{lastMetrics: want, collectedAt: time.Now()}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if got := m.Collect(); got != want {
				t.Errorf("cached sample changed: %+v", got)
			}
		})
	}
	wg.Wait()
}

func TestDirectorySize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "part"), []byte("123456"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := getDirSize(dir); got != 6 {
		t.Fatalf("directory size: got %d, want 6", got)
	}
}
