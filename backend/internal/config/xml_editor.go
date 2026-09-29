package config

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/system"
)

type ConfigFileInfo struct {
	Name         string    `json:"name"`
	RelativePath string    `json:"relative_path"`
	FullPath     string    `json:"full_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsDirectory  bool      `json:"is_directory"`
}

type XMLValidationResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ListConfigFiles scans config directory for xml files
func ListConfigFiles() ([]ConfigFileInfo, error) {
	baseDir := GetConfigDir()
	var list []ConfigFileInfo

	// Ensure basic files exist in dev mode if missing
	ensureDefaultFiles(baseDir)

	_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if strings.Contains(info.Name(), ".bak.") || strings.HasSuffix(info.Name(), "~") {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xml") {
			rel, _ := filepath.Rel(baseDir, path)
			list = append(list, ConfigFileInfo{
				Name:         info.Name(),
				RelativePath: filepath.ToSlash(rel),
				FullPath:     path,
				Size:         info.Size(),
				ModTime:      info.ModTime(),
			})
		}
		return nil
	})

	return list, nil
}

func ensureDefaultFiles(baseDir string) {
	mainCfg := filepath.Join(baseDir, "config.xml")
	if !fileExists(mainCfg) {
		defaultContent := `<clickhouse>
    <logger>
        <level>information</level>
        <log>/var/log/clickhouse-server/clickhouse-server.log</log>
        <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
        <size>1000M</size>
        <count>10</count>
    </logger>
    <http_port>8123</http_port>
    <tcp_port>9000</tcp_port>
    <listen_host>0.0.0.0</listen_host>
    <max_connections>4096</max_connections>
    <keep_alive_timeout>10</keep_alive_timeout>
    <max_concurrent_queries>100</max_concurrent_queries>
    <path>/var/lib/clickhouse/</path>
    <tmp_path>/var/lib/clickhouse/tmp/</tmp_path>
    <user_files_path>/var/lib/clickhouse/user_files/</user_files_path>
</clickhouse>`
		_ = os.WriteFile(mainCfg, []byte(defaultContent), 0644)
	}

	usersCfg := filepath.Join(baseDir, "users.xml")
	if !fileExists(usersCfg) {
		defaultUsers := `<clickhouse>
    <profiles>
        <default>
            <max_memory_usage>10000000000</max_memory_usage>
            <use_uncompressed_cache>0</use_uncompressed_cache>
            <load_balancing>random</load_balancing>
        </default>
        <readonly>
            <readonly>1</readonly>
        </readonly>
    </profiles>
    <users>
        <default>
            <password></password>
            <networks>
                <ip>::/0</ip>
            </networks>
            <profile>default</profile>
            <quota>default</quota>
        </default>
    </users>
    <quotas>
        <default>
            <interval>
                <duration>3600</duration>
                <queries>0</queries>
                <errors>0</errors>
                <result_rows>0</result_rows>
                <read_rows>0</read_rows>
                <execution_time>0</execution_time>
            </interval>
        </default>
    </quotas>
</clickhouse>`
		_ = os.WriteFile(usersCfg, []byte(defaultUsers), 0644)
	}

	confD := filepath.Join(baseDir, "config.d")
	_ = os.MkdirAll(confD, 0755)
	systemLogsCfg := filepath.Join(confD, "system-logs.xml")
	if !fileExists(systemLogsCfg) {
		defaultLogsCfg := `<clickhouse>
    <logger>
        <level replace="1">information</level>
    </logger>
    <!-- 降低内置监控日志写入频率并限制保留周期 (TTL 3天)，避免日志膨胀占用磁盘与内存 -->
    <metric_log replace="1">
        <database>system</database>
        <table>metric_log</table>
        <engine>ENGINE = MergeTree PARTITION BY toYYYYMM(event_date) ORDER BY (event_date, event_time) TTL event_date + INTERVAL 3 DAY DELETE</engine>
        <flush_interval_milliseconds>30000</flush_interval_milliseconds>
        <collect_interval_milliseconds>15000</collect_interval_milliseconds>
    </metric_log>
    <asynchronous_metric_log replace="1">
        <database>system</database>
        <table>asynchronous_metric_log</table>
        <engine>ENGINE = MergeTree PARTITION BY toYYYYMM(event_date) ORDER BY (event_date, event_time) TTL event_date + INTERVAL 3 DAY DELETE</engine>
        <flush_interval_milliseconds>60000</flush_interval_milliseconds>
    </asynchronous_metric_log>
    <!-- 关闭非必要采样分析日志 -->
    <trace_log remove="1"/>
    <processors_profile_log remove="1"/>
</clickhouse>`
		_ = os.WriteFile(systemLogsCfg, []byte(defaultLogsCfg), 0644)
	}
}

// ReadConfigFile reads config file content
func ReadConfigFile(relativePath string) (string, error) {
	baseDir := GetConfigDir()
	cleanRel := filepath.Clean(relativePath)
	if !filepath.IsLocal(relativePath) {
		return "", fmt.Errorf("invalid path")
	}

	fullPath := filepath.Join(baseDir, cleanRel)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ValidateXMLContent tests if XML syntax is valid
func ValidateXMLContent(content string) XMLValidationResult {
	decoder := xml.NewDecoder(strings.NewReader(content))
	depth, roots := 0, 0
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			line, _ := decoder.InputPos()
			return XMLValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("XML 语法错误: %v (行号: %d)", err, line),
				Line:    line,
			}
		}
		switch value := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(value)) != "" {
				return XMLValidationResult{Message: "XML 根元素外不能包含文本"}
			}
		}
	}
	if roots != 1 {
		return XMLValidationResult{Message: "XML 必须包含且仅包含一个根元素"}
	}
	return XMLValidationResult{Valid: true, Message: "XML 格式有效"}
}

// SaveConfigFile validates, backups, and saves XML file
func SaveConfigFile(relativePath, newContent, reason, username string) error {
	val := ValidateXMLContent(newContent)
	if !val.Valid {
		return fmt.Errorf("XML 语法验证未通过: %s", val.Message)
	}

	baseDir := GetConfigDir()
	cleanRel := filepath.Clean(relativePath)
	if !filepath.IsLocal(relativePath) {
		return fmt.Errorf("invalid path")
	}

	fullPath := filepath.Join(baseDir, cleanRel)

	// If file exists, create timestamped backup
	oldContent, readErr := os.ReadFile(fullPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	if readErr == nil {
		BackupConfigFile(fullPath, string(oldContent), reason, username)
	}

	_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write config file failed: %w", err)
	}

	// Test ClickHouse validate-config if binary exists on Linux
	if system.IsLinux() {
		if out, err := system.DefaultServiceController.ValidateConfig(); err != nil {
			var restoreErr error
			if readErr == nil {
				restoreErr = os.WriteFile(fullPath, oldContent, 0644)
			} else {
				restoreErr = os.Remove(fullPath)
			}
			if restoreErr != nil {
				return fmt.Errorf("配置验证失败 (%v)，恢复原配置失败: %w", err, restoreErr)
			}
			return fmt.Errorf("ClickHouse 验证配置警告/错误: %s (%v)", out, err)
		}
	}

	return nil
}

// ListConfigBackups fetches historical backups for a file
func ListConfigBackups(relativePath string) ([]database.ConfigBackup, error) {
	if !filepath.IsLocal(relativePath) {
		return nil, fmt.Errorf("invalid path")
	}
	baseDir := GetConfigDir()
	fullPath := filepath.Join(baseDir, filepath.Clean(relativePath))

	var backups []database.ConfigBackup
	if database.DB == nil {
		return backups, nil
	}

	err := database.DB.Where("file_path = ?", fullPath).Order("created_at desc").Limit(30).Find(&backups).Error
	return backups, err
}

// RollbackConfig restores a specific backup ID
func RollbackConfig(backupID uint, username string) error {
	var backup database.ConfigBackup
	if err := database.DB.First(&backup, backupID).Error; err != nil {
		return err
	}

	relativePath, err := filepath.Rel(GetConfigDir(), backup.FilePath)
	if err != nil {
		return err
	}
	return SaveConfigFile(relativePath, backup.Content, fmt.Sprintf("Rollback to backup #%d", backupID), username)
}
