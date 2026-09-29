import React, { useState, useEffect, useRef } from 'react'
import {
  Row,
  Col,
  Card,
  Tree,
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Checkbox,
  Tooltip,
  message,
  Popconfirm,
  Dropdown,
  Tabs,
  Typography,
  Drawer,
  Upload,
  Alert,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DatabaseOutlined,
  TableOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  SyncOutlined,
  CodeOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  CopyOutlined,
  DisconnectOutlined,
  LinkOutlined,
  CloudUploadOutlined,
  UploadOutlined,
  EyeOutlined,
  ClearOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type {
  DatabaseInfo,
  TableInfo,
  ColumnInfo,
  VisualColumn,
  VisualCreateTableRequest,
  ImportPreviewResult,
} from '../../types'
import { formatBytes, formatNumber, formatDateTime } from '../../utils/format'

const { Text, Title } = Typography

interface DatabasesPageProps {
  onNavigateToBrowser: (db: string, table: string) => void
  onNavigateToSQL: (sql: string) => void
  isDark?: boolean
}

export const DatabasesPage: React.FC<DatabasesPageProps> = ({
  onNavigateToBrowser,
  onNavigateToSQL,
  isDark = false,
}) => {
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [selectedDb, setSelectedDb] = useState<string>('default')
  const [tables, setTables] = useState<TableInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [tablesLoading, setTablesLoading] = useState(false)
  const tableRequest = useRef(0)

  // Create DB Modal
  const [createDbOpen, setCreateDbOpen] = useState(false)
  const [dbForm] = Form.useForm()

  // Drop DB Modal
  const [dropDbModalOpen, setDropDbModalOpen] = useState(false)
  const [dropDbConfirm, setDropDbConfirm] = useState('')
  const [dbToDrop, setDbToDrop] = useState('')

  // Show Create Table SQL Modal
  const [showSqlModal, setShowSqlModal] = useState(false)
  const [tableSql, setTableSql] = useState('')
  const [sqlTableName, setSqlTableName] = useState('')

  // Visual Table Creator Modal
  const [visualModalOpen, setVisualModalOpen] = useState(false)
  const [visualForm] = Form.useForm()
  const [visualColumns, setVisualColumns] = useState<VisualColumn[]>([
    { name: 'id', type: 'UInt64', nullable: false, default: '', comment: 'Primary ID', codec: '' },
    { name: 'created_at', type: 'DateTime', nullable: false, default: 'now()', comment: 'Creation time', codec: '' },
    { name: 'title', type: 'String', nullable: false, default: '', comment: 'Item title', codec: '' },
  ])
  const [generatedSql, setGeneratedSql] = useState('')

  // Drop Table Modal
  const [dropTableModalOpen, setDropTableModalOpen] = useState(false)
  const [tableToDrop, setTableToDrop] = useState('')
  const [dropTableConfirm, setDropTableConfirm] = useState('')

  // Truncate Table Modal
  const [truncateModalOpen, setTruncateModalOpen] = useState(false)
  const [tableToTruncate, setTableToTruncate] = useState('')
  const [truncateConfirm, setTruncateConfirm] = useState('')

  // Columns Drawer
  const [colsDrawerOpen, setColsDrawerOpen] = useState(false)
  const [colsDrawerTable, setColsDrawerTable] = useState('')
  const [currentColumns, setCurrentColumns] = useState<ColumnInfo[]>([])
  const [colsLoading, setColsLoading] = useState(false)

  // Add Column Modal
  const [addColumnOpen, setAddColumnOpen] = useState(false)
  const [addColumnForm] = Form.useForm()

  // Drop Column Modal
  const [dropColOpen, setDropColOpen] = useState(false)
  const [colToDrop, setColToDrop] = useState('')
  const [dropColConfirm, setDropColConfirm] = useState('')

  // Edit Column Comment Modal
  const [editCommentOpen, setEditCommentOpen] = useState(false)
  const [colToComment, setColToComment] = useState('')
  const [colCommentValue, setColCommentValue] = useState('')

  // Data Import Wizard Modal
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importTable, setImportTable] = useState('')
  const [importFormat, setImportFormat] = useState('CSV')
  const [importDataText, setImportDataText] = useState('')
  const [importPreviewData, setImportPreviewData] = useState<ImportPreviewResult | null>(null)
  const [importing, setImporting] = useState(false)
  const [previewing, setPreviewing] = useState(false)

  const fetchDatabases = async () => {
    setLoading(true)
    try {
      const res = await api.listDatabases()
      if (res.data.data) {
        setDatabases(res.data.data)
        if (!res.data.data.some((d) => d.name === selectedDb) && res.data.data.length > 0) {
          setSelectedDb(res.data.data[0].name)
        }
      }
    } catch {
      setDatabases([])
    } finally {
      setLoading(false)
    }
  }

  const fetchTables = async (dbName: string) => {
    if (!dbName) return
    const id = ++tableRequest.current
    setTablesLoading(true)
    try {
      const res = await api.listTables(dbName)
      if (id === tableRequest.current && res.data.data) {
        setTables(res.data.data)
      }
    } catch {
      if (id === tableRequest.current) setTables([])
    } finally {
      if (id === tableRequest.current) setTablesLoading(false)
    }
  }

  useEffect(() => {
    fetchDatabases()
  }, [])

  useEffect(() => {
    if (selectedDb) {
      setTables([])
      fetchTables(selectedDb)
    }
  }, [selectedDb])

  const handleCreateDatabase = async (values: any) => {
    try {
      await api.createDatabase(values)
      message.success('数据库创建成功')
      setCreateDbOpen(false)
      dbForm.resetFields()
      fetchDatabases()
    } catch {
      // Handled
    }
  }

  const handleDropDatabase = async () => {
    const expected = `DROP ${dbToDrop}`
    if (dropDbConfirm !== expected) {
      message.error(`请输入 '${expected}' 确认删除`)
      return
    }
    try {
      await api.dropDatabase(dbToDrop, dropDbConfirm)
      message.success(`数据库 ${dbToDrop} 已成功删除`)
      setDropDbModalOpen(false)
      setDropDbConfirm('')
      fetchDatabases()
    } catch {
      // Handled
    }
  }

  const handleShowCreateSql = async (table: string) => {
    try {
      const res = await api.getShowCreateTable(selectedDb, table)
      setSqlTableName(table)
      setTableSql(res.data.data?.sql || '')
      setShowSqlModal(true)
    } catch {
      // Handled
    }
  }

  const handleDropTable = async () => {
    const expected = `DROP ${tableToDrop}`
    if (dropTableConfirm !== expected) {
      message.error(`请输入 '${expected}' 确认删除`)
      return
    }
    try {
      await api.dropTable(selectedDb, tableToDrop, dropTableConfirm)
      message.success(`数据表 ${tableToDrop} 已成功删除`)
      setDropTableModalOpen(false)
      setDropTableConfirm('')
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  const handleTruncateTable = async () => {
    const expected = `TRUNCATE ${tableToTruncate}`
    if (truncateConfirm !== expected) {
      message.error(`请输入 '${expected}' 确认清空`)
      return
    }
    try {
      await api.truncateTable(selectedDb, tableToTruncate, truncateConfirm)
      message.success(`数据表 ${tableToTruncate} 数据已清空`)
      setTruncateModalOpen(false)
      setTruncateConfirm('')
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  const handleOptimizeTable = async (table: string) => {
    try {
      await api.optimizeTable({ database: selectedDb, table, final: true })
      message.success(`数据表 ${table} OPTIMIZE FINAL 执行已触发`)
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  // Column management handlers
  const fetchTableColumns = async (table: string) => {
    setColsLoading(true)
    try {
      const res = await api.getTableColumns(selectedDb, table)
      if (res.data.data) {
        setCurrentColumns(res.data.data)
      }
    } finally {
      setColsLoading(false)
    }
  }

  const handleOpenColsDrawer = (table: string) => {
    setColsDrawerTable(table)
    setColsDrawerOpen(true)
    fetchTableColumns(table)
  }

  const handleAddColumn = async (values: any) => {
    try {
      await api.addColumn({
        database: selectedDb,
        table: colsDrawerTable,
        column_name: values.column_name,
        type: values.type,
        nullable: values.nullable,
        default_expression: values.default_expression,
        comment: values.comment,
        codec: values.codec,
        after_column: values.after_column,
      })
      message.success(`字段 [${values.column_name}] 添加成功`)
      setAddColumnOpen(false)
      addColumnForm.resetFields()
      fetchTableColumns(colsDrawerTable)
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  const handleDropColumn = async () => {
    const expected = `DROP ${colToDrop}`
    if (dropColConfirm !== expected) {
      message.error(`请输入 '${expected}' 确认删除`)
      return
    }
    try {
      await api.dropColumn({
        database: selectedDb,
        table: colsDrawerTable,
        column_name: colToDrop,
        confirmation: dropColConfirm,
      })
      message.success(`字段 [${colToDrop}] 已删除`)
      setDropColOpen(false)
      setDropColConfirm('')
      fetchTableColumns(colsDrawerTable)
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  const handleSaveColumnComment = async () => {
    try {
      await api.modifyColumnComment({
        database: selectedDb,
        table: colsDrawerTable,
        column_name: colToComment,
        comment: colCommentValue,
      })
      message.success(`字段 [${colToComment}] 注释修改成功`)
      setEditCommentOpen(false)
      fetchTableColumns(colsDrawerTable)
    } catch {
      // Handled
    }
  }

  // Import wizard handlers
  const handleOpenImport = (table?: string) => {
    setImportTable(table || (tables[0]?.name || ''))
    setImportFormat('CSV')
    setImportDataText('')
    setImportPreviewData(null)
    setImportModalOpen(true)
  }

  const handlePreviewImport = async () => {
    if (!importDataText.trim()) {
      message.warning('请输入或上传导入数据')
      return
    }
    setPreviewing(true)
    try {
      const res = await api.previewImportData({ data: importDataText, format: importFormat })
      if (res.data.data) {
        setImportPreviewData(res.data.data)
      }
    } finally {
      setPreviewing(false)
    }
  }

  const handleExecuteImport = async () => {
    if (!importTable) {
      message.error('请选择目标数据表')
      return
    }
    if (!importDataText.trim()) {
      message.error('导入数据不能为空')
      return
    }
    setImporting(true)
    try {
      const res = await api.importData({
        database: selectedDb,
        table: importTable,
        format: importFormat,
        data: importDataText,
      })
      message.success(res.data.message || '数据导入成功！')
      setImportModalOpen(false)
      fetchTables(selectedDb)
    } finally {
      setImporting(false)
    }
  }

  const handleFileUpload = (file: File) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      if (content) {
        setImportDataText(content)
        message.success(`已载入文件 [${file.name}] (${(file.size / 1024).toFixed(1)} KB)`)
      }
    }
    reader.readAsText(file)
    return false // prevent auto upload
  }

  // Visual column helper
  const addColumn = () => {
    setVisualColumns([
      ...visualColumns,
      { name: '', type: 'String', nullable: false, default: '', comment: '', codec: '' },
    ])
  }

  const removeColumn = (index: number) => {
    setVisualColumns(visualColumns.filter((_, i) => i !== index))
  }

  const updateColumn = (index: number, field: keyof VisualColumn, value: any) => {
    const next = [...visualColumns]
    next[index] = { ...next[index], [field]: value }
    setVisualColumns(next)
  }

  const handlePreviewVisualSql = async () => {
    let values: any
    try {
      values = await visualForm.validateFields(['table_name'])
    } catch {
      message.warning('请先输入数据表名称')
      return
    }
    const tableName = values.table_name?.trim()
    if (!tableName) {
      message.warning('请先输入数据表名称')
      return
    }
    const validCols = visualColumns.filter(c => c.name.trim() !== '')
    if (validCols.length === 0) {
      message.warning('请至少添加一个包含有效字段名的列')
      return
    }
    const allValues = visualForm.getFieldsValue()
    const req: VisualCreateTableRequest = {
      database: selectedDb,
      table_name: tableName,
      engine: allValues.engine || 'MergeTree',
      engine_args: allValues.engine_args,
      columns: visualColumns,
      order_by: allValues.order_by || 'id',
      primary_key: allValues.primary_key,
      partition_by: allValues.partition_by,
      ttl: allValues.ttl,
      settings: allValues.settings,
      execute_now: false,
    }
    try {
      const res = await api.visualCreateTable(req)
      setGeneratedSql(res.data.data?.sql || '')
    } catch {
      // Handled
    }
  }

  const handleExecuteVisualSql = async () => {
    let values: any
    try {
      values = await visualForm.validateFields()
    } catch {
      return
    }
    const tableName = values.table_name?.trim()
    if (!tableName) {
      message.warning('请输入数据表名称')
      return
    }
    const validCols = visualColumns.filter(c => c.name.trim() !== '')
    if (validCols.length === 0) {
      message.warning('请至少添加一个包含有效字段名的列')
      return
    }
    const req: VisualCreateTableRequest = {
      database: selectedDb,
      table_name: tableName,
      engine: values.engine || 'MergeTree',
      engine_args: values.engine_args,
      columns: visualColumns,
      order_by: values.order_by || 'id',
      primary_key: values.primary_key,
      partition_by: values.partition_by,
      ttl: values.ttl,
      settings: values.settings,
      execute_now: true,
    }
    try {
      await api.visualCreateTable(req)
      message.success(`数据表 ${tableName} 创建成功`)
      setVisualModalOpen(false)
      visualForm.resetFields()
      fetchTables(selectedDb)
    } catch {
      // Handled
    }
  }

  const tableColumns: ColumnsType<TableInfo> = [
    {
      title: '表名',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record) => (
        <Space>
          <TableOutlined style={{ color: '#987b2e' }} />
          <Button
            type="link"
            style={{ padding: 0, fontWeight: 600 }}
            onClick={() => onNavigateToBrowser(selectedDb, name)}
          >
            {name}
          </Button>
          {record.comment && <Text type="secondary" style={{ fontSize: 12 }}>({record.comment})</Text>}
        </Space>
      ),
    },
    {
      title: 'Engine',
      dataIndex: 'engine',
      key: 'engine',
      render: (engine: string) => <Tag color="cyan">{engine}</Tag>,
    },
    {
      title: '总行数',
      dataIndex: 'total_rows',
      key: 'total_rows',
      sorter: (a, b) => Number(a.total_rows) - Number(b.total_rows),
      render: (rows: number) => formatNumber(rows),
    },
    {
      title: '磁盘容量 (压缩后)',
      dataIndex: 'compressed_bytes',
      key: 'compressed_bytes',
      sorter: (a, b) => Number(a.compressed_bytes) - Number(b.compressed_bytes),
      render: (b: number, r) => (
        <Tooltip title={`未压缩: ${formatBytes(r.uncompressed_bytes || 0)}`}>
          <span>{formatBytes(b || r.total_bytes || 0)}</span>
        </Tooltip>
      ),
    },
    {
      title: 'Part 数量',
      dataIndex: 'parts_count',
      key: 'parts_count',
      render: (parts: number) => <Tag>{parts}</Tag>,
    },
    {
      title: 'Primary Key / Order By',
      key: 'keys',
      render: (_, r) => (
        <span style={{ fontSize: 12, fontFamily: 'monospace' }}>
          {r.order_by || r.primary_key || '-'}
        </span>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 280,
      render: (_, record) => (
        <Space size="small">
          <Button
            size="small"
            type="primary"
            ghost
            icon={<SearchOutlined />}
            onClick={() => onNavigateToBrowser(selectedDb, record.name)}
          >
            浏览
          </Button>

          <Button
            size="small"
            icon={<TableOutlined />}
            onClick={() => handleOpenColsDrawer(record.name)}
          >
            结构/字段
          </Button>

          <Button
            size="small"
            icon={<CloudUploadOutlined />}
            onClick={() => handleOpenImport(record.name)}
          >
            导入
          </Button>

          <Dropdown menu={{ items: [
            { key: 'ddl', label: '建表语句', onClick: () => handleShowCreateSql(record.name) },
            { key: 'optimize', label: '合并数据', onClick: () => handleOptimizeTable(record.name) },
            { type: 'divider' },
            { key: 'truncate', label: '清空数据', danger: true, onClick: () => { setTableToTruncate(record.name); setTruncateModalOpen(true) } },
            { key: 'drop', label: '删除表', danger: true, onClick: () => { setTableToDrop(record.name); setDropTableModalOpen(true) } },
          ] }}><Button size="small">更多</Button></Dropdown>
        </Space>
      ),
    },
  ]

  const clickhouseTypes = [
    'UInt8', 'UInt16', 'UInt32', 'UInt64', 'UInt128', 'UInt256',
    'Int8', 'Int16', 'Int32', 'Int64', 'Int128', 'Int256',
    'Float32', 'Float64', 'Decimal(18, 4)',
    'String', 'FixedString(32)',
    'Date', 'Date32', 'DateTime', 'DateTime64(3)',
    'UUID', 'IPv4', 'IPv6',
    'Array(String)', 'Array(UInt64)', 'Map(String, String)',
    'LowCardinality(String)',
  ]

  const isSystemDb = (name: string) => ['system', 'information_schema', 'INFORMATION_SCHEMA'].includes(name)
  const [truncatingLogs, setTruncatingLogs] = useState(false)

  const handleTruncateSystemLogs = async () => {
    Modal.confirm({
      title: '确认清理系统日志表数据？',
      content: '此操作将清空 system.asynchronous_metric_log、system.metric_log、system.trace_log 等内部监控日志数据，立即释放磁盘空间，完全不影响您的业务数据与表结构。',
      okText: '立即清理',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setTruncatingLogs(true)
        try {
          const res = await api.truncateSystemLogs()
          message.success(res.data.message || '系统日志表已成功清空！')
          fetchTables(selectedDb)
          fetchDatabases()
        } finally {
          setTruncatingLogs(false)
        }
      },
    })
  }

  const sortedDatabases = [...databases].sort((a, b) => {
    if (a.name === 'default') return -1
    if (b.name === 'default') return 1
    const aSys = isSystemDb(a.name)
    const bSys = isSystemDb(b.name)
    if (aSys && !bSys) return 1
    if (!aSys && bSys) return -1
    return a.name.localeCompare(b.name)
  })

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Row gutter={[16, 16]}>
        {/* Left: Databases Tree List */}
        <Col xs={24} md={6}>
          <Card
            title={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Space>
                  <DatabaseOutlined />
                  <span>数据库列表 ({databases.length})</span>
                </Space>
                <Button
                  size="small"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setCreateDbOpen(true)}
                >
                  新建
                </Button>
              </div>
            }
            size="small"
            style={{ borderRadius: 10, minHeight: 600 }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {sortedDatabases.map((db) => {
                const isSelected = db.name === selectedDb
                const isSys = isSystemDb(db.name)
                return (
                  <div
                    key={db.name}
                    onClick={() => setSelectedDb(db.name)}
                    style={{
                      padding: '10px 12px',
                      borderRadius: 8,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      background: isSelected ? 'rgba(250, 173, 20, 0.12)' : 'transparent',
                      border: isSelected ? '1px solid #faad14' : '1px solid transparent',
                      transition: 'all 0.2s ease',
                    }}
                  >
                    <Space>
                      <DatabaseOutlined style={{ color: isSelected ? '#faad14' : 'inherit' }} />
                      <span style={{ fontWeight: isSelected ? 600 : 400 }}>{db.name}</span>
                    </Space>
                    <Space size="small">
                      {isSys && <Tag color="default" style={{ fontSize: 10, margin: 0 }}>系统库</Tag>}
                      <Tag style={{ fontSize: 11 }}>{db.tables_count} 表</Tag>
                      {!isSys && db.name !== 'default' && (
                        <Button
                          size="small"
                          type="text"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={(e) => {
                            e.stopPropagation()
                            setDbToDrop(db.name)
                            setDropDbModalOpen(true)
                          }}
                        />
                      )}
                    </Space>
                  </div>
                )
              })}
            </div>
          </Card>
        </Col>

        {/* Right: Tables in Selected Database */}
        <Col xs={24} md={18}>
          <Card
            title={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Space>
                  <TableOutlined />
                  <span>数据库: <strong>{selectedDb}</strong></span>
                  <Tag color="blue">{tables.length} 张数据表</Tag>
                  {isSystemDb(selectedDb) && <Tag color="default">系统只读库</Tag>}
                </Space>
                <Space>
                  {!isSystemDb(selectedDb) ? (
                    <>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          visualForm.setFieldsValue({
                            table_name: '',
                            engine: 'MergeTree',
                            order_by: 'id',
                            primary_key: 'id',
                          })
                          setVisualModalOpen(true)
                        }}
                      >
                        可视化建表
                      </Button>
                      <Button
                        type="primary"
                        ghost
                        icon={<CloudUploadOutlined />}
                        onClick={() => handleOpenImport()}
                      >
                        数据导入向导
                      </Button>
                    </>
                  ) : selectedDb === 'system' ? (
                    <Button
                      danger
                      icon={<ClearOutlined />}
                      onClick={handleTruncateSystemLogs}
                      loading={truncatingLogs}
                    >
                      清理系统日志表
                    </Button>
                  ) : null}
                  <Button
                    icon={<SyncOutlined />}
                    onClick={() => fetchTables(selectedDb)}
                  >
                    刷新
                  </Button>
                </Space>
              </div>
            }
            size="small"
            style={{ borderRadius: 10, minHeight: 600 }}
          >
            {selectedDb === 'system' && (
              <Alert
                style={{ marginBottom: 16 }}
                message="ClickHouse 内置系统库 (system)"
                description="包含 ClickHouse 运行指标、日志表 (如 metric_log、asynchronous_metric_log) 及字典元数据。本面板已对系统库与用户业务统计进行隔离。若磁盘空间紧张，可点击右上角「清理系统日志表」清空历史监控数据。"
                type="info"
                showIcon
              />
            )}
            <Table
              dataSource={tables}
              columns={tableColumns}
              rowKey="name"
              loading={tablesLoading}
              pagination={{ pageSize: 15, showSizeChanger: true }}
              size="middle"
            />
          </Card>
        </Col>
      </Row>

      {/* Modal: Create Database */}
      <Modal
        title="新建 ClickHouse 数据库"
        open={createDbOpen}
        onCancel={() => setCreateDbOpen(false)}
        onOk={() => dbForm.submit()}
      >
        <Form form={dbForm} layout="vertical" onFinish={handleCreateDatabase}>
          <Form.Item
            name="name"
            label="数据库名称"
            rules={[
              { required: true, message: '请输入数据库名称' },
              { pattern: /^[a-zA-Z0-9_]+$/, message: '仅支持字母、数字及下划线' },
            ]}
          >
            <Input placeholder="例如: analytics_db" />
          </Form.Item>
          <Form.Item name="engine" label="Database Engine (默认 Ordinary / Atomic)">
            <Select
              allowClear
              placeholder="默认 (Atomic 推荐)"
              options={[
                { label: 'Atomic (默认推荐)', value: 'Atomic' },
                { label: 'Ordinary (兼容模式)', value: 'Ordinary' },
                { label: 'Memory (纯内存临时库)', value: 'Memory' },
              ]}
            />
          </Form.Item>
          <Form.Item name="comment" label="备注说明">
            <Input placeholder="数据库用途说明" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Modal: Drop Database Confirmation */}
      <Modal
        title={`危险操作：删除数据库 [${dbToDrop}] 确认`}
        open={dropDbModalOpen}
        onCancel={() => {
          setDropDbModalOpen(false)
          setDropDbConfirm('')
        }}
        onOk={handleDropDatabase}
        okButtonProps={{ danger: true, disabled: dropDbConfirm !== `DROP ${dbToDrop}` }}
        okText="确认删除数据库"
      >
        <p style={{ color: '#ff4d4f' }}>
          删除数据库将永久清空该数据库下的所有表、视图及数据，操作不可撤销！
        </p>
        <p>请输入 <strong>DROP {dbToDrop}</strong> 确认删除：</p>
        <Input
          placeholder={`输入 DROP ${dbToDrop}`}
          value={dropDbConfirm}
          onChange={(e) => setDropDbConfirm(e.target.value)}
        />
      </Modal>

      {/* Modal: Show CREATE TABLE DDL */}
      <Modal
        title={`CREATE TABLE DDL: ${sqlTableName}`}
        open={showSqlModal}
        onCancel={() => setShowSqlModal(false)}
        width={720}
        footer={[
          <Button
            key="copy"
            icon={<CopyOutlined />}
            onClick={() => {
              navigator.clipboard.writeText(tableSql)
              message.success('已复制到剪贴板')
            }}
          >
            复制 SQL
          </Button>,
          <Button
            key="console"
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={() => {
              setShowSqlModal(false)
              onNavigateToSQL(tableSql)
            }}
          >
            在 SQL 控制台执行
          </Button>,
          <Button key="close" onClick={() => setShowSqlModal(false)}>
            关闭
          </Button>,
        ]}
      >
        <pre
          style={{
            background: isDark ? '#141414' : '#f8fafc',
            color: isDark ? '#faad14' : '#0f172a',
            border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
            padding: 16,
            borderRadius: 8,
            maxHeight: 400,
            overflow: 'auto',
            fontFamily: 'Consolas, Monaco, monospace',
            fontSize: 13,
          }}
        >
          {tableSql}
        </pre>
      </Modal>

      {/* Modal: Visual Table Creator */}
      <Modal
        title={`在数据库 [${selectedDb}] 中可视化新建表`}
        open={visualModalOpen}
        onCancel={() => setVisualModalOpen(false)}
        width={960}
        footer={[
          <Button key="preview" onClick={handlePreviewVisualSql}>
            预览 SQL
          </Button>,
          <Button key="create" type="primary" onClick={handleExecuteVisualSql}>
            立即创建表
          </Button>,
        ]}
      >
        <Form form={visualForm} layout="vertical">
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="table_name"
                label="表名"
                rules={[{ required: true, message: '请输入表名' }]}
              >
                <Input placeholder="例如: user_events" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="engine" label="Engine 表引擎" initialValue="MergeTree">
                <Select
                  options={[
                    { label: 'MergeTree (核心推荐引擎)', value: 'MergeTree' },
                    { label: 'ReplacingMergeTree (按排序键去重)', value: 'ReplacingMergeTree' },
                    { label: 'SummingMergeTree (数值聚合求和)', value: 'SummingMergeTree' },
                    { label: 'AggregatingMergeTree (聚合中间状态)', value: 'AggregatingMergeTree' },
                    { label: 'CollapsingMergeTree (标记消除更新/删除)', value: 'CollapsingMergeTree' },
                    { label: 'Memory (内存临时表)', value: 'Memory' },
                    { label: 'Log (简单轻量日志表)', value: 'Log' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="engine_args"
                label="引擎参数 (Engine Args)"
                tooltip="如 ReplacingMergeTree 的版本列 (例如: ver) 或 CollapsingMergeTree 的标记列 (例如: sign)"
              >
                <Input placeholder="可选，例如: ver 或 sign" />
              </Form.Item>
            </Col>
          </Row>

          {/* Column Builder */}
          <div style={{ marginBottom: 12, fontWeight: 600, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>字段列表 ({visualColumns.length})</span>
            <Button size="small" type="dashed" icon={<PlusOutlined />} onClick={addColumn}>
              添加字段
            </Button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxHeight: 260, overflowY: 'auto', paddingRight: 4 }}>
            {visualColumns.map((col, idx) => (
              <Row key={idx} gutter={8} align="middle">
                <Col span={6}>
                  <Input
                    placeholder="字段名 (例如: user_id)"
                    value={col.name}
                    onChange={(e) => updateColumn(idx, 'name', e.target.value)}
                  />
                </Col>
                <Col span={6}>
                  <Select
                    showSearch
                    placeholder="数据类型"
                    value={col.type}
                    onChange={(v) => updateColumn(idx, 'type', v)}
                    options={clickhouseTypes.map((t) => ({ label: t, value: t }))}
                  />
                </Col>
                <Col span={3}>
                  <Checkbox
                    checked={col.nullable}
                    onChange={(e) => updateColumn(idx, 'nullable', e.target.checked)}
                  >
                    Nullable
                  </Checkbox>
                </Col>
                <Col span={4}>
                  <Input
                    placeholder="Default 默认值"
                    value={col.default}
                    onChange={(e) => updateColumn(idx, 'default', e.target.value)}
                  />
                </Col>
                <Col span={4}>
                  <Input
                    placeholder="注释"
                    value={col.comment}
                    onChange={(e) => updateColumn(idx, 'comment', e.target.value)}
                  />
                </Col>
                <Col span={1}>
                  <Button
                    type="text"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => removeColumn(idx)}
                    disabled={visualColumns.length <= 1}
                  />
                </Col>
              </Row>
            ))}
          </div>

          <Row gutter={16} style={{ marginTop: 16 }}>
            <Col span={8}>
              <Form.Item name="order_by" label="ORDER BY (必须包含排序列)" initialValue="id">
                <Input placeholder="例如: id, created_at" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="primary_key" label="PRIMARY KEY (可选)">
                <Input placeholder="例如: id" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="partition_by" label="PARTITION BY (按分区划分)">
                <Input placeholder="例如: toYYYYMM(created_at)" />
              </Form.Item>
            </Col>
          </Row>

          {generatedSql && (
            <div style={{ marginTop: 12 }}>
              <div style={{ fontWeight: 600, fontSize: 12, marginBottom: 4 }}>实时生成 SQL 预览:</div>
              <pre
                style={{
                  background: isDark ? '#141414' : '#f8fafc',
                  color: isDark ? '#52c41a' : '#15803d',
                  border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
                  padding: 12,
                  borderRadius: 6,
                  fontFamily: 'Consolas, Monaco, monospace',
                  fontSize: 12,
                  maxHeight: 140,
                  overflow: 'auto',
                }}
              >
                {generatedSql}
              </pre>
            </div>
          )}
        </Form>
      </Modal>

      {/* Modal: Drop Table Confirmation */}
      <Modal
        title={`危险操作：删除数据表 [${tableToDrop}] 确认`}
        open={dropTableModalOpen}
        onCancel={() => {
          setDropTableModalOpen(false)
          setDropTableConfirm('')
        }}
        onOk={handleDropTable}
        okButtonProps={{ danger: true, disabled: dropTableConfirm !== `DROP ${tableToDrop}` }}
        okText="确认删除表"
      >
        <p style={{ color: '#ff4d4f' }}>
          删除数据表将永久清除表结构与磁盘中的所有数据 Part！
        </p>
        <p>请输入 <strong>DROP {tableToDrop}</strong> 确认操作：</p>
        <Input
          placeholder={`输入 DROP ${tableToDrop}`}
          value={dropTableConfirm}
          onChange={(e) => setDropTableConfirm(e.target.value)}
        />
      </Modal>

      {/* Modal: Truncate Table Confirmation */}
      <Modal
        title={`危险操作：清空数据表 [${tableToTruncate}] 确认`}
        open={truncateModalOpen}
        onCancel={() => {
          setTruncateModalOpen(false)
          setTruncateConfirm('')
        }}
        onOk={handleTruncateTable}
        okButtonProps={{ danger: true, disabled: truncateConfirm !== `TRUNCATE ${tableToTruncate}` }}
        okText="确认清空表"
      >
        <p style={{ color: '#ff4d4f' }}>
          TRUNCATE 操作将立即清空表中的全部行数据，保留表结构！
        </p>
        <p>请输入 <strong>TRUNCATE {tableToTruncate}</strong> 确认操作：</p>
        <Input
          placeholder={`输入 TRUNCATE ${tableToTruncate}`}
          value={truncateConfirm}
          onChange={(e) => setTruncateConfirm(e.target.value)}
        />
      </Modal>

      {/* Drawer: Column Structure Management */}
      <Drawer
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingRight: 24 }}>
            <Space>
              <TableOutlined />
              <span>数据表 [<strong>{colsDrawerTable}</strong>] 字段结构与模式管理</span>
            </Space>
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => {
                addColumnForm.resetFields()
                setAddColumnOpen(true)
              }}
            >
              添加新字段
            </Button>
          </div>
        }
        open={colsDrawerOpen}
        onClose={() => setColsDrawerOpen(false)}
        width={860}
      >
        <Table
          dataSource={currentColumns}
          rowKey="name"
          loading={colsLoading}
          pagination={false}
          size="small"
          columns={[
            {
              title: '字段名',
              dataIndex: 'name',
              key: 'name',
              render: (name, r) => (
                <Space>
                  <strong style={{ fontFamily: 'monospace' }}>{name}</strong>
                  {r.is_in_primary_key && <Tag color="gold">PK 排序键</Tag>}
                  {r.is_in_partition_key && <Tag color="blue">分区键</Tag>}
                </Space>
              ),
            },
            {
              title: '数据类型',
              dataIndex: 'type',
              key: 'type',
              render: (t) => <Tag color="cyan">{t}</Tag>,
            },
            {
              title: '默认值',
              key: 'default',
              render: (_, r) => (
                <span style={{ fontSize: 12, fontFamily: 'monospace' }}>
                  {r.default_expression ? `${r.default_kind || 'DEFAULT'} ${r.default_expression}` : '-'}
                </span>
              ),
            },
            {
              title: '压缩编码 (Codec)',
              dataIndex: 'compression_codec',
              key: 'compression_codec',
              render: (c) => (c ? <Tag>{c}</Tag> : <Text type="secondary">-</Text>),
            },
            {
              title: '注释说明',
              dataIndex: 'comment',
              key: 'comment',
              render: (c) => c || <Text type="secondary">-</Text>,
            },
            {
              title: '操作',
              key: 'action',
              width: 140,
              render: (_, r) => (
                <Space size="small">
                  <Button
                    size="small"
                    type="link"
                    icon={<EditOutlined />}
                    onClick={() => {
                      setColToComment(r.name)
                      setColCommentValue(r.comment || '')
                      setEditCommentOpen(true)
                    }}
                  >
                    注释
                  </Button>
                  <Button
                    size="small"
                    type="link"
                    danger
                    icon={<DeleteOutlined />}
                    disabled={r.is_in_primary_key || r.is_in_partition_key}
                    onClick={() => {
                      setColToDrop(r.name)
                      setDropColConfirm('')
                      setDropColOpen(true)
                    }}
                  >
                    删除
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Drawer>

      {/* Modal: Add Column */}
      <Modal
        title={`在表 [${colsDrawerTable}] 中添加新字段 (ALTER TABLE ADD COLUMN)`}
        open={addColumnOpen}
        onCancel={() => setAddColumnOpen(false)}
        onOk={() => addColumnForm.submit()}
        okText="确认添加字段"
      >
        <Form form={addColumnForm} layout="vertical" onFinish={handleAddColumn}>
          <Form.Item
            name="column_name"
            label="字段名称"
            rules={[
              { required: true, message: '请输入字段名称' },
              { pattern: /^[a-zA-Z0-9_]+$/, message: '仅支持字母、数字及下划线' },
            ]}
          >
            <Input placeholder="例如: user_status" />
          </Form.Item>

          <Row gutter={12}>
            <Col span={16}>
              <Form.Item
                name="type"
                label="数据类型"
                initialValue="String"
                rules={[{ required: true }]}
              >
                <Select
                  showSearch
                  options={clickhouseTypes.map((t) => ({ label: t, value: t }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="nullable" valuePropName="checked" label="可为空">
                <Checkbox>Nullable</Checkbox>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="default_expression" label="默认值表达式 (Default Expression)">
            <Input placeholder="例如: '' 或 0 或 now()" />
          </Form.Item>

          <Form.Item name="comment" label="字段注释">
            <Input placeholder="用途说明" />
          </Form.Item>

          <Form.Item name="after_column" label="插入位置 (AFTER COLUMN，可选)">
            <Select
              allowClear
              placeholder="默认添加在最后"
              options={currentColumns.map((c) => ({ label: `在 [${c.name}] 之后`, value: c.name }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Modal: Drop Column Confirmation */}
      <Modal
        title={`危险操作：删除字段 [${colToDrop}] 确认`}
        open={dropColOpen}
        onCancel={() => {
          setDropColOpen(false)
          setDropColConfirm('')
        }}
        onOk={handleDropColumn}
        okButtonProps={{ danger: true, disabled: dropColConfirm !== `DROP ${colToDrop}` }}
        okText="确认删除字段"
      >
        <p style={{ color: '#ff4d4f' }}>
          删除字段将从数据表结构及物理存储中彻底移除该列的所有数据！
        </p>
        <p>请输入 <strong>DROP {colToDrop}</strong> 确认操作：</p>
        <Input
          placeholder={`输入 DROP ${colToDrop}`}
          value={dropColConfirm}
          onChange={(e) => setDropColConfirm(e.target.value)}
        />
      </Modal>

      {/* Modal: Edit Column Comment */}
      <Modal
        title={`修改字段 [${colToComment}] 注释`}
        open={editCommentOpen}
        onCancel={() => setEditCommentOpen(false)}
        onOk={handleSaveColumnComment}
        okText="保存注释"
      >
        <Form layout="vertical">
          <Form.Item label="新注释内容">
            <Input
              placeholder="请输入字段注释"
              value={colCommentValue}
              onChange={(e) => setColCommentValue(e.target.value)}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Modal: Data Import Wizard */}
      <Modal
        title={
          <Space>
            <CloudUploadOutlined />
            <span>ClickHouse 数据导入向导 (批量写入数据)</span>
          </Space>
        }
        open={importModalOpen}
        onCancel={() => setImportModalOpen(false)}
        width={840}
        footer={[
          <Button key="cancel" onClick={() => setImportModalOpen(false)}>
            取消
          </Button>,
          <Button
            key="preview"
            icon={<EyeOutlined />}
            loading={previewing}
            onClick={handlePreviewImport}
          >
            解析前 10 行预览
          </Button>,
          <Button
            key="import"
            type="primary"
            icon={<CloudUploadOutlined />}
            loading={importing}
            onClick={handleExecuteImport}
          >
            开始批量导入
          </Button>,
        ]}
      >
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontWeight: 600 }}>目标数据表:</div>
            <Select
              style={{ width: '100%' }}
              value={importTable}
              onChange={(v) => setImportTable(v)}
              options={tables.map((t) => ({ label: `${selectedDb}.${t.name}`, value: t.name }))}
            />
          </Col>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontWeight: 600 }}>数据格式 (FORMAT):</div>
            <Select
              style={{ width: '100%' }}
              value={importFormat}
              onChange={(v) => setImportFormat(v)}
              options={[
                { label: 'CSV (逗号分隔，无列名表头)', value: 'CSV' },
                { label: 'CSVWithNames (逗号分隔，首行为列名表头)', value: 'CSVWithNames' },
                { label: 'TSV (Tab 制表符分隔，无列名表头)', value: 'TSV' },
                { label: 'TSVWithNames (Tab 制表符分隔，首行为列名)', value: 'TSVWithNames' },
                { label: 'JSONEachRow (每行一个标准 JSON 对象)', value: 'JSONEachRow' },
              ]}
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontWeight: 600 }}>粘贴导入数据或上传数据文件:</span>
          <Upload
            beforeUpload={handleFileUpload}
            showUploadList={false}
            accept=".csv,.tsv,.json,.txt"
          >
            <Button size="small" icon={<UploadOutlined />}>
              从本地文件上传载入 (.csv / .tsv / .json)
            </Button>
          </Upload>
        </div>

        <Input.TextArea
          rows={6}
          placeholder="在此粘贴待导入的数据文本内容，或者点击上方按钮上传本地文件..."
          value={importDataText}
          onChange={(e) => setImportDataText(e.target.value)}
          style={{ fontFamily: 'monospace', fontSize: 12, marginBottom: 12 }}
        />

        {importPreviewData && (
          <div style={{ marginTop: 12 }}>
            <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 6, display: 'flex', justifyContent: 'space-between' }}>
              <span>数据解析预览 (前 {importPreviewData.rows.length} 行):</span>
              <Tag color="cyan">估算总行数: ~{importPreviewData.total_lines_estimate}</Tag>
            </div>
            <div style={{ maxHeight: 200, overflow: 'auto', border: '1px solid rgba(140, 140, 140, 0.2)', borderRadius: 6 }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12, fontFamily: 'monospace' }}>
                <thead>
                  <tr style={{ background: 'rgba(140, 140, 140, 0.1)', textAlign: 'left' }}>
                    {importPreviewData.headers.map((h, i) => (
                      <th key={i} style={{ padding: '6px 10px', borderBottom: '1px solid rgba(140, 140, 140, 0.2)' }}>
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {importPreviewData.rows.map((row, rIdx) => (
                    <tr key={rIdx} style={{ borderBottom: '1px solid rgba(140, 140, 140, 0.1)' }}>
                      {row.map((cell, cIdx) => (
                        <td key={cIdx} style={{ padding: '6px 10px', whiteSpace: 'nowrap' }}>
                          {cell}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
