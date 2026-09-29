package api

import (
	"fmt"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/auth"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/security"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func Login(c *gin.Context) {
	clientIP := c.ClientIP()

	// 1. IP Whitelist check
	if !security.CheckIPWhitelist(clientIP) {
		audit.RecordFailed(c, "LOGIN_BLOCKED", clientIP, nil)
		utils.Forbidden(c, "您的客户端 IP 不在面板允许的访问白名单中")
		return
	}

	// 2. Brute-force rate limit check
	if allowed, reason := security.DefaultLoginLimiter.CheckAllowed(clientIP); !allowed {
		audit.RecordFailed(c, "LOGIN_LOCKED", clientIP, nil)
		utils.Fail(c, 429, 429, "登录已受限", reason, "请等待锁定期结束")
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "用户名和密码不能为空", err.Error(), "")
		return
	}

	var user database.AdminUser
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		attempts := security.DefaultLoginLimiter.RecordFailure(clientIP)
		remaining := security.MaxFailedAttempts - attempts
		if remaining < 0 {
			remaining = 0
		}
		audit.RecordFailed(c, "LOGIN", req.Username, err)
		utils.BadRequest(c, "账号或密码错误", fmt.Sprintf("用户名或密码不匹配 (剩余重试次数: %d)", remaining), "请核对面板管理员用户名及密码")
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		attempts := security.DefaultLoginLimiter.RecordFailure(clientIP)
		remaining := security.MaxFailedAttempts - attempts
		if remaining < 0 {
			remaining = 0
		}
		audit.RecordFailed(c, "LOGIN", req.Username, nil)
		utils.BadRequest(c, "账号或密码错误", fmt.Sprintf("用户名或密码不匹配 (剩余重试次数: %d)", remaining), "请检查密码大小写")
		return
	}

	// Reset failure attempts upon successful login
	security.DefaultLoginLimiter.ResetFailure(clientIP)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		utils.InternalError(c, "生成凭据失败", err, "")
		return
	}

	audit.RecordSuccess(c, "LOGIN", req.Username)

	utils.Success(c, gin.H{
		"token":                token,
		"username":             user.Username,
		"role":                 user.Role,
		"must_change_password": user.MustChangePassword,
	})
}

func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求格式错误", err.Error(), "")
		return
	}

	if len(req.NewPassword) < 8 {
		utils.BadRequest(c, "密码长度过短", "新密码至少需要 8 个字符", "请设置更复杂的安全密码")
		return
	}
	if req.NewPassword == req.OldPassword {
		utils.BadRequest(c, "新密码不能与原密码相同", "", "请设置新的安全密码")
		return
	}

	userID := c.GetUint("userID")
	var user database.AdminUser
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.BadRequest(c, "用户不存在", err.Error(), "")
		return
	}

	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		utils.BadRequest(c, "原密码不正确", "", "请输入正确的原密码")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.InternalError(c, "密码加密失败", err, "")
		return
	}

	user.PasswordHash = newHash
	user.MustChangePassword = false
	if err := database.DB.Save(&user).Error; err != nil {
		utils.InternalError(c, "更新密码失败", err, "")
		return
	}

	audit.RecordSuccess(c, "CHANGE_PASSWORD", user.Username)
	utils.SuccessWithMessage(c, "密码修改成功", nil)
}

func GetMe(c *gin.Context) {
	userID := c.GetUint("userID")
	var user database.AdminUser
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":                   user.ID,
		"username":             user.Username,
		"role":                 user.Role,
		"must_change_password": user.MustChangePassword,
	})
}
