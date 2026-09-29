package clickhouse

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type DatabaseInfo struct {
	Name         string `json:"name"`
	Engine       string `json:"engine"`
	DataPath     string `json:"data_path"`
	MetadataPath string `json:"metadata_path"`
	Comment      string `json:"comment"`
	TablesCount  uint64 `json:"tables_count"`
	TotalBytes   uint64 `json:"total_bytes"`
	TotalRows    uint64 `json:"total_rows"`
}

type TableInfo struct {
	Database          string    `json:"database"`
	Name              string    `json:"name"`
	Engine            string    `json:"engine"`
	TotalRows         uint64    `json:"total_rows"`
	TotalBytes        uint64    `json:"total_bytes"`
	CompressedBytes   uint64    `json:"compressed_bytes"`
	UncompressedBytes uint64    `json:"uncompressed_bytes"`
	PartsCount        uint64    `json:"parts_count"`
	PrimaryKey        string    `json:"primary_key"`
	OrderBy           string    `json:"order_by"`
	PartitionKey      string    `json:"partition_key"`
	TTL               string    `json:"ttl"`
	LastModified      time.Time `json:"last_modified"`
	Comment           string    `json:"comment"`
}

type ColumnInfo struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	DefaultKind      string `json:"default_kind"`
	DefaultExpr      string `json:"default_expression"`
	Comment          string `json:"comment"`
	IsInPrimaryKey   bool   `json:"is_in_primary_key"`
	IsInPartitionKey bool   `json:"is_in_partition_key"`
	CompressionCodec string `json:"compression_codec"`
}

type VisualColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
	Comment  string `json:"comment"`
	Codec    string `json:"codec"`
}

type VisualCreateTableRequest struct {
	Database    string         `json:"database"`
	TableName   string         `json:"table_name"`
	Engine      string         `json:"engine"` // MergeTree, ReplacingMergeTree, etc.
	EngineArgs  string         `json:"engine_args"`
	Columns     []VisualColumn `json:"columns"`
	OrderBy     string         `json:"order_by"`
	PrimaryKey  string         `json:"primary_key"`
	PartitionBy string         `json:"partition_by"`
	SampleBy    string         `json:"sample_by"`
	TTL         string         `json:"ttl"`
	Settings    string         `json:"settings"`
	ExecuteNow  bool           `json:"execute_now"`
}

