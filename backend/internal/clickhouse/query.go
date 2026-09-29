package clickhouse

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

type QueryColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type QueryResult struct {
	QueryID     string                   `json:"query_id"`
	Columns     []QueryColumn            `json:"columns"`
	Rows        []map[string]interface{} `json:"rows"`
	TotalRows   int                      `json:"total_rows"`
	ReadRows    uint64                   `json:"read_rows"`
	ReadBytes   uint64                   `json:"read_bytes"`
	ElapsedMs   int64                    `json:"elapsed_ms"`
	MemoryBytes int64                    `json:"memory_bytes"`
}

type BrowserDataRequest struct {
	Database  string `json:"database"`
	Table     string `json:"table"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortField string `json:"sort_field"`
	SortOrder string `json:"sort_order"` // ASC / DESC
	Where     string `json:"where"`
	Columns   string `json:"columns"` // comma separated or empty for all
}

type BrowserDataResponse struct {
	QueryResult
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	TotalRows uint64 `json:"total_rows_estimate"`
}

var isSelectRegex = regexp.MustCompile(`(?is)^(?:\s|--[^\n]*(?:\n|$)|/\*.*?\*/)*(SELECT|WITH|SHOW|DESCRIBE|DESC|EXPLAIN|EXISTS)\b`)

func (s *ClickHouseService) Query(ctx context.Context, querySQL string, maxRows int) (*QueryResult, error) {
	return s.QueryWithDatabase(ctx, querySQL, maxRows, "")
}

func (s *ClickHouseService) QueryWithDatabase(ctx context.Context, querySQL string, maxRows int, defaultDb string) (*QueryResult, error) {
	settings := clickhouse.Settings{}
	if defaultDb != "" && defaultDb != s.GetConfig().Database {
		cfg := s.GetConfig()
		cfg.Database = defaultDb
		s = &ClickHouseService{config: cfg}
		conn, err := s.GetConn(ctx)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
	}
	if maxRows > 0 {
		settings["max_result_rows"] = maxRows
		settings["result_overflow_mode"] = "break"
	}
	queryID := uuid.New().String()
	var readRows, readBytes uint64
	ctx = clickhouse.Context(ctx, clickhouse.WithSettings(settings), clickhouse.WithQueryID(queryID),
		clickhouse.WithProgress(func(p *clickhouse.Progress) {
			readRows += p.Rows
			readBytes += p.Bytes
		}))

	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(querySQL)
	isSelect := isSelectRegex.MatchString(trimmed)

	startTime := time.Now()

	// If it's a non-SELECT statement (DDL/DML), use Exec
	if !isSelect {
		if err := conn.Exec(ctx, trimmed); err != nil {
			return nil, err
		}
		elapsed := time.Since(startTime).Milliseconds()
		return &QueryResult{
			QueryID:   queryID,
			Columns:   []QueryColumn{{Name: "status", Type: "String"}},
			Rows:      []map[string]interface{}{{"status": "Query executed successfully"}},
			TotalRows: 1,
			ElapsedMs: elapsed,
		}, nil
	}

	rows, err := conn.Query(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	colTypes := rows.ColumnTypes()
	columns := make([]QueryColumn, 0, len(colTypes))
	for _, ct := range colTypes {
		columns = append(columns, QueryColumn{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		})
	}

	results := make([]map[string]interface{}, 0)
	rowCount := 0

	for rows.Next() {
		if maxRows > 0 && rowCount >= maxRows {
			break
		}

		valPtrs := make([]interface{}, len(columns))
		for i, ct := range colTypes {
			valPtrs[i] = reflect.New(ct.ScanType()).Interface()
		}

		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col.Name] = queryValue(reflect.ValueOf(valPtrs[i]).Elem().Interface())
		}
		results = append(results, rowMap)
		rowCount++
	}
	// Close drains the driver's background reader before reading progress or late errors.
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	elapsed := time.Since(startTime).Milliseconds()

	return &QueryResult{
		QueryID:   queryID,
		Columns:   columns,
		Rows:      results,
		TotalRows: len(results),
		ReadRows:  readRows,
		ReadBytes: readBytes,
		ElapsedMs: elapsed,
	}, nil
}

// Preserve numbers that JavaScript or JSON cannot represent, including nested values.
func queryValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05.999999999")
	case *big.Int:
		if v == nil {
			return nil
		}
		return v.String()
	case big.Int:
		return v.String()
	case uint64:
		if v > 1<<53-1 {
			return strconv.FormatUint(v, 10)
		}
	case int64:
		if v > 1<<53-1 || v < -(1<<53-1) {
			return strconv.FormatInt(v, 10)
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return strconv.FormatFloat(v, 'g', -1, 64)
		}
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return strconv.FormatFloat(float64(v), 'g', -1, 32)
		}
	}
	if _, ok := value.(json.Marshaler); ok {
		return value
	}
	if _, ok := value.(encoding.TextMarshaler); ok {
		return value
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return queryValue(v.Elem().Interface())
	case reflect.Array, reflect.Slice:
		items := make([]interface{}, v.Len())
		for i := range items {
			items[i] = queryValue(v.Index(i).Interface())
		}
		return items
	case reflect.Map:
		items := make(map[string]interface{}, v.Len())
		for _, key := range v.MapKeys() {
			items[fmt.Sprint(key.Interface())] = queryValue(v.MapIndex(key).Interface())
		}
		return items
	}
	return value
}

func (s *ClickHouseService) Execute(ctx context.Context, sql string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) GetData(ctx context.Context, req BrowserDataRequest) (*BrowserDataResponse, error) {
	if req.Database == "" || req.Table == "" {
		return nil, fmt.Errorf("database and table are required")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 5000 {
		req.PageSize = 100
	}

	if req.Page-1 > int(^uint(0)>>1)/req.PageSize {
		return nil, fmt.Errorf("page is too large")
	}

	offset := (req.Page - 1) * req.PageSize

	cols := "*"
	if req.Columns != "" {
		cols = req.Columns
	}

	whereClause := ""
	if strings.TrimSpace(req.Where) != "" {
		whereClause = fmt.Sprintf("WHERE %s", req.Where)
	}

	orderClause := ""
	if req.SortField != "" {
		order := "ASC"
		if strings.ToUpper(req.SortOrder) == "DESC" {
			order = "DESC"
		}
		orderClause = fmt.Sprintf("ORDER BY `%s` %s", escapeIdent(req.SortField), order)
	}

	// 1. Get approximate or total rows
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	var totalRows uint64
	if whereClause == "" {
		// Fast O(1) row count from system.tables
		_ = conn.QueryRow(ctx, "SELECT total_rows FROM system.tables WHERE database = ? AND name = ?", req.Database, req.Table).Scan(&totalRows)
	}

	if totalRows == 0 {
		countSQL := fmt.Sprintf("SELECT count() FROM `%s`.`%s` %s", escapeIdent(req.Database), escapeIdent(req.Table), whereClause)
		countCtx, countCancel := context.WithTimeout(ctx, 3*time.Second)
		err := conn.QueryRow(countCtx, countSQL).Scan(&totalRows)
		countCancel()
		if err != nil {
			return nil, err
		}
	}

	// 2. Fetch page data
	querySQL := fmt.Sprintf("SELECT %s FROM `%s`.`%s` %s %s LIMIT %d OFFSET %d",
		cols,
		escapeIdent(req.Database),
		escapeIdent(req.Table),
		whereClause,
		orderClause,
		req.PageSize,
		offset,
	)

	qr, err := s.Query(ctx, querySQL, req.PageSize)
	if err != nil {
		return nil, err
	}

	return &BrowserDataResponse{
		QueryResult: *qr,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalRows:   totalRows,
	}, nil
}
