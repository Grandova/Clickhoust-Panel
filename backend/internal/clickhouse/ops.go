package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type MetricsSummary struct {
	QueryPreSecond      float64          `json:"query_per_second"`
	InsertPreSecond     float64          `json:"insert_per_second"`
	SelectPreSecond     float64          `json:"select_per_second"`
	RowsReadPreSecond   float64          `json:"rows_read_per_second"`
	BytesReadPreSecond  float64          `json:"bytes_read_per_second"`
	RowsWritePreSecond  float64          `json:"rows_write_per_second"`
	BytesWritePreSecond float64          `json:"bytes_write_per_second"`
	CurrentConnections  uint64           `json:"current_connections"`
	CurrentQueries      uint64           `json:"current_queries"`
	MemoryTracking      uint64           `json:"memory_tracking"`
	MaxMemoryTracking   uint64           `json:"max_memory_tracking"`
	DiskUsageBytes      uint64           `json:"disk_usage_bytes"`
	TotalDatabases      int              `json:"total_databases"`
	TotalTables         int              `json:"total_tables"`
	TotalRows           uint64           `json:"total_rows"`
	TotalBytes          uint64           `json:"total_bytes"`
	CompressedBytes     uint64           `json:"compressed_bytes"`
	UncompressedBytes   uint64           `json:"uncompressed_bytes"`
	TodayQueries        uint64           `json:"today_queries"`
	TodayFailedQueries  uint64           `json:"today_failed_queries"`
	RawMetrics          map[string]int64 `json:"raw_metrics"`
	Timestamp           int64            `json:"timestamp"`
}

type ProcessInfo struct {
	QueryID     string    `json:"query_id"`
	User        string    `json:"user"`
	Address     string    `json:"address"`
	Elapsed     float64   `json:"elapsed"`
	ReadRows    uint64    `json:"read_rows"`
	ReadBytes   uint64    `json:"read_bytes"`
	MemoryUsage int64     `json:"memory_usage"`
	Query       string    `json:"query"`
	StartTime   time.Time `json:"start_time"`
}

type QueryLogItem struct {
	QueryID          string    `json:"query_id"`
	Query            string    `json:"query"`
	User             string    `json:"user"`
	ClientAddress    string    `json:"client_address"`
	Database         string    `json:"database"`
	Type             string    `json:"type"` // QueryFinish, ExceptionBeforeStart, ExceptionWhileProcessing
	EventTime        time.Time `json:"event_time"`
	DurationMs       int64     `json:"duration_ms"`
	ReadRows         uint64    `json:"read_rows"`
	ReadBytes        uint64    `json:"read_bytes"`
	ResultRows       uint64    `json:"result_rows"`
	ResultBytes      uint64    `json:"result_bytes"`
	MemoryPeak       int64     `json:"memory_peak"`
	ExceptionCode    int32     `json:"exception_code"`
	ExceptionMessage string    `json:"exception_message"`
}

type PartInfo struct {
	Database         string    `json:"database"`
	Table            string    `json:"table"`
	Partition        string    `json:"partition"`
	Name             string    `json:"name"`
	Rows             uint64    `json:"rows"`
	BytesOnDisk      uint64    `json:"bytes_on_disk"`
	DataCompressed   uint64    `json:"data_compressed"`
	DataUncompressed uint64    `json:"data_uncompressed"`
	Marks            uint64    `json:"marks"`
	Active           bool      `json:"active"`
	ModificationTime time.Time `json:"modification_time"`
	DiskName         string    `json:"disk_name"`
	Level            uint32    `json:"level"`
}

type MergeInfo struct {
	Database       string  `json:"database"`
	Table          string  `json:"table"`
	Elapsed        float64 `json:"elapsed"`
	Progress       float64 `json:"progress"`
	NumParts       uint64  `json:"num_parts"`
	ResultPartName string  `json:"result_part_name"`
	RowsRead       uint64  `json:"rows_read"`
	RowsWritten    uint64  `json:"rows_written"`
	MemoryUsage    int64   `json:"memory_usage"`
	ThreadID       uint64  `json:"thread_id"`
}

