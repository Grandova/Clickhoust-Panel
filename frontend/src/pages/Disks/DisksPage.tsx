import React, { useState, useEffect } from 'react'
import {
  Card,
  Row,
  Col,
  Progress,
  Tag,
  Table,
  Space,
  Button,
  Alert,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { HddOutlined, ReloadOutlined, WarningOutlined } from '@ant-design/icons'
import { api } from '../../api/client'
import type { DiskInfo } from '../../types'
import { formatBytes } from '../../utils/format'

export const DisksPage: React.FC = () => {
  const [disks, setDisks] = useState<DiskInfo[]>([])
  const [loading, setLoading] = useState(false)

  const fetchDisks = async () => {
    setLoading(true)
    try {
      const res = await api.getDisks()
      if (res.data.data) {
        setDisks(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDisks()
  }, [])

  const columns: ColumnsType<DiskInfo> = [
    {
      title: '磁盘名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space>
          <HddOutlined style={{ color: '#987b2e' }} />
          <strong>{name}</strong>
        </Space>
      ),
    },
    {
      title: '存储类型',
      dataIndex: 'type',
      key: 'type',
      render: (t: string) => <Tag color="blue">{t || 'local'}</Tag>,
    },
    {
      title: '挂载路径',
      dataIndex: 'path',
      key: 'path',
      render: (p: string) => <code style={{ fontSize: 12 }}>{p}</code>,
    },
    {
      title: '总容量',
      dataIndex: 'total_space',
      key: 'total_space',
      render: (t: number) => formatBytes(t),
    },
    {
      title: '已使用',
      dataIndex: 'used_space',
      key: 'used_space',
      render: (u: number) => formatBytes(u),
    },
    {
      title: '剩余空间',
      dataIndex: 'free_space',
      key: 'free_space',
      render: (f: number) => formatBytes(f),
    },
    {
      title: '使用率',
      dataIndex: 'used_percent',
      key: 'used_percent',
      sorter: (a, b) => a.used_percent - b.used_percent,
      render: (pct: number) => {
        let stroke = '#52c41a'
        if (pct >= 90) stroke = '#ff4d4f'
        else if (pct >= 80) stroke = '#faad14'
        return (
          <div style={{ width: 140 }}>
            <Progress percent={Math.round(pct)} size="small" strokeColor={stroke} />
          </div>
        )
      },
    },
  ]

  const hasHighUsage = disks.some((d) => d.used_percent >= 80)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {hasHighUsage && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message="磁盘容量预警"
          description="部分存储磁盘占用率已超过 80% 警戒阈值。当磁盘打满后，ClickHouse 将无法写入新的 Part 并可能导致查询阻塞或报错，请及时清理数据或扩容！"
        />
      )}

      {/* Disk Gauges Grid */}
      <Row gutter={[16, 16]}>
        {disks.map((disk) => {
          let statusColor = '#52c41a'
          let tagColor = 'success'
          let text = '正常'

          if (disk.used_percent >= 90) {
            statusColor = '#ff4d4f'
            tagColor = 'error'
            text = '紧急 (>=90%)'
          } else if (disk.used_percent >= 80) {
            statusColor = '#faad14'
            tagColor = 'warning'
            text = '告警 (>=80%)'
          }

          return (
            <Col xs={24} sm={12} md={8} key={disk.name}>
              <Card size="small" style={{ borderRadius: 10 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <Space>
                    <HddOutlined style={{ fontSize: 18, color: statusColor }} />
                    <span style={{ fontWeight: 600, fontSize: 15 }}>{disk.name}</span>
                  </Space>
                  <Tag color={tagColor}>{text}</Tag>
                </div>

                <div style={{ textAlign: 'center', margin: '12px 0' }}>
                  <Progress
                    type="dashboard"
                    percent={Math.round(disk.used_percent)}
                    strokeColor={statusColor}
                    size={130}
                  />
                </div>

                <div style={{ fontSize: 12, display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ opacity: 0.7 }}>已用 / 总量:</span>
                    <span><strong>{formatBytes(disk.used_space)}</strong> / {formatBytes(disk.total_space)}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ opacity: 0.7 }}>可用剩余:</span>
                    <span><strong>{formatBytes(disk.free_space)}</strong></span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ opacity: 0.7 }}>挂载路径:</span>
                    <code style={{ fontSize: 11 }}>{disk.path}</code>
                  </div>
                </div>
              </Card>
            </Col>
          )
        })}
      </Row>

      {/* Disks Table */}
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Space>
              <HddOutlined />
              <span>ClickHouse 存储策略与磁盘概况 (system.disks)</span>
            </Space>
            <Button icon={<ReloadOutlined />} onClick={fetchDisks}>
              刷新
            </Button>
          </div>
        }
      >
        <Table
          dataSource={disks}
          columns={columns}
          rowKey="name"
          loading={loading}
          pagination={false}
          size="middle"
        />
      </Card>
    </div>
  )
}
