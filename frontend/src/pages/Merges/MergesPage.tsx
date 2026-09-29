import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Progress,
  Tag,
  Button,
  Space,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { SyncOutlined, ReloadOutlined } from '@ant-design/icons'
import { api } from '../../api/client'
import type { MergeInfo } from '../../types'
import { formatBytes, formatNumber } from '../../utils/format'

export const MergesPage: React.FC = () => {
  const [merges, setMerges] = useState<MergeInfo[]>([])
  const [loading, setLoading] = useState(false)

  const fetchMerges = async () => {
    setLoading(true)
    try {
      const res = await api.getMerges()
      if (res.data.data) {
        setMerges(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMerges()
    const timer = setInterval(fetchMerges, 3000)
    return () => clearInterval(timer)
  }, [])

  const columns: ColumnsType<MergeInfo> = [
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
      title: '合并进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (p: number) => {
        const percent = Math.min(100, Math.round(p * 100))
        return <Progress percent={percent} size="small" status="active" />
      },
    },
    {
      title: '已耗时',
      dataIndex: 'elapsed',
      key: 'elapsed',
      render: (e: number) => `${e.toFixed(1)}s`,
    },
    {
      title: '参与 Parts 数',
      dataIndex: 'num_parts',
      key: 'num_parts',
      render: (n: number) => <Tag color="blue">{n}</Tag>,
    },
    {
      title: '生成新 Part 名称',
      dataIndex: 'result_part_name',
      key: 'result_part_name',
      ellipsis: true,
      render: (name: string) => <code style={{ fontSize: 12 }}>{name}</code>,
    },
    {
      title: '读取行 / 写入行',
      key: 'rows_io',
      render: (_, r) => (
        <span>
          {formatNumber(r.rows_read)} / {formatNumber(r.rows_written)}
        </span>
      ),
    },
    {
      title: '内存消耗',
      dataIndex: 'memory_usage',
      key: 'memory_usage',
      render: (m: number) => formatBytes(m),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <SyncOutlined spin={merges.length > 0} />
            <span>后台 Merge 合并监控 (system.merges)</span>
            <Tag color={merges.length > 0 ? 'processing' : 'default'}>
              {merges.length} 个后台任务进行中
            </Tag>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={fetchMerges}>
            刷新
          </Button>
        </div>
      }
    >
      <Table
        dataSource={merges}
        columns={columns}
        rowKey={(r, idx) => r.result_part_name || idx || Math.random()}
        loading={loading}
        pagination={false}
        size="middle"
        locale={{ emptyText: '当前没有正在执行的后台 Merge 任务' }}
      />
    </Card>
  )
}
