package api

import (
	"context"
	"fmt"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type RestoreBackupRequest struct {
	BackupID     string `json:"backup_id" binding:"required"`
	Disk         string `json:"disk"`
	Confirmation string `json:"confirmation" binding:"required"` // RESTORE
}

func ListBackups(c *gin.Context) {
	var records []database.BackupRecord
	if database.DB != nil {
		database.DB.Order("started_at desc").Find(&records)
	}
	utils.Success(c, records)
}

func CreateBackup(c *gin.Context) {
	var req clickhouse.BackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	backupID, err := clickhouse.DefaultService.CreateBackup(ctx, req)
	if err != nil {
		audit.RecordFailed(c, "CREATE_BACKUP", req.Database, err)
		utils.ClickHouseError(c, "创建备份任务失败", err, "")
		return
	}

	audit.RecordSuccess(c, "CREATE_BACKUP", fmt.Sprintf("%s (%s)", req.Database, backupID))
	utils.SuccessWithMessage(c, "备份任务已提交，ClickHouse 正在后台执行原生备份", gin.H{"backup_id": backupID})
}

func RestoreBackup(c *gin.Context) {
	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "RESTORE" {
		utils.BadRequest(c, "危险操作：恢复备份需要确认", "确认不匹配", "请输入 RESTORE 确认恢复操作")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	audit.Record(c, "RESTORE_BACKUP", req.BackupID, "STARTING", nil)

	err := clickhouse.DefaultService.RestoreBackup(ctx, clickhouse.RestoreRequest{
		BackupID: req.BackupID,
		Disk:     req.Disk,
	})
	if err != nil {
		audit.RecordFailed(c, "RESTORE_BACKUP", req.BackupID, err)
		utils.ClickHouseError(c, "恢复备份失败", err, "")
		return
	}

	audit.RecordSuccess(c, "RESTORE_BACKUP", req.BackupID)
	utils.SuccessWithMessage(c, "备份恢复成功", nil)
}

func DeleteBackupRecord(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "缺少 id", "", "")
		return
	}

	if database.DB != nil {
		database.DB.Delete(&database.BackupRecord{}, id)
	}
	utils.SuccessWithMessage(c, "备份记录已移除", nil)
}
