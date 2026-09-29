package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/security"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type ConnectionSettingsRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Secure   bool   `json:"secure"`
	Protocol string `json:"protocol"` // native or http
}

type SecuritySettingsRequest struct {
	IPWhitelist string `json:"ip_whitelist"`
}

type UnlockIPRequest struct {
	IP string `json:"ip" binding:"required"`
}

func GetConnectionSettings(c *gin.Context) {
	var cfg clickhouse.ConnectionConfig

	// Try reading persisted connection setting from SQLite first
	var setting database.PanelSetting
	if database.DB != nil && database.DB.Where("key = ?", "ch_connection").First(&setting).Error == nil {
		if err := json.Unmarshal([]byte(setting.Value), &cfg); err == nil {
			cfg.Password = "******"
			utils.Success(c, cfg)
			return
		}
	}

	cfg = clickhouse.DefaultService.GetConfig()
	// Mask password for display
	cfg.Password = "******"
	utils.Success(c, cfg)
}

func UpdateConnectionSettings(c *gin.Context) {
	var req ConnectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误", err.Error(), "")
		return
	}

	currentCfg := clickhouse.DefaultService.GetConfig()
	password := req.Password
	if password == "******" {
		password = currentCfg.Password
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "native"
	}

	newCfg := clickhouse.ConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		Password: password,
		Database: req.Database,
		Secure:   req.Secure,
		Protocol: protocol,
	}

	// Test connection with new config
	testSvc := &clickhouse.ClickHouseService{}
	testSvc.SetConfig(newCfg)

	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := testSvc.GetConn(testCtx)
	if err != nil {
		audit.RecordFailed(c, "TEST_CH_CONNECTION", req.Host, err)
		utils.ClickHouseError(c, "连接测试未通过", err, "请检查连接地址、端口及账号密码是否正确")
		return
	}
	defer conn.Close()

	version, _ := testSvc.GetVersion(testCtx)

	// Persist to panel SQLite database
	if database.DB != nil {
		jsonBytes, err := json.Marshal(newCfg)
		if err == nil {
			setting := database.PanelSetting{
				Key:       "ch_connection",
				Value:     string(jsonBytes),
				UpdatedAt: time.Now(),
			}
			if err := database.DB.Save(&setting).Error; err != nil {
				utils.InternalError(c, "保存连接配置失败", err, "")
				return
			}
		}
	}
	clickhouse.DefaultService.SetConfig(newCfg)

	audit.RecordSuccess(c, "UPDATE_CH_CONNECTION", req.Host)
	utils.SuccessWithMessage(c, "ClickHouse 连接配置更新成功，已成功建立握手", gin.H{
		"version": version,
	})
}

func GetSecuritySettings(c *gin.Context) {
	var setting database.PanelSetting
	whitelist := ""
	if database.DB != nil {
		if err := database.DB.Where("key = ?", "ip_whitelist").First(&setting).Error; err == nil {
			whitelist = setting.Value
		}
	}
	utils.Success(c, gin.H{
		"ip_whitelist": whitelist,
		"client_ip":    c.ClientIP(),
		"locked_ips":   security.DefaultLoginLimiter.GetLockedIPs(),
	})
}

func UpdateSecuritySettings(c *gin.Context) {
	var req SecuritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误", err.Error(), "")
		return
	}

	clientIP := c.ClientIP()
	if err := security.ValidateIPWhitelist(req.IPWhitelist, clientIP); err != nil {
		utils.BadRequest(c, "白名单规则校验失败", err.Error(), "请确保输入的每个 IP/CIDR 合法，且必须包含当前访问 IP 以免被锁死")
		return
	}

	if database.DB != nil {
		setting := database.PanelSetting{
			Key:       "ip_whitelist",
			Value:     req.IPWhitelist,
			UpdatedAt: time.Now(),
		}
		if err := database.DB.Save(&setting).Error; err != nil {
			utils.InternalError(c, "保存安全设置失败", err, "")
			return
		}
	}

	audit.RecordSuccess(c, "UPDATE_SECURITY_SETTINGS", "ip_whitelist")
	utils.SuccessWithMessage(c, "安全白名单设置已更新生效", nil)
}

func UnlockSecurityIP(c *gin.Context) {
	var req UnlockIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "缺少 IP 参数", err.Error(), "")
		return
	}

	security.DefaultLoginLimiter.UnlockIP(req.IP)
	audit.RecordSuccess(c, "UNLOCK_IP", req.IP)
	utils.SuccessWithMessage(c, fmt.Sprintf("已成功解除客户端 IP [%s] 的锁定状态", req.IP), nil)
}
