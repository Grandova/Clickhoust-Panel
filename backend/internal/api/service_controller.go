package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/system"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type ConfirmActionRequest struct {
	Confirmation string `json:"confirmation"` // User must enter "CONFIRM" for dangerous actions
}

func GetServiceStatus(c *gin.Context) {
	status, err := system.DefaultServiceController.GetStatus()
	if err != nil {
		utils.InternalError(c, "获取服务状态失败", err, "")
		return
	}
	utils.Success(c, status)
}

func StartService(c *gin.Context) {
	if err := system.DefaultServiceController.Start(); err != nil {
		audit.RecordFailed(c, "START_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "启动 ClickHouse 服务失败", err, "请检查 ClickHouse 配置文件是否存在错误，或查看系统日志")
		return
	}
	audit.RecordSuccess(c, "START_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "ClickHouse 服务启动成功", nil)
}

func StopService(c *gin.Context) {
	var req ConfirmActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "STOP" {
		utils.BadRequest(c, "危险操作：停止服务需要二次确认", "确认字段不匹配", "请在确认输入框中输入 STOP 确认停止服务")
		return
	}

	if err := system.DefaultServiceController.Stop(); err != nil {
		audit.RecordFailed(c, "STOP_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "停止 ClickHouse 服务失败", err, "")
		return
	}
	audit.RecordSuccess(c, "STOP_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "ClickHouse 服务已停止", nil)
}

func RestartService(c *gin.Context) {
	// First validate config if possible
	if system.IsLinux() {
		if out, err := system.DefaultServiceController.ValidateConfig(); err != nil {
			utils.Fail(c, 400, utils.CodeConfigCheckErr, "配置文件验证未通过，已禁止直接重启 ClickHouse", out, "请在配置中心修复配置文件后再尝试重启")
			return
		}
	}

	if err := system.DefaultServiceController.Restart(); err != nil {
		audit.RecordFailed(c, "RESTART_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "重启 ClickHouse 服务失败", err, "服务可能因配置错误未能拉起，请查看 systemd 日志")
		return
	}
	audit.RecordSuccess(c, "RESTART_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "ClickHouse 服务重启成功", nil)
}

func ReloadService(c *gin.Context) {
	if err := system.DefaultServiceController.Reload(); err != nil {
		audit.RecordFailed(c, "RELOAD_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "热重载 ClickHouse 配置失败", err, "")
		return
	}
	audit.RecordSuccess(c, "RELOAD_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "ClickHouse 配置热重载成功", nil)
}

func EnableService(c *gin.Context) {
	if err := system.DefaultServiceController.Enable(); err != nil {
		audit.RecordFailed(c, "ENABLE_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "设置开机自启失败", err, "")
		return
	}
	audit.RecordSuccess(c, "ENABLE_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "已成功设置 ClickHouse 开机自启", nil)
}

func DisableService(c *gin.Context) {
	if err := system.DefaultServiceController.Disable(); err != nil {
		audit.RecordFailed(c, "DISABLE_SERVICE", "clickhouse-server", err)
		utils.InternalError(c, "取消开机自启失败", err, "")
		return
	}
	audit.RecordSuccess(c, "DISABLE_SERVICE", "clickhouse-server")
	utils.SuccessWithMessage(c, "已取消 ClickHouse 开机自启", nil)
}

func ValidateServiceConfig(c *gin.Context) {
	out, err := system.DefaultServiceController.ValidateConfig()
	if err != nil {
		utils.Fail(c, 200, utils.CodeConfigCheckErr, "配置校验存在错误或警告", fmt.Sprintf("%v", err), out)
		return
	}
	utils.SuccessWithMessage(c, "配置文件语法与有效性校验通过", gin.H{"output": out})
}

func TruncateSystemLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	tables, err := clickhouse.DefaultService.TruncateSystemLogs(ctx)
	if err != nil {
		audit.RecordFailed(c, "TRUNCATE_SYSTEM_LOGS", "system", err)
		utils.ClickHouseError(c, "清理系统日志表失败", err, "")
		return
	}
	audit.RecordSuccess(c, "TRUNCATE_SYSTEM_LOGS", strings.Join(tables, ", "))
	utils.SuccessWithMessage(c, fmt.Sprintf("已成功清空 %d 个内置系统日志/指标表数据，已释放磁盘空间", len(tables)), gin.H{
		"truncated_tables": tables,
	})
}