func (s *ClickHouseService) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			d.name,
			d.engine,
			d.data_path,
			d.metadata_path,
			d.comment,
			count(t.name) AS tables_count,
			coStorage.total_bytes,
			coStorage.total_rows
		FROM system.databases AS d
		LEFT JOIN system.tables AS t ON t.database = d.name
		LEFT JOIN (
			SELECT 
				database, 
				sum(bytes_on_disk) AS total_bytes, 
				sum(rows) AS total_rows 
			FROM system.parts 
			WHERE active = 1 
			GROUP BY database
		) AS coStorage ON coStorage.database = d.name
		GROUP BY d.name, d.engine, d.data_path, d.metadata_path, d.comment, coStorage.total_bytes, coStorage.total_rows
		ORDER BY d.name
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		// Fallback query if system.parts or complex join has permission limits
		return s.listDatabasesFallback(ctx)
	}
	defer rows.Close()

	var result []DatabaseInfo
	for rows.Next() {
		var item DatabaseInfo
		var totalBytes, totalRows *uint64
		if err := rows.Scan(
			&item.Name,
			&item.Engine,
			&item.DataPath,
			&item.MetadataPath,
			&item.Comment,
			&item.TablesCount,
			&totalBytes,
			&totalRows,
		); err != nil {
			return nil, err
		}
		if totalBytes != nil {
			item.TotalBytes = *totalBytes
		}
		if totalRows != nil {
			item.TotalRows = *totalRows
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *ClickHouseService) listDatabasesFallback(ctx context.Context) ([]DatabaseInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, "SELECT name, engine, data_path, metadata_path, comment FROM system.databases ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DatabaseInfo
	for rows.Next() {
		var item DatabaseInfo
		if err := rows.Scan(&item.Name, &item.Engine, &item.DataPath, &item.MetadataPath, &item.Comment); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *ClickHouseService) CreateDatabase(ctx context.Context, name, engine, comment string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", escapeIdent(name))
	if engine != "" {
		q += fmt.Sprintf(" ENGINE = %s", engine)
	}
	if comment != "" {
		q += fmt.Sprintf(" COMMENT '%s'", escapeString(comment))
	}
	return conn.Exec(ctx, q)
}

func (s *ClickHouseService) DropDatabase(ctx context.Context, name string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", escapeIdent(name)))
}

func (s *ClickHouseService) RenameDatabase(ctx context.Context, oldName, newName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("RENAME DATABASE `%s` TO `%s`", escapeIdent(oldName), escapeIdent(newName)))
}

func (s *ClickHouseService) ListTables(ctx context.Context, dbName string) ([]TableInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			t.database,
			t.name,
			t.engine,
			coalesce(p.total_rows, t.total_rows, 0) AS total_rows,
			coalesce(p.total_bytes, t.total_bytes, 0) AS total_bytes,
			coalesce(p.compressed_bytes, 0) AS compressed_bytes,
			coalesce(p.uncompressed_bytes, 0) AS uncompressed_bytes,
			coalesce(p.parts_count, 0) AS parts_count,
			t.primary_key,
			t.sorting_key,
			t.partition_key,
			extract(t.create_table_query, '(?is)\\bTTL\\s+(.+?)(?:\\s+SETTINGS\\b|$)') as ttl,
			coalesce(p.max_mod_time, toDateTime('1970-01-01 00:00:00')) as last_mod,
			t.comment
		FROM system.tables AS t
		LEFT JOIN (
			SELECT 
				database, 
				table, 
				sum(rows) AS total_rows, 
				sum(bytes_on_disk) AS total_bytes,
				sum(data_compressed_bytes) AS compressed_bytes,
				sum(data_uncompressed_bytes) AS uncompressed_bytes,
				count() AS parts_count,
				max(modification_time) AS max_mod_time
			FROM system.parts 
			WHERE database = ? AND active = 1 
			GROUP BY database, table
		) AS p ON p.database = t.database AND p.table = t.name
		WHERE t.database = ?
		ORDER BY t.name
	`

	rows, err := conn.Query(ctx, query, dbName, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TableInfo
	for rows.Next() {
		var item TableInfo
		var totalRows, totalBytes, compBytes, uncompBytes *uint64
		if err := rows.Scan(
			&item.Database,
			&item.Name,
			&item.Engine,
			&totalRows,
			&totalBytes,
			&compBytes,
			&uncompBytes,
			&item.PartsCount,
			&item.PrimaryKey,
			&item.OrderBy,
			&item.PartitionKey,
			&item.TTL,
			&item.LastModified,
			&item.Comment,
		); err != nil {
			return nil, err
		}
		if totalRows != nil {
			item.TotalRows = *totalRows
		}
		if totalBytes != nil {
			item.TotalBytes = *totalBytes
		}
		if compBytes != nil {
			item.CompressedBytes = *compBytes
		}
		if uncompBytes != nil {
			item.UncompressedBytes = *uncompBytes
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *ClickHouseService) GetColumns(ctx context.Context, dbName, tableName string) ([]ColumnInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			name,
			type,
			default_kind,
			default_expression,
			comment,
			is_in_primary_key,
			is_in_partition_key,
			compression_codec
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position
	`
	rows, err := conn.Query(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var inPk, inPart uint8
		if err := rows.Scan(
			&c.Name,
			&c.Type,
			&c.DefaultKind,
			&c.DefaultExpr,
			&c.Comment,
			&inPk,
			&inPart,
			&c.CompressionCodec,
		); err != nil {
			return nil, err
		}
		c.IsInPrimaryKey = (inPk == 1)
		c.IsInPartitionKey = (inPart == 1)
		cols = append(cols, c)
	}
	return cols, nil
}

func (s *ClickHouseService) ShowCreateTable(ctx context.Context, dbName, tableName string) (string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return "", err
	}

	var ddl string
	row := conn.QueryRow(ctx, fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName)))
	if err := row.Scan(&ddl); err != nil {
		return "", err
	}
	return ddl, nil
}

func (s *ClickHouseService) DropTable(ctx context.Context, dbName, tableName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName)))
}

func (s *ClickHouseService) TruncateTable(ctx context.Context, dbName, tableName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName)))
}

func (s *ClickHouseService) RenameTable(ctx context.Context, dbName, oldName, newName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`", escapeIdent(dbName), escapeIdent(oldName), escapeIdent(dbName), escapeIdent(newName)))
}

func (s *ClickHouseService) OptimizeTable(ctx context.Context, dbName, tableName string, final bool, partition string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	q := fmt.Sprintf("OPTIMIZE TABLE `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName))
	if partition != "" {
		q += fmt.Sprintf(" PARTITION %s", partition)
	}
	if final {
		q += " FINAL"
	}
	return conn.Exec(ctx, q)
}

func (s *ClickHouseService) DetachTable(ctx context.Context, dbName, tableName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("DETACH TABLE `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName)))
}

func (s *ClickHouseService) AttachTable(ctx context.Context, dbName, tableName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("ATTACH TABLE `%s`.`%s`", escapeIdent(dbName), escapeIdent(tableName)))
}

