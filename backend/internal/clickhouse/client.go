package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Secure   bool   `json:"secure"`
	Protocol string `json:"protocol"` // native (default) or http
}

func DefaultConfig() ConnectionConfig {
	return ConnectionConfig{
		Host:     "127.0.0.1",
		Port:     9000,
		User:     "default",
		Password: "",
		Database: "default",
		Secure:   false,
		Protocol: "native",
	}
}

type ClickHouseService struct {
	mu              sync.RWMutex
	config          ConnectionConfig
	conn            driver.Conn
	keepAliveOnce   sync.Once
	stopKeepAlive   chan struct{}
	metricsMu       sync.Mutex
	lastMetrics     *MetricsSummary
	lastEvents      [7]uint64
	lastMetricsTime time.Time
	lastTotalsTime  time.Time
}

var DefaultService = &ClickHouseService{
	config:        DefaultConfig(),
	stopKeepAlive: make(chan struct{}),
}

func (s *ClickHouseService) SetConfig(cfg ConnectionConfig) {
	s.mu.Lock()
	s.config = cfg
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()
	s.metricsMu.Lock()
	s.lastMetrics = nil
	s.lastMetricsTime = time.Time{}
	s.lastTotalsTime = time.Time{}
	s.metricsMu.Unlock()

	if s.stopKeepAlive == nil {
		return
	}
	// Eagerly pre-warm connection in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.GetConn(ctx)
	}()
}

func (s *ClickHouseService) GetConfig() ConnectionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetConn returns the persistent long-connection pool instantly without blocking on Ping()
func (s *ClickHouseService) GetConn(ctx context.Context) (driver.Conn, error) {
	// 1. Fast path: Read-lock check for existing active connection pool
	s.mu.RLock()
	if s.conn != nil {
		c := s.conn
		s.mu.RUnlock()
		return c, nil
	}
	s.mu.RUnlock()

	// 2. Slow path: Acquire write lock to initialize the connection pool
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-checked locking
	if s.conn != nil {
		return s.conn, nil
	}

	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	opts := &clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: s.config.Database,
			Username: s.config.User,
			Password: s.config.Password,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		// Production-grade connection pool configuration for high-performance reuse
		MaxOpenConns:     30,
		MaxIdleConns:     15,
		ConnMaxLifetime:  2 * time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenRoundRobin,
	}

	if s.config.Protocol == "http" {
		opts.Protocol = clickhouse.HTTP
		// The panel uses password authentication, including an empty password.
		opts.HttpHeaders = map[string]string{"X-ClickHouse-SSL-Certificate-Auth": "off"}
	} else if s.config.Protocol == "native" || s.config.Protocol == "" {
		opts.Protocol = clickhouse.Native
	} else {
		return nil, fmt.Errorf("unsupported ClickHouse protocol: %s", s.config.Protocol)
	}

	if s.config.Secure {
		opts.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("connect to ClickHouse at %s failed: %w", addr, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := conn.Ping(pingCtx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping ClickHouse at %s failed: %w (is ClickHouse running?)", addr, err)
	}

	s.conn = conn

	// Start background keepalive loop once
	s.keepAliveOnce.Do(func() {
		if s.stopKeepAlive != nil {
			go s.startKeepAliveLoop()
		}
	})

	return conn, nil
}

// startKeepAliveLoop maintains active long connections in the background
func (s *ClickHouseService) startKeepAliveLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopKeepAlive:
			return
		case <-ticker.C:
			s.mu.RLock()
			conn := s.conn
			s.mu.RUnlock()

			if conn != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if err := conn.Ping(ctx); err != nil {
					log.Printf("[ClickHouse] Background keepalive ping failed: %v, resetting connection pool", err)
					s.mu.Lock()
					if s.conn == conn {
						_ = s.conn.Close()
						s.conn = nil
					}
					s.mu.Unlock()
				}
				cancel()
			}
		}
	}
}

func (s *ClickHouseService) Ping(ctx context.Context) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Ping(ctx)
}

func (s *ClickHouseService) GetVersion(ctx context.Context) (string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return "", err
	}

	var version string
	row := conn.QueryRow(ctx, "SELECT version()")
	if err := row.Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

func CheckPortOpen(host string, port int, timeout time.Duration) bool {
	target := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
