import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Input,
  message,
  Tag,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ThunderboltOutlined,
  ReloadOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { ProcessInfo } from '../../types'
import { formatBytes, formatNumber } from '../../utils/format'

export const ProcessesPage: React.FC<{ isDark?: boolean }> = ({ isDark = false }) => {
  const [processes, setProcesses] = useState<ProcessInfo[]>([])
  const [loading, setLoading] = useState(false)

  // Kill query modal
  const [killModalOpen, setKillModalOpen] = useState(false)
  const [queryToKill, setQueryToKill] = useState<ProcessInfo | null>(null)
  const [confirmInput, setConfirmInput] = useState('')

  const fetchProcesses = async () => {
    setLoading(true)
    try {
      const res = await api.getProcesses()
      if (res.data.data) {
        setProcesses(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProcesses()
    const timer = setInterval(fetchProcesses, 3000)
    return () => clearInterval(timer)
  }, [])

  const handleKill = async () => {
    if (!queryToKill || confirmInput !== 'KILL') {
      message.error('请输入 KILL 确认强制终止')
      return
    }
    try {
      await api.killQuery(queryToKill.query_id, confirmInput)
      message.success(`已终止查询 ${queryToKill.query_id}`)
      setKillModalOpen(false)
      setConfirmInput('')
      setQueryToKill(null)
      fetchProcesses()
    } catch {
      // Handled
    }
  }

  const columns: ColumnsType<ProcessInfo> = [
    {
      title: 'Query ID',
      dataIndex: 'query_id',
      key: 'query_id',
      render: (id: string) => <code style={{ fontSize: 12 }}>{id}</code>,
    },
    {
      title: '用户',
      dataIndex: 'user',
      key: 'user',
      render: (u: string) => <Tag color="blue">{u}</Tag>,
    },
    {
      title: '客户端主机',
      dataIndex: 'address',
      key: 'address',
      render: (a: string) => a || '127.0.0.1',
    },
    {
      title: '已执行时间',
      dataIndex: 'elapsed',
      key: 'elapsed',
      sorter: (a, b) => a.elapsed - b.elapsed,
      render: (e: number) => {
        const color = e > 10 ? 'red' : e > 3 ? 'orange' : 'green'
        return <Tag color={color}>{e.toFixed(2)}s</Tag>
      },
    },
    {
      title: '已扫描行数',
      dataIndex: 'read_rows',
      key: 'read_rows',
      render: (r: number) => formatNumber(r),
    },
    {
      title: '已扫描字节',
      dataIndex: 'read_bytes',
      key: 'read_bytes',
      render: (b: number) => formatBytes(b),
    },
    {
      title: '内存占用',
      dataIndex: 'memory_usage',
      key: 'memory_usage',
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
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button
          danger
          size="small"
          icon={<StopOutlined />}
          onClick={() => {
            setQueryToKill(record)
            setKillModalOpen(true)
          }}
        >
          Kill Query
        </Button>
      ),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <ThunderboltOutlined />
            <span>运行中的查询 (system.processes)</span>
            <Tag color="gold">{processes.length} 个正在执行</Tag>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={fetchProcesses}>
            刷新
          </Button>
        </div>
      }
    >
      <Table
        dataSource={processes}
        columns={columns}
        rowKey="query_id"
        loading={loading}
        pagination={{ pageSize: 20 }}
        size="middle"
      />

      <Modal
        title="危险操作：强制终止正在运行的查询"
        open={killModalOpen}
        onCancel={() => {
          setKillModalOpen(false)
          setConfirmInput('')
        }}
        onOk={handleKill}
        okButtonProps={{ danger: true, disabled: confirmInput !== 'KILL' }}
        okText="确认终止查询"
      >
        <p style={{ color: '#ff4d4f' }}>
          强制终止查询将向 ClickHouse 发送 KILL QUERY 命令，打断客户端数据流！
        </p>
        <div style={{ marginBottom: 12 }}>
          <strong>Query ID:</strong> <code>{queryToKill?.query_id}</code>
        </div>
        <div style={{ marginBottom: 12 }}>
          <strong>SQL:</strong>
          <pre
            style={{
              background: isDark ? '#141414' : '#f8fafc',
              color: isDark ? '#faad14' : '#0f172a',
              border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
              padding: 8,
              borderRadius: 6,
              maxHeight: 120,
              overflow: 'auto',
              fontFamily: 'Consolas, Monaco, monospace',
            }}
          >
            {queryToKill?.query}
          </pre>
        </div>
        <p>请输入 <strong>KILL</strong> 确认操作：</p>
        <Input
          placeholder="输入 KILL"
          value={confirmInput}
          onChange={(e) => setConfirmInput(e.target.value)}
        />
      </Modal>
    </Card>
  )
}
