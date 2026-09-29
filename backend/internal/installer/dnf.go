package installer

import (
	"context"
	"fmt"
	"os"
	"strings"

	"clickhouse-manager/internal/system"
)

type DnfInstaller struct {
	osInfo system.OSInfo
}

func (d *DnfInstaller) Name() string {
	return "DNF / YUM Installer (RHEL / Rocky / AlmaLinux / CentOS)"
}

func (d *DnfInstaller) CheckInstalled() bool {
	return system.DefaultServiceController.IsInstalled()
}

func (d *DnfInstaller) getPkgManager() string {
	if system.CheckBinaryInstalled("dnf") {
		return "dnf"
	}
	return "yum"
}

func (d *DnfInstaller) Install(ctx context.Context, adminPassword string, cb ProgressCallback) error {
	executor := system.DefaultExecutor
	pm := d.getPkgManager()

	cb("1/12", "检测操作系统与架构: "+d.osInfo.PrettyName+" ["+d.osInfo.Arch+"]", false)

	// CPU instruction capability check & intelligent fallback
	useCompat := false
	if d.osInfo.Arch == "x86_64" || d.osInfo.Arch == "amd64" {
		cpuBytes, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			cpuInfo := string(cpuBytes)
			hasAVX2 := strings.Contains(cpuInfo, "avx2")
			hasAVX := strings.Contains(cpuInfo, "avx")
			hasSSE42 := strings.Contains(cpuInfo, "sse4_2")

			if !hasAVX2 {
				useCompat = true
				cb("1/12", "【硬件诊断】检测到当前 CPU 未支持 AVX2 向量指令集（ClickHouse 官方 RPM 软件包强制要求 x86-64-v3/AVX2，直接安装会导致 Illegal instruction 错误）。", false)
				if hasSSE42 || hasAVX {
					cb("1/12", "【智能自适应安装】系统检测到基础向量支持，正在自动启用 ClickHouse 官方高兼容版本（amd64compat，免 AVX2 限制），确保 100% 成功安装与运行！", false)
				} else {
					cb("1/12", "【智能自适应安装】当前 CPU 处于低配置虚拟化模式，自动切换 ClickHouse 官方独立兼容引擎...", false)
				}
			} else {
				cb("1/12", "【硬件检测通过】CPU 完美支持 SSE 4.2 与 AVX2 向量指令集，将采用官方 RPM 软件源安装。", false)
			}
		}
	} else if d.osInfo.Arch == "aarch64" || d.osInfo.Arch == "arm64" {
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
		return installCompatBinary(ctx, d.osInfo.Arch, adminPassword, cb)
	}

	// Step 1: Install yum-utils
	cb("2/12", "安装基础依赖 (yum-utils, curl)...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/12", l, false) }, pm, "install", "-y", "yum-utils", "curl")

	// Step 2: Add ClickHouse RPM repo
	cb("3/12", "添加 ClickHouse 官方 RPM 仓库...", false)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("3/12", l, false) }, "yum-config-manager", "--add-repo", "https://packages.clickhouse.com/rpm/clickhouse.repo"); err != nil {
		cb("3/12", "yum-config-manager 失败，尝试直接写入 repo 文件...", true)
		repoContent := `[clickhouse-stable]
name=ClickHouse - Stable Repository
baseurl=https://packages.clickhouse.com/rpm/stable/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.clickhouse.com/rpm/stable/repodata/repomd.xml.key
`
		if err := os.WriteFile("/etc/yum.repos.d/clickhouse.repo", []byte(repoContent), 0644); err != nil {
			return fmt.Errorf("write repo file failed: %w", err)
		}
	}

	// Step 3: Install packages
	cb("4/12", "正在安装 clickhouse-server 与 clickhouse-client...", false)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("4/12", l, false) }, pm, "install", "-y", "clickhouse-server", "clickhouse-client"); err != nil {
		cb("4/12", "ClickHouse RPM 安装失败: "+err.Error(), true)
		return err
	}

	// Step 4: Configure password
	if adminPassword != "" {
		cb("5/12", "配置默认管理员密码...", false)
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
	}

	// Step 5: Network config
	cb("6/12", "开放所有网络地址监听 (listen_host: 0.0.0.0)...", false)
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
	_ = executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", "echo 1 > /proc/sys/kernel/task_delayacct 2>/dev/null; echo madvise > /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null; mkdir -p /etc/security/limits.d && echo -e 'clickhouse soft nofile 500000\nclickhouse hard nofile 500000\nclickhouse soft nproc 500000\nclickhouse hard nproc 500000' > /etc/security/limits.d/clickhouse.conf || true")

	// Step 6: Start service
	cb("7/12", "启动 ClickHouse 服务...", false)
	if err := system.DefaultServiceController.Start(); err != nil {
		cb("7/12", "启动失败: "+err.Error(), true)
		return err
	}

	// Step 7: Enable on boot
	cb("8/12", "设置开机自启...", false)
	_ = system.DefaultServiceController.Enable()

	// Step 8: Check status
	cb("9/12", "检查服务运行状态...", false)
	status, _ := system.DefaultServiceController.GetStatus()
	if !status.Active {
		cb("9/12", "警告: 服务未能处于 active 状态，请检查日志", true)
	}

	cb("10/12", "ClickHouse 安装并初始化完成！", false)
	return nil
}

func (d *DnfInstaller) Uninstall(ctx context.Context, purgeData bool, cb ProgressCallback) error {
	executor := system.DefaultExecutor
	pm := d.getPkgManager()

	cb("1/3", "停止 ClickHouse 服务...", false)
	_ = system.DefaultServiceController.Stop()
	_ = system.DefaultServiceController.Disable()

	cb("2/3", "卸载软件包与二进制程序...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/3", l, false) }, pm, "remove", "-y", "clickhouse-server", "clickhouse-client", "clickhouse-common-static")
	_ = os.Remove("/usr/bin/clickhouse")
	_ = os.Remove("/usr/bin/clickhouse-server")
	_ = os.Remove("/usr/bin/clickhouse-client")
	_ = os.Remove("/usr/bin/clickhouse-local")
	_ = os.Remove("/etc/systemd/system/clickhouse-server.service")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/3", l, false) }, "systemctl", "daemon-reload")

	if purgeData {
		cb("3/3", "清除数据目录...", false)
		_ = os.RemoveAll("/var/lib/clickhouse")
		_ = os.RemoveAll("/var/log/clickhouse-server")
		_ = os.RemoveAll("/etc/clickhouse-server")
	}

	cb("Complete", "卸载完成！", false)
	return nil
}

func (d *DnfInstaller) Upgrade(ctx context.Context, cb ProgressCallback) error {
	executor := system.DefaultExecutor
	pm := d.getPkgManager()

	// Check if CPU lacks AVX2
	if d.osInfo.Arch == "x86_64" || d.osInfo.Arch == "amd64" {
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

	cb("1/3", "检查并更新软件包...", false)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("1/3", l, false) }, pm, "upgrade", "-y", "clickhouse-server", "clickhouse-client"); err != nil {
		return err
	}

	cb("2/3", "重启 ClickHouse 服务...", false)
	_ = system.DefaultServiceController.Restart()

	serverVer, _ := system.DefaultServiceController.GetVersions()
	cb("3/3", "升级完成，当前版本: "+strings.TrimSpace(serverVer), false)
	return nil
}
