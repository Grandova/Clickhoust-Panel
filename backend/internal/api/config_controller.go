package api

import (
	"fmt"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/config"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type SaveFormConfigRequest struct {
	Values map[string]string `json:"values" binding:"required"`
}

type SaveFileConfigRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
	Reason  string `json:"reason"`
}

type RollbackConfigRequest struct {
	BackupID uint `json:"backup_id" binding:"required"`
}

type ValidateXMLRequest struct {
	Content string `json:"content" binding:"required"`
}

func GetFormConfig(c *gin.Context) {
	items := config.GetFormConfigs()
	utils.Success(c, items)
}

func SaveFormConfig(c *gin.Context) {
	var req SaveFormConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误", err.Error(), "")
		return
	}

	username, _ := c.Get("username")
	if err := config.SaveFormConfigs(req.Values, username.(string)); err != nil {
		audit.RecordFailed(c, "UPDATE_CONFIG_FORM", "ch_manager_overrides.xml", err)
		utils.InternalError(c, "保存配置失败", err, "")
		return
	}

	audit.RecordSuccess(c, "UPDATE_CONFIG_FORM", "ch_manager_overrides.xml")
	utils.SuccessWithMessage(c, "配置修改已保存！如果修改了需要重启的配置项，请在服务管理中重启或重新加载生效", nil)
}

func ListConfigFiles(c *gin.Context) {
	files, err := config.ListConfigFiles()
	if err != nil {
		utils.InternalError(c, "读取配置文件列表失败", err, "")
		return
	}
	utils.Success(c, files)
}

func ReadConfigFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		utils.BadRequest(c, "缺少 path 参数", "", "请指定需读取的文件相对路径")
		return
	}

	content, err := config.ReadConfigFile(path)
	if err != nil {
		utils.InternalError(c, "读取配置文件失败", err, "")
		return
	}

	utils.Success(c, gin.H{
		"path":    path,
		"content": content,
	})
}

func ValidateXML(c *gin.Context) {
	var req ValidateXMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	result := config.ValidateXMLContent(req.Content)
	utils.Success(c, result)
}

func SaveConfigFile(c *gin.Context) {
	var req SaveFileConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	username, _ := c.Get("username")
	reason := req.Reason
	if reason == "" {
		reason = "Web Editor Manual Save"
	}

	if err := config.SaveConfigFile(req.Path, req.Content, reason, username.(string)); err != nil {
		audit.RecordFailed(c, "SAVE_CONFIG_XML", req.Path, err)
		utils.Fail(c, 400, utils.CodeConfigCheckErr, "保存失败：XML 语法或配置校验错误", err.Error(), "请检查 XML 标签闭合及格式，未通过校验前禁止覆盖配置文件")
		return
	}

	audit.RecordSuccess(c, "SAVE_CONFIG_XML", req.Path)
	utils.SuccessWithMessage(c, "配置文件已安全保存，已自动创建历史备份版本", nil)
}

func ListConfigBackups(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		utils.BadRequest(c, "缺少 path 参数", "", "")
		return
	}

	backups, err := config.ListConfigBackups(path)
	if err != nil {
		utils.InternalError(c, "获取备份历史失败", err, "")
		return
	}
	utils.Success(c, backups)
}

func RollbackConfig(c *gin.Context) {
	var req RollbackConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	username, _ := c.Get("username")
	if err := config.RollbackConfig(req.BackupID, username.(string)); err != nil {
		audit.RecordFailed(c, "ROLLBACK_CONFIG", fmt.Sprintf("backup_%d", req.BackupID), err)
		utils.InternalError(c, "恢复历史版本失败", err, "")
		return
	}

	audit.RecordSuccess(c, "ROLLBACK_CONFIG", fmt.Sprintf("backup_%d", req.BackupID))
	utils.SuccessWithMessage(c, "已成功恢复至指定历史版本", nil)
}
