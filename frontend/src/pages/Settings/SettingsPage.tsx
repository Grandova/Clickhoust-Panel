import React, { useState, useEffect } from 'react'
import {
  Card,
  Form,
  Input,
  InputNumber,
  Checkbox,
  Button,
  Space,
  Tag,
  Divider,
  message,
  Alert,
  Tabs,
  Select,
  Table,
  Typography,
  Descriptions,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ToolOutlined,
  SaveOutlined,
  CheckCircleOutlined,
  SafetyCertificateOutlined,
  InfoCircleOutlined,
  UnlockOutlined,
  GlobalOutlined,
  PlusOutlined,
  DesktopOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { LockedIPInfo, DashboardSummary } from '../../types'

const { Text, Paragraph } = Typography
const { TextArea } = Input

export const SettingsPage: React.FC = () => {
  const [connForm] = Form.useForm()
  const [secForm] = Form.useForm()
  const [savingConn, setSavingConn] = useState(false)
  const [savingSec, setSavingSec] = useState(false)

  const [clientIP, setClientIP] = useState<string>('')
  const [lockedIPs, setLockedIPs] = useState<LockedIPInfo[]>([])
  const [dashboardData, setDashboardData] = useState<DashboardSummary | null>(null)

  const fetchConnectionSettings = async () => {
    try {
      const res = await api.getConnectionSettings()
      if (res.data.data) {
        connForm.setFieldsValue({
          protocol: res.data.data.protocol || 'native',
          host: res.data.data.host || '127.0.0.1',
          port: res.data.data.port || 9000,
          user: res.data.data.user || 'default',
          password: res.data.data.password || '',
          database: res.data.data.database || 'default',
          secure: res.data.data.secure || false,
        })
      }
    } catch {
      // Handled
    }
  }

  const fetchSecuritySettings = async () => {
    try {
      const res = await api.getSecuritySettings()
      if (res.data.data) {
        setClientIP(res.data.data.client_ip || '')
        secForm.setFieldsValue({
          ip_whitelist: res.data.data.ip_whitelist || '',
        })
        setLockedIPs(res.data.data.locked_ips || [])
      }
    } catch {
      // Handled
    }
  }

  const fetchSysInfo = async () => {
    try {
      const res = await api.getDashboardSummary()
      if (res.data.data) {
        setDashboardData(res.data.data)
      }
    } catch {
      // Handled
    }
  }

  useEffect(() => {
    fetchConnectionSettings()
  }, [])

  const handleSaveConnection = async (values: any) => {
    setSavingConn(true)
    try {
      await api.updateConnectionSettings(values)
      message.success('连接成功，配置已保存')
      fetchConnectionSettings()
    } catch {
    } finally {
      setSavingConn(false)
    }
  }

  const handleSaveSecurity = async (values: any) => {
    setSavingSec(true)
    try {
      await api.updateSecuritySettings({ ip_whitelist: values.ip_whitelist })
      message.success('面板访问白名单已成功更新生效！')
      fetchSecuritySettings()
    } catch {
    } finally {
      setSavingSec(false)
    }
  }

  const handleAddCurrentIP = () => {
    if (!clientIP) return
    const currentVal = secForm.getFieldValue('ip_whitelist') || ''
    const list = currentVal
      .split(',')
      .map((s: string) => s.trim())
      .filter(Boolean)
    if (!list.includes(clientIP)) {
      list.push(clientIP)
      secForm.setFieldsValue({ ip_whitelist: list.join(', ') })
      message.info(`已将当前 IP [${clientIP}] 添加至输入框`)
    } else {
      message.warning(`当前 IP [${clientIP}] 已在列表中`)
    }
  }

  const handleUnlockIP = async (ip: string) => {
    try {
      await api.unlockIP(ip)
      message.success(`已解除 IP [${ip}] 的安全锁定`)
      fetchSecuritySettings()
    } catch {
      // Handled
    }
  }

  const lockedColumns: ColumnsType<LockedIPInfo> = [
    {
      title: '受限 IP 地址',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip) => <Tag color="error">{ip}</Tag>,
    },
    {
      title: '失败尝试次数',
      dataIndex: 'failed_attempts',
      key: 'failed_attempts',
      render: (count) => <span style={{ color: '#ff4d4f', fontWeight: 600 }}>{count} 次</span>,
    },
    {
      title: '锁定触发时间',
      dataIndex: 'locked_at',
      key: 'locked_at',
    },
    {
      title: '剩余锁定时长',
      dataIndex: 'remaining_mins',
      key: 'remaining_mins',
      render: (mins) => <span>约 {mins} 分钟</span>,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button
          size="small"
          type="link"
          icon={<UnlockOutlined />}
          onClick={() => handleUnlockIP(record.ip)}
        >
          立即解锁
        </Button>
      ),
    },
  ]

  return (
    <Card
      title={
        <Space>
          <ToolOutlined />
          <span>连接与设置</span>
        </Space>
      }
    >
      <Tabs
        defaultActiveKey="connection"
        onChange={(key) => {
          if (key === 'security') fetchSecuritySettings()
          if (key === 'system') fetchSysInfo()
        }}
        items={[
          {
            key: 'connection',
            label: (
              <Space>
                <DesktopOutlined />
                <span>数据库连接</span>
              </Space>
            ),
            children: (
              <div style={{ maxWidth: 680, paddingTop: 12 }}>
                <p style={{ color: 'var(--panel-muted)', marginBottom: 24 }}>填写 ClickHouse 地址与账号，测试通过后自动保存。</p>

                <Form form={connForm} layout="vertical" onFinish={handleSaveConnection}
                  onValuesChange={(changed, values) => {
                    if ('protocol' in changed || 'secure' in changed) {
                      const previous = connForm.getFieldValue('port')
                      if ([9000, 9440, 8123, 8443].includes(previous)) {
                        connForm.setFieldValue('port', values.protocol === 'http' ? (values.secure ? 8443 : 8123) : (values.secure ? 9440 : 9000))
                      }
                    }
                  }}>

                  <Form.Item
                    name="protocol"
                    label="连接方式"
                    initialValue="native"
                    rules={[{ required: true }]}
                  >
                    <Select
                      options={[
                        { label: 'Native · 通常使用 9000 端口', value: 'native' },
                        { label: 'HTTP · 通常使用 8123 端口', value: 'http' },
                      ]}
                    />
                  </Form.Item>

                  <Form.Item
                    name="host"
                    label="服务器地址"
                    rules={[{ required: true, message: '请输入主机地址' }]}
                    initialValue="127.0.0.1"
                  >
                    <Input placeholder="例如: 127.0.0.1 或 clickhouse.internal" />
                  </Form.Item>

                  <Form.Item
                    name="port"
                    label="端口"
                    rules={[{ required: true, message: '请输入端口号' }]}
                    initialValue={9000}
                  >
                    <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                  </Form.Item>

                  <Form.Item
                    name="user"
                    label="用户名"
                    initialValue="default"
                  >
                    <Input placeholder="默认: default" />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    label="密码"
                  >
                    <Input.Password placeholder="密码 (若无密码留空即可)" />
                  </Form.Item>

                  <Form.Item
                    name="database"
                    label="默认数据库"
                    initialValue="default"
                  >
                    <Input placeholder="默认: default" />
                  </Form.Item>

                  <Form.Item name="secure" valuePropName="checked">
                    <Checkbox>使用 TLS 加密连接</Checkbox>
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type="primary"
                      icon={<SaveOutlined />}
                      htmlType="submit"
                      loading={savingConn}
                    >
                      连接并保存
                    </Button>
                  </Form.Item>
                </Form>
              </div>
            ),
          },
          {
            key: 'security',
            label: (
              <Space>
                <SafetyCertificateOutlined />
                <span>访问安全</span>
              </Space>
            ),
            children: (
              <div style={{ maxWidth: 720, paddingTop: 12 }}>
                <Alert
                  message={
                    <Space>
                      <span>当前访问客户端 IP:</span>
                      <Tag color="blue" style={{ fontSize: 13, fontWeight: 600 }}>
                        {clientIP || '正在检测...'}
                      </Tag>
                    </Space>
                  }
                  description="为防止管理员误操作导致锁死，系统在保存白名单时会进行防锁死安全校验：保存的白名单必须包含您当前的客户端 IP 或所在网段。"
                  type="warning"
                  showIcon
                  style={{ marginBottom: 20 }}
                />

                <Form form={secForm} layout="vertical" onFinish={handleSaveSecurity}>
                  <Form.Item
                    name="ip_whitelist"
                    label={
                      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', alignItems: 'center' }}>
                        <span>面板访问 IP 白名单 (IP Whitelist)</span>
                        {clientIP && (
                          <Button
                            type="dashed"
                            size="small"
                            icon={<PlusOutlined />}
                            onClick={handleAddCurrentIP}
                          >
                            快捷添加当前 IP ({clientIP})
                          </Button>
                        )}
                      </div>
                    }
                    extra="支持单个 IP (如: 192.168.1.50) 或 CIDR 网段 (如: 10.0.0.0/8, 192.168.1.0/24)。多个以英文逗号分隔。留空或输入 * 表示不限制，允许任意网络访问。"
                  >
                    <TextArea
                      rows={3}
                      placeholder="例如: 127.0.0.1, 192.168.1.0/24, 10.0.0.1 (留空允许所有 IP)"
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type="primary"
                      icon={<SaveOutlined />}
                      htmlType="submit"
                      loading={savingSec}
                    >
                      保存安全白名单设置
                    </Button>
                  </Form.Item>
                </Form>

                <Divider style={{ margin: '24px 0' }} />

                <div style={{ marginBottom: 16 }}>
                  <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 6 }}>
                    暴力破解防御与 IP 锁定记录
                  </div>
                  <Paragraph type="secondary" style={{ fontSize: 13 }}>
                    系统内置登录防暴力破解机制：同一 IP 连续输错密码达 5 次后，系统将自动对该 IP 实施 15 分钟的安全限制。在此期间该 IP 无法提交登录尝试。
                  </Paragraph>
                </div>

                {lockedIPs.length > 0 ? (
                  <Table
                    columns={lockedColumns}
                    dataSource={lockedIPs}
                    rowKey="ip"
                    size="small"
                    pagination={false}
                  />
                ) : (
                  <div
                    style={{
                      padding: 16,
                      background: 'rgba(82, 196, 26, 0.08)',
                      borderRadius: 8,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      color: '#52c41a',
                    }}
                  >
                    <CheckCircleOutlined />
                    <span>当前无被安全锁定的客户端 IP，系统安全状态正常。</span>
                  </div>
                )}
              </div>
            ),
          },
          {
            key: 'system',
            label: (
              <Space>
                <InfoCircleOutlined />
                <span>系统与环境信息</span>
              </Space>
            ),
            children: (
              <div style={{ maxWidth: 760, paddingTop: 12 }}>
                <Descriptions bordered column={2} size="middle">
                  <Descriptions.Item label="面板系统版本">
                    <Tag color="cyan">ClickHouse Manager v1.0.0</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="服务端技术栈">
                    <span>Go (Gin + GORM + Pure-Go SQLite)</span>
                  </Descriptions.Item>

                  <Descriptions.Item label="前端技术栈">
                    <span>React 19 + TypeScript + Ant Design v5 + Monaco</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="持久化存储模式">
                    <span>SQLite WAL 模式 (零外部依赖单二进制分发)</span>
                  </Descriptions.Item>

                  <Descriptions.Item label="宿主操作系统">
                    <span>{dashboardData?.os_info?.pretty_name || dashboardData?.os_info?.os || 'Linux'}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="CPU 硬件架构">
                    <span>{dashboardData?.os_info?.arch || 'x86_64'}</span>
                  </Descriptions.Item>

                  <Descriptions.Item label="ClickHouse 目标版本">
                    <Tag color="geekblue">{dashboardData?.ch_version || '未连接 / 离线'}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="systemd 服务状态">
                    {dashboardData?.service?.active ? (
                      <Tag color="success">running (PID: {dashboardData.service.pid})</Tag>
                    ) : (
                      <Tag color="default">stopped</Tag>
                    )}
                  </Descriptions.Item>
                </Descriptions>
              </div>
            ),
          },
        ]}
      />
    </Card>
  )
}
