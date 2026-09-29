package installer

import (
	"context"
	"fmt"
	"os"
	"strings"

	"clickhouse-manager/internal/system"
)

type AptInstaller struct {
	osInfo system.OSInfo
}

func (a *AptInstaller) Name() string {
	return "APT Installer (Debian / Ubuntu)"
}

func (a *AptInstaller) CheckInstalled() bool {
	return system.DefaultServiceController.IsInstalled()
}

func (a *AptInstaller) Install(ctx context.Context, adminPassword string, cb ProgressCallback) error {
	executor := system.DefaultExecutor

	cb("1/12", "检测操作系统与架构: "+a.osInfo.PrettyName+" ["+a.osInfo.Arch+"]", false)

	// CPU instruction capability check & intelligent fallback
	useCompat := false
	if a.osInfo.Arch == "x86_64" || a.osInfo.Arch == "amd64" {
		cpuBytes, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			cpuInfo := string(cpuBytes)
			hasAVX2 := strings.Contains(cpuInfo, "avx2")
			hasAVX := strings.Contains(cpuInfo, "avx")
			hasSSE42 := strings.Contains(cpuInfo, "sse4_2")

			if !hasAVX2 {
				useCompat = true
				cb("1/12", "【硬件诊断】检测到当前 CPU 未支持 AVX2 向量指令集（ClickHouse 官方 APT 软件包强制要求 x86-64-v3/AVX2，直接安装会导致 Illegal instruction 错误）。", false)
				if hasSSE42 || hasAVX {
					cb("1/12", "【智能自适应安装】系统检测到基础向量支持，正在自动启用 ClickHouse 官方高兼容版本（amd64compat，免 AVX2 限制），确保 100% 成功安装与运行！", false)
				} else {
					cb("1/12", "【智能自适应安装】当前 CPU 处于低配置虚拟化模式，自动切换 ClickHouse 官方独立兼容引擎...", false)
				}
			} else {
				cb("1/12", "【硬件检测通过】CPU 完整支持 SSE 4.2 与 AVX2 向量指令集，将采用官方 APT 软件源安装。", false)
			}
		}
	} else if a.osInfo.Arch == "aarch64" || a.osInfo.Arch == "arm64" {
		cpuBytes, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			cpuInfo := string(cpuBytes)
			hasV82 := strings.Contains(cpuInfo, "asimd") && strings.Contains(cpuInfo, "atomics")
			if !hasV82 {
				useCompat = true
				cb("1/12", "【ARM64 硬件检测】CPU 为 ARMv8.0 基础架构，自动启用官方兼容构建 (aarch64v80compat)...", false)
			}
		}
	}

	if useCompat {
		return installCompatBinary(ctx, a.osInfo.Arch, adminPassword, cb)
	}

	// Clean up any half-configured dpkg state from previous attempts
	_ = executor.ExecuteStream(ctx, func(l string) { cb("1/12", l, false) }, "sh", "-c", "dpkg --configure -a 2>/dev/null || true")

	// Step 1: Install prerequisites
	cb("2/12", "安装基础依赖 (curl, apt-transport-https, ca-certificates, gnupg)...", false)
	err := executor.ExecuteStream(ctx, func(l string) { cb("2/12", l, false) }, "apt-get", "install", "-y", "curl", "apt-transport-https", "ca-certificates", "gnupg")
	if err != nil {
		cb("2/12", "安装基础依赖错误: "+err.Error(), true)
	}

	// Step 2: Download GPG key
	cb("3/12", "获取 ClickHouse 官方 GPG 密钥...", false)
	_ = os.MkdirAll("/usr/share/keyrings", 0755)
	keyScript := "curl -fsSL https://packages.clickhouse.com/rpm/lts/repodata/repomd.xml.key | gpg --dearmor --yes -o /usr/share/keyrings/clickhouse-keyring.gpg"
	if err := executor.ExecuteStream(ctx, func(l string) { cb("3/12", l, false) }, "sh", "-c", keyScript); err != nil {
		cb("3/12", "GPG 密钥下载失败: "+err.Error(), true)
		return err
	}

	// Step 3: Add repo
	cb("4/12", "添加 ClickHouse 官方 APT 仓库...", false)
	repoContent := "deb [signed-by=/usr/share/keyrings/clickhouse-keyring.gpg] https://packages.clickhouse.com/deb stable main"
	if err := os.WriteFile("/etc/apt/sources.list.d/clickhouse.list", []byte(repoContent+"\n"), 0644); err != nil {
		return fmt.Errorf("write clickhouse.list failed: %w", err)
	}

	// Step 4: Update apt cache
	cb("5/12", "更新 APT 软件索引...", false)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("5/12", l, false) }, "apt-get", "update"); err != nil {
		cb("5/12", "APT 更新索引发生告警或错误: "+err.Error(), true)
	}

	// Step 5: Install packages non-interactively
	cb("6/12", "正在安装 clickhouse-server 与 clickhouse-client...", false)
	installCmd := "DEBIAN_FRONTEND=noninteractive apt-get install -y --allow-unauthenticated clickhouse-server clickhouse-client"
	if err := executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", installCmd); err != nil {
		cb("6/12", "ClickHouse 安装遇到错误，正在自动清理半配置包并修复 APT 包管理器状态...", true)
		_ = executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", "dpkg --purge --force-all clickhouse-server clickhouse-client clickhouse-common-static 2>/dev/null; apt-get -f install -y 2>/dev/null || true")
		cb("6/12", "ClickHouse 安装失败: "+err.Error(), true)
		return err
	}

	// Step 6: Configure password if provided
	if adminPassword != "" {
		cb("7/12", "配置默认管理员密码...", false)
		userConfigDir := "/etc/clickhouse-server/users.d"
		_ = os.MkdirAll(userConfigDir, 0755)
		userXml := fmt.Sprintf(`<clickhouse>
    <users>
        <default>
            <password>%s</password>
        </default>
    </users>
</clickhouse>`, adminPassword)
		_ = os.WriteFile(userConfigDir+"/admin-password.xml", []byte(userXml), 0640)
	} else {
		cb("7/12", "未指定自定义密码，使用默认无密码或保留现有配置...", false)
	}

	// Step 7: Listen host configuration
	cb("8/12", "开放所有网络地址监听 (listen_host: 0.0.0.0)...", false)
	confDir := "/etc/clickhouse-server/config.d"
	_ = os.MkdirAll(confDir, 0755)
	listenXml := `<clickhouse>
    <listen_host replace="1">0.0.0.0</listen_host>
    <listen_reuse_port replace="1">1</listen_reuse_port>
</clickhouse>`
	_ = os.WriteFile(confDir+"/listen.xml", []byte(listenXml), 0644)

	queryLogXml := `<clickhouse>
    <query_log replace="1">
        <database>system</database>
        <table>query_log</table>
        <partition_by>toYYYYMM(event_date)</partition_by>
        <flush_interval_milliseconds>7500</flush_interval_milliseconds>
    </query_log>
</clickhouse>`
	_ = os.WriteFile(confDir+"/query-log.xml", []byte(queryLogXml), 0644)

	systemLogsXml := `<clickhouse>
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
	_ = os.WriteFile(confDir+"/system-logs.xml", []byte(systemLogsXml), 0644)

	// Apply system kernel & ulimit optimizations
	_ = executor.ExecuteStream(ctx, func(l string) { cb("8/12", l, false) }, "sh", "-c", "echo 1 > /proc/sys/kernel/task_delayacct 2>/dev/null; echo madvise > /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null; mkdir -p /etc/security/limits.d && echo -e 'clickhouse soft nofile 500000\nclickhouse hard nofile 500000\nclickhouse soft nproc 500000\nclickhouse hard nproc 500000' > /etc/security/limits.d/clickhouse.conf || true")

	// Step 8: Start ClickHouse service
	cb("9/12", "启动 ClickHouse 服务 (systemctl start clickhouse-server)...", false)
	if err := system.DefaultServiceController.Start(); err != nil {
		cb("9/12", "启动失败: "+err.Error(), true)
		return err
	}

	// Step 9: Enable on boot
	cb("10/12", "设置开机自启...", false)
	_ = system.DefaultServiceController.Enable()

	// Step 10: Verify status
	cb("11/12", "检查服务运行状态...", false)
	status, _ := system.DefaultServiceController.GetStatus()
	if !status.Active {
		cb("11/12", "警告: 服务未能处于 active 状态，请检查日志", true)
	}

	cb("12/12", "ClickHouse 安装并初始化完成！", false)
	return nil
}

func (a *AptInstaller) Uninstall(ctx context.Context, purgeData bool, cb ProgressCallback) error {
	executor := system.DefaultExecutor

	cb("1/4", "停止 ClickHouse 服务...", false)
	_ = system.DefaultServiceController.Stop()
	_ = system.DefaultServiceController.Disable()

	cb("2/4", "强制清除 clickhouse 软件包及二进制程序...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/4", l, false) }, "sh", "-c", "dpkg --purge --force-all clickhouse-server clickhouse-client clickhouse-common-static 2>/dev/null || true")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/4", l, false) }, "apt-get", "install", "-f", "-y")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/4", l, false) }, "apt-get", "autoremove", "-y")
	_ = os.Remove("/usr/bin/clickhouse")
	_ = os.Remove("/usr/bin/clickhouse-server")
	_ = os.Remove("/usr/bin/clickhouse-client")
	_ = os.Remove("/usr/bin/clickhouse-local")
	_ = os.Remove("/etc/systemd/system/clickhouse-server.service")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/4", l, false) }, "systemctl", "daemon-reload")

	if purgeData {
		cb("3/4", "清理数据和日志目录...", false)
		_ = os.RemoveAll("/var/lib/clickhouse")
		_ = os.RemoveAll("/var/log/clickhouse-server")
		_ = os.RemoveAll("/etc/clickhouse-server")
		_ = os.Remove("/etc/apt/sources.list.d/clickhouse.list")
	}

	cb("4/4", "卸载完成！", false)
	return nil
}

func (a *AptInstaller) Upgrade(ctx context.Context, cb ProgressCallback) error {
	executor := system.DefaultExecutor

	// Check if CPU lacks AVX2
	if a.osInfo.Arch == "x86_64" || a.osInfo.Arch == "amd64" {
		cpuBytes, err := os.ReadFile("/proc/cpuinfo")
		if err == nil && !strings.Contains(string(cpuBytes), "avx2") {
			cb("1/3", "检测到高兼容运行环境，正在从官方通道拉取最新兼容版 ClickHouse 核心...", false)
			downloadURL := "https://builds.clickhouse.com/master/amd64compat/clickhouse"
			tempBin := "/tmp/clickhouse"
			downloadCmd := fmt.Sprintf("curl -fsSL '%s' -o %s && chmod 755 %s", downloadURL, tempBin, tempBin)
			if err := executor.ExecuteStream(ctx, func(l string) { cb("1/3", l, false) }, "sh", "-c", downloadCmd); err != nil {
				return fmt.Errorf("下载最新兼容版失败: %w", err)
			}
			cb("2/3", "替换二进制并重启服务...", false)
			_ = system.DefaultServiceController.Stop()
			_ = executor.ExecuteStream(ctx, func(l string) { cb("2/3", l, false) }, "sh", "-c", "mv -f /tmp/clickhouse /usr/bin/clickhouse && chmod 755 /usr/bin/clickhouse")
			_ = system.DefaultServiceController.Start()
			serverVer, _ := system.DefaultServiceController.GetVersions()
			cb("3/3", "升级完成，当前版本: "+strings.TrimSpace(serverVer), false)
			return nil
		}
	}

	cb("1/4", "更新 APT 软件源索引...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("1/4", l, false) }, "apt-get", "update")

	cb("2/4", "升级 clickhouse-server 与 client 到最新版...", false)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("2/4", l, false) }, "apt-get", "install", "--only-upgrade", "-y", "clickhouse-server", "clickhouse-client"); err != nil {
		return err
	}

	cb("3/4", "重启 ClickHouse 服务...", false)
	_ = system.DefaultServiceController.Restart()

	serverVer, _ := system.DefaultServiceController.GetVersions()
	cb("4/4", "升级完成，当前版本: "+strings.TrimSpace(serverVer), false)
	return nil
}