// GenerateCreateTableSQL constructs ClickHouse CREATE TABLE DDL
func GenerateCreateTableSQL(req VisualCreateTableRequest) (string, error) {
	if req.Database == "" || req.TableName == "" {
		return "", fmt.Errorf("database and table name cannot be empty")
	}
	if len(req.Columns) == 0 {
		return "", fmt.Errorf("at least one column is required")
	}

	var colDefs []string
	for _, col := range req.Columns {
		if col.Name == "" || col.Type == "" {
			continue
		}
		typeDef := col.Type
		if col.Nullable && !strings.HasPrefix(strings.ToLower(col.Type), "nullable") && !strings.HasPrefix(strings.ToLower(col.Type), "lowcardinality") {
			typeDef = fmt.Sprintf("Nullable(%s)", col.Type)
		}

		colStr := fmt.Sprintf("    `%s` %s", escapeIdent(col.Name), typeDef)
		if col.Default != "" {
			colStr += fmt.Sprintf(" DEFAULT %s", col.Default)
		}
		if col.Codec != "" {
			colStr += fmt.Sprintf(" CODEC(%s)", col.Codec)
		}
		if col.Comment != "" {
			colStr += fmt.Sprintf(" COMMENT '%s'", escapeString(col.Comment))
		}
		colDefs = append(colDefs, colStr)
	}

	engine := req.Engine
	if engine == "" {
		engine = "MergeTree"
	}
	if req.EngineArgs != "" && !strings.Contains(engine, "(") {
		engine = fmt.Sprintf("%s(%s)", engine, req.EngineArgs)
	}

	sql := fmt.Sprintf("CREATE TABLE `%s`.`%s` (\n%s\n) ENGINE = %s\n",
		escapeIdent(req.Database),
		escapeIdent(req.TableName),
		strings.Join(colDefs, ",\n"),
		engine,
	)

	if req.OrderBy != "" {
		sql += fmt.Sprintf("ORDER BY (%s)\n", req.OrderBy)
	} else if strings.Contains(engine, "MergeTree") {
		// MergeTree requires ORDER BY or tuple()
		sql += "ORDER BY tuple()\n"
	}

	if req.PrimaryKey != "" {
		sql += fmt.Sprintf("PRIMARY KEY (%s)\n", req.PrimaryKey)
	}

	if req.PartitionBy != "" {
		sql += fmt.Sprintf("PARTITION BY (%s)\n", req.PartitionBy)
	}

	if req.SampleBy != "" {
		sql += fmt.Sprintf("SAMPLE BY (%s)\n", req.SampleBy)
	}

	if req.TTL != "" {
		sql += fmt.Sprintf("TTL %s\n", req.TTL)
	}

	if req.Settings != "" {
		sql += fmt.Sprintf("SETTINGS %s\n", req.Settings)
	}

	return strings.TrimSpace(sql), nil
}

func (s *ClickHouseService) VisualCreateTable(ctx context.Context, req VisualCreateTableRequest) (string, error) {
	sql, err := GenerateCreateTableSQL(req)
	if err != nil {
		return "", err
	}

	if req.ExecuteNow {
		conn, err := s.GetConn(ctx)
		if err != nil {
			return sql, err
		}
		if err := conn.Exec(ctx, sql); err != nil {
			return sql, fmt.Errorf("execute CREATE TABLE failed: %w", err)
		}
	}

	return sql, nil
}

func escapeIdent(s string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`").Replace(s)
}

func escapeString(s string) string {
	return strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(s)
}

type AddColumnRequest struct {
	Database    string `json:"database"`
	Table       string `json:"table"`
	ColumnName  string `json:"column_name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	DefaultExpr string `json:"default_expression"`
	Comment     string `json:"comment"`
	Codec       string `json:"codec"`
	AfterColumn string `json:"after_column"`
}

func (s *ClickHouseService) AddColumn(ctx context.Context, req AddColumnRequest) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	colType := req.Type
	if req.Nullable && !strings.HasPrefix(strings.ToLower(colType), "nullable") && !strings.HasPrefix(strings.ToLower(colType), "lowcardinality") {
		colType = fmt.Sprintf("Nullable(%s)", colType)
	}

	sql := fmt.Sprintf("ALTER TABLE `%s`.`%s` ADD COLUMN IF NOT EXISTS `%s` %s",
		escapeIdent(req.Database),
		escapeIdent(req.Table),
		escapeIdent(req.ColumnName),
		colType,
	)

	if req.DefaultExpr != "" {
		sql += fmt.Sprintf(" DEFAULT %s", req.DefaultExpr)
	}
	if req.Codec != "" {
		sql += fmt.Sprintf(" CODEC(%s)", req.Codec)
	}
	if req.Comment != "" {
		sql += fmt.Sprintf(" COMMENT '%s'", escapeString(req.Comment))
	}
	if req.AfterColumn != "" {
		sql += fmt.Sprintf(" AFTER `%s`", escapeIdent(req.AfterColumn))
	}

	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) DropColumn(ctx context.Context, dbName, tableName, columnName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("ALTER TABLE `%s`.`%s` DROP COLUMN IF EXISTS `%s`",
		escapeIdent(dbName),
		escapeIdent(tableName),
		escapeIdent(columnName),
	)
	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) ModifyColumnComment(ctx context.Context, dbName, tableName, columnName, comment string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT COLUMN IF EXISTS `%s` '%s'",
		escapeIdent(dbName),
		escapeIdent(tableName),
		escapeIdent(columnName),
		escapeString(comment),
	)
	return conn.Exec(ctx, sql)
}

