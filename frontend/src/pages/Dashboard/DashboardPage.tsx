import React, { useState, useEffect } from 'react'
import { Row, Col, Card, Statistic, Tag, Button, Alert, Space } from 'antd'
import { DatabaseOutlined, SearchOutlined, CodeOutlined, ArrowRightOutlined, ReloadOutlined } from '@ant-design/icons'
import { Button as HeroButton } from '@heroui/react'
import { api } from '../../api/client'
import type { DashboardSummary } from '../../types'
import { formatBytes, formatNumber } from '../../utils/format'

interface DashboardPageProps {
  onNavigate: (key: string) => void
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigate }) => {
  const [data, setData] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [refresh, setRefresh] = useState(0)

  useEffect(() => {
    let cancelled = false
    let timer: ReturnType<typeof setTimeout>
    const fetchData = async () => {
      try {
        const res = await api.getDashboardSummary()
        if (!cancelled) {
          setData(res.data.data)
          setError(false)
        }
      } catch {
        if (!cancelled) setError(true)
      } finally {
        if (!cancelled) {
          setLoading(false)
          timer = setTimeout(() => {
            if (document.visibilityState === 'visible') fetchData()
          }, 10000)
        }
      }
    }
    const onVisible = () => {
      if (document.visibilityState === 'visible') setRefresh((n) => n + 1)
    }
    setLoading(true)
    fetchData()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      cancelled = true
      clearTimeout(timer)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [refresh])

  return (
    <div className="home-page">
      <div className="dashboard-heading">
        <div><h1>首页</h1><p>连接数据库，开始查看和查询数据。</p></div>
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => setRefresh((n) => n + 1)}>刷新</Button>
      </div>
      {error && <Alert type="error" showIcon title="状态获取失败" description="请检查面板网络连接后重试。" />}
      <Card className="connection-card" loading={loading && !data}>
        <div className="connection-heading">
          <Space><span className="connection-dot" data-connected={!!data?.ch_reachable} /><strong>ClickHouse 连接</strong></Space>
          <Tag color={data?.ch_reachable ? 'success' : 'default'}>{data ? (data.ch_reachable ? '已连接' : '未连接') : '检测中'}</Tag>
        </div>
        <h2>{data?.ch_reachable ? '数据库已就绪' : '从连接数据库开始'}</h2>
        <p>{data?.ch_reachable ? `ClickHouse ${data.ch_version} · 可以查看数据或执行 SQL。` : '填写服务器地址和账号，连接本地或远程 ClickHouse。'}</p>
        <HeroButton onPress={() => onNavigate(data?.ch_reachable ? 'data-browser' : 'settings')}>
          {data?.ch_reachable ? '查看数据' : '连接数据库'}<ArrowRightOutlined />
        </HeroButton>
      </Card>
      <Row gutter={[16, 16]}>
        {[
          { key: 'databases', icon: <DatabaseOutlined />, title: '数据库与表', text: '查看表结构，创建表或导入数据。' },
          { key: 'data-browser', icon: <SearchOutlined />, title: '查看数据', text: '选择一张表，直接浏览和导出数据。' },
          { key: 'sql-console', icon: <CodeOutlined />, title: 'SQL 查询', text: '编写 SQL，查询和分析数据。' },
        ].map((item) => (
          <Col xs={24} md={8} key={item.key}>
            <button className="home-action" onClick={() => onNavigate(item.key)}>
              {item.icon}<strong>{item.title}</strong><span>{item.text}</span><ArrowRightOutlined />
            </button>
          </Col>
        ))}
      </Row>
      {data?.ch_reachable && data.metrics && <Card size="small">
        <Row gutter={[24, 16]}>
          <Col xs={12} md={6}><Statistic title="数据库" value={data.metrics.total_databases} /></Col>
          <Col xs={12} md={6}><Statistic title="数据表" value={data.metrics.total_tables} /></Col>
          <Col xs={12} md={6}><Statistic title="总行数" value={formatNumber(data.metrics.total_rows)} /></Col>
          <Col xs={12} md={6}><Statistic title="数据占用" value={formatBytes(data.metrics.total_bytes)} /></Col>
        </Row>
      </Card>}
      <div className="home-footer"><span>监控、备份及本机服务管理位于「高级功能」。</span><Button type="text" onClick={() => onNavigate('monitoring')}>查看监控 <ArrowRightOutlined /></Button></div>
    </div>
  )
}
