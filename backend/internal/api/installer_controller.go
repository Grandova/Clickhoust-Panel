package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/installer"
	"clickhouse-manager/internal/system"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin WebSocket for local/dev
	},
}

type InstallRequest struct {
	AdminPassword string `json:"admin_password"`
}

type UninstallRequest struct {
	Confirmation string `json:"confirmation"` // User must enter "UNINSTALL"
	PurgeData    bool   `json:"purge_data"`
}

func GetInstallerInfo(c *gin.Context) {
	osInfo := system.DetectOS()
	inst, err := installer.GetInstaller()
	name := "Unsupported"
	if err == nil {
		name = inst.Name()
	}

	utils.Success(c, gin.H{
		"os_info":       osInfo,
		"installer":     name,
		"installed":     system.DefaultServiceController.IsInstalled(),
		"is_installing": installer.DefaultManager.IsRunning(),
	})
}

func InstallClickHouse(c *gin.Context) {
	if installer.DefaultManager.IsRunning() {
		utils.BadRequest(c, "安装任务已在进行中", "请勿重复触发安装", "可通过实时日志窗口查看安装进度")
		return
	}

	var req InstallRequest
	_ = c.ShouldBindJSON(&req)

	inst, err := installer.GetInstaller()
	if err != nil {
		utils.InternalError(c, "获取对应系统的安装器失败", err, "目前支持 Debian, Ubuntu, CentOS, Rocky Linux, AlmaLinux 等主流发行版")
		return
	}

	installer.DefaultManager.ClearLogs()
	audit.Record(c, "INSTALL_CLICKHOUSE", inst.Name(), "STARTED", nil)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		installer.DefaultManager.Broadcast(fmt.Sprintf("=== 开始安装 ClickHouse [%s] ===", time.Now().Format("2006-01-02 15:04:05")))
		cb := func(step, msg string, isError bool) {
			prefix := "[INFO]"
			if isError {
				prefix = "[ERROR]"
			}
			installer.DefaultManager.Broadcast(fmt.Sprintf("%s %s %s", prefix, step, msg))
		}

		err := inst.Install(ctx, req.AdminPassword, cb)
		if err != nil {
			installer.DefaultManager.Broadcast(fmt.Sprintf("[FATAL] 安装失败: %v", err))
			audit.Record(nil, "INSTALL_CLICKHOUSE", inst.Name(), "FAILED", err)
		} else {
			installer.DefaultManager.Broadcast("=== ClickHouse 安装并启动成功 ===")
			audit.Record(nil, "INSTALL_CLICKHOUSE", inst.Name(), "SUCCESS", nil)
		}
	}()

	utils.SuccessWithMessage(c, "安装任务已在后台启动，请查看实时日志", nil)
}

func UninstallClickHouse(c *gin.Context) {
	var req UninstallRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "UNINSTALL" {
		utils.BadRequest(c, "危险操作：卸载 ClickHouse 需要二次确认", "确认字段不匹配", "请在确认输入框中输入 UNINSTALL 确认卸载")
		return
	}

	inst, err := installer.GetInstaller()
	if err != nil {
		utils.InternalError(c, "不支持当前系统卸载", err, "")
		return
	}

	audit.Record(c, "UNINSTALL_CLICKHOUSE", inst.Name(), "STARTED", nil)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		installer.DefaultManager.Broadcast("=== 开始卸载 ClickHouse ===")
		cb := func(step, msg string, isError bool) {
			installer.DefaultManager.Broadcast(fmt.Sprintf("[%s] %s", step, msg))
		}

		if err := inst.Uninstall(ctx, req.PurgeData, cb); err != nil {
			installer.DefaultManager.Broadcast(fmt.Sprintf("[ERROR] 卸载失败: %v", err))
			audit.Record(nil, "UNINSTALL_CLICKHOUSE", inst.Name(), "FAILED", err)
		} else {
			installer.DefaultManager.Broadcast("=== ClickHouse 卸载完成 ===")
			audit.Record(nil, "UNINSTALL_CLICKHOUSE", inst.Name(), "SUCCESS", nil)
		}
	}()

	utils.SuccessWithMessage(c, "卸载任务已提交后台执行", nil)
}

func UpgradeClickHouse(c *gin.Context) {
	inst, err := installer.GetInstaller()
	if err != nil {
		utils.InternalError(c, "不支持当前系统升级", err, "")
		return
	}

	audit.Record(c, "UPGRADE_CLICKHOUSE", inst.Name(), "STARTED", nil)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		installer.DefaultManager.Broadcast("=== 开始升级 ClickHouse ===")
		cb := func(step, msg string, isError bool) {
			installer.DefaultManager.Broadcast(fmt.Sprintf("[%s] %s", step, msg))
		}

		if err := inst.Upgrade(ctx, cb); err != nil {
			installer.DefaultManager.Broadcast(fmt.Sprintf("[ERROR] 升级失败: %v", err))
			audit.Record(nil, "UPGRADE_CLICKHOUSE", inst.Name(), "FAILED", err)
		} else {
			installer.DefaultManager.Broadcast("=== ClickHouse 升级完成 ===")
			audit.Record(nil, "UPGRADE_CLICKHOUSE", inst.Name(), "SUCCESS", nil)
		}
	}()

	utils.SuccessWithMessage(c, "升级任务已提交后台执行", nil)
}

func StreamInstallLogsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := installer.DefaultManager.Subscribe()
	defer installer.DefaultManager.Unsubscribe(ch)

	// Keepalive ping ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case logLine, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(logLine)); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
