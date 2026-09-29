import React, { useState, useEffect, useRef } from 'react'
import {
  Row,
  Col,
  Card,
  Radio,
  Space,
  Button,
} from 'antd'
import {
  LineChartOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react/esm/core'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])
import { api } from '../../api/client'

interface MonitoringPageProps {
  isDark: boolean
}

export const MonitoringPage: React.FC<MonitoringPageProps> = ({ isDark }) => {
  const inFlight = useRef(false)
  const [timeRange, setTimeRange] = useState('5m')
  const [samples, setMetrics] = useState<any[]>([])
  const windowMs = ({ '5m': 300000, '15m': 900000, '1h': 3600000, '6h': 21600000, '24h': 86400000 } as Record<string, number>)[timeRange]
  const metrics = samples.filter((m) => m.timestamp >= Date.now() - windowMs)

  const fetchMetrics = async () => {
    if (inFlight.current || document.visibilityState !== 'visible') return
    inFlight.current = true
    try {
      const res = await api.getRealtimeMetrics()
      if (res.data.data) {
        const item = res.data.data
        const timeStr = new Date().toLocaleTimeString()
        setMetrics((prev) => [
          ...prev.filter((m) => m.timestamp >= Date.now() - 86400000),
          {
            time: timeStr,
            timestamp: Date.now(),
            qps: item.ch?.query_per_second || 0,
            inserts: item.ch?.insert_per_second || 0,
            selects: item.ch?.select_per_second || 0,
            conn: item.ch?.current_connections || 0,
            cpu: item.host?.cpu_percent || 0,
            memPercent: item.host?.mem_percent || 0,
            memBytes: item.ch?.memory_tracking || 0,
            netRx: (item.host?.net_rx_bytes_sec || 0) / 1024, // KB/s
            netTx: (item.host?.net_tx_bytes_sec || 0) / 1024,
            readBytes: (item.ch?.bytes_read_per_second || 0) / 1024,
            writeBytes: (item.ch?.bytes_write_per_second || 0) / 1024,
          },
        ])
      }
    } catch {
      // The API interceptor displays the error.
    } finally {
      inFlight.current = false
    }
  }

  useEffect(() => {
    fetchMetrics()
    const timer = setInterval(fetchMetrics, 10000)
    return () => clearInterval(timer)
  }, [])

  const commonGrid = { top: 36, right: 20, bottom: 28, left: 55 }
  const axisLine = { lineStyle: { color: isDark ? '#444' : '#ddd' } }
  const splitLine = { lineStyle: { color: isDark ? '#222' : '#f0f0f0' } }
  const commonLegend = {
    top: 4,
    right: 16,
    icon: 'circle',
    itemWidth: 8,
    itemHeight: 8,
    itemGap: 14,
    textStyle: { color: isDark ? '#a1a1aa' : '#4b5563', fontSize: 12 },
  }
  const createXAxis = () => ({
    type: 'category' as const,
    data: metrics.map((m) => m.time),
    axisLine,
    axisLabel: { hideOverlap: true, color: isDark ? '#888' : '#666', fontSize: 11 },
  })

  const qpsOption = {
    backgroundColor: 'transparent',
    tooltip: { confine: true, trigger: 'axis' },
    legend: commonLegend,
    grid: commonGrid,
    xAxis: createXAxis(),
    yAxis: { type: 'value', splitLine },
    series: [
      { name: 'Query/s', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.qps), itemStyle: { color: '#b89943' } },
      { name: 'Select/s', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.selects), itemStyle: { color: '#747164' } },
      { name: 'Insert/s', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.inserts), itemStyle: { color: '#7a8c67' } },
    ],
  }

  const connOption = {
    backgroundColor: 'transparent',
    tooltip: { confine: true, trigger: 'axis' },
    legend: commonLegend,
    grid: commonGrid,
    xAxis: createXAxis(),
    yAxis: { type: 'value', splitLine },
    series: [
      {
        name: '活跃连接数 (TCP + HTTP)',
        type: 'line',
        sampling: 'lttb',
        smooth: true,
        data: metrics.map((m) => m.conn),
        itemStyle: { color: '#a58c56' },
        areaStyle: { opacity: 0.15 },
      },
    ],
  }

  const cpuMemOption = {
    backgroundColor: 'transparent',
    tooltip: { confine: true, trigger: 'axis' },
    legend: commonLegend,
    grid: commonGrid,
    xAxis: createXAxis(),
    yAxis: { type: 'value', max: 100, splitLine },
    series: [
      { name: 'CPU 占用率 %', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.cpu), itemStyle: { color: '#ad7955' } },
      { name: 'RAM 占用率 %', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.memPercent), itemStyle: { color: '#7a8c67' } },
    ],
  }

  const ioOption = {
    backgroundColor: 'transparent',
    tooltip: { confine: true, trigger: 'axis', formatter: (params: any) => `${params[0]?.name}<br/>${params.map((p: any) => `${p.seriesName}: ${p.value.toFixed(1)} KB/s`).join('<br/>')}` },
    legend: commonLegend,
    grid: commonGrid,
    xAxis: createXAxis(),
    yAxis: { type: 'value', splitLine },
    series: [
      { name: '网卡接收 (RX)', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.netRx), itemStyle: { color: '#987b2e' } },
      { name: '网卡发送 (TX)', type: 'line', sampling: 'lttb', smooth: true, data: metrics.map((m) => m.netTx), itemStyle: { color: '#747164' } },
    ],
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card size="small" style={{ borderRadius: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space>
            <LineChartOutlined />
            <span style={{ fontWeight: 600 }}>实时监控 · 当前页面采样</span>
          </Space>
          <Space>
            <Radio.Group value={timeRange} onChange={(e) => setTimeRange(e.target.value)} size="small">
              <Radio.Button value="5m">5分钟</Radio.Button>
              <Radio.Button value="15m">15分钟</Radio.Button>
              <Radio.Button value="1h">1小时</Radio.Button>
              <Radio.Button value="6h">6小时</Radio.Button>
              <Radio.Button value="24h">24小时</Radio.Button>
            </Radio.Group>
            <Button icon={<ReloadOutlined />} size="small" onClick={fetchMetrics}>
              刷新
            </Button>
          </Space>
        </div>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="查询并发与吞吐 (QPS / Insert / Select)" size="small" style={{ borderRadius: 10 }}>
            <ReactECharts echarts={echarts} option={qpsOption} style={{ height: 260 }} />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="当前客户端活跃连接数 (TCP / HTTP Connections)" size="small" style={{ borderRadius: 10 }}>
            <ReactECharts echarts={echarts} option={connOption} style={{ height: 260 }} />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="面板所在主机 · CPU / 内存 (%)" size="small" style={{ borderRadius: 10 }}>
            <ReactECharts echarts={echarts} option={cpuMemOption} style={{ height: 260 }} />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="面板所在主机 · 网络 (KB/s)" size="small" style={{ borderRadius: 10 }}>
            <ReactECharts echarts={echarts} option={ioOption} style={{ height: 260 }} />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
