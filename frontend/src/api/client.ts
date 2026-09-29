import axios, { AxiosError } from 'axios'
import { notification } from 'antd'
import type {
  ApiResponse,
  DashboardSummary,
  DatabaseInfo,
  TableInfo,
  ColumnInfo,
  VisualCreateTableRequest,
  QueryResult,
  ProcessInfo,
  QueryLogItem,
  PartInfo,
  MergeInfo,
  MutationInfo,
  DiskInfo,
  ClickHouseUser,
  ConfigItemMeta,
  ConfigFileInfo,
  ConfigBackup,
  BackupRecord,
  AuditLog,
  ServiceStatus,
  OSInfo,
  SecuritySettingsData,
  AddColumnRequest,
  ImportDataRequest,
  ImportDataResult,
  ImportPreviewResult,
  ClusterNodeInfo,
  ReplicaInfo,
  DictionaryInfo,
} from '../types'

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,
})

// Request interceptor to attach JWT token
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor
apiClient.interceptors.response.use(
  (response) => {
    const res = response.data as ApiResponse
    if (res && typeof res.code === 'number' && res.code !== 0) {
      notification.error({
        title: res.message || '操作失败',
        description: res.error_detail
          ? `${res.error_detail} \n${res.suggestion ? `建议: ${res.suggestion}` : ''}`
          : res.suggestion || '请检查系统状态',
        duration: 6,
      })
      return Promise.reject(new Error(res.message || 'Error'))
    }
    return response
  },
  (error: AxiosError<ApiResponse>) => {
    if (error.response?.status === 401 && error.config?.url !== '/auth/login') {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      window.dispatchEvent(new Event('storage'))
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
      return Promise.reject(error)
    }

    const errData = error.response?.data
    const msg = errData?.message || error.message || '网络请求错误'
    const detail = errData?.error_detail || ''
    const suggestion = errData?.suggestion || ''

    notification.error({
      title: msg,
      description: `${detail} ${suggestion ? `\n建议: ${suggestion}` : ''}`,
      duration: 6,
    })

    return Promise.reject(error)
  }
)

