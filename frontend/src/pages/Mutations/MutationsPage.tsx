import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Modal,
  Input,
  Tooltip,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ToolOutlined,
  ReloadOutlined,
  StopOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { MutationInfo } from '../../types'
import { formatDateTime } from '../../utils/format'

export const MutationsPage: React.FC = () => {
  const [mutations, setMutations] = useState<MutationInfo[]>([])
  const [loading, setLoading] = useState(false)

  // Kill modal
  const [killModalOpen, setKillModalOpen] = useState(false)
  const [mutationToKill, setMutationToKill] = useState<MutationInfo | null>(null)
  const [confirmInput, setConfirmInput] = useState('')

  const fetchMutations = async () => {
    setLoading(true)
    try {
      const res = await api.getMutations()
      if (res.data.data) {
        setMutations(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMutations()
    const timer = setInterval(fetchMutations, 4000)
    return () => clearInterval(timer)
  }, [])

  const handleKill = async () => {
    if (!mutationToKill || confirmInput !== 'KILL') {
      message.error('请输入 KILL 确认强制终止')
      return
    }
    try {
      await api.killMutation({
        database: mutationToKill.database,
        table: mutationToKill.table,
        mutation_id: mutationToKill.mutation_id,
        confirmation: confirmInput,
      })
      message.success(`已终止 Mutation ${mutationToKill.mutation_id}`)
      setKillModalOpen(false)
      setConfirmInput('')
      setMutationToKill(null)
      fetchMutations()
    } catch {
      // Handled
    }
  }

  const columns: ColumnsType<MutationInfo> = [
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
      title: 'Mutation ID',
      dataIndex: 'mutation_id',
      key: 'mutation_id',
      render: (id: string) => <code style={{ fontSize: 12 }}>{id}</code>,
    },
    {
      title: '变更指令',
      dataIndex: 'command',
      key: 'command',
      ellipsis: true,
      render: (cmd: string) => (
        <Tooltip title={cmd}>
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{cmd}</span>
        </Tooltip>
      ),
    },
    {
      title: '待处理 Part',
      dataIndex: 'parts_to_do',
      key: 'parts_to_do',
      render: (p: number) => (p > 0 ? <Tag color="warning">{p} parts</Tag> : <Tag color="success">0</Tag>),
    },
    {
      title: '完成状态',
      dataIndex: 'is_done',
      key: 'is_done',
      render: (done: boolean) =>
        done ? <Tag color="success">已完成</Tag> : <Tag color="processing">正在执行</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'create_time',
      key: 'create_time',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '最新失败原因',
      dataIndex: 'latest_fail_reason',
      key: 'latest_fail_reason',
      ellipsis: true,
      render: (reason: string) =>
        reason ? (
          <Tooltip title={reason}>
            <Tag color="error" icon={<ExclamationCircleOutlined />}>
              失败
            </Tag>
          </Tooltip>
        ) : (
          <span style={{ color: '#52c41a' }}>无异常</span>
        ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) =>
        !record.is_done ? (
          <Button
            danger
            size="small"
            icon={<StopOutlined />}
            onClick={() => {
              setMutationToKill(record)
              setKillModalOpen(true)
            }}
          >
            Kill
          </Button>
        ) : null,
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <ToolOutlined />
            <span>异步 Mutation 任务管理 (system.mutations)</span>
            <Tag>{mutations.length} 个任务记录</Tag>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={fetchMutations}>
            刷新
          </Button>
        </div>
      }
    >
      <Table
        dataSource={mutations}
        columns={columns}
        rowKey="mutation_id"
        loading={loading}
        pagination={{ pageSize: 20 }}
        size="middle"
      />

      {/* Modal: Kill Mutation Confirmation */}
      <Modal
        title="危险操作：强制终止 Mutation 任务"
        open={killModalOpen}
        onCancel={() => {
          setKillModalOpen(false)
          setConfirmInput('')
        }}
        onOk={handleKill}
        okButtonProps={{ danger: true, disabled: confirmInput !== 'KILL' }}
        okText="确认终止 Mutation"
      >
        <p style={{ color: '#ff4d4f' }}>
          强制终止 Mutation 将向 ClickHouse 执行 KILL MUTATION 命令，中断尚未完成的 Part 修改！
        </p>
        <div style={{ marginBottom: 12 }}>
          <strong>表:</strong> {mutationToKill?.database}.{mutationToKill?.table}
        </div>
        <div style={{ marginBottom: 12 }}>
          <strong>Command:</strong> <code>{mutationToKill?.command}</code>
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
