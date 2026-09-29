package api

import (
	"strconv"

	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

func ListAuditLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("page_size", "20")
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	search := c.Query("search")
	action := c.Query("action")

	query := database.DB.Model(&database.AuditLog{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR target LIKE ? OR error LIKE ?", like, like, like)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	query.Count(&total)

	var logs []database.AuditLog
	query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs)

	utils.Success(c, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
