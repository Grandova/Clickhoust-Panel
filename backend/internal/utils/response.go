package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code        int         `json:"code"`
	Message     string      `json:"message"`
	Data        interface{} `json:"data,omitempty"`
	ErrorDetail string      `json:"error_detail,omitempty"`
	Suggestion  string      `json:"suggestion,omitempty"`
}

const (
	CodeSuccess        = 0
	CodeBadRequest     = 400
	CodeUnauthorized   = 401
	CodeForbidden      = 403
	CodeNotFound       = 404
	CodeInternalError  = 500
	CodeClickHouseErr  = 5001
	CodeSystemdErr     = 5002
	CodeInstallErr     = 5003
	CodeConfigCheckErr = 5004
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "Success",
		Data:    data,
	})
}

func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

func Fail(c *gin.Context, httpStatus int, code int, message string, detail string, suggestion string) {
	c.JSON(httpStatus, Response{
		Code:        code,
		Message:     message,
		ErrorDetail: detail,
		Suggestion:  suggestion,
	})
}

func BadRequest(c *gin.Context, message string, detail string, suggestion string) {
	if suggestion == "" {
		suggestion = "请检查请求参数格式和必填字段是否正确"
	}
	Fail(c, http.StatusBadRequest, CodeBadRequest, message, detail, suggestion)
}

func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, message, "Token 无效或已过期", "请重新登录获取最新有效凭证")
}

func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, CodeForbidden, message, "权限不足", "请使用更高权限管理员账号操作")
}

func InternalError(c *gin.Context, message string, err error, suggestion string) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	if suggestion == "" {
		suggestion = "请查看服务日志或联系系统管理员"
	}
	Fail(c, http.StatusInternalServerError, CodeInternalError, message, detail, suggestion)
}

func ClickHouseError(c *gin.Context, message string, err error, suggestion string) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	if strings.Contains(detail, "Not enough privileges") || strings.Contains(detail, "ACCESS_DENIED") {
		suggestion = "当前 ClickHouse 连接账号权限不足。请在连接与设置中使用有对应 GRANT OPTION 的管理账号；按数据库授予需要的权限，避免使用 ALL。具体缺少的权限见错误详情。"
	} else if strings.Contains(detail, "ACCESS_STORAGE_READONLY") || strings.Contains(detail, "storage is readonly") {
		suggestion = "该用户由 ClickHouse 配置文件管理，无法通过 SQL 修改。请在服务器用户配置中修改，或创建由 SQL 管理的新用户。"
	} else if suggestion == "" {
		suggestion = "请检查 ClickHouse 服务是否正常运行、网络端口是否畅通以及账号权限"
	}
	Fail(c, http.StatusOK, CodeClickHouseErr, message, detail, suggestion)
}
