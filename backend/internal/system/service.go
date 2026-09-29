package system

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ServiceStatus struct {
	Installed     bool   `json:"installed"`
	ServerVersion string `json:"server_version"`
	ClientVersion string `json:"client_version"`
	Active        bool   `json:"active"`
	SubState      string `json:"sub_state"` // running, dead, failed
	Enabled       bool   `json:"enabled"`   // enabled on boot
	PID           int    `json:"pid"`
	Uptime        string `json:"uptime"`
	UptimeSec     int64  `json:"uptime_sec"`
	MemoryBytes   int64  `json:"memory_bytes"`
	StatusText    string `json:"status_text"`
}

type ServiceController struct {
	serviceName string
	executor    *CommandExecutor
}

var DefaultServiceController = &ServiceController{
	serviceName: "clickhouse-server",
	executor:    DefaultExecutor,
}

func (s *ServiceController) IsInstalled() bool {
	if !IsLinux() {
		// On non-Linux check clickhouse binary or fallback
		return CheckBinaryInstalled("clickhouse-server") || CheckBinaryInstalled("clickhouse")
	}
	return CheckBinaryInstalled("clickhouse-server") || CheckBinaryInstalled("clickhouse")
}

func (s *ServiceController) GetVersions() (serverVer, clientVer string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if out, err := s.executor.Execute(ctx, "clickhouse-server", "--version"); err == nil {
		serverVer = strings.TrimSpace(out)
	} else if out, err := s.executor.Execute(ctx, "clickhouse", "server", "--version"); err == nil {
		serverVer = strings.TrimSpace(out)
	}

	if out, err := s.executor.Execute(ctx, "clickhouse-client", "--version"); err == nil {
		clientVer = strings.TrimSpace(out)
	} else if out, err := s.executor.Execute(ctx, "clickhouse", "client", "--version"); err == nil {
		clientVer = strings.TrimSpace(out)
	}

	return serverVer, clientVer
}

func (s *ServiceController) GetStatus() (ServiceStatus, error) {
	status := ServiceStatus{
		Installed: s.IsInstalled(),
	}

	status.ServerVersion, status.ClientVersion = s.GetVersions()

	if !IsLinux() {
		status.SubState = "unsupported"
		status.StatusText = "Unsupported: systemd service management requires Linux"
		return status, nil
	}

	if !status.Installed {
		status.StatusText = "ClickHouse is not installed on this system"
		return status, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check active
	if out, err := s.executor.Execute(ctx, "systemctl", "is-active", s.serviceName); err == nil {
		status.SubState = strings.TrimSpace(out)
		status.Active = (status.SubState == "active")
	} else {
		status.SubState = strings.TrimSpace(out)
		status.Active = false
	}

	// Check enabled
	if out, err := s.executor.Execute(ctx, "systemctl", "is-enabled", s.serviceName); err == nil {
		status.Enabled = strings.TrimSpace(out) == "enabled"
	}

	// Get full status text for details
	if out, _ := s.executor.Execute(ctx, "systemctl", "status", s.serviceName); out != "" {
		status.StatusText = out

		// Parse Main PID
		rePID := regexp.MustCompile(`Main PID:\s*(\d+)`)
		if matches := rePID.FindStringSubmatch(out); len(matches) > 1 {
			status.PID, _ = strconv.Atoi(matches[1])
		}
	}

	// Get exact uptime and memory via systemctl show
	if showOut, err := s.executor.Execute(ctx, "systemctl", "show", s.serviceName, "--property=MainPID,ActiveEnterTimestamp,MemoryCurrent"); err == nil {
		lines := strings.Split(showOut, "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				switch k {
				case "MainPID":
					if status.PID == 0 {
						status.PID, _ = strconv.Atoi(v)
					}
				case "MemoryCurrent":
					if v != "[not set]" {
						status.MemoryBytes, _ = strconv.ParseInt(v, 10, 64)
					}
				case "ActiveEnterTimestamp":
					if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", v); err == nil {
						uptime := time.Since(t)
						status.UptimeSec = int64(uptime.Seconds())
						status.Uptime = formatUptime(uptime)
					}
				}
			}
		}
	}

	return status, nil
}

func (s *ServiceController) Start() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "start", s.serviceName)
	if err != nil {
		return fmt.Errorf("start failed: %s (%w)", out, err)
	}
	return nil
}

func (s *ServiceController) Stop() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "stop", s.serviceName)
	if err != nil {
		return fmt.Errorf("stop failed: %s (%w)", out, err)
	}
	return nil
}

func (s *ServiceController) Restart() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "restart", s.serviceName)
	if err != nil {
		return fmt.Errorf("restart failed: %s (%w)", out, err)
	}
	return nil
}

