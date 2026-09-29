package config

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/system"
)

type ConfigItemMeta struct {
	Key             string   `json:"key"`
	Category        string   `json:"category"` // network, memory, query, cache, mergetree, log
	CurrentValue    string   `json:"current_value"`
	DefaultValue    string   `json:"default_value"`
	Description     string   `json:"description"`
	Type            string   `json:"type"` // int, float, string, select
	RequiresRestart bool     `json:"requires_restart"`
	Recommended     string   `json:"recommended"`
	Options         []string `json:"options,omitempty"`
}

type ConfigCatalog struct {
	Items []ConfigItemMeta `json:"items"`
}

func GetConfigDir() string {
	if system.IsLinux() && fileExists("/etc/clickhouse-server") {
		return "/etc/clickhouse-server"
	}
	dir := "data/clickhouse_config"
	_ = os.MkdirAll(filepath.Join(dir, "config.d"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "users.d"), 0755)
	return dir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetPredefinedConfigMetas() []ConfigItemMeta {
	return []ConfigItemMeta{
		// Network
		{Key: "listen_host", Category: "network", DefaultValue: "0.0.0.0", Description: "监听的客户端网络地址 (0.0.0.0 监听全部网卡 IPv4)", Type: "string", RequiresRestart: true, Recommended: "0.0.0.0 或 ::"},
		{Key: "http_port", Category: "network", DefaultValue: "8123", Description: "HTTP 客户端连接端口", Type: "int", RequiresRestart: true, Recommended: "8123"},
		{Key: "tcp_port", Category: "network", DefaultValue: "9000", Description: "原生 Native TCP 协议通信端口", Type: "int", RequiresRestart: true, Recommended: "9000"},
		{Key: "https_port", Category: "network", DefaultValue: "8443", Description: "HTTPS 加密连接端口", Type: "int", RequiresRestart: true, Recommended: "8443"},
		{Key: "tcp_port_secure", Category: "network", DefaultValue: "9440", Description: "加密 Native TCP 协议端口", Type: "int", RequiresRestart: true, Recommended: "9440"},
		{Key: "interserver_http_port", Category: "network", DefaultValue: "9009", Description: "副本节点间数据交换的 HTTP 端口", Type: "int", RequiresRestart: true, Recommended: "9009"},
		{Key: "max_connections", Category: "network", DefaultValue: "4096", Description: "服务允许的最大并发网络连接数", Type: "int", RequiresRestart: true, Recommended: "1024 - 10000"},
		{Key: "keep_alive_timeout", Category: "network", DefaultValue: "10", Description: "HTTP Keep-Alive 保持连接超时秒数", Type: "int", RequiresRestart: true, Recommended: "5 - 60"},
		{Key: "max_concurrent_queries", Category: "network", DefaultValue: "100", Description: "最大允许并发执行的查询数", Type: "int", RequiresRestart: false, Recommended: "50 - 500"},

		// Memory
		{Key: "max_server_memory_usage", Category: "memory", DefaultValue: "0", Description: "整个 ClickHouse 服务能使用的最大内存字节数 (0 表示不单独硬性限制)", Type: "int", RequiresRestart: false, Recommended: "物理内存的 80%-90%"},
		{Key: "max_server_memory_usage_to_ram_ratio", Category: "memory", DefaultValue: "0.9", Description: "服务可用内存占系统总 RAM 的最大比例", Type: "float", RequiresRestart: false, Recommended: "0.8 - 0.95"},
		{Key: "max_memory_usage", Category: "memory", DefaultValue: "10000000000", Description: "单次用户查询的最大内存限制 (字节)", Type: "int", RequiresRestart: false, Recommended: "物理内存的 50% 左右"},
		{Key: "max_memory_usage_for_user", Category: "memory", DefaultValue: "0", Description: "单个用户所有并发查询累计的最大内存限制 (0 为不限制)", Type: "int", RequiresRestart: false, Recommended: "0 或根据用户分配"},

		// Query
		{Key: "max_execution_time", Category: "query", DefaultValue: "60", Description: "单个查询的最长执行超时时间 (秒)", Type: "int", RequiresRestart: false, Recommended: "30 - 300"},
		{Key: "max_threads", Category: "query", DefaultValue: "0", Description: "查询管道处理最大线程数 (0 表示按 CPU 核心数自动决定)", Type: "int", RequiresRestart: false, Recommended: "0 或 CPU 物理核心数"},
		{Key: "max_block_size", Category: "query", DefaultValue: "65536", Description: "数据块在管道中传递时的最大行数", Type: "int", RequiresRestart: false, Recommended: "65536"},
		{Key: "max_insert_block_size", Category: "query", DefaultValue: "1048545", Description: "写入表时缓冲区的最大块大小 (行)", Type: "int", RequiresRestart: false, Recommended: "100000 - 1048576"},
		{Key: "max_result_rows", Category: "query", DefaultValue: "0", Description: "单次查询返回结果行数上限 (0 为不限制)", Type: "int", RequiresRestart: false, Recommended: "0 或 100000"},
		{Key: "max_result_bytes", Category: "query", DefaultValue: "0", Description: "单次查询返回结果字节数上限 (0 为不限制)", Type: "int", RequiresRestart: false, Recommended: "0 或 1073741824"},

		// Cache
		{Key: "mark_cache_size", Category: "cache", DefaultValue: "5368709120", Description: "MergeTree 表 mark 索引缓存大小 (字节)", Type: "int", RequiresRestart: true, Recommended: "2GB - 8GB"},
		{Key: "uncompressed_cache_size", Category: "cache", DefaultValue: "0", Description: "解压后数据块缓存大小 (字节, 默认 0 不建议大批量开启)", Type: "int", RequiresRestart: true, Recommended: "0 或 2GB"},
		{Key: "query_cache", Category: "cache", DefaultValue: "1", Description: "是否启用查询结果缓存 (0 关闭, 1 开启)", Type: "select", RequiresRestart: false, Recommended: "0 或 1", Options: []string{"0", "1"}},

		// MergeTree
		{Key: "background_pool_size", Category: "mergetree", DefaultValue: "16", Description: "后台 Merge 和 Mutation 操作的线程池大小", Type: "int", RequiresRestart: true, Recommended: "16 - 64"},
		{Key: "background_merges_mutations_concurrency_ratio", Category: "mergetree", DefaultValue: "2", Description: "并发 Merge/Mutation 任务与 background_pool_size 的比率", Type: "int", RequiresRestart: true, Recommended: "2"},
		{Key: "parts_to_delay_insert", Category: "mergetree", DefaultValue: "150", Description: "分区内 part 达到该数量时延迟写入以缓解合并压力", Type: "int", RequiresRestart: false, Recommended: "150 - 300"},
		{Key: "parts_to_throw_insert", Category: "mergetree", DefaultValue: "300", Description: "分区内 part 达到该数量时直接拒绝写入报错", Type: "int", RequiresRestart: false, Recommended: "300 - 600"},

		// Logger
		{Key: "logger_level", Category: "log", DefaultValue: "information", Description: "ClickHouse 服务日志记录级别", Type: "select", RequiresRestart: true, Recommended: "information 或 warning", Options: []string{"trace", "debug", "information", "notice", "warning", "error", "critical"}},
		{Key: "logger_log", Category: "log", DefaultValue: "/var/log/clickhouse-server/clickhouse-server.log", Description: "主服务日志文件路径", Type: "string", RequiresRestart: true, Recommended: "/var/log/clickhouse-server/clickhouse-server.log"},
		{Key: "logger_errorlog", Category: "log", DefaultValue: "/var/log/clickhouse-server/clickhouse-server.err.log", Description: "错误日志文件路径", Type: "string", RequiresRestart: true, Recommended: "/var/log/clickhouse-server/clickhouse-server.err.log"},
		{Key: "logger_size", Category: "log", DefaultValue: "1000M", Description: "单日志文件滚动前最大容量", Type: "string", RequiresRestart: true, Recommended: "500M - 2000M"},
		{Key: "logger_count", Category: "log", DefaultValue: "10", Description: "保留的历史归档日志文件数量", Type: "int", RequiresRestart: true, Recommended: "5 - 30"},
	}
}

// GetFormConfigs reads drop-in overrides and returns the configuration catalog with current values
func GetFormConfigs() []ConfigItemMeta {
	items := GetPredefinedConfigMetas()
	cfgDir := GetConfigDir()
	overrideFile := filepath.Join(cfgDir, "config.d", "ch_manager_overrides.xml")

	values := make(map[string]string)
	if content, err := os.ReadFile(overrideFile); err == nil {
		type Root struct {
			XMLName xml.Name
			Nodes   []ConfigNode `xml:",any"`
		}
		var root Root
		if err := xml.Unmarshal(content, &root); err == nil {
			for _, n := range root.Nodes {
				values[n.XMLName.Local] = string(n.Content)
			}
		}
	}

	for i := range items {
		if val, exists := values[items[i].Key]; exists && val != "" {
			items[i].CurrentValue = val
		} else {
			items[i].CurrentValue = items[i].DefaultValue
		}
	}
	return items
}

type ConfigNode struct {
	XMLName xml.Name
	Content []byte `xml:",innerxml"`
}

// SaveFormConfigs writes modified configs into /etc/clickhouse-server/config.d/ch_manager_overrides.xml with automatic backup
func SaveFormConfigs(updatedValues map[string]string, username string) error {
	cfgDir := GetConfigDir()
	targetPath := filepath.Join(cfgDir, "config.d", "ch_manager_overrides.xml")

	// Backup before saving
	if fileExists(targetPath) {
		oldContent, _ := os.ReadFile(targetPath)
		if len(oldContent) > 0 {
			BackupConfigFile(targetPath, string(oldContent), "Form Config Auto-Backup", username)
		}
	}

	var xmlLines []string
	xmlLines = append(xmlLines, "<clickhouse>")
	for k, v := range updatedValues {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		xmlLines = append(xmlLines, fmt.Sprintf("    <%s>%s</%s>", k, v, k))
	}
	xmlLines = append(xmlLines, "</clickhouse>")

	content := strings.Join(xmlLines, "\n") + "\n"
	return os.WriteFile(targetPath, []byte(content), 0644)
}

func BackupConfigFile(filePath, content, reason, username string) {
	ts := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.bak.%s", filePath, ts)
	_ = os.WriteFile(backupPath, []byte(content), 0644)

	if database.DB != nil {
		database.DB.Create(&database.ConfigBackup{
			FilePath:  filePath,
			Content:   content,
			Reason:    reason,
			CreatedBy: username,
			CreatedAt: time.Now(),
		})
	}
}
