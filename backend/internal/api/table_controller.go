package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickhouse-manager/internal/audit"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/utils"

	"github.com/gin-gonic/gin"
)

type TableActionRequest struct {
	Database     string `json:"database" binding:"required"`
	Table        string `json:"table" binding:"required"`
	Confirmation string `json:"confirmation"`
}

type RenameTableRequest struct {
	Database string `json:"database" binding:"required"`
	OldName  string `json:"old_name" binding:"required"`
	NewName  string `json:"new_name" binding:"required"`
}

type OptimizeTableRequest struct {
	Database  string `json:"database" binding:"required"`
	Table     string `json:"table" binding:"required"`
	Final     bool   `json:"final"`
	Partition string `json:"partition"`
}

func ListTables(c *gin.Context) {
	dbName := c.Query("database")
	if dbName == "" {
		dbName = "default"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	list, err := clickhouse.DefaultService.ListTables(ctx, dbName)
	if err != nil {
		utils.ClickHouseError(c, "获取数据表列表失败", err, "")
		return
	}
	utils.Success(c, list)
}

func GetTableColumns(c *gin.Context) {
	dbName := c.Query("database")
	tableName := c.Query("table")
	if dbName == "" || tableName == "" {
		utils.BadRequest(c, "缺少 database 或 table 参数", "", "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cols, err := clickhouse.DefaultService.GetColumns(ctx, dbName, tableName)
	if err != nil {
		utils.ClickHouseError(c, "获取字段信息失败", err, "")
		return
	}
	utils.Success(c, cols)
}

func GetShowCreateTable(c *gin.Context) {
	dbName := c.Query("database")
	tableName := c.Query("table")
	if dbName == "" || tableName == "" {
		utils.BadRequest(c, "缺少 database 或 table 参数", "", "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	ddl, err := clickhouse.DefaultService.ShowCreateTable(ctx, dbName, tableName)
	if err != nil {
		utils.ClickHouseError(c, "获取建表 SQL 失败", err, "")
		return
	}
	utils.Success(c, gin.H{"sql": ddl})
}

func VisualCreateTable(c *gin.Context) {
	var req clickhouse.VisualCreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误", err.Error(), "")
		return
	}

	req.TableName = strings.TrimSpace(req.TableName)
	req.Database = strings.TrimSpace(req.Database)
	if req.Database == "" {
		utils.BadRequest(c, "数据库名称不能为空", "目标数据库未指定", "请选择有效的目标数据库后重试")
		return
	}
	if req.TableName == "" {
		utils.BadRequest(c, "表名不能为空", "请输入需要创建的数据表名称", "请在表单中输入数据表名称后重试")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	sql, err := clickhouse.DefaultService.VisualCreateTable(ctx, req)
	if err != nil {
		audit.RecordFailed(c, "VISUAL_CREATE_TABLE", fmt.Sprintf("%s.%s", req.Database, req.TableName), err)
		if strings.Contains(err.Error(), "cannot be empty") || strings.Contains(err.Error(), "at least one column") {
			utils.BadRequest(c, "建表参数错误", err.Error(), "请在表单中填写表名并添加至少一个有效字段")
			return
		}
		utils.ClickHouseError(c, "可视化建表失败", err, "")
		return
	}

	if req.ExecuteNow {
		audit.RecordSuccess(c, "VISUAL_CREATE_TABLE", fmt.Sprintf("%s.%s", req.Database, req.TableName))
		utils.SuccessWithMessage(c, "数据表创建成功", gin.H{"sql": sql})
	} else {
		utils.SuccessWithMessage(c, "SQL 生成成功 (预览模式)", gin.H{"sql": sql})
	}
}

func DropTable(c *gin.Context) {
	var req TableActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	expected := fmt.Sprintf("DROP %s", req.Table)
	if req.Confirmation != expected {
		utils.BadRequest(c, "危险操作：删除表需要二次确认", "确认内容错误", fmt.Sprintf("请输入 '%s' 确认删除操作", expected))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.DropTable(ctx, req.Database, req.Table); err != nil {
		audit.RecordFailed(c, "DROP_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "删除数据表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "DROP_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table))
	utils.SuccessWithMessage(c, "数据表删除成功", nil)
}

func TruncateTable(c *gin.Context) {
	var req TableActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	expected := fmt.Sprintf("TRUNCATE %s", req.Table)
	if req.Confirmation != expected {
		utils.BadRequest(c, "危险操作：清空表需要二次确认", "确认内容错误", fmt.Sprintf("请输入 '%s' 确认清空操作", expected))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.TruncateTable(ctx, req.Database, req.Table); err != nil {
		audit.RecordFailed(c, "TRUNCATE_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "清空数据表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "TRUNCATE_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table))
	utils.SuccessWithMessage(c, "数据表已成功清空", nil)
}

func RenameTable(c *gin.Context) {
	var req RenameTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.RenameTable(ctx, req.Database, req.OldName, req.NewName); err != nil {
		audit.RecordFailed(c, "RENAME_TABLE", fmt.Sprintf("%s.%s", req.Database, req.OldName), err)
		utils.ClickHouseError(c, "重命名表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "RENAME_TABLE", fmt.Sprintf("%s.%s -> %s", req.Database, req.OldName, req.NewName))
	utils.SuccessWithMessage(c, "数据表重命名成功", nil)
}

func OptimizeTable(c *gin.Context) {
	var req OptimizeTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.OptimizeTable(ctx, req.Database, req.Table, req.Final, req.Partition); err != nil {
		audit.RecordFailed(c, "OPTIMIZE_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "Optimize 表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "OPTIMIZE_TABLE", fmt.Sprintf("%s.%s (final=%v)", req.Database, req.Table, req.Final))
	utils.SuccessWithMessage(c, "数据表合并优化已触发执行", nil)
}

func DetachTable(c *gin.Context) {
	var req TableActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.DetachTable(ctx, req.Database, req.Table); err != nil {
		audit.RecordFailed(c, "DETACH_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "卸载表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "DETACH_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table))
	utils.SuccessWithMessage(c, "数据表已卸载 (DETACH)", nil)
}

func AttachTable(c *gin.Context) {
	var req TableActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.AttachTable(ctx, req.Database, req.Table); err != nil {
		audit.RecordFailed(c, "ATTACH_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "挂载表失败", err, "")
		return
	}

	audit.RecordSuccess(c, "ATTACH_TABLE", fmt.Sprintf("%s.%s", req.Database, req.Table))
	utils.SuccessWithMessage(c, "数据表已挂载 (ATTACH)", nil)
}

type DropColumnRequest struct {
	Database     string `json:"database" binding:"required"`
	Table        string `json:"table" binding:"required"`
	ColumnName   string `json:"column_name" binding:"required"`
	Confirmation string `json:"confirmation"`
}

type ModifyColumnCommentRequest struct {
	Database   string `json:"database" binding:"required"`
	Table      string `json:"table" binding:"required"`
	ColumnName string `json:"column_name" binding:"required"`
	Comment    string `json:"comment"`
}

type PreviewImportRequest struct {
	Data   string `json:"data" binding:"required"`
	Format string `json:"format"`
}

func AddColumn(c *gin.Context) {
	var req clickhouse.AddColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.AddColumn(ctx, req); err != nil {
		audit.RecordFailed(c, "ADD_COLUMN", fmt.Sprintf("%s.%s.%s", req.Database, req.Table, req.ColumnName), err)
		utils.ClickHouseError(c, "添加字段失败", err, "")
		return
	}

	audit.RecordSuccess(c, "ADD_COLUMN", fmt.Sprintf("%s.%s.%s (%s)", req.Database, req.Table, req.ColumnName, req.Type))
	utils.SuccessWithMessage(c, fmt.Sprintf("字段 [%s] 添加成功", req.ColumnName), nil)
}

func DropColumn(c *gin.Context) {
	var req DropColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	expected := fmt.Sprintf("DROP %s", req.ColumnName)
	if req.Confirmation != expected {
		utils.BadRequest(c, "危险操作：删除字段需要二次确认", "确认内容错误", fmt.Sprintf("请输入 '%s' 确认删除字段", expected))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.DropColumn(ctx, req.Database, req.Table, req.ColumnName); err != nil {
		audit.RecordFailed(c, "DROP_COLUMN", fmt.Sprintf("%s.%s.%s", req.Database, req.Table, req.ColumnName), err)
		utils.ClickHouseError(c, "删除字段失败", err, "")
		return
	}

	audit.RecordSuccess(c, "DROP_COLUMN", fmt.Sprintf("%s.%s.%s", req.Database, req.Table, req.ColumnName))
	utils.SuccessWithMessage(c, fmt.Sprintf("字段 [%s] 已删除", req.ColumnName), nil)
}

func ModifyColumnComment(c *gin.Context) {
	var req ModifyColumnCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := clickhouse.DefaultService.ModifyColumnComment(ctx, req.Database, req.Table, req.ColumnName, req.Comment); err != nil {
		audit.RecordFailed(c, "COMMENT_COLUMN", fmt.Sprintf("%s.%s.%s", req.Database, req.Table, req.ColumnName), err)
		utils.ClickHouseError(c, "修改字段注释失败", err, "")
		return
	}

	audit.RecordSuccess(c, "COMMENT_COLUMN", fmt.Sprintf("%s.%s.%s", req.Database, req.Table, req.ColumnName))
	utils.SuccessWithMessage(c, "字段注释修改成功", nil)
}

func ImportData(c *gin.Context) {
	var req clickhouse.ImportDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误", err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	result, err := clickhouse.DefaultService.ImportData(ctx, req)
	if err != nil {
		audit.RecordFailed(c, "IMPORT_DATA", fmt.Sprintf("%s.%s", req.Database, req.Table), err)
		utils.ClickHouseError(c, "数据导入失败", err, "请检查数据格式与表字段类型是否严格对应")
		return
	}

	audit.RecordSuccess(c, "IMPORT_DATA", fmt.Sprintf("%s.%s (%d rows)", req.Database, req.Table, result.InsertedRows))
	utils.SuccessWithMessage(c, fmt.Sprintf("成功导入约 %d 行数据 (耗时 %d ms)", result.InsertedRows, result.ElapsedMs), result)
}

func PreviewImportData(c *gin.Context) {
	var req PreviewImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误", err.Error(), "")
		return
	}

	res, err := clickhouse.PreviewImportData(req.Data, req.Format)
	if err != nil {
		utils.BadRequest(c, "数据解析预览失败", err.Error(), "")
		return
	}

	utils.Success(c, res)
}