type MutationInfo struct {
	Database         string    `json:"database"`
	Table            string    `json:"table"`
	MutationID       string    `json:"mutation_id"`
	Command          string    `json:"command"`
	CreateTime       time.Time `json:"create_time"`
	PartsToDo        int64     `json:"parts_to_do"`
	IsDone           bool      `json:"is_done"`
	LatestFailedPart string    `json:"latest_failed_part"`
	LatestFailReason string    `json:"latest_fail_reason"`
	LatestFailTime   time.Time `json:"latest_fail_time"`
}

type DiskInfo struct {
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	FreeSpace     uint64  `json:"free_space"`
	TotalSpace    uint64  `json:"total_space"`
	UsedSpace     uint64  `json:"used_space"`
	UsedPercent   float64 `json:"used_percent"`
	KeepFreeSpace uint64  `json:"keep_free_space"`
	Type          string  `json:"type"`
	StoragePolicy string  `json:"storage_policy"`
}

func (s *ClickHouseService) GetProcesses(ctx context.Context) ([]ProcessInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			query_id,
			user,
			client_hostname,
			elapsed,
			read_rows,
			read_bytes,
			memory_usage,
			query,
			toDateTime(now() - elapsed)
		FROM system.processes
		ORDER BY elapsed DESC
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProcessInfo
	for rows.Next() {
		var p ProcessInfo
		if err := rows.Scan(
			&p.QueryID,
			&p.User,
			&p.Address,
			&p.Elapsed,
			&p.ReadRows,
			&p.ReadBytes,
			&p.MemoryUsage,
			&p.Query,
			&p.StartTime,
		); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

func (s *ClickHouseService) KillQuery(ctx context.Context, queryID string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("KILL QUERY WHERE query_id = '%s' SYNC", escapeString(queryID))
	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) GetQueryLog(ctx context.Context, minDurationMs int64, limit int, offset int, search string) ([]QueryLogItem, int64, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, 0, err
	}

	whereParts := []string{"1=1"}
	if minDurationMs > 0 {
		whereParts = append(whereParts, fmt.Sprintf("query_duration_ms >= %d", minDurationMs))
	}
	if search != "" {
		esc := escapeString(search)
		whereParts = append(whereParts, fmt.Sprintf("(query ILIKE '%%%s%%' OR query_id ILIKE '%%%s%%' OR user ILIKE '%%%s%%')", esc, esc, esc))
	}
	whereClause := strings.Join(whereParts, " AND ")

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Check if table exists
	var hasQueryLog uint8
	_ = conn.QueryRow(ctx, "SELECT 1 FROM system.tables WHERE database = 'system' AND name = 'query_log'").Scan(&hasQueryLog)
	if hasQueryLog == 0 {
		return []QueryLogItem{}, 0, nil
	}

	// Count total matching
	var total uint64
	countSQL := fmt.Sprintf("SELECT count() FROM system.query_log WHERE %s", whereClause)
	_ = conn.QueryRow(ctx, countSQL).Scan(&total)

	query := fmt.Sprintf(`
		SELECT 
			query_id,
			query,
			user,
			client_hostname,
			current_database,
			type,
			event_time,
			toInt64(query_duration_ms),
			read_rows,
			read_bytes,
			result_rows,
			result_bytes,
			toInt64(memory_usage),
			exception_code,
			exception
		FROM system.query_log
		WHERE %s
		ORDER BY event_time DESC
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	rows, err := conn.Query(ctx, query)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "doesn't exist") || strings.Contains(strings.ToLower(err.Error()), "unknown_table") {
			return []QueryLogItem{}, 0, nil
		}
		return nil, 0, err
	}
	defer rows.Close()

	var list []QueryLogItem
	for rows.Next() {
		var item QueryLogItem
		if err := rows.Scan(
			&item.QueryID,
			&item.Query,
			&item.User,
			&item.ClientAddress,
			&item.Database,
			&item.Type,
			&item.EventTime,
			&item.DurationMs,
			&item.ReadRows,
			&item.ReadBytes,
			&item.ResultRows,
			&item.ResultBytes,
			&item.MemoryPeak,
			&item.ExceptionCode,
			&item.ExceptionMessage,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, int64(total), rows.Err()
}

func (s *ClickHouseService) GetParts(ctx context.Context, dbName, tableName string, activeOnly bool) ([]PartInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	whereParts := []string{"1=1"}
	if dbName != "" {
		whereParts = append(whereParts, fmt.Sprintf("database = '%s'", escapeString(dbName)))
	}
	if tableName != "" {
		whereParts = append(whereParts, fmt.Sprintf("table = '%s'", escapeString(tableName)))
	}
	if activeOnly {
		whereParts = append(whereParts, "active = 1")
	}

	query := fmt.Sprintf(`
		SELECT 
			database,
			table,
			partition,
			name,
			rows,
			bytes_on_disk,
			data_compressed_bytes,
			data_uncompressed_bytes,
			marks,
			active,
			modification_time,
			disk_name,
			level
		FROM system.parts
		WHERE %s
		ORDER BY modification_time DESC
		LIMIT 1000
	`, strings.Join(whereParts, " AND "))

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PartInfo
	for rows.Next() {
		var p PartInfo
		var active uint8
		if err := rows.Scan(
			&p.Database,
			&p.Table,
			&p.Partition,
			&p.Name,
			&p.Rows,
			&p.BytesOnDisk,
			&p.DataCompressed,
			&p.DataUncompressed,
			&p.Marks,
			&active,
			&p.ModificationTime,
			&p.DiskName,
			&p.Level,
		); err != nil {
			return nil, err
		}
		p.Active = (active == 1)
		list = append(list, p)
	}
	return list, nil
}

func (s *ClickHouseService) GetMerges(ctx context.Context) ([]MergeInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			database,
			table,
			elapsed,
			progress,
			num_parts,
			result_part_name,
			rows_read,
			rows_written,
			memory_usage,
			thread_id
		FROM system.merges
		ORDER BY elapsed DESC
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MergeInfo
	for rows.Next() {
		var m MergeInfo
		if err := rows.Scan(
			&m.Database,
			&m.Table,
			&m.Elapsed,
			&m.Progress,
			&m.NumParts,
			&m.ResultPartName,
			&m.RowsRead,
			&m.RowsWritten,
			&m.MemoryUsage,
			&m.ThreadID,
		); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (s *ClickHouseService) GetMutations(ctx context.Context, dbName, tableName string) ([]MutationInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	where := "1=1"
	if dbName != "" {
		where += fmt.Sprintf(" AND database = '%s'", escapeString(dbName))
	}
	if tableName != "" {
		where += fmt.Sprintf(" AND table = '%s'", escapeString(tableName))
	}

	query := fmt.Sprintf(`
		SELECT 
			database,
			table,
			mutation_id,
			command,
			create_time,
			parts_to_do,
			is_done,
			latest_failed_part,
			latest_fail_reason,
			latest_fail_time
		FROM system.mutations
		WHERE %s
		ORDER BY create_time DESC
		LIMIT 500
	`, where)

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MutationInfo
	for rows.Next() {
		var m MutationInfo
		var isDone uint8
		if err := rows.Scan(
			&m.Database,
			&m.Table,
			&m.MutationID,
			&m.Command,
			&m.CreateTime,
			&m.PartsToDo,
			&isDone,
			&m.LatestFailedPart,
			&m.LatestFailReason,
			&m.LatestFailTime,
		); err != nil {
			return nil, err
		}
		m.IsDone = (isDone == 1)
		list = append(list, m)
	}
	return list, nil
}

func (s *ClickHouseService) KillMutation(ctx context.Context, dbName, tableName, mutationID string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("KILL MUTATION WHERE database = '%s' AND table = '%s' AND mutation_id = '%s'",
		escapeString(dbName),
		escapeString(tableName),
		escapeString(mutationID),
	)
	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) GetDisks(ctx context.Context) ([]DiskInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			name,
			path,
			free_space,
			total_space,
			keep_free_space,
			type
		FROM system.disks
		ORDER BY name
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DiskInfo
	for rows.Next() {
		var d DiskInfo
		if err := rows.Scan(
			&d.Name,
			&d.Path,
			&d.FreeSpace,
			&d.TotalSpace,
			&d.KeepFreeSpace,
			&d.Type,
		); err != nil {
			return nil, err
		}
		if d.TotalSpace > 0 {
			d.UsedSpace = d.TotalSpace - d.FreeSpace
			d.UsedPercent = (float64(d.UsedSpace) / float64(d.TotalSpace)) * 100.0
		}
		list = append(list, d)
	}
	return list, nil
}

func (s *ClickHouseService) GetMetrics(ctx context.Context) (*MetricsSummary, error) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	if s.lastMetrics != nil && time.Since(s.lastMetricsTime) < time.Second {
		return s.lastMetrics, nil
	}
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	summary := &MetricsSummary{
		RawMetrics: make(map[string]int64),
		Timestamp:  time.Now().Unix(),
	}

	// 1. Current metrics
	if rows, err := conn.Query(ctx, "SELECT metric, value FROM system.metrics WHERE metric IN ('Query', 'TCPConnection', 'HTTPConnection', 'MemoryTracking', 'MaxMemoryTracking')"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var m string
			var v int64
			if err := rows.Scan(&m, &v); err == nil {
				summary.RawMetrics[m] = v
				switch m {
				case "Query":
					summary.CurrentQueries = uint64(v)
				case "TCPConnection", "HTTPConnection":
					summary.CurrentConnections += uint64(v)
				case "MemoryTracking":
					summary.MemoryTracking = uint64(v)
				case "MaxMemoryTracking":
					summary.MaxMemoryTracking = uint64(v)
				}
			}
		}
	}

	// system.events contains cumulative counters, not per-second rates.
	rows, err := conn.Query(ctx, "SELECT event, value FROM system.events WHERE event IN ('Query', 'SelectQuery', 'InsertQuery', 'InsertedRows', 'InsertedBytes', 'SelectedRows', 'SelectedBytes')")
	if err != nil {
		return nil, err
	}
	var events [7]uint64
	for rows.Next() {
		var event string
		var value uint64
		if err := rows.Scan(&event, &value); err != nil {
			rows.Close()
			return nil, err
		}
		for i, name := range [...]string{"Query", "SelectQuery", "InsertQuery", "InsertedRows", "InsertedBytes", "SelectedRows", "SelectedBytes"} {
			if event == name {
				events[i] = value
				break
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := time.Now()
	if !s.lastMetricsTime.IsZero() {
		seconds := now.Sub(s.lastMetricsTime).Seconds()
		for i, rate := range []*float64{&summary.QueryPreSecond, &summary.SelectPreSecond, &summary.InsertPreSecond, &summary.RowsWritePreSecond, &summary.BytesWritePreSecond, &summary.RowsReadPreSecond, &summary.BytesReadPreSecond} {
			if events[i] >= s.lastEvents[i] {
				*rate = float64(events[i]-s.lastEvents[i]) / seconds
			}
		}
	}

	// Metadata and query-log totals do not need the same sampling rate as live counters.
	if s.lastMetrics != nil && time.Since(s.lastTotalsTime) < 30*time.Second {
		summary.TotalDatabases = s.lastMetrics.TotalDatabases
		summary.TotalTables = s.lastMetrics.TotalTables
		summary.TotalRows = s.lastMetrics.TotalRows
		summary.TotalBytes = s.lastMetrics.TotalBytes
		summary.CompressedBytes = s.lastMetrics.CompressedBytes
		summary.UncompressedBytes = s.lastMetrics.UncompressedBytes
		summary.DiskUsageBytes = s.lastMetrics.DiskUsageBytes
		summary.TodayQueries = s.lastMetrics.TodayQueries
		summary.TodayFailedQueries = s.lastMetrics.TodayFailedQueries
	} else {
		// 3. Database & Table totals (excluding system internal databases)
		var totalDatabases, totalTables uint64
		_ = conn.QueryRow(ctx, "SELECT count() FROM system.databases WHERE name NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA')").Scan(&totalDatabases)
		_ = conn.QueryRow(ctx, "SELECT count() FROM system.tables WHERE database NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA')").Scan(&totalTables)
		summary.TotalDatabases, summary.TotalTables = int(totalDatabases), int(totalTables)

		// 4. Rows & bytes (exclude system databases so user data isn't mixed with internal metric logs)
		var totalRows, totalBytes, compBytes, uncompBytes *uint64
		_ = conn.QueryRow(ctx, "SELECT sum(rows), sum(bytes_on_disk), sum(data_compressed_bytes), sum(data_uncompressed_bytes) FROM system.parts WHERE active = 1 AND database NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA')").Scan(
			&totalRows, &totalBytes, &compBytes, &uncompBytes,
		)
		if totalRows != nil {
			summary.TotalRows = *totalRows
		}
		if totalBytes != nil {
			summary.TotalBytes = *totalBytes
			summary.DiskUsageBytes = *totalBytes
		}
		if compBytes != nil {
			summary.CompressedBytes = *compBytes
		}
		if uncompBytes != nil {
			summary.UncompressedBytes = *uncompBytes
		}

		// Fetch host disk usage from system.disks
		var diskUsed, diskTotal uint64
		if err := conn.QueryRow(ctx, "SELECT sum(total_space - free_space), sum(total_space) FROM system.disks").Scan(&diskUsed, &diskTotal); err == nil && diskUsed > 0 {
			summary.DiskUsageBytes = diskUsed
		}

		// 5. Today's queries from query_log (if table exists)
		var hasQueryLog uint8
		_ = conn.QueryRow(ctx, "SELECT 1 FROM system.tables WHERE database = 'system' AND name = 'query_log'").Scan(&hasQueryLog)
		if hasQueryLog == 1 {
			_ = conn.QueryRow(ctx, "SELECT count(), countIf(type IN ('ExceptionBeforeStart', 'ExceptionWhileProcessing')) FROM system.query_log WHERE toDate(event_time) = today() AND type != 'QueryStart'").Scan(
				&summary.TodayQueries, &summary.TodayFailedQueries,
			)
		}

		s.lastTotalsTime = time.Now()
	}

	s.lastMetrics, s.lastEvents, s.lastMetricsTime = summary, events, now
	return summary, nil
}

// TruncateSystemLogs purges internal ClickHouse metric and trace tables to release disk space
func (s *ClickHouseService) TruncateSystemLogs(ctx context.Context) ([]string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	targetTables := []string{
		"asynchronous_metric_log",
		"metric_log",
		"trace_log",
		"text_log",
		"processors_profile_log",
		"query_log",
		"part_log",
		"crash_log",
	}

	var truncated []string
	for _, tbl := range targetTables {
		var exists uint8
		_ = conn.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM system.tables WHERE database = 'system' AND name = '%s'", tbl)).Scan(&exists)
		if exists == 1 {
			if err := conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE system.`%s`", tbl)); err == nil {
				truncated = append(truncated, tbl)
			}
		}
	}
	return truncated, nil
}

type ClusterNodeInfo struct {
	Cluster     string `json:"cluster"`
	ShardNum    uint32 `json:"shard_num"`
	ShardWeight uint32 `json:"shard_weight"`
	ReplicaNum  uint32 `json:"replica_num"`
	HostName    string `json:"host_name"`
	HostAddress string `json:"host_address"`
	Port        uint16 `json:"port"`
	IsLocal     bool   `json:"is_local"`
	User        string `json:"user"`
	DefaultDb   string `json:"default_database"`
}

type ReplicaInfo struct {
	Database        string    `json:"database"`
	Table           string    `json:"table"`
	IsLeader        bool      `json:"is_leader"`
	IsReadonly      bool      `json:"is_readonly"`
	AbsoluteDelay   uint64    `json:"absolute_delay"`
	QueueSize       uint32    `json:"queue_size"`
	InsertsInQueue  uint32    `json:"inserts_in_queue"`
	MergesInQueue   uint32    `json:"merges_in_queue"`
	LogPointer      uint64    `json:"log_pointer"`
	LastQueueUpdate time.Time `json:"last_queue_update"`
}

type DictionaryInfo struct {
	Database         string    `json:"database"`
	Name             string    `json:"name"`
	Status           string    `json:"status"` // LOADED, LOADING, FAILED, NOT_LOADED
	Origin           string    `json:"origin"`
	Type             string    `json:"type"`
	ElementCount     uint64    `json:"element_count"`
	BytesAllocated   uint64    `json:"bytes_allocated"`
	Source           string    `json:"source"`
	LoadingStartTime time.Time `json:"loading_start_time"`
	LastException    string    `json:"last_exception"`
}

func (s *ClickHouseService) GetClusters(ctx context.Context) ([]ClusterNodeInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			cluster,
			shard_num,
			shard_weight,
			replica_num,
			host_name,
			host_address,
			port,
			is_local,
			user,
			default_database
		FROM system.clusters
		ORDER BY cluster, shard_num, replica_num
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return []ClusterNodeInfo{}, nil // Gracefully handle standalone ClickHouse with no cluster defined
	}
	defer rows.Close()

	var list []ClusterNodeInfo
	for rows.Next() {
		var c ClusterNodeInfo
		var isLocal uint8
		if err := rows.Scan(
			&c.Cluster,
			&c.ShardNum,
			&c.ShardWeight,
			&c.ReplicaNum,
			&c.HostName,
			&c.HostAddress,
			&c.Port,
			&isLocal,
			&c.User,
			&c.DefaultDb,
		); err != nil {
			return nil, err
		}
		c.IsLocal = (isLocal == 1)
		list = append(list, c)
	}
	return list, nil
}

func (s *ClickHouseService) GetReplicas(ctx context.Context) ([]ReplicaInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			database,
			table,
			is_leader,
			is_readonly,
			absolute_delay,
			queue_size,
			inserts_in_queue,
			merges_in_queue,
			log_pointer,
			last_queue_update
		FROM system.replicas
		ORDER BY database, table
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return []ReplicaInfo{}, nil // Return empty if no replicated tables
	}
	defer rows.Close()

	var list []ReplicaInfo
	for rows.Next() {
		var r ReplicaInfo
		var isLeader, isReadonly uint8
		if err := rows.Scan(
			&r.Database,
			&r.Table,
			&isLeader,
			&isReadonly,
			&r.AbsoluteDelay,
			&r.QueueSize,
			&r.InsertsInQueue,
			&r.MergesInQueue,
			&r.LogPointer,
			&r.LastQueueUpdate,
		); err != nil {
			return nil, err
		}
		r.IsLeader = (isLeader == 1)
		r.IsReadonly = (isReadonly == 1)
		list = append(list, r)
	}
	return list, nil
}

func (s *ClickHouseService) GetDictionaries(ctx context.Context) ([]DictionaryInfo, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			database,
			name,
			status,
			origin,
			type,
			element_count,
			bytes_allocated,
			source,
			loading_start_time,
			last_exception
		FROM system.dictionaries
		ORDER BY name
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return []DictionaryInfo{}, nil
	}
	defer rows.Close()

	var list []DictionaryInfo
	for rows.Next() {
		var d DictionaryInfo
		if err := rows.Scan(
			&d.Database,
			&d.Name,
			&d.Status,
			&d.Origin,
			&d.Type,
			&d.ElementCount,
			&d.BytesAllocated,
			&d.Source,
			&d.LoadingStartTime,
			&d.LastException,
		); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (s *ClickHouseService) ReloadDictionary(ctx context.Context, dbName, dictName string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	target := fmt.Sprintf("`%s`.`%s`", escapeIdent(dbName), escapeIdent(dictName))
	if dbName == "" {
		target = fmt.Sprintf("`%s`", escapeIdent(dictName))
	}
	return conn.Exec(ctx, fmt.Sprintf("SYSTEM RELOAD DICTIONARY %s", target))
}
