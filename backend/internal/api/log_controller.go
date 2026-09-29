package api

import (
	"context"
	"strconv"

	"clickhouse-manager/internal/logs"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func GetLogs(c *gin.Context) {
	logType := c.DefaultQuery("type", "server")
	linesStr := c.DefaultQuery("lines", "500")
	lines, _ := strconv.Atoi(linesStr)
	filter := c.Query("filter")
	level := c.Query("level")

	res, err := logs.ReadRecentLogs(logs.LogQuery{
		Type:   logs.LogType(logType),
		Lines:  lines,
		Filter: filter,
		Level:  level,
	})
	if err != nil {
		utils.InternalError(c, "读取日志文件失败", err, "")
		return
	}

	utils.Success(c, res)
}

func StreamLogsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	logType := c.DefaultQuery("type", "server")
	filePath := logs.GetLogPath(logs.LogType(logType))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read close from client
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	err = logs.TailFileStream(ctx, filePath, func(line string) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(line))
	})
	if err != nil && ctx.Err() == nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[Tail Error] "+err.Error()))
	}
}
