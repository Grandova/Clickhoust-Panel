import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Input,
  Select,
  Button,
  Space,
  Tag,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  SecurityScanOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { AuditLog } from '../../types'
import { formatDateTime } from '../../utils/format'

export const AuditLogPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [action, setAction] = useState('')
  const [loading, setLoading] = useState(false)

  const fetchLogs = async (p = page, ps = pageSize) => {
    setLoading(true)
    try {
      const res = await api.listAuditLogs({
        page: p,
        page_size: ps,
        search,
        action,
      })
      if (res.data.data) {
        setLogs(res.data.data.items)
        setTotal(res.data.data.total)
        setPage(p)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs(1, pageSize)
  }, [action])

  const columns: ColumnsType<AuditLog> = [
    {
      title: '时刻',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '操作用户',
      dataIndex: 'username',
      key: 'username',
      render: (u: string) => <Tag color="blue">{u}</Tag>,
    },
    {
      title: '客户端 IP',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip: string) => <span style={{ fontFamily: 'monospace' }}>{ip}</span>,
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      key: 'action',
      render: (act: string) => {
        let color = 'default'
        if (act.includes('LOGIN')) color = 'cyan'
        else if (act.includes('DROP') || act.includes('DELETE') || act.includes('STOP') || act.includes('UNINSTALL')) color = 'error'
        else if (act.includes('CREATE') || act.includes('START') || act.includes('INSTALL')) color = 'success'
        else if (act.includes('UPDATE') || act.includes('RESTART') || act.includes('SAVE')) color = 'gold'
        return <Tag color={color}>{act}</Tag>
      },
    },
    {
      title: '操作对象 / 目标',
      dataIndex: 'target',
      key: 'target',
      ellipsis: true,
      render: (t: string) => <code style={{ fontSize: 12 }}>{t}</code>,
    },
    {
      title: '执行结果',
      dataIndex: 'result',
      key: 'result',
      render: (r: string, record) =>
        r === 'SUCCESS' ? (
          <Tag color="success" icon={<CheckCircleOutlined />}>成功</Tag>
        ) : (
          <Tooltip title={record.error || '失败'}>
            <Tag color="error" icon={<CloseCircleOutlined />}>失败</Tag>
          </Tooltip>
        ),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space wrap>
            <SecurityScanOutlined />
            <span>系统与运维安全操作审计日志</span>

            <Select
              allowClear
              placeholder="操作类型过滤"
              style={{ width: 180 }}
              value={action || undefined}
              onChange={(val) => setAction(val || '')}
              options={[
                { label: '全部操作', value: '' },
                { label: '登录 (LOGIN)', value: 'LOGIN' },
                { label: '安装 ClickHouse', value: 'INSTALL_CLICKHOUSE' },
                { label: '服务启停操作', value: 'RESTART_SERVICE' },
                { label: '数据库操作 (CREATE/DROP)', value: 'DROP_DATABASE' },
                { label: '数据表修改操作', value: 'DROP_TABLE' },
                { label: '高危 SQL 拦截', value: 'EXECUTE_DANGEROUS_SQL' },
                { label: '配置保存修改', value: 'SAVE_CONFIG_XML' },
              ]}
            />

            <Input
              placeholder="搜索用户、目标或报错关键词..."
              prefix={<SearchOutlined />}
              style={{ width: 260 }}
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
        dataSource={logs}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPageSize(ps)
            fetchLogs(p, ps)
          },
          showTotal: (tot) => `共 ${tot} 条审计记录`,
        }}
        size="middle"
      />
    </Card>
  )
}
