package api

import (
	"context"
	"strings"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type ExecuteQueryRequest struct {
	SQL      string `json:"sql" binding:"required"`
	MaxRows  int    `json:"max_rows"`
	Database string `json:"database"`
}

type SaveFavoriteRequest struct {
	Title    string `json:"title" binding:"required"`
	SQLText  string `json:"sql_text" binding:"required"`
	Database string `json:"database"`
}

func ExecuteQuery(c *gin.Context) {
	var req ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "SQL 语句不能为空", err.Error(), "")
		return
	}

	trimmed := strings.TrimSpace(req.SQL)
	if trimmed == "" {
		utils.BadRequest(c, "SQL 语句不能为空", "", "")
		return
	}
	if req.MaxRows <= 0 {
		req.MaxRows = 1000
	}
	if req.MaxRows > 10000 {
		req.MaxRows = 10000
	}

	// Check dangerous SQL statements and record audit
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "DROP ") || strings.HasPrefix(upper, "TRUNCATE ") || strings.HasPrefix(upper, "ALTER TABLE ") {
		audit.Record(c, "EXECUTE_DANGEROUS_SQL", trimmed, "EXECUTING", nil)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	result, err := clickhouse.DefaultService.QueryWithDatabase(ctx, req.SQL, req.MaxRows, req.Database)
	if err != nil {
		utils.ClickHouseError(c, "SQL 执行失败", err, "请检查 SQL 语法、表名或字段名是否正确")
		return
	}

	utils.Success(c, result)
}

func BrowseTableData(c *gin.Context) {
	var req clickhouse.BrowserDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	res, err := clickhouse.DefaultService.GetData(ctx, req)
	if err != nil {
		utils.ClickHouseError(c, "获取数据失败", err, "请检查筛选条件语法是否符合 ClickHouse SQL 规范")
		return
	}

	utils.Success(c, res)
}

func ListFavorites(c *gin.Context) {
	var list []database.QueryFavorite
	if database.DB != nil {
		database.DB.Order("created_at desc").Find(&list)
	}
	utils.Success(c, list)
}

func SaveFavorite(c *gin.Context) {
	var req SaveFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	username, _ := c.Get("username")
	fav := database.QueryFavorite{
		Title:     req.Title,
		SQLText:   req.SQLText,
		Database:  req.Database,
		CreatedBy: username.(string),
		CreatedAt: time.Now(),
	}

	if database.DB != nil {
		if err := database.DB.Create(&fav).Error; err != nil {
			utils.InternalError(c, "保存收藏失败", err, "")
			return
		}
	}

	utils.SuccessWithMessage(c, "SQL 已保存至收藏夹", fav)
}

func DeleteFavorite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "缺少 id", "", "")
		return
	}

	if database.DB != nil {
		database.DB.Delete(&database.QueryFavorite{}, id)
	}
	utils.SuccessWithMessage(c, "收藏已移除", nil)
}