export const api = {
  // Auth
  login: (data: { username: string; password: string }) =>
    apiClient.post<ApiResponse<{ token: string; username: string; role: string; must_change_password: boolean }>>('/auth/login', data),
  changePassword: (data: { old_password: string; new_password: string }) =>
    apiClient.post<ApiResponse>('/auth/change-password', data),
  getMe: () => apiClient.get<ApiResponse<{ id: number; username: string; role: string; must_change_password: boolean }>>('/auth/me'),

  // Dashboard
  getDashboardSummary: () => apiClient.get<ApiResponse<DashboardSummary>>('/dashboard/summary'),
  getRealtimeMetrics: () => apiClient.get<ApiResponse>('/dashboard/metrics'),

  // Service Management
  getServiceStatus: () => apiClient.get<ApiResponse<ServiceStatus>>('/service/status'),
  startService: () => apiClient.post<ApiResponse>('/service/start'),
  stopService: (confirmation: string) => apiClient.post<ApiResponse>('/service/stop', { confirmation }),
  restartService: () => apiClient.post<ApiResponse>('/service/restart'),
  reloadService: () => apiClient.post<ApiResponse>('/service/reload'),
  enableService: () => apiClient.post<ApiResponse>('/service/enable'),
  disableService: () => apiClient.post<ApiResponse>('/service/disable'),
  validateServiceConfig: () => apiClient.post<ApiResponse<{ output: string }>>('/service/validate'),
  truncateSystemLogs: () => apiClient.post<ApiResponse<{ truncated_tables: string[] }>>('/service/truncate-logs'),

  // Installer
  getInstallerInfo: () =>
    apiClient.get<ApiResponse<{ os_info: OSInfo; installer: string; installed: boolean; is_installing: boolean }>>('/installer/info'),
  installClickHouse: (adminPassword?: string) =>
    apiClient.post<ApiResponse>('/installer/install', { admin_password: adminPassword }),
  uninstallClickHouse: (confirmation: string, purgeData: boolean) =>
    apiClient.post<ApiResponse>('/installer/uninstall', { confirmation, purge_data: purgeData }),
  upgradeClickHouse: () => apiClient.post<ApiResponse>('/installer/upgrade'),

  // Config Center
  getFormConfig: () => apiClient.get<ApiResponse<ConfigItemMeta[]>>('/config/form'),
  saveFormConfig: (values: Record<string, string>) => apiClient.post<ApiResponse>('/config/form', { values }),
  listConfigFiles: () => apiClient.get<ApiResponse<ConfigFileInfo[]>>('/config/files'),
  readConfigFile: (path: string) => apiClient.get<ApiResponse<{ path: string; content: string }>>('/config/file', { params: { path } }),
  saveConfigFile: (path: string, content: string, reason?: string) =>
    apiClient.post<ApiResponse>('/config/file', { path, content, reason }),
  validateXML: (content: string) => apiClient.post<ApiResponse<{ valid: boolean; message: string; line?: number }>>('/config/validate-xml', { content }),
  listConfigBackups: (path: string) => apiClient.get<ApiResponse<ConfigBackup[]>>('/config/backups', { params: { path } }),
  rollbackConfig: (backupId: number) => apiClient.post<ApiResponse>('/config/rollback', { backup_id: backupId }),

  // Databases
  listDatabases: () => apiClient.get<ApiResponse<DatabaseInfo[]>>('/databases'),
  createDatabase: (data: { name: string; engine?: string; comment?: string }) =>
    apiClient.post<ApiResponse>('/databases', data),
  dropDatabase: (name: string, confirmation: string) =>
    apiClient.delete<ApiResponse>(`/databases/${encodeURIComponent(name)}`, { data: { confirmation } }),
  renameDatabase: (oldName: string, newName: string) =>
    apiClient.post<ApiResponse>('/databases/rename', { old_name: oldName, new_name: newName }),

  // Tables
  listTables: (database: string) => apiClient.get<ApiResponse<TableInfo[]>>('/tables', { params: { database } }),
  getTableColumns: (database: string, table: string) =>
    apiClient.get<ApiResponse<ColumnInfo[]>>('/tables/columns', { params: { database, table } }),
  getShowCreateTable: (database: string, table: string) =>
    apiClient.get<ApiResponse<{ sql: string }>>('/tables/create-sql', { params: { database, table } }),
  visualCreateTable: (req: VisualCreateTableRequest) =>
    apiClient.post<ApiResponse<{ sql: string }>>('/tables/visual-create', req),
  dropTable: (database: string, table: string, confirmation: string) =>
    apiClient.delete<ApiResponse>('/tables', { data: { database, table, confirmation } }),
  truncateTable: (database: string, table: string, confirmation: string) =>
    apiClient.post<ApiResponse>('/tables/truncate', { database, table, confirmation }),
  renameTable: (database: string, oldName: string, newName: string) =>
    apiClient.post<ApiResponse>('/tables/rename', { database, old_name: oldName, new_name: newName }),
  optimizeTable: (data: { database: string; table: string; final: boolean; partition?: string }) =>
    apiClient.post<ApiResponse>('/tables/optimize', data),
  detachTable: (database: string, table: string) => apiClient.post<ApiResponse>('/tables/detach', { database, table }),
  attachTable: (database: string, table: string) => apiClient.post<ApiResponse>('/tables/attach', { database, table }),
  addColumn: (data: AddColumnRequest) => apiClient.post<ApiResponse>('/tables/columns/add', data),
  dropColumn: (data: { database: string; table: string; column_name: string; confirmation: string }) =>
    apiClient.post<ApiResponse>('/tables/columns/drop', data),
  modifyColumnComment: (data: { database: string; table: string; column_name: string; comment: string }) =>
    apiClient.post<ApiResponse>('/tables/columns/comment', data),
  importData: (data: ImportDataRequest) => apiClient.post<ApiResponse<ImportDataResult>>('/tables/import', data),
  previewImportData: (data: { data: string; format: string }) =>
    apiClient.post<ApiResponse<ImportPreviewResult>>('/tables/import-preview', data),

  // Query & Data Browser
  executeQuery: (sql: string, maxRows = 1000, database?: string) =>
    apiClient.post<ApiResponse<QueryResult>>('/query/execute', { sql, max_rows: maxRows, database }, { timeout: 125000 }),
  browseData: (params: {
    database: string
    table: string
    page: number
    page_size: number
    sort_field?: string
    sort_order?: string
    where?: string
    columns?: string
  }) =>
    apiClient.post<ApiResponse<QueryResult & { page: number; page_size: number; total_rows_estimate: number }>>(
      '/query/browser',
      params
    ),
  listFavorites: () => apiClient.get<ApiResponse>('/query/favorites'),
  saveFavorite: (data: { title: string; sql_text: string; database?: string }) =>
    apiClient.post<ApiResponse>('/query/favorites', data),
  deleteFavorite: (id: number) => apiClient.delete<ApiResponse>(`/query/favorites/${id}`),

  // Ops & Monitoring
  getProcesses: () => apiClient.get<ApiResponse<ProcessInfo[]>>('/ops/processes'),
  killQuery: (queryId: string, confirmation: string) =>
    apiClient.post<ApiResponse>('/ops/processes/kill', { query_id: queryId, confirmation }),
  getQueryLog: (params: { min_duration_ms?: number; limit?: number; offset?: number; search?: string }) =>
    apiClient.get<ApiResponse<{ items: QueryLogItem[]; total: number }>>('/ops/query-log', { params }),
  getParts: (params?: { database?: string; table?: string; active_only?: boolean }) =>
    apiClient.get<ApiResponse<PartInfo[]>>('/ops/parts', { params }),
  getMerges: () => apiClient.get<ApiResponse<MergeInfo[]>>('/ops/merges'),
  getMutations: (params?: { database?: string; table?: string }) =>
    apiClient.get<ApiResponse<MutationInfo[]>>('/ops/mutations', { params }),
  killMutation: (data: { database: string; table: string; mutation_id: string; confirmation: string }) =>
    apiClient.post<ApiResponse>('/ops/mutations/kill', data),
  getDisks: () => apiClient.get<ApiResponse<DiskInfo[]>>('/ops/disks'),
  getClusters: () => apiClient.get<ApiResponse<ClusterNodeInfo[]>>('/ops/clusters'),
  getReplicas: () => apiClient.get<ApiResponse<ReplicaInfo[]>>('/ops/replicas'),
  getDictionaries: () => apiClient.get<ApiResponse<DictionaryInfo[]>>('/ops/dictionaries'),
  reloadDictionary: (data: { database?: string; name: string }) =>
    apiClient.post<ApiResponse>('/ops/dictionaries/reload', data),

  // Users
  listUsers: () => apiClient.get<ApiResponse<ClickHouseUser[]>>('/users'),
  createUser: (data: any) => apiClient.post<ApiResponse>('/users', data),
  alterUser: (data: any) => apiClient.put<ApiResponse>('/users', data),
  dropUser: (name: string, confirmation: string) =>
    apiClient.delete<ApiResponse>(`/users/${encodeURIComponent(name)}`, { data: { confirmation } }),
  grantPrivileges: (data: { username: string; privileges: string[]; database: string; table: string; with_grant_option?: boolean }) =>
    apiClient.post<ApiResponse>('/users/grant', data),
  revokePrivileges: (data: { username: string; privileges: string[]; database: string; table: string }) =>
    apiClient.post<ApiResponse>('/users/revoke', data),
  listProfiles: () => apiClient.get<ApiResponse<string[]>>('/users/profiles'),

  // Logs
  getLogs: (params: { type: string; lines?: number; filter?: string; level?: string }) =>
    apiClient.get<ApiResponse<{ lines: string[]; total: number; file_path: string }>>('/logs', { params }),

  // Backups
  listBackups: () => apiClient.get<ApiResponse<BackupRecord[]>>('/backups'),
  createBackup: (data: { database: string; table?: string; disk?: string; backup_id?: string }) =>
    apiClient.post<ApiResponse<{ backup_id: string }>>('/backups', data),
  restoreBackup: (data: { backup_id: string; disk?: string; confirmation: string }) =>
    apiClient.post<ApiResponse>('/backups/restore', data, { timeout: 1805000 }),
  deleteBackup: (id: number) => apiClient.delete<ApiResponse>(`/backups/${id}`),

  // Audit & Settings
  listAuditLogs: (params: { page?: number; page_size?: number; search?: string; action?: string }) =>
    apiClient.get<ApiResponse<{ items: AuditLog[]; total: number; page: number; page_size: number }>>('/audit/logs', { params }),
  getConnectionSettings: () => apiClient.get<ApiResponse>('/settings/connection'),
  updateConnectionSettings: (data: any) => apiClient.post<ApiResponse>('/settings/connection', data),
  getSecuritySettings: () => apiClient.get<ApiResponse<SecuritySettingsData>>('/settings/security'),
  updateSecuritySettings: (data: { ip_whitelist: string }) => apiClient.post<ApiResponse>('/settings/security', data),
  unlockIP: (ip: string) => apiClient.post<ApiResponse>('/settings/security/unlock', { ip }),
}
