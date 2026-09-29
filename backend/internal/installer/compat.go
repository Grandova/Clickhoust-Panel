package installer

import (
	"context"
	"fmt"
	"os"

	"clickhouse-manager/internal/system"
)

// installCompatBinary installs ClickHouse using the official compatibility binary
// which runs on any x86_64 / SSE hardware without AVX2 instruction set restrictions.
func installCompatBinary(ctx context.Context, arch string, adminPassword string, cb ProgressCallback) error {
	executor := system.DefaultExecutor

	// Step 2: Install download tools & purge any broken packages
	cb("2/12", "环境预检并清理旧安装残留 (curl, ca-certificates)...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/12", l, false) }, "sh", "-c", "dpkg --purge --force-all clickhouse-server clickhouse-client clickhouse-common-static 2>/dev/null || true")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("2/12", l, false) }, "sh", "-c", "which curl &>/dev/null || (apt-get update && apt-get install -y curl ca-certificates) || (yum install -y curl ca-certificates) || true")

	// Step 3: Determine download URL
	downloadURL := "https://builds.clickhouse.com/master/amd64compat/clickhouse"
	if arch == "aarch64" || arch == "arm64" {
		downloadURL = "https://builds.clickhouse.com/master/aarch64v80compat/clickhouse"
	}

	cb("3/12", fmt.Sprintf("正在从官方构建源下载 ClickHouse 兼容版程序 (免 AVX2 指令集限制)..."), false)
	tempBin := "/tmp/clickhouse"
	downloadCmd := fmt.Sprintf("curl -fsSL '%s' -o %s && chmod 755 %s", downloadURL, tempBin, tempBin)
	if err := executor.ExecuteStream(ctx, func(l string) { cb("3/12", l, false) }, "sh", "-c", downloadCmd); err != nil {
		cb("3/12", "下载 ClickHouse 兼容二进制失败: "+err.Error(), true)
		return err
	}

	// Step 4: Deploy binary & create symlinks
	cb("4/12", "部署二进制并创建系统软链接 (clickhouse-server, clickhouse-client)...", false)
	setupBinCmd := `
		mv -f /tmp/clickhouse /usr/bin/clickhouse
		chmod 755 /usr/bin/clickhouse
		ln -sf /usr/bin/clickhouse /usr/bin/clickhouse-server
		ln -sf /usr/bin/clickhouse /usr/bin/clickhouse-client
		ln -sf /usr/bin/clickhouse /usr/bin/clickhouse-local
	`
	if err := executor.ExecuteStream(ctx, func(l string) { cb("4/12", l, false) }, "sh", "-c", setupBinCmd); err != nil {
		cb("4/12", "部署二进制失败: "+err.Error(), true)
		return err
	}

	// Step 5: System user & directory structure
	cb("5/12", "配置 clickhouse 系统用户与工作目录...", false)
	setupDirsCmd := `
		id -u clickhouse &>/dev/null || useradd -r -s /bin/false -d /var/lib/clickhouse clickhouse || true
		mkdir -p /etc/clickhouse-server/config.d /etc/clickhouse-server/users.d /var/lib/clickhouse /var/log/clickhouse-server
	`
	_ = executor.ExecuteStream(ctx, func(l string) { cb("5/12", l, false) }, "sh", "-c", setupDirsCmd)

	// Step 6: Initialize configuration files & systemd service
	cb("6/12", "初始化配置文件与 systemd 守护进程...", false)
	// Try native install command non-interactively to generate default configs
	_ = executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", "echo '' | /usr/bin/clickhouse install 2>/dev/null || true")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", "systemctl stop clickhouse-server 2>/dev/null; pkill -9 -f clickhouse 2>/dev/null || true")

	// Ensure systemd service file is configured
	servicePath := "/etc/systemd/system/clickhouse-server.service"
	serviceContent := `[Unit]
Description=ClickHouse Server (database)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=clickhouse
Group=clickhouse
Restart=always
RestartSec=3
ExecStartPre=-/bin/sh -c 'echo madvise > /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null || true'
ExecStartPre=-/bin/sh -c 'echo 1 > /proc/sys/kernel/task_delayacct 2>/dev/null || true'
ExecStart=/usr/bin/clickhouse-server --config-file=/etc/clickhouse-server/config.xml
LimitNOFILE=500000
LimitNPROC=500000
TasksMax=infinity

[Install]
WantedBy=multi-user.target
`
	_ = os.WriteFile(servicePath, []byte(serviceContent), 0644)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("6/12", l, false) }, "sh", "-c", "echo 1 > /proc/sys/kernel/task_delayacct 2>/dev/null; echo madvise > /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null; mkdir -p /etc/security/limits.d && echo -e 'clickhouse soft nofile 500000\nclickhouse hard nofile 500000\nclickhouse soft nproc 500000\nclickhouse hard nproc 500000' > /etc/security/limits.d/clickhouse.conf || true")

	// Ensure default config.xml exists
	configPath := "/etc/clickhouse-server/config.xml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfigXml := `<clickhouse>
    <logger>
        <level>information</level>
        <log>/var/log/clickhouse-server/clickhouse-server.log</log>
        <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
        <size>1000M</size>
        <count>10</count>
    </logger>
    <http_port>8123</http_port>
    <tcp_port>9000</tcp_port>
    <path>/var/lib/clickhouse/</path>
    <tmp_path>/var/lib/clickhouse/tmp/</tmp_path>
    <user_files_path>/var/lib/clickhouse/user_files/</user_files_path>
    <users_config>users.xml</users_config>
    <listen_host>0.0.0.0</listen_host>
