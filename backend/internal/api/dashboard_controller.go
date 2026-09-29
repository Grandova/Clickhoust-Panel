package api

import (
	"context"
	"time"

	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/system"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

func GetDashboardSummary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. Host Metrics
	hostMetrics := system.DefaultCollector.Collect()

	// 2. Service Status
	serviceStatus, _ := system.DefaultServiceController.GetStatus()

	// 3. ClickHouse DB & Metrics Summary
	chVersion, chErr := clickhouse.DefaultService.GetVersion(ctx)
	chReachable := (chErr == nil)

	var chMetrics *clickhouse.MetricsSummary
	if chReachable {
		chMetrics, _ = clickhouse.DefaultService.GetMetrics(ctx)
	}

	utils.Success(c, gin.H{
		"host":         hostMetrics,
		"service":      serviceStatus,
		"ch_reachable": chReachable,
		"ch_version":   chVersion,
		"metrics":      chMetrics,
		"os_info":      system.DetectOS(),
	})
}

func GetRealtimeMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	host := system.DefaultCollector.Collect()
	chMetrics, _ := clickhouse.DefaultService.GetMetrics(ctx)

	utils.Success(c, gin.H{
		"timestamp": time.Now().Unix(),
		"host":      host,
		"ch":        chMetrics,
	})
}
