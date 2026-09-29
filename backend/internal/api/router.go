package api

import (
	"time"

	"clickhouse-manager/internal/auth"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/security"
	"clickhouse-manager/internal/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)

	// CORS configuration for local dev and production
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Sec-WebSocket-Protocol", "Upgrade", "Connection"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "time": time.Now().Unix()})
	})

	v1 := r.Group("/api/v1")
	// Enforce IP whitelist at the root API group
	v1.Use(security.IPWhitelistMiddleware())
	{
		// Public
		v1.POST("/auth/login", Login)

		// Protected
		authorized := v1.Group("")
		authorized.Use(auth.Middleware())
		{
			// Auth
			authorized.POST("/auth/change-password", ChangePassword)
			authorized.GET("/auth/me", GetMe)
			authorized.Use(func(c *gin.Context) {
				var user database.AdminUser
				if err := database.DB.First(&user, c.GetUint("userID")).Error; err != nil {
					utils.Unauthorized(c, "用户不存在")
					c.Abort()
					return
				}
				if user.MustChangePassword {
					utils.Forbidden(c, "首次登录必须先修改初始密码")
					c.Abort()
					return
				}
				c.Next()
			})

			// Dashboard
			authorized.GET("/dashboard/summary", GetDashboardSummary)
			authorized.GET("/dashboard/metrics", GetRealtimeMetrics)

			// Service management
			authorized.GET("/service/status", GetServiceStatus)
			authorized.POST("/service/start", StartService)
			authorized.POST("/service/stop", StopService)
			authorized.POST("/service/restart", RestartService)
			authorized.POST("/service/reload", ReloadService)
			authorized.POST("/service/enable", EnableService)
			authorized.POST("/service/disable", DisableService)
			authorized.POST("/service/validate", ValidateServiceConfig)
			authorized.POST("/service/truncate-logs", TruncateSystemLogs)

			// Installer
			authorized.GET("/installer/info", GetInstallerInfo)
			authorized.POST("/installer/install", InstallClickHouse)
			authorized.POST("/installer/uninstall", UninstallClickHouse)
			authorized.POST("/installer/upgrade", UpgradeClickHouse)
			authorized.GET("/installer/logs/ws", StreamInstallLogsWS)

			// Configuration Center
			authorized.GET("/config/form", GetFormConfig)
			authorized.POST("/config/form", SaveFormConfig)
			authorized.GET("/config/files", ListConfigFiles)
			authorized.GET("/config/file", ReadConfigFile)
			authorized.POST("/config/file", SaveConfigFile)
			authorized.POST("/config/validate-xml", ValidateXML)
			authorized.GET("/config/backups", ListConfigBackups)
			authorized.POST("/config/rollback", RollbackConfig)

			// Databases
			authorized.GET("/databases", ListDatabases)
			authorized.POST("/databases", CreateDatabase)
			authorized.DELETE("/databases/:name", DropDatabase)
			authorized.POST("/databases/rename", RenameDatabase)

			// Tables
			authorized.GET("/tables", ListTables)
			authorized.GET("/tables/columns", GetTableColumns)
			authorized.GET("/tables/create-sql", GetShowCreateTable)
			authorized.POST("/tables/visual-create", VisualCreateTable)
			authorized.DELETE("/tables", DropTable)
			authorized.POST("/tables/truncate", TruncateTable)
			authorized.POST("/tables/rename", RenameTable)
			authorized.POST("/tables/optimize", OptimizeTable)
			authorized.POST("/tables/detach", DetachTable)
			authorized.POST("/tables/attach", AttachTable)
			authorized.POST("/tables/columns/add", AddColumn)
			authorized.POST("/tables/columns/drop", DropColumn)
			authorized.POST("/tables/columns/comment", ModifyColumnComment)
			authorized.POST("/tables/import", ImportData)
			authorized.POST("/tables/import-preview", PreviewImportData)

			// Queries
			authorized.POST("/query/execute", ExecuteQuery)
			authorized.POST("/query/browser", BrowseTableData)
			authorized.GET("/query/favorites", ListFavorites)
			authorized.POST("/query/favorites", SaveFavorite)
			authorized.DELETE("/query/favorites/:id", DeleteFavorite)

			// Ops & Monitoring
			authorized.GET("/ops/processes", GetProcesses)
			authorized.POST("/ops/processes/kill", KillQuery)
			authorized.GET("/ops/query-log", GetQueryLog)
			authorized.GET("/ops/parts", GetParts)
			authorized.GET("/ops/merges", GetMerges)
			authorized.GET("/ops/mutations", GetMutations)
			authorized.POST("/ops/mutations/kill", KillMutation)
			authorized.GET("/ops/disks", GetDisks)
			authorized.GET("/ops/clusters", GetClusters)
			authorized.GET("/ops/replicas", GetReplicas)
			authorized.GET("/ops/dictionaries", GetDictionaries)
			authorized.POST("/ops/dictionaries/reload", ReloadDictionary)

			// Users
			authorized.GET("/users", ListUsers)
			authorized.POST("/users", CreateUser)
			authorized.PUT("/users", AlterUser)
			authorized.DELETE("/users/:name", DropUser)
			authorized.POST("/users/grant", GrantPrivileges)
			authorized.POST("/users/revoke", RevokePrivileges)
			authorized.GET("/users/profiles", ListProfiles)

			// Logs
			authorized.GET("/logs", GetLogs)
			authorized.GET("/logs/ws", StreamLogsWS)

			// Backups
			authorized.GET("/backups", ListBackups)
			authorized.POST("/backups", CreateBackup)
			authorized.POST("/backups/restore", RestoreBackup)
			authorized.DELETE("/backups/:id", DeleteBackupRecord)

			// Audit & Settings
			authorized.GET("/audit/logs", ListAuditLogs)
			authorized.GET("/settings/connection", GetConnectionSettings)
			authorized.POST("/settings/connection", UpdateConnectionSettings)
			authorized.GET("/settings/security", GetSecuritySettings)
			authorized.POST("/settings/security", UpdateSecuritySettings)
			authorized.POST("/settings/security/unlock", UnlockSecurityIP)
		}
	}

	return r
}
