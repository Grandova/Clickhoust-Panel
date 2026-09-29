import React, { useState, useEffect } from 'react'
import {
  Card,
  Tabs,
  Form,
  Input,
  Select,
  Button,
  Tag,
  Space,
  Row,
  Col,
  Alert,
  message,
  Tooltip,
} from 'antd'
import {
  SettingOutlined,
  SaveOutlined,
  SearchOutlined,
  ReloadOutlined,
  InfoCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { ConfigItemMeta } from '../../types'

export const ConfigCenterPage: React.FC = () => {
  const [items, setItems] = useState<ConfigItemMeta[]>([])
  const [search, setSearch] = useState('')
  const [activeTab, setActiveTab] = useState('all')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const fetchConfigs = async () => {
    setLoading(true)
    try {
      const res = await api.getFormConfig()
      if (res.data.data) {
        setItems(res.data.data)
        const initialVals: Record<string, string> = {}
        res.data.data.forEach((item) => {
          initialVals[item.key] = item.current_value
        })
        form.setFieldsValue(initialVals)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchConfigs()
  }, [])

  const handleSave = async (values: any) => {
    setSaving(true)
    try {
      await api.saveFormConfig(values)
      message.success('配置已保存并自动生成历史备份！如涉及需重启项请前往服务管理重启。')
      fetchConfigs()
    } finally {
      setSaving(false)
    }
  }

  // Filter items
  const filteredItems = items.filter((item) => {
    const matchCategory = activeTab === 'all' || item.category === activeTab
    const matchSearch =
      !search ||
      item.key.toLowerCase().includes(search.toLowerCase()) ||
      item.description.toLowerCase().includes(search.toLowerCase())
    return matchCategory && matchSearch
  })

  const categories = [
    { key: 'all', label: '全部参数' },
    { key: 'network', label: '网络通信 (Network)' },
    { key: 'memory', label: '内存管理 (Memory)' },
    { key: 'query', label: '查询限制 (Query)' },
    { key: 'cache', label: '缓存设置 (Cache)' },
    { key: 'mergetree', label: 'MergeTree 引擎' },
    { key: 'log', label: '日志配置 (Logger)' },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space>
            <SettingOutlined />
            <span>ClickHouse 可视化配置中心</span>
            <Tag color="cyan">Web 表单安全修改模式</Tag>
          </Space>
          <Space>
            <Input
              placeholder="搜索参数名称或说明 (如 memory)..."
              prefix={<SearchOutlined />}
              style={{ width: 280 }}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
            />
            <Button icon={<ReloadOutlined />} onClick={fetchConfigs}>
              重置
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={() => form.submit()}
              loading={saving}
            >
              保存所有配置变更
            </Button>
          </Space>
        </div>
      }
    >
      <Alert
        message="配置修改安全保证"
        description="所有通过表单修改的参数都会安全写入 /etc/clickhouse-server/config.d/ch_manager_overrides.xml 中进行覆盖，且在写入前自动创建时间戳备份版本，避免破坏原始配置文件。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Tabs
        activeKey={activeTab}
        onChange={(k) => setActiveTab(k)}
        items={categories.map((c) => ({ key: c.key, label: c.label }))}
        style={{ marginBottom: 16 }}
      />

      <Form form={form} layout="vertical" onFinish={handleSave}>
        <Row gutter={[24, 16]}>
          {filteredItems.map((item) => (
            <Col xs={24} md={12} key={item.key}>
              <Card
                size="small"
                style={{
                  borderRadius: 8,
                  height: '100%',
                  background: 'rgba(140, 140, 140, 0.03)',
                  border: '1px solid rgba(140, 140, 140, 0.15)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <Space>
                    <code style={{ fontSize: 14, fontWeight: 700 }}>{item.key}</code>
                    <Tag color="blue">{item.category}</Tag>
                  </Space>
                  {item.requires_restart ? (
                    <Tooltip title="修改此项后需要重启 ClickHouse 服务才能生效">
                      <Tag color="volcano" icon={<WarningOutlined />}>需重启</Tag>
                    </Tooltip>
                  ) : (
                    <Tag color="green">热重载生效</Tag>
                  )}
                </div>

                <div style={{ fontSize: 12, opacity: 0.8, marginBottom: 12 }}>
                  {item.description}
                </div>

                <Form.Item name={item.key} style={{ marginBottom: 8 }}>
                  {item.type === 'select' && item.options ? (
                    <Select
                      options={item.options.map((opt) => ({ label: opt, value: opt }))}
                    />
                  ) : (
                    <Input placeholder={`默认值: ${item.default_value}`} />
                  )}
                </Form.Item>

                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, opacity: 0.65 }}>
                  <span>默认值: <code>{item.default_value}</code></span>
                  <span>推荐: {item.recommended}</span>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </Form>
    </Card>
  )
}
