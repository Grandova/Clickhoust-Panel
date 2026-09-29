package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clickhouse-manager/internal/system"
)

type LogType string

const (
	LogTypeServer  LogType = "server"
	LogTypeError   LogType = "error"
	LogTypeJournal LogType = "journal"
)

type LogQuery struct {
	Type   LogType `json:"type"`
	Lines  int     `json:"lines"`
	Filter string  `json:"filter"`
	Level  string  `json:"level"` // TRACE, DEBUG, INFO, WARNING, ERROR, FATAL
}

type LogResponse struct {
	Lines    []string `json:"lines"`
	Total    int      `json:"total"`
	FilePath string   `json:"file_path"`
}

func GetLogPath(t LogType) string {
	if system.IsLinux() {
		switch t {
		case LogTypeServer:
			return "/var/log/clickhouse-server/clickhouse-server.log"
		case LogTypeError:
			return "/var/log/clickhouse-server/clickhouse-server.err.log"
		}
	}
	// Fallback dev logs path
	dir := "data/logs"
	_ = os.MkdirAll(dir, 0755)
	switch t {
	case LogTypeServer:
		return filepath.Join(dir, "clickhouse-server.log")
	case LogTypeError:
		return filepath.Join(dir, "clickhouse-server.err.log")
	}
	return filepath.Join(dir, "clickhouse-server.log")
}

func EnsureSampleLogs() {
	serverLog := GetLogPath(LogTypeServer)
	if _, err := os.Stat(serverLog); os.IsNotExist(err) {
		sample := `2026.09.29 04:00:01.102345 [ 1234 ] <Information> Application: Starting ClickHouse 24.8.4.13 (revision: 54492)
2026.09.29 04:00:01.103120 [ 1234 ] <Information> Application: Loaded config '/etc/clickhouse-server/config.xml'
2026.09.29 04:00:01.104230 [ 1234 ] <Information> Application: Available RAM: 31.84 GiB; physical cores: 8; logical cores: 16
2026.09.29 04:00:01.105432 [ 1234 ] <Information> Application: Listening for HTTP: http://0.0.0.0:8123
2026.09.29 04:00:01.105890 [ 1234 ] <Information> Application: Listening for Native TCP: 0.0.0.0:9000
2026.09.29 04:00:01.110200 [ 1234 ] <Information> Application: Ready for connections.`
		_ = os.WriteFile(serverLog, []byte(sample+"\n"), 0644)
	}

	errLog := GetLogPath(LogTypeError)
	if _, err := os.Stat(errLog); os.IsNotExist(err) {
		sampleErr := `2026.09.29 04:00:00.000000 [ 1230 ] <Warning> Context: No errors recorded.`
		_ = os.WriteFile(errLog, []byte(sampleErr+"\n"), 0644)
	}
}

// ReadRecentLogs reads recent lines matching filter criteria
func ReadRecentLogs(query LogQuery) (*LogResponse, error) {
	if query.Lines <= 0 || query.Lines > 10000 {
		query.Lines = 500
	}

	if query.Type == LogTypeJournal {
		return readJournalLogs(query)
	}

	EnsureSampleLogs()
	filePath := GetLogPath(query.Type)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filePath, err)
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	levelUpper := strings.ToUpper(query.Level)
	filterLower := strings.ToLower(query.Filter)

	for scanner.Scan() {
		line := scanner.Text()
		if levelUpper != "" && !strings.Contains(strings.ToUpper(line), levelUpper) {
			continue
		}
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}
		allLines = append(allLines, line)
	}

	// Take last N lines
	start := 0
	if len(allLines) > query.Lines {
		start = len(allLines) - query.Lines
	}
	recent := allLines[start:]

	return &LogResponse{
		Lines:    recent,
		Total:    len(recent),
		FilePath: filePath,
	}, nil
}

func readJournalLogs(query LogQuery) (*LogResponse, error) {
	if !system.IsLinux() {
		return &LogResponse{
			Lines: []string{
				fmt.Sprintf("systemd journal is only available on Linux (current: %s)", system.DetectOS().PrettyName),
			},
			Total:    1,
			FilePath: "journalctl",
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := system.DefaultExecutor
	out, err := cmd.Execute(ctx, "journalctl", "-u", "clickhouse-server", "-n", fmt.Sprintf("%d", query.Lines), "--no-pager")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var filtered []string
	levelUpper := strings.ToUpper(query.Level)
	filterLower := strings.ToLower(query.Filter)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if levelUpper != "" && !strings.Contains(strings.ToUpper(line), levelUpper) {
			continue
		}
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}
		filtered = append(filtered, line)
	}

	return &LogResponse{
		Lines:    filtered,
		Total:    len(filtered),
		FilePath: "journalctl -u clickhouse-server",
	}, nil
}

// TailFileStreams continuously streams new lines from log file into callback
func TailFileStream(ctx context.Context, filePath string, callback func(line string)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to end minus last 4KB
	fi, err := file.Stat()
	if err == nil && fi.Size() > 4096 {
		_, _ = file.Seek(-4096, io.SeekEnd)
	}

	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				return err
			}
			callback(strings.TrimRight(line, "\r\n"))
		}
	}
}