</clickhouse>`
		_ = os.WriteFile(configPath, []byte(defaultConfigXml), 0644)
	}

	// Ensure default users.xml exists
	usersPath := "/etc/clickhouse-server/users.xml"
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		defaultUsersXml := `<clickhouse>
    <profiles>
        <default>
            <max_memory_usage>10000000000</max_memory_usage>
            <use_uncompressed_cache>0</use_uncompressed_cache>
            <load_balancing>random</load_balancing>
        </default>
    </profiles>
    <users>
        <default>
            <password></password>
            <networks>
                <ip>::/0</ip>
            </networks>
            <profile>default</profile>
            <quota>default</quota>
            <access_management>1</access_management>
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
		_ = os.WriteFile(usersPath, []byte(defaultUsersXml), 0640)
	}

	// Step 7: Configure password if provided
	userConfigDir := "/etc/clickhouse-server/users.d"
	_ = os.MkdirAll(userConfigDir, 0755)
	_ = os.Remove(userConfigDir + "/default-password.xml")
	if adminPassword != "" {
		cb("7/12", "配置默认管理员密码...", false)
		userXml := fmt.Sprintf(`<clickhouse>
    <users>
        <default>
            <password>%s</password>
        </default>
    </users>
</clickhouse>`, adminPassword)
		_ = os.WriteFile(userConfigDir+"/admin-password.xml", []byte(userXml), 0640)
	} else {
		cb("7/12", "未指定自定义密码，允许无密码本地登录...", false)
		_ = os.Remove(userConfigDir + "/admin-password.xml")
	}

	// Step 8: Configure listen_host
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

	// Ensure correct ownership
	_ = executor.ExecuteStream(ctx, func(l string) { cb("8/12", l, false) }, "sh", "-c", "chown -R clickhouse:clickhouse /etc/clickhouse-server /var/lib/clickhouse /var/log/clickhouse-server")

	// Step 9: Reload systemd & start ClickHouse
	cb("9/12", "重载 systemd 守护进程并启动 ClickHouse...", false)
	_ = executor.ExecuteStream(ctx, func(l string) { cb("9/12", l, false) }, "sh", "-c", "systemctl stop clickhouse-server 2>/dev/null; pkill -9 -f clickhouse 2>/dev/null || true")
	_ = executor.ExecuteStream(ctx, func(l string) { cb("9/12", l, false) }, "systemctl", "daemon-reload")
	if err := system.DefaultServiceController.Start(); err != nil {
		cb("9/12", "启动 ClickHouse 服务失败: "+err.Error(), true)
		return err
	}

	// Step 10: Enable on boot
	cb("10/12", "设置开机自启...", false)
	_ = system.DefaultServiceController.Enable()

	// Step 11: Check status
	cb("11/12", "检查服务运行状态...", false)
	status, _ := system.DefaultServiceController.GetStatus()
	if !status.Active {
		cb("11/12", "警告: 服务未能处于 active 状态，请检查日志", true)
	}

	cb("12/12", "ClickHouse (高兼容版) 安装并初始化完成！", false)
	return nil
}
