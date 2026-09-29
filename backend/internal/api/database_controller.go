package api

import (
	"context"
	"fmt"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type CreateDatabaseRequest struct {
	Name    string `json:"name" binding:"required"`
	Engine  string `json:"engine"`
	Comment string `json:"comment"`
}

type RenameDatabaseRequest struct {
	OldName string `json:"old_name" binding:"required"`
	NewName string `json:"new_name" binding:"required"`
}

type DropDatabaseRequest struct {
	Confirmation string `json:"confirmation" binding:"required"`
}

func ListDatabases(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.ListDatabases(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取数据库列表失败", err, "请确保 ClickHouse 服务正常运行，并检查连接配置")
		return
	}
	utils.Success(c, list)
}

func CreateDatabase(c *gin.Context) {
	var req CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.CreateDatabase(ctx, req.Name, req.Engine, req.Comment); err != nil {
		audit.RecordFailed(c, "CREATE_DATABASE", req.Name, err)
		utils.ClickHouseError(c, "创建数据库失败", err, "")
		return
	}

	audit.RecordSuccess(c, "CREATE_DATABASE", req.Name)
	utils.SuccessWithMessage(c, "数据库创建成功", nil)
}

func DropDatabase(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.BadRequest(c, "缺少数据库名称", "", "")
		return
	}

	var req DropDatabaseRequest
	expected := fmt.Sprintf("DROP %s", name)
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != expected {
		utils.BadRequest(c, "危险操作：删除数据库需要输入完整确认指令", "确认内容错误", fmt.Sprintf("请输入 '%s' 确认删除操作", expected))
		return
	}

	// Protect system and default databases
	if name == "system" || name == "information_schema" {
		utils.Forbidden(c, "禁止删除系统核心数据库")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.DropDatabase(ctx, name); err != nil {
		audit.RecordFailed(c, "DROP_DATABASE", name, err)
		utils.ClickHouseError(c, "删除数据库失败", err, "")
		return
	}

	audit.RecordSuccess(c, "DROP_DATABASE", name)
	utils.SuccessWithMessage(c, "数据库删除成功", nil)
}

func RenameDatabase(c *gin.Context) {
	var req RenameDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.RenameDatabase(ctx, req.OldName, req.NewName); err != nil {
		audit.RecordFailed(c, "RENAME_DATABASE", req.OldName, err)
		utils.ClickHouseError(c, "重命名数据库失败", err, "")
		return
	}

	audit.RecordSuccess(c, "RENAME_DATABASE", req.OldName+" -> "+req.NewName)
	utils.SuccessWithMessage(c, "数据库重命名成功", nil)
}
