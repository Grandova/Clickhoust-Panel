import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Select,
  Checkbox,
  Space,
  Button,
  Tag,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  AppstoreOutlined,
  ReloadOutlined,
  DatabaseOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { PartInfo, DatabaseInfo } from '../../types'
import { formatBytes, formatNumber, formatDateTime } from '../../utils/format'

export const PartsPage: React.FC = () => {
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [selectedDb, setSelectedDb] = useState<string>('')
  const [activeOnly, setActiveOnly] = useState(true)
  const [parts, setParts] = useState<PartInfo[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api.listDatabases().then((res) => {
      if (res.data.data) {
        setDatabases(res.data.data)
      }
    })
  }, [])

  const fetchParts = async () => {
    setLoading(true)
    try {
      const res = await api.getParts({
        database: selectedDb,
        active_only: activeOnly,
      })
      if (res.data.data) {
        setParts(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchParts()
  }, [selectedDb, activeOnly])

  const columns: ColumnsType<PartInfo> = [
    {
      title: '库 / 表',
      key: 'db_tbl',
      render: (_, r) => (
        <span>
          <Tag color="cyan">{r.database}</Tag>
          <strong>{r.table}</strong>
        </span>
      ),
    },
    {
      title: 'Partition 分区',
      dataIndex: 'partition',
      key: 'partition',
      render: (p: string) => <Tag color="blue">{p || 'all'}</Tag>,
    },
    {
      title: 'Part 标识名',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      render: (n: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{n}</span>,
    },
    {
      title: '行数',
      dataIndex: 'rows',
      key: 'rows',
      sorter: (a, b) => Number(a.rows) - Number(b.rows),
      render: (r: number) => formatNumber(r),
    },
    {
      title: '磁盘容量 (压缩)',
      dataIndex: 'bytes_on_disk',
      key: 'bytes_on_disk',
      sorter: (a, b) => Number(a.bytes_on_disk) - Number(b.bytes_on_disk),
      render: (b: number, r) => (
        <Tooltip title={`未压缩: ${formatBytes(r.data_uncompressed)}`}>
          <span>{formatBytes(b)}</span>
        </Tooltip>
      ),
    },
    {
      title: 'Marks 标记数',
      dataIndex: 'marks',
      key: 'marks',
      render: (m: number) => formatNumber(m),
    },
    {
      title: '状态',
      dataIndex: 'active',
      key: 'active',
      render: (act: boolean) =>
        act ? <Tag color="success">Active</Tag> : <Tag color="default">Inactive</Tag>,
    },
    {
      title: '最后修改时刻',
      dataIndex: 'modification_time',
      key: 'modification_time',
      render: (t: string) => formatDateTime(t),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space wrap>
            <AppstoreOutlined />
            <span>ClickHouse 数据分片管理 (system.parts)</span>
            <Select
              allowClear
              placeholder="全部数据库"
              style={{ width: 160 }}
              value={selectedDb || undefined}
              onChange={(val) => setSelectedDb(val || '')}
              options={databases.map((d) => ({ label: d.name, value: d.name }))}
            />
            <Checkbox checked={activeOnly} onChange={(e) => setActiveOnly(e.target.checked)}>
              仅展示活跃 Part (Active = 1)
            </Checkbox>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={fetchParts}>
            刷新
          </Button>
        </div>
      }
    >
      <Table
        dataSource={parts}
        columns={columns}
        rowKey="name"
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        size="middle"
      />
    </Card>
  )
}
