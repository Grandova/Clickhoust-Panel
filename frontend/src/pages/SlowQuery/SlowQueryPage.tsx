import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Select,
  Input,
  Button,
  Space,
  Tag,
  Drawer,
  Descriptions,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ClockCircleOutlined,
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { QueryLogItem } from '../../types'
import { formatBytes, formatNumber, formatDurationMs, formatDateTime } from '../../utils/format'

export const SlowQueryPage: React.FC<{ isDark?: boolean }> = ({ isDark = false }) => {
  const [threshold, setThreshold] = useState<number>(1000) // ms
  const [search, setSearch] = useState('')
  const [items, setItems] = useState<QueryLogItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  // Drawer
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<QueryLogItem | null>(null)

  const fetchLogs = async (p = page, ps = pageSize) => {
    setLoading(true)
    try {
      const res = await api.getQueryLog({
        min_duration_ms: threshold,
        search,
        limit: ps,
        offset: (p - 1) * ps,
      })
      if (res.data.data) {
        setItems(res.data.data.items)
        setTotal(res.data.data.total)
        setPage(p)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs(1, pageSize)
  }, [threshold])

  const columns: ColumnsType<QueryLogItem> = [
    {
      title: '执行时间',
      dataIndex: 'event_time',
      key: 'event_time',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
      sorter: (a, b) => a.duration_ms - b.duration_ms,
      render: (ms: number) => {
        const color = ms > 5000 ? 'red' : ms > 2000 ? 'orange' : 'gold'
        return <Tag color={color}>{formatDurationMs(ms)}</Tag>
      },
    },
    {
      title: '数据库 / 用户',
      key: 'db_user',
      render: (_, r) => (
        <span>
          <Tag color="cyan">{r.database || 'default'}</Tag>
          <Tag color="blue">{r.user}</Tag>
        </span>
      ),
    },
    {
      title: '扫描行数',
      dataIndex: 'read_rows',
      key: 'read_rows',
      render: (r: number) => formatNumber(r),
    },
    {
      title: '扫描大小',
      dataIndex: 'read_bytes',
      key: 'read_bytes',
      render: (b: number) => formatBytes(b),
    },
    {
      title: '内存峰值',
      dataIndex: 'memory_peak',
      key: 'memory_peak',
      render: (m: number) => formatBytes(m),
    },
    {
      title: 'SQL 语句',
      dataIndex: 'query',
      key: 'query',
      ellipsis: true,
      render: (q: string) => (
        <Tooltip title={q}>
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{q}</span>
        </Tooltip>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (_, r) =>
        r.type === 'QueryFinish' ? (
          <Tag color="success">完成</Tag>
        ) : (
          <Tooltip title={r.exception_message || '异常'}>
            <Tag color="error" icon={<ExclamationCircleOutlined />}>
              异常
            </Tag>
          </Tooltip>
        ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            setSelectedItem(record)
            setDrawerOpen(true)
          }}
        >
          详情
        </Button>
      ),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space wrap>
            <ClockCircleOutlined />
            <span>慢查询分析 (system.query_log)</span>
            <Select
              style={{ width: 140 }}
              value={threshold}
              onChange={(val) => setThreshold(val)}
              options={[
                { label: '慢查询 >= 500ms', value: 500 },
                { label: '慢查询 >= 1s', value: 1000 },
                { label: '慢查询 >= 3s', value: 3000 },
                { label: '慢查询 >= 5s', value: 5000 },
                { label: '慢查询 >= 10s', value: 10000 },
              ]}
            />

            <Input
              placeholder="搜索 SQL、Query ID 或用户名"
              style={{ width: 280 }}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onPressEnter={() => fetchLogs(1, pageSize)}
            />

            <Button type="primary" icon={<SearchOutlined />} onClick={() => fetchLogs(1, pageSize)}>
              搜索
            </Button>
          </Space>

          <Button icon={<ReloadOutlined />} onClick={() => fetchLogs(page, pageSize)}>
            刷新
          </Button>
        </div>
      }
    >
      <Table
        dataSource={items}
        columns={columns}
        rowKey="query_id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPageSize(ps)
            fetchLogs(p, ps)
          },
          showTotal: (tot) => `共 ${tot} 条慢查询记录`,
        }}
        size="middle"
      />

      {/* Query Detail Drawer */}
      <Drawer
        title="慢查询明细"
        placement="right"
        width={680}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      >
        {selectedItem && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="Query ID" span={2}>
                <code>{selectedItem.query_id}</code>
              </Descriptions.Item>
              <Descriptions.Item label="执行用户">{selectedItem.user}</Descriptions.Item>
              <Descriptions.Item label="当前数据库">{selectedItem.database}</Descriptions.Item>
              <Descriptions.Item label="客户端地址">{selectedItem.client_address}</Descriptions.Item>
              <Descriptions.Item label="执行耗时">
                <Tag color="red">{formatDurationMs(selectedItem.duration_ms)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="扫描行数">{formatNumber(selectedItem.read_rows)}</Descriptions.Item>
              <Descriptions.Item label="扫描数据量">{formatBytes(selectedItem.read_bytes)}</Descriptions.Item>
              <Descriptions.Item label="返回行数">{formatNumber(selectedItem.result_rows)}</Descriptions.Item>
              <Descriptions.Item label="返回数据量">{formatBytes(selectedItem.result_bytes)}</Descriptions.Item>
              <Descriptions.Item label="峰值内存" span={2}>{formatBytes(selectedItem.memory_peak)}</Descriptions.Item>
              <Descriptions.Item label="执行时刻" span={2}>{formatDateTime(selectedItem.event_time)}</Descriptions.Item>
            </Descriptions>

            {selectedItem.exception_message && (
              <div>
                <div style={{ fontWeight: 600, color: '#ff4d4f', marginBottom: 4 }}>异常信息:</div>
                <pre
                  style={{
                    background: 'rgba(255, 77, 79, 0.1)',
                    color: '#ff4d4f',
                    padding: 12,
                    borderRadius: 6,
                    whiteSpace: 'pre-wrap',
                  }}
                >
                  {selectedItem.exception_message}
                </pre>
              </div>
            )}

            <div>
              <div style={{ fontWeight: 600, marginBottom: 6 }}>完整 SQL 语句:</div>
              <pre
                style={{
                  background: isDark ? '#141414' : '#f8fafc',
                  color: isDark ? '#faad14' : '#0f172a',
                  border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
                  padding: 14,
                  borderRadius: 8,
                  maxHeight: 280,
                  overflow: 'auto',
                  fontFamily: 'Consolas, Monaco, monospace',
                  fontSize: 13,
                }}
              >
                {selectedItem.query}
              </pre>
            </div>
          </div>
        )}
      </Drawer>
    </Card>
  )
}
