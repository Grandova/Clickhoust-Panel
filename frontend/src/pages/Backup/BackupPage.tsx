import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Tag,
  message,
  Descriptions,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  SaveOutlined,
  ReloadOutlined,
  PlusOutlined,
  UndoOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  SyncOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { BackupRecord, DatabaseInfo, TableInfo } from '../../types'
import { formatDateTime } from '../../utils/format'

export const BackupPage: React.FC = () => {
  const [backups, setBackups] = useState<BackupRecord[]>([])
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [tables, setTables] = useState<TableInfo[]>([])
  const [loading, setLoading] = useState(false)

  // Create Backup Modal
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [createForm] = Form.useForm()
  const [selectedDb, setSelectedDb] = useState<string>('')

  // Restore Modal
  const [restoreModalOpen, setRestoreModalOpen] = useState(false)
  const [recordToRestore, setRecordToRestore] = useState<BackupRecord | null>(null)
  const [restoreConfirmInput, setRestoreConfirmInput] = useState('')

  const fetchBackups = async () => {
    setLoading(true)
    try {
      const res = await api.listBackups()
      if (res.data.data) {
        setBackups(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchBackups()
    api.listDatabases().then((res) => {
      if (res.data.data) {
        setDatabases(res.data.data)
      }
    })
  }, [])

  const handleDbChange = (dbName: string) => {
    setSelectedDb(dbName)
    if (dbName) {
      api.listTables(dbName).then((res) => {
        if (res.data.data) {
          setTables(res.data.data)
        }
      })
    } else {
      setTables([])
    }
  }

  const handleCreateBackup = async (values: any) => {
    try {
      await api.createBackup(values)
      message.success('备份任务已提交，ClickHouse 正在执行原生备份')
      setCreateModalOpen(false)
      createForm.resetFields()
      fetchBackups()
    } catch {
      // Handled
    }
  }

  const handleRestoreBackup = async () => {
    if (!recordToRestore || restoreConfirmInput !== 'RESTORE') {
      message.error('请输入 RESTORE 确认恢复操作')
      return
    }
    try {
      await api.restoreBackup({
        backup_id: recordToRestore.backup_id,
        disk: recordToRestore.disk,
        confirmation: restoreConfirmInput,
      })
      message.success(`备份 ${recordToRestore.backup_id} 恢复完成`)
      setRestoreModalOpen(false)
      setRestoreConfirmInput('')
      setRecordToRestore(null)
      fetchBackups()
    } catch {
      // Handled
    }
  }

  const handleDeleteRecord = async (id: number) => {
    try {
      await api.deleteBackup(id)
      message.success('备份记录已移除')
      fetchBackups()
    } catch {
      // Handled
    }
  }

  const columns: ColumnsType<BackupRecord> = [
    {
      title: 'Backup ID',
      dataIndex: 'backup_id',
      key: 'backup_id',
      render: (id: string) => <code style={{ fontSize: 12 }}>{id}</code>,
    },
    {
      title: '目标数据库',
      dataIndex: 'database_name',
      key: 'database_name',
      render: (db: string) => (db ? <Tag color="blue">{db}</Tag> : <Tag color="purple">全部数据库</Tag>),
    },
    {
      title: '目标数据表',
      dataIndex: 'table_name',
      key: 'table_name',
      render: (t: string) => t || '全库所有表',
    },
    {
      title: '存储磁盘',
      dataIndex: 'disk',
      key: 'disk',
      render: (d: string) => <Tag>{d}</Tag>,
    },
    {
      title: '备份状态',
      dataIndex: 'status',
      key: 'status',
      render: (st: string) => {
        if (st === 'SUCCESS') return <Tag color="success" icon={<CheckCircleOutlined />}>已完成</Tag>
        if (st === 'RUNNING') return <Tag color="processing" icon={<SyncOutlined spin />}>备份中</Tag>
        return <Tag color="error" icon={<CloseCircleOutlined />}>失败</Tag>
      },
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '完成时间',
      dataIndex: 'completed_at',
      key: 'completed_at',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space size="small">
          {record.status === 'SUCCESS' && (
            <Button
              size="small"
              icon={<UndoOutlined />}
              onClick={() => {
                setRecordToRestore(record)
                setRestoreModalOpen(true)
              }}
            >
              恢复
            </Button>
          )}
          <Button
            size="small"
            danger
            type="text"
            icon={<DeleteOutlined />}
            onClick={() => handleDeleteRecord(record.id)}
          />
        </Space>
      ),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <SaveOutlined />
            <span>ClickHouse 原生备份与恢复 (BACKUP / RESTORE)</span>
            <Tag color="cyan">支持 Native Disk / S3 架构</Tag>
          </Space>
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalOpen(true)}
            >
              创建新备份
            </Button>
            <Button icon={<ReloadOutlined />} onClick={fetchBackups}>
              刷新
            </Button>
          </Space>
        </div>
      }
    >
      <Table
        dataSource={backups}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 15 }}
        size="middle"
      />

      {/* Modal: Create Backup */}
      <Modal
        title="创建 ClickHouse 数据库备份"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => createForm.submit()}
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreateBackup}>
          <Form.Item
            name="database"
            label="备份数据库 (留空备份全实例所有库)"
          >
            <Select
              allowClear
              placeholder="选择指定数据库或全量备份"
              onChange={handleDbChange}
              options={databases.map((d) => ({ label: d.name, value: d.name }))}
            />
          </Form.Item>

          {selectedDb && (
            <Form.Item name="table" label="指定数据表 (可选，默认该库全部数据表)">
              <Select
                allowClear
                placeholder="全部表"
                options={tables.map((t) => ({ label: t.name, value: t.name }))}
              />
            </Form.Item>
          )}

          <Form.Item name="disk" label="目标存储磁盘 (Disk)" initialValue="default">
            <Select
              options={[
                { label: 'default 本地存储', value: 'default' },
                { label: 'backups 专属备份盘', value: 'backups' },
              ]}
            />
          </Form.Item>

          <Form.Item name="backup_id" label="自定义 Backup 标识 (可选)">
            <Input placeholder="留空则按时间戳自动命名" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Modal: Restore Backup Confirmation */}
      <Modal
        title={`危险操作：恢复备份 [${recordToRestore?.backup_id}] 确认`}
        open={restoreModalOpen}
        onCancel={() => {
          setRestoreModalOpen(false)
          setRestoreConfirmInput('')
        }}
        onOk={handleRestoreBackup}
        okButtonProps={{ danger: true, disabled: restoreConfirmInput !== 'RESTORE' }}
        okText="确认恢复"
      >
        <p style={{ color: '#ff4d4f' }}>
          恢复备份将向 ClickHouse 执行 RESTORE 命令覆盖或恢复表数据！
        </p>
        <p>请输入 <strong>RESTORE</strong> 确认执行：</p>
        <Input
          placeholder="输入 RESTORE"
          value={restoreConfirmInput}
          onChange={(e) => setRestoreConfirmInput(e.target.value)}
        />
      </Modal>
    </Card>
  )
}