func (s *ServiceController) Reload() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "reload", s.serviceName)
	if err != nil {
		return fmt.Errorf("reload failed: %s (%w)", out, err)
	}
	return nil
}

func (s *ServiceController) Enable() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "enable", s.serviceName)
	if err != nil {
		return fmt.Errorf("enable failed: %s (%w)", out, err)
	}
	return nil
}

func (s *ServiceController) Disable() error {
	if !IsLinux() {
		return fmt.Errorf("Unsupported: systemd service management requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := s.executor.Execute(ctx, "systemctl", "disable", s.serviceName)
	if err != nil {
		return fmt.Errorf("disable failed: %s (%w)", out, err)
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (s *ServiceController) ValidateConfig() (string, error) {
	configDir := "/etc/clickhouse-server"
	if !IsLinux() || !fileExists(configDir) {
		configDir = "data/clickhouse_config"
	}

	// 1. Deep XML syntax check on every .xml file in configDir
	var xmlErrors []string
	var checkedFiles []string
	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".xml") {
			return nil
		}
		if strings.Contains(info.Name(), ".bak.") || strings.HasSuffix(info.Name(), "~") {
			return nil
		}
		rel, _ := filepath.Rel(configDir, path)
		checkedFiles = append(checkedFiles, filepath.ToSlash(rel))

		content, rErr := os.ReadFile(path)
		if rErr != nil {
			xmlErrors = append(xmlErrors, fmt.Sprintf("[%s] 读取失败: %v", rel, rErr))
			return nil
		}
		dec := xml.NewDecoder(bytes.NewReader(content))
		depth, roots := 0, 0
		for {
			token, tErr := dec.Token()
			if tErr != nil {
				if tErr == io.EOF {
					break
				}
				line, _ := dec.InputPos()
				xmlErrors = append(xmlErrors, fmt.Sprintf("[%s] 第 %d 行 XML 语法错误: %v", rel, line, tErr))
				break
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
					xmlErrors = append(xmlErrors, fmt.Sprintf("[%s] 根元素外不能包含文本", rel))
				}
			}
		}
		if roots != 1 {
			xmlErrors = append(xmlErrors, fmt.Sprintf("[%s] XML 必须包含且仅包含一个根元素", rel))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(checkedFiles) == 0 {
		return "", fmt.Errorf("未找到可校验的 XML 配置文件")
	}

	if len(xmlErrors) > 0 {
		return "", fmt.Errorf("XML 配置文件语法校验未通过:\n%s", strings.Join(xmlErrors, "\n"))
	}

	// 2. If clickhouse or clickhouse-extract-from-config binary exists, test merging via extract-from-config
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mainConfig := filepath.Join(configDir, "config.xml")
	var toolCmd string
	var toolArgs []string

	if p, err := exec.LookPath("clickhouse-extract-from-config"); err == nil {
		toolCmd = p
		toolArgs = []string{"--config-file=" + mainConfig, "--key=path"}
	} else if p, err := exec.LookPath("clickhouse"); err == nil {
		toolCmd = p
		toolArgs = []string{"extract-from-config", "--config-file=" + mainConfig, "--key=path"}
	} else if p, err := exec.LookPath("clickhouse-server"); err == nil {
		toolCmd = p
		toolArgs = []string{"extract-from-config", "--config-file=" + mainConfig, "--key=path"}
	}

	var chOutput string
	if !fileExists(mainConfig) {
		return "", fmt.Errorf("未找到主配置文件: %s", mainConfig)
	}
	if toolCmd == "" {
		return "", fmt.Errorf("Unsupported: XML 语法检查通过，但未找到 ClickHouse 配置校验程序")
	}
	if toolCmd != "" {
		out, err := s.executor.Execute(ctx, toolCmd, toolArgs...)
		if err != nil {
			return "", fmt.Errorf("ClickHouse 预处理器解析配置失败: %s (%w)", strings.TrimSpace(out), err)
		}
		chOutput = strings.TrimSpace(out)
	}

	msg := fmt.Sprintf("✓ 成功校验 %d 个 XML 配置文件，XML 闭合与语法结构均合法。\n", len(checkedFiles))
	if chOutput != "" {
		msg += fmt.Sprintf("✓ ClickHouse 配置预处理器 (extract-from-config) 解析通过。\n  数据目录 (path): %s\n  此检查不验证所有运行参数，也不执行配置重载。", chOutput)
	} else {
		msg += "✓ 配置文件语法校验全部通过 (已通过原生 XML 词法结构验证)。"
	}
	return msg, nil
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
