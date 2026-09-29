package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type KillQueryRequest struct {
	QueryID      string `json:"query_id" binding:"required"`
	Confirmation string `json:"confirmation" binding:"required"` // KILL
}

type KillMutationRequest struct {
	Database     string `json:"database" binding:"required"`
	Table        string `json:"table" binding:"required"`
	MutationID   string `json:"mutation_id" binding:"required"`
	Confirmation string `json:"confirmation" binding:"required"` // KILL
}

func GetProcesses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetProcesses(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取运行中查询失败", err, "")
		return
	}
	utils.Success(c, list)
}

func KillQuery(c *gin.Context) {
	var req KillQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "KILL" {
		utils.BadRequest(c, "危险操作：中断查询需要确认", "确认不匹配", "请输入 KILL 确认强制终止查询")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.KillQuery(ctx, req.QueryID); err != nil {
		audit.RecordFailed(c, "KILL_QUERY", req.QueryID, err)
		utils.ClickHouseError(c, "终止查询失败", err, "")
		return
	}

	audit.RecordSuccess(c, "KILL_QUERY", req.QueryID)
	utils.SuccessWithMessage(c, "已成功终止指定查询", nil)
}

func GetQueryLog(c *gin.Context) {
	minDurationStr := c.DefaultQuery("min_duration_ms", "500")
	minDuration, _ := strconv.ParseInt(minDurationStr, 10, 64)

	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	offsetStr := c.DefaultQuery("offset", "0")
	offset, _ := strconv.Atoi(offsetStr)

	search := c.Query("search")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	list, total, err := clickhouse.DefaultService.GetQueryLog(ctx, minDuration, limit, offset, search)
	if err != nil {
		utils.ClickHouseError(c, "获取查询日志失败", err, "")
		return
	}

	utils.Success(c, gin.H{
		"items": list,
		"total": total,
	})
}

func GetParts(c *gin.Context) {
	dbName := c.Query("database")
	tableName := c.Query("table")
	activeOnly := c.Query("active_only") != "false"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetParts(ctx, dbName, tableName, activeOnly)
	if err != nil {
		utils.ClickHouseError(c, "获取 Part 分区列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetMerges(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetMerges(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取后台 Merge 列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetMutations(c *gin.Context) {
	dbName := c.Query("database")
	tableName := c.Query("table")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetMutations(ctx, dbName, tableName)
	if err != nil {
		utils.ClickHouseError(c, "获取 Mutation 列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func KillMutation(c *gin.Context) {
	var req KillMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "KILL" {
		utils.BadRequest(c, "危险操作：中断 Mutation 需要确认", "确认不匹配", "请输入 KILL 确认终止")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.KillMutation(ctx, req.Database, req.Table, req.MutationID); err != nil {
		audit.RecordFailed(c, "KILL_MUTATION", fmt.Sprintf("%s.%s (%s)", req.Database, req.Table, req.MutationID), err)
		utils.ClickHouseError(c, "终止 Mutation 失败", err, "")
		return
	}

	audit.RecordSuccess(c, "KILL_MUTATION", fmt.Sprintf("%s.%s (%s)", req.Database, req.Table, req.MutationID))
	utils.SuccessWithMessage(c, "已成功终止指定 Mutation 任务", nil)
}

func GetDisks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetDisks(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取磁盘列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetClusters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetClusters(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取集群列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetReplicas(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetReplicas(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取副本状态失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetDictionaries(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.GetDictionaries(ctx)
	if err != nil {
		utils.ClickHouseError(c, "获取外部字典列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

type ReloadDictionaryRequest struct {
	Database string `json:"database"`
	Name     string `json:"name" binding:"required"`
}

func ReloadDictionary(c *gin.Context) {
	var req ReloadDictionaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "缺少字典名称参数", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.ReloadDictionary(ctx, req.Database, req.Name); err != nil {
		audit.RecordFailed(c, "RELOAD_DICTIONARY", fmt.Sprintf("%s.%s", req.Database, req.Name), err)
		utils.ClickHouseError(c, "重载字典失败", err, "")
		return
	}

	audit.RecordSuccess(c, "RELOAD_DICTIONARY", fmt.Sprintf("%s.%s", req.Database, req.Name))
	utils.SuccessWithMessage(c, fmt.Sprintf("字典 [%s] 重载成功", req.Name), nil)
}
