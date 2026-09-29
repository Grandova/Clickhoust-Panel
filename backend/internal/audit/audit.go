package audit

import (
	"time"

	"clickhouse-manager/internal/database"

	"github.com/gin-gonic/gin"
)

func Record(c *gin.Context, action string, target string, result string, err error) {
	username := "system"
	ip := "127.0.0.1"

	if c != nil {
		if val, exists := c.Get("username"); exists {
			username = val.(string)
		}
		ip = c.ClientIP()
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	entry := database.AuditLog{
		Username:  username,
		IP:        ip,
		Action:    action,
		Target:    target,
		Result:    result,
		Error:     errStr,
		CreatedAt: time.Now(),
	}

	if database.DB != nil {
		database.DB.Create(&entry)
	}
}

func RecordSuccess(c *gin.Context, action string, target string) {
	Record(c, action, target, "SUCCESS", nil)
}

func RecordFailed(c *gin.Context, action string, target string, err error) {
	Record(c, action, target, "FAILED", err)
}
