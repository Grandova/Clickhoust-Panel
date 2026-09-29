package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type DropUserRequest struct {
	Confirmation string `json:"confirmation" binding:"required"`
}

func ListUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	users, err := clickhouse.DefaultService.ListUsers(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取 ClickHouse 用户列表失败", err, "")
		return
	}
	utils.Success(c, users)
}

func CreateUser(c *gin.Context) {
	var req clickhouse.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.CreateUser(ctx, req); err != nil {
		audit.RecordFailed(c, "CREATE_CH_USER", req.Username, err)
		utils.ClickHouseError(c, "创建用户失败", err, "")
		return
	}

	audit.RecordSuccess(c, "CREATE_CH_USER", req.Username)
	utils.SuccessWithMessage(c, "用户创建成功", nil)
}

func AlterUser(c *gin.Context) {
	var req clickhouse.AlterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.AlterUser(ctx, req); err != nil {
		audit.RecordFailed(c, "ALTER_CH_USER", req.Username, err)
		utils.ClickHouseError(c, "修改用户失败", err, "")
		return
	}

	// 若修改的用户为面板当前连接账号，自动同步连接密码，避免连接中断
	currentCfg := clickhouse.DefaultService.GetConfig()
	syncedConn := false
	if req.NewPassword != "" && req.Username == currentCfg.User {
		newCfg := currentCfg
		newCfg.Password = req.NewPassword
		clickhouse.DefaultService.SetConfig(newCfg)
		if database.DB != nil {
			settingJSON, _ := json.Marshal(newCfg)
			database.DB.Save(&database.PanelSetting{
				Key:   "ch_connection",
				Value: string(settingJSON),
			})
		}
		syncedConn = true
	}

	audit.RecordSuccess(c, "ALTER_CH_USER", req.Username)
	if syncedConn {
		utils.SuccessWithMessage(c, "用户密码已修改成功，面板连接凭据已自动同步更新", nil)
	} else {
		utils.SuccessWithMessage(c, "用户配置修改成功", nil)
	}
}

func DropUser(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.BadRequest(c, "缺少用户名", "", "")
		return
	}

	var req DropUserRequest
	expected := fmt.Sprintf("DROP %s", name)
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != expected {
		utils.BadRequest(c, "危险操作：删除用户需要二次确认", "确认不匹配", fmt.Sprintf("请输入 '%s' 确认删除操作", expected))
		return
	}

	if name == "default" {
		utils.Forbidden(c, "禁止删除默认 default 用户")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.DropUser(ctx, name); err != nil {
		audit.RecordFailed(c, "DROP_CH_USER", name, err)
		utils.ClickHouseError(c, "删除用户失败", err, "")
		return
	}

	audit.RecordSuccess(c, "DROP_CH_USER", name)
	utils.SuccessWithMessage(c, "用户删除成功", nil)
}

func GrantPrivileges(c *gin.Context) {
	var req clickhouse.GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.GrantPrivileges(ctx, req); err != nil {
		audit.RecordFailed(c, "GRANT_PRIVILEGES", fmt.Sprintf("%s on %s.%s", req.Username, req.Database, req.Table), err)
		utils.ClickHouseError(c, "授权失败", err, "")
		return
	}

	audit.RecordSuccess(c, "GRANT_PRIVILEGES", fmt.Sprintf("%s on %s.%s", req.Username, req.Database, req.Table))
	utils.SuccessWithMessage(c, "授权操作成功", nil)
}

func RevokePrivileges(c *gin.Context) {
	var req clickhouse.GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.RevokePrivileges(ctx, req); err != nil {
		audit.RecordFailed(c, "REVOKE_PRIVILEGES", fmt.Sprintf("%s on %s.%s", req.Username, req.Database, req.Table), err)
		utils.ClickHouseError(c, "撤销权限失败", err, "")
		return
	}

	audit.RecordSuccess(c, "REVOKE_PRIVILEGES", fmt.Sprintf("%s on %s.%s", req.Username, req.Database, req.Table))
	utils.SuccessWithMessage(c, "权限撤销成功", nil)
}

func ListProfiles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	profiles, err := clickhouse.DefaultService.ListProfiles(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取 Profile 列表失败", err, "")
		return
	}
	utils.Success(c, profiles)
}
