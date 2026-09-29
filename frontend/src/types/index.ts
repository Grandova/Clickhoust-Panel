export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
  error_detail?: string
  suggestion?: string
}

export interface OSInfo {
  os: string
  platform: string
  version: string
  arch: string
  pretty_name: string
  pkg_type: string
}

export interface ServiceStatus {
  installed: boolean
  server_version: string
  client_version: string
  active: boolean
  sub_state: string
  enabled: boolean
  pid: number
  uptime: string
  uptime_sec: number
  memory_bytes: number
  status_text: string
}

export interface HostMetrics {
  cpu_percent: number
  cpu_count: number
  mem_total: number
  mem_used: number
  mem_free: number
  mem_percent: number
  disk_total: number
  disk_used: number
  disk_free: number
  disk_percent: number
  net_rx_bytes_sec: number
  net_tx_bytes_sec: number
  data_dir_bytes: number
  log_dir_bytes: number
  timestamp: number
}

export interface MetricsSummary {
  query_per_second: number
  insert_per_second: number
  select_per_second: number
  rows_read_per_second: number
  bytes_read_per_second: number
  rows_write_per_second: number
  bytes_write_per_second: number
  current_connections: number
  current_queries: number
  memory_tracking: number
  max_memory_tracking: number
  disk_usage_bytes: number
  total_databases: number
  total_tables: number
  total_rows: number
  total_bytes: number
  compressed_bytes: number
  uncompressed_bytes: number
  today_queries: number
  today_failed_queries: number
  raw_metrics: Record<string, number>
  timestamp: number
}

export interface DashboardSummary {
  host: HostMetrics
  service: ServiceStatus
  ch_reachable: boolean
  ch_version: string
  metrics?: MetricsSummary
  os_info: OSInfo
}

export interface DatabaseInfo {
  name: string
  engine: string
  data_path: string
  metadata_path: string
  comment: string
  tables_count: number
  total_bytes: number
  total_rows: number
}

export interface TableInfo {
  database: string
  name: string
  engine: string
  total_rows: number
  total_bytes: number
  compressed_bytes: number
  uncompressed_bytes: number
  parts_count: number
  primary_key: string
  order_by: string
  partition_key: string
  ttl: string
  last_modified: string
  comment: string
}

export interface ColumnInfo {
  name: string
  type: string
  default_kind: string
  default_expression: string
  comment: string
  is_in_primary_key: boolean
  is_in_partition_key: boolean
  compression_codec: string
}

export interface VisualColumn {
  name: string
  type: string
  nullable: boolean
  default: string
  comment: string
  codec: string
}

export interface VisualCreateTableRequest {
  database: string
  table_name: string
  engine: string
  engine_args?: string
  columns: VisualColumn[]
  order_by?: string
  primary_key?: string
  partition_by?: string
  sample_by?: string
  ttl?: string
  settings?: string
  execute_now: boolean
}

export interface QueryColumn {
  name: string
  type: string
}

export interface QueryResult {
  query_id: string
  columns: QueryColumn[]
  rows: Record<string, any>[]
  total_rows: number
  read_rows?: number
  read_bytes?: number
  elapsed_ms: number
  memory_bytes?: number
}

export interface ProcessInfo {
  query_id: string
  user: string
  address: string
  elapsed: number
  read_rows: number
  read_bytes: number
  memory_usage: number
  query: string
  start_time: string
}

export interface QueryLogItem {
  query_id: string
  query: string
  user: string
  client_address: string
  database: string
  type: string
  event_time: string
  duration_ms: number
  read_rows: number
  read_bytes: number
  result_rows: number
  result_bytes: number
  memory_peak: number
  exception_code: number
  exception_message: string
}

export interface PartInfo {
  database: string
  table: string
  partition: string
  name: string
  rows: number
  bytes_on_disk: number
  data_compressed: number
  data_uncompressed: number
  marks: number
  active: boolean
  modification_time: string
  disk_name: string
  level: number
}

export interface MergeInfo {
  database: string
  table: string
  elapsed: number
  progress: number
  num_parts: number
  result_part_name: string
  rows_read: number
  rows_written: number
  memory_usage: number
  thread_id: number
}

export interface MutationInfo {
  database: string
  table: string
  mutation_id: string
  command: string
  create_time: string
  parts_to_do: number
  is_done: boolean
  latest_failed_part: string
  latest_fail_reason: string
  latest_fail_time: string
}

export interface DiskInfo {
  name: string
  path: string
  free_space: number
  total_space: number
  used_space: number
  used_percent: number
  keep_free_space: number
  type: string
  storage_policy: string
}

export interface ClickHouseUser {
  name: string
  storage?: string
  auth_type: string
  host_ip: string[]
  default_database: string
  profile: string
  quota: string
  grants: string[]
}

export interface ConfigItemMeta {
  key: string
  category: string
  current_value: string
  default_value: string
  description: string
  type: 'int' | 'float' | 'string' | 'select'
  requires_restart: boolean
  recommended: string
  options?: string[]
}

export interface ConfigFileInfo {
  name: string
  relative_path: string
  full_path: string
  size: number
  mod_time: string
  is_directory: boolean
}

export interface ConfigBackup {
  id: number
  file_path: string
  content: string
  reason: string
  created_by: string
  created_at: string
}

export interface BackupRecord {
  id: number
  backup_id: string
  database_name: string
  table_name: string
  disk: string
  size: number
  status: string
  started_at: string
  completed_at?: string
  error?: string
}

export interface AuditLog {
  id: number
  username: string
  ip: string
  action: string
  target: string
  result: string
  error?: string
  created_at: string
}

export interface LockedIPInfo {
  ip: string
  failed_attempts: number
  locked_at: string
  remaining_mins: number
}

export interface SecuritySettingsData {
  ip_whitelist: string
  client_ip: string
  locked_ips: LockedIPInfo[]
}

export interface AddColumnRequest {
  database: string
  table: string
  column_name: string
  type: string
  nullable?: boolean
  default_expression?: string
  comment?: string
  codec?: string
  after_column?: string
}

export interface ImportDataRequest {
  database: string
  table: string
  format: string
  data: string
}

export interface ImportDataResult {
  inserted_rows: number
  elapsed_ms: number
}

export interface ImportPreviewResult {
  headers: string[]
  rows: string[][]
  total_lines_estimate: number
}

export interface ClusterNodeInfo {
  cluster: string
  shard_num: number
  shard_weight: number
  replica_num: number
  host_name: string
  host_address: string
  port: number
  is_local: boolean
  user: string
  default_database: string
}

export interface ReplicaInfo {
  database: string
  table: string
  is_leader: boolean
  is_readonly: boolean
  absolute_delay: number
  queue_size: number
  inserts_in_queue: number
  merges_in_queue: number
  log_pointer: number
  last_queue_update: string
}

export interface DictionaryInfo {
  database: string
  name: string
  status: string
  origin: string
  type: string
  element_count: number
  bytes_allocated: number
  source: string
  loading_start_time: string
  last_exception: string
}