type ImportDataRequest struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Format   string `json:"format"` // CSV, CSVWithNames, TSV, TSVWithNames, JSONEachRow
	Data     string `json:"data"`
}

type ImportDataResult struct {
	InsertedRows int   `json:"inserted_rows"`
	ElapsedMs    int64 `json:"elapsed_ms"`
}

type ImportPreviewResult struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
	Total   int        `json:"total_lines_estimate"`
}

func (s *ClickHouseService) ImportData(ctx context.Context, req ImportDataRequest) (*ImportDataResult, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	format := req.Format
	if format == "" {
		format = "CSV"
	}

	trimmedData := strings.TrimSpace(req.Data)
	if trimmedData == "" {
		return nil, fmt.Errorf("import data is empty")
	}

	startTime := time.Now()
	sql := fmt.Sprintf("INSERT INTO `%s`.`%s` FORMAT %s\n%s",
		escapeIdent(req.Database),
		escapeIdent(req.Table),
		format,
		trimmedData,
	)

	if err := conn.Exec(ctx, sql); err != nil {
		return nil, err
	}

	elapsed := time.Since(startTime).Milliseconds()

	// Estimate line count
	lines := strings.Split(trimmedData, "\n")
	inserted := len(lines)
	if strings.Contains(strings.ToLower(format), "names") && inserted > 0 {
		inserted--
	}

	return &ImportDataResult{
		InsertedRows: inserted,
		ElapsedMs:    elapsed,
	}, nil
}

func PreviewImportData(data, format string) (*ImportPreviewResult, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("preview data is empty")
	}

	if format == "" || format == "CSV" || format == "CSVWithNames" {
		reader := csv.NewReader(strings.NewReader(trimmed))
		reader.FieldsPerRecord = -1
		result := &ImportPreviewResult{Rows: make([][]string, 0)}
		if format == "CSVWithNames" {
			headers, err := reader.Read()
			if err != nil {
				return nil, err
			}
			result.Headers = headers
		}
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			result.Total++
			if len(result.Rows) < 10 {
				result.Rows = append(result.Rows, row)
			}
		}
		if len(result.Headers) == 0 && len(result.Rows) > 0 {
			for i := range result.Rows[0] {
				result.Headers = append(result.Headers, fmt.Sprintf("Col_%d", i+1))
			}
		}
		return result, nil
	}

	lines := strings.Split(trimmed, "\n")
	totalLines := len(lines)
	maxPreview := 10
	if len(lines) > maxPreview {
		lines = lines[:maxPreview]
	}

	isJSON := strings.Contains(strings.ToUpper(format), "JSON")
	isTSV := strings.Contains(strings.ToUpper(format), "TSV")

	result := &ImportPreviewResult{
		Total: totalLines,
	}

	if isJSON {
		result.Headers = []string{"JSON Record Preview"}
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if t != "" {
				result.Rows = append(result.Rows, []string{t})
			}
		}
		return result, nil
	}

	sep := ","
	if isTSV {
		sep = "\t"
	}

	withNames := strings.Contains(strings.ToUpper(format), "NAMES")
	startIdx := 0

	if withNames && len(lines) > 0 {
		headerLine := strings.TrimSpace(lines[0])
		result.Headers = parseDelimitedLine(headerLine, sep)
		startIdx = 1
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		row := parseDelimitedLine(line, sep)
		result.Rows = append(result.Rows, row)
	}

	if len(result.Headers) == 0 && len(result.Rows) > 0 {
		for c := 0; c < len(result.Rows[0]); c++ {
			result.Headers = append(result.Headers, fmt.Sprintf("Col_%d", c+1))
		}
	}

	return result, nil
}

func parseDelimitedLine(line, sep string) []string {
	parts := strings.Split(line, sep)
	var cleaned []string
	for _, p := range parts {
		c := strings.TrimSpace(p)
		c = strings.Trim(c, "\"")
		cleaned = append(cleaned, c)
	}
	return cleaned
}
