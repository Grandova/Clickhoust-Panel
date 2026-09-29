package installer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"clickhouse-manager/internal/system"
)

type ProgressCallback func(step string, message string, isError bool)

type Installer interface {
	Name() string
	Install(ctx context.Context, adminPassword string, cb ProgressCallback) error
	Uninstall(ctx context.Context, purgeData bool, cb ProgressCallback) error
	Upgrade(ctx context.Context, cb ProgressCallback) error
	CheckInstalled() bool
}

type InstallManager struct {
	mu           sync.Mutex
	isInstalling bool
	lastLogLines []string
	subscribers  map[chan string]bool
}

var DefaultManager = &InstallManager{
	subscribers: make(map[chan string]bool),
}

func (m *InstallManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isInstalling
}

func (m *InstallManager) Subscribe() chan string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan string, 100)
	m.subscribers[ch] = true
	// send cached logs
	for _, l := range m.lastLogLines {
		ch <- l
	}
	return ch
}

func (m *InstallManager) Unsubscribe(ch chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscribers, ch)
	close(ch)
}

func (m *InstallManager) Broadcast(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastLogLines = append(m.lastLogLines, line)
	if len(m.lastLogLines) > 1000 {
		m.lastLogLines = m.lastLogLines[len(m.lastLogLines)-1000:]
	}
	for ch := range m.subscribers {
		select {
		case ch <- line:
		default:
		}
	}
}

func (m *InstallManager) ClearLogs() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastLogLines = nil
}

func GetInstaller() (Installer, error) {
	osInfo := system.DetectOS()
	switch osInfo.PkgType {
	case "deb":
		return &AptInstaller{osInfo: osInfo}, nil
	case "rpm":
		return &DnfInstaller{osInfo: osInfo}, nil
	default:
		if !system.IsLinux() {
			return &DevMockInstaller{osInfo: osInfo}, nil
		}
		return nil, fmt.Errorf("unsupported OS package manager for platform: %s", osInfo.Platform)
	}
}

type DevMockInstaller struct {
	osInfo system.OSInfo
}

func (d *DevMockInstaller) Name() string {
	return "Dev Mock Installer (Non-Linux)"
}

func (d *DevMockInstaller) CheckInstalled() bool {
	return system.DefaultServiceController.IsInstalled()
}

func (d *DevMockInstaller) Install(ctx context.Context, adminPassword string, cb ProgressCallback) error {
	steps := []string{
		"检测操作系统... " + d.osInfo.PrettyName,
		"检测 CPU 架构... " + d.osInfo.Arch,
		"模拟添加 ClickHouse Repository...",
		"模拟安装 clickhouse-server...",
		"模拟安装 clickhouse-client...",
		"创建基础配置...",
		"初始化管理员账号 (default)...",
		"启动 ClickHouse 服务...",
		"设置开机自启...",
		"测试数据库连接...",
		"ClickHouse 运行成功！",
	}

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(600 * time.Millisecond):
			cb(fmt.Sprintf("[%d/%d]", i+1, len(steps)), step, false)
		}
	}
	return nil
}

func (d *DevMockInstaller) Uninstall(ctx context.Context, purgeData bool, cb ProgressCallback) error {
	cb("1/3", "停止 ClickHouse 服务...", false)
	time.Sleep(500 * time.Millisecond)
	cb("2/3", "卸载 clickhouse-server 及相关组件...", false)
	time.Sleep(500 * time.Millisecond)
	if purgeData {
		cb("3/3", "清除数据及配置文件 /var/lib/clickhouse...", false)
	} else {
		cb("3/3", "保留现有数据目录...", false)
	}
	time.Sleep(500 * time.Millisecond)
	cb("Complete", "ClickHouse 卸载完成", false)
	return nil
}

func (d *DevMockInstaller) Upgrade(ctx context.Context, cb ProgressCallback) error {
	cb("1/3", "检查 ClickHouse 软件源更新...", false)
	time.Sleep(600 * time.Millisecond)
	cb("2/3", "升级 clickhouse-server 到最新稳定版...", false)
	time.Sleep(600 * time.Millisecond)
	cb("3/3", "重启 ClickHouse 服务验证升级...", false)
	time.Sleep(600 * time.Millisecond)
	cb("Complete", "ClickHouse 升级成功", false)
	return nil
}
