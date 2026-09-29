import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Select,
  Input,
  Button,
  Table,
  Space,
  Alert,
  Empty,
  Pagination,
  Tooltip,
  Dropdown,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  SearchOutlined,
  ReloadOutlined,
  DownloadOutlined,
  FilterOutlined,
  DatabaseOutlined,
  TableOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { DatabaseInfo, TableInfo, QueryResult } from '../../types'
import { formatNumber, formatDurationMs } from '../../utils/format'

interface DataBrowserPageProps {
  initialDb?: string
  initialTable?: string
}

export const DataBrowserPage: React.FC<DataBrowserPageProps> = ({
  initialDb = 'default',
  initialTable = '',
}) => {
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [tables, setTables] = useState<TableInfo[]>([])
  const [selectedDb, setSelectedDb] = useState<string>(initialDb)
  const [selectedTable, setSelectedTable] = useState<string>(initialTable)

  // Browser query state
  const [whereClause, setWhereClause] = useState('')
  const [filterOpen, setFilterOpen] = useState(false)
  const [error, setError] = useState('')
  const [pageSize, setPageSize] = useState<number>(100)
  const [currentPage, setCurrentPage] = useState<number>(1)
  const [sortField, setSortField] = useState<string>('')
  const [sortOrder, setSortOrder] = useState<string>('ASC')

  const [loading, setLoading] = useState(false)
  const [queryResult, setQueryResult] = useState<QueryResult | null>(null)
  const [totalEstimated, setTotalEstimated] = useState<number>(0)

  const requestId = useRef(0)

  // Load databases
  useEffect(() => {
    api.listDatabases().then((res) => {
      if (res.data.data) {
        setDatabases(res.data.data)
        if (res.data.data.length && !res.data.data.some((db) => db.name === initialDb)) {
          setSelectedDb(res.data.data[0].name)
          setSelectedTable('')
        }
      }
    }).catch(() => setError('无法加载数据库，请检查连接设置。'))
  }, [])

  // Load tables when selectedDb changes
  useEffect(() => {
    if (!selectedDb) return
    let cancelled = false
    api.listTables(selectedDb).then((res) => {
      if (!cancelled && res.data.data) {
        setTables(res.data.data)
        if (!selectedTable || !res.data.data.some((t) => t.name === selectedTable)) {
          if (res.data.data.length > 0) {
            setSelectedTable(res.data.data[0].name)
          } else {
            setSelectedTable('')
          }
        }
      }
    }).catch(() => { if (!cancelled) setError('无法加载数据表，请检查数据库连接。') })
    return () => { cancelled = true }
  }, [selectedDb])

  const loadData = async (page = currentPage, size = pageSize, field = sortField, order = sortOrder) => {
    const id = ++requestId.current
    if (!selectedDb || !selectedTable) { setLoading(false); return }
    setLoading(true)
    setError('')
    try {
      const res = await api.browseData({
        database: selectedDb,
        table: selectedTable,
        page,
        page_size: size,
        sort_field: field,
        sort_order: order,
        where: whereClause,
      })
      if (id === requestId.current && res.data.data) {
        setQueryResult(res.data.data)
        setTotalEstimated(res.data.data.total_rows_estimate || 0)
        setCurrentPage(page)
      }
    } catch {
      if (id === requestId.current) {
        setQueryResult(null)
        setTotalEstimated(0)
        setError('查询失败，请检查连接或筛选条件后重试。')
      }
    } finally {
      if (id === requestId.current) setLoading(false)
    }
  }

  useEffect(() => {
    loadData(1, pageSize)
  }, [selectedDb, selectedTable])

  const handleExportCSV = () => {
    if (!queryResult || !queryResult.rows.length) {
      message.warning('当前无数据可导出')
      return
    }
    const headers = queryResult.columns.map((c) => `"${c.name.replace(/"/g, '""')}"`).join(',')
    const rows = queryResult.rows.map((row) =>
      queryResult.columns
        .map((c) => {
          const val = row[c.name]
          if (val === null || val === undefined) return ''
          const str = String(val).replace(/"/g, '""')
          return `"${str}"`
        })
        .join(',')
    )
    const csvContent = 'data:text/csv;charset=utf-8,' + encodeURIComponent('\uFEFF' + [headers, ...rows].join('\n'))
    const encodedUri = csvContent
    const link = document.createElement('a')
    link.setAttribute('href', encodedUri)
    link.setAttribute('download', `${selectedDb}_${selectedTable}_export.csv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleExportTSV = () => {
    if (!queryResult || !queryResult.rows.length) {
      message.warning('当前无数据可导出')
      return
    }
    const headers = queryResult.columns.map((c) => c.name).join('\t')
    const rows = queryResult.rows.map((row) =>
      queryResult.columns
        .map((c) => {
          const val = row[c.name]
          if (val === null || val === undefined) return '\\N'
          return String(val).replace(/\t/g, ' ').replace(/\n/g, '\\n')
        })
        .join('\t')
    )
    const tsvContent = 'data:text/tab-separated-values;charset=utf-8,' + encodeURIComponent('\uFEFF' + [headers, ...rows].join('\n'))
    const encodedUri = tsvContent
    const link = document.createElement('a')
    link.setAttribute('href', encodedUri)
    link.setAttribute('download', `${selectedDb}_${selectedTable}_export.tsv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleExportJSON = () => {
    if (!queryResult || !queryResult.rows.length) {
      message.warning('当前无数据可导出')
      return
    }
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(queryResult.rows, null, 2))
    const link = document.createElement('a')
    link.setAttribute('href', dataStr)
    link.setAttribute('download', `${selectedDb}_${selectedTable}_export.json`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // Generate table columns dynamically from ClickHouse schema
  const dynamicColumns: ColumnsType<Record<string, any>> =
    queryResult?.columns.map((col) => ({
      title: (
        <Tooltip title={`Type: ${col.type}`}>
          <div>
            <strong>{col.name}</strong>
            <div style={{ fontSize: 11, fontWeight: 'normal', opacity: 0.6 }}>{col.type}</div>
          </div>
        </Tooltip>
      ),
      dataIndex: ['row', col.name],
      key: col.name,
      ellipsis: true,
      sorter: true,
      sortOrder: sortField === col.name ? (sortOrder === 'DESC' ? 'descend' : 'ascend') : null,
      render: (val: any) => {
        if (val === null || val === undefined) {
          return <span style={{ color: '#888', fontStyle: 'italic' }}>NULL</span>
        }
        if (typeof val === 'object') {
          return <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{JSON.stringify(val)}</span>
        }
        return <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>
      },
    })) || []

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space wrap>
            <Select
              style={{ width: 160 }}
              aria-label="数据库"
              value={selectedDb}
              onChange={(val) => {
                setSelectedDb(val)
                setSelectedTable('')
                setTables([])
                setQueryResult(null)
                setTotalEstimated(0)
                setWhereClause('')
                setSortField('')
              }}
              options={databases.map((d) => ({
                label: (
                  <Space>
                    <DatabaseOutlined />
                    <span>{d.name}</span>
                  </Space>
                ),
                value: d.name,
              }))}
            />

            <Select
              style={{ width: 220 }}
              aria-label="数据表"
              value={selectedTable || undefined}
              onChange={(val) => {
                setSelectedTable(val)
                setQueryResult(null)
                setWhereClause('')
                setSortField('')
              }}
              placeholder="选择数据表"
              options={tables.map((t) => ({
                label: (
                  <Space>
                    <TableOutlined />
                    <span>{t.name}</span>
                  </Space>
                ),
                value: t.name,
              }))}
            />

            <Button icon={<FilterOutlined />} type={filterOpen ? 'primary' : 'default'} onClick={() => setFilterOpen(!filterOpen)}>筛选</Button>
          </Space>

          <Space>
            <Dropdown
              menu={{
                items: [
                  { key: 'csv', label: '导出 CSV 文件', onClick: handleExportCSV },
                  { key: 'tsv', label: '导出 TSV 文件', onClick: handleExportTSV },
                  { key: 'json', label: '导出 JSON 数据', onClick: handleExportJSON },
                ],
              }}
            >
              <Button icon={<DownloadOutlined />} disabled={!queryResult?.rows.length}>导出本页</Button>
            </Dropdown>
            <Button icon={<ReloadOutlined />} onClick={() => loadData(currentPage, pageSize)} disabled={!selectedTable}>
              刷新
            </Button>
          </Space>
        </div>
      }
    >
      {error && <Alert type="error" showIcon title={error} style={{ marginBottom: 16 }} />}
      {filterOpen && <Space.Compact style={{ width: '100%', marginBottom: 16 }}>
        <Input aria-label="筛选条件" placeholder="SQL 条件，例如 id > 100" value={whereClause} onChange={(e) => setWhereClause(e.target.value)} onPressEnter={() => loadData(1, pageSize)} />
        <Button type="primary" icon={<SearchOutlined />} onClick={() => loadData(1, pageSize)}>应用</Button>
      </Space.Compact>}
      {queryResult && <div style={{ marginBottom: 12, color: 'var(--panel-muted)', fontSize: 12 }}>
        本页 {queryResult.total_rows} 行 · 耗时 {formatDurationMs(queryResult.elapsed_ms)}
      </div>}

      {/* Grid */}
      <Table
        dataSource={queryResult?.rows.map((row, key) => ({ key, row })) || []}
        columns={dynamicColumns}
        rowKey="key"
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={selectedTable ? '没有匹配的数据' : '选择数据库和数据表以开始浏览'} /> }}
        loading={loading}
        pagination={false}
        scroll={{ x: 'max-content', y: 540 }}
        size="small"
        onChange={(_, __, sorter: any) => {
          const field = sorter.order ? String(sorter.columnKey) : ''
          const order = sorter.order === 'descend' ? 'DESC' : 'ASC'
          setSortField(field)
          setSortOrder(order)
          loadData(1, pageSize, field, order)
        }}
      />

      {/* Custom pagination with limit safety options: 100, 500, 1000, 5000 */}
      <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 12 }}>
        <Pagination
          current={currentPage}
          pageSize={pageSize}
          total={totalEstimated || queryResult?.total_rows || 0}
          pageSizeOptions={['100', '500', '1000', '5000']}
          showSizeChanger
          onChange={(page, size) => {
            setPageSize(size)
            loadData(size !== pageSize ? 1 : page, size)
          }}
          showTotal={(total) => `共约 ${formatNumber(total)} 条记录`}
        />
      </div>
    </Card>
  )
}
