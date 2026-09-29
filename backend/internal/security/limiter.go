package security

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type failRecord struct {
	attempts int
	lockTime time.Time
}

type LoginLimiter struct {
	mu      sync.Mutex
	records map[string]*failRecord
}

var DefaultLoginLimiter = &LoginLimiter{
	records: make(map[string]*failRecord),
}

const (
	MaxFailedAttempts = 5
	LockoutDuration   = 15 * time.Minute
)

// CheckAllowed checks whether an IP is currently locked out
func (l *LoginLimiter) CheckAllowed(ip string) (bool, string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, exists := l.records[ip]
	if !exists {
		return true, ""
	}

	if rec.attempts >= MaxFailedAttempts {
		remaining := time.Until(rec.lockTime.Add(LockoutDuration))
		if remaining > 0 {
			mins := int(remaining.Minutes()) + 1
			return false, fmt.Sprintf("登录失败次数过多（超过 %d 次），该 IP 已被安全锁定，请在 %d 分钟后再试", MaxFailedAttempts, mins)
		}
		// Lock expired, reset
		delete(l.records, ip)
	}

	return true, ""
}

// RecordFailure increments failed login attempts
func (l *LoginLimiter) RecordFailure(ip string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, exists := l.records[ip]
	if !exists {
		rec = &failRecord{attempts: 0}
		l.records[ip] = rec
	}

	rec.attempts++
	rec.lockTime = time.Now()
	return rec.attempts
}

// ResetFailure clears failures upon successful login
func (l *LoginLimiter) ResetFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, ip)
}

// GetLockedIPs returns all currently locked IP addresses and remaining minutes
func (l *LoginLimiter) GetLockedIPs() []map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	var result []map[string]interface{}
	now := time.Now()
	for ip, rec := range l.records {
		if rec.attempts >= MaxFailedAttempts {
			remaining := rec.lockTime.Add(LockoutDuration).Sub(now)
			if remaining > 0 {
				result = append(result, map[string]interface{}{
					"ip":              ip,
					"failed_attempts": rec.attempts,
					"locked_at":       rec.lockTime.Format("2006-01-02 15:04:05"),
					"remaining_mins":  int(remaining.Minutes()) + 1,
				})
			}
		}
	}
	return result
}

// UnlockIP manually unlocks an IP address
func (l *LoginLimiter) UnlockIP(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, ip)
}

// ValidateIPWhitelist validates syntax of IP whitelist and ensures current admin IP is not locked out
func ValidateIPWhitelist(whitelistStr, clientIP string) error {
	trimmed := strings.TrimSpace(whitelistStr)
	if trimmed == "" || trimmed == "*" {
		return nil // Whitelist disabled, allowed
	}

	rawItems := strings.Split(trimmed, ",")
	var parsedNets []*net.IPNet
	var parsedIPs []net.IP

	for _, raw := range rawItems {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			_, ipNet, err := net.ParseCIDR(item)
			if err != nil {
				return fmt.Errorf("无效的 CIDR 网段格式: '%s' (%v)", item, err)
			}
			parsedNets = append(parsedNets, ipNet)
		} else {
			ip := net.ParseIP(item)
			if ip == nil {
				return fmt.Errorf("无效的 IP 地址: '%s'", item)
			}
			parsedIPs = append(parsedIPs, ip)
		}
	}

	if len(parsedNets) == 0 && len(parsedIPs) == 0 {
		return nil
	}

	currIP := net.ParseIP(clientIP)
	if currIP == nil {
		return nil // Unknown/mock client IP
	}

	matched := false
	for _, ip := range parsedIPs {
		if ip.Equal(currIP) {
			matched = true
			break
		}
	}
	if !matched {
		for _, ipNet := range parsedNets {
			if ipNet.Contains(currIP) {
				matched = true
				break
			}
		}
	}

	if !matched {
		return fmt.Errorf("安全保护拦截：白名单配置中未包含您当前的客户端 IP [%s]，保存该配置将导致您立即失去面板访问权限！请将当前 IP 或网段加入列表后再保存", clientIP)
	}

	return nil
}

// CheckIPWhitelist verifies if incoming client IP is authorized
func CheckIPWhitelist(clientIP string) bool {
	if database.DB == nil {
		return true
	}

	var setting database.PanelSetting
	if err := database.DB.Where("key = ?", "ip_whitelist").First(&setting).Error; err != nil || setting.Value == "" {
		return true // Whitelist not configured, all allowed
	}

	trimmedList := strings.TrimSpace(setting.Value)
	if trimmedList == "" || trimmedList == "*" {
		return true
	}

	currIP := net.ParseIP(clientIP)

	allowedIPs := strings.Split(trimmedList, ",")
	for _, raw := range allowedIPs {
		allowed := strings.TrimSpace(raw)
		if allowed == "" {
			continue
		}
		if allowed == clientIP {
			return true
		}
		if currIP != nil {
			if ip := net.ParseIP(allowed); ip != nil && ip.Equal(currIP) {
				return true
			}
			if strings.Contains(allowed, "/") {
				_, ipNet, err := net.ParseCIDR(allowed)
				if err == nil && ipNet.Contains(currIP) {
					return true
				}
			}
		}
	}

	return false
}

// IPWhitelistMiddleware creates a Gin middleware that enforces the configured IP whitelist
func IPWhitelistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CheckIPWhitelist(c.ClientIP()) {
			utils.Forbidden(c, "您的客户端 IP 不在面板允许的访问白名单中")
			c.Abort()
			return
		}
		c.Next()
	}
}
