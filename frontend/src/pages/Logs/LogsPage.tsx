import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Radio,
  Select,
  Input,
  Button,
  Space,
  Tag,
  Switch,
  message,
  Tooltip,
} from 'antd'
import {
  FileTextOutlined,
  SearchOutlined,
  ReloadOutlined,
  CopyOutlined,
  DownloadOutlined,
  ClearOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'

export const LogsPage: React.FC = () => {
  const [logType, setLogType] = useState<string>('server')
  const [lines, setLines] = useState<number>(500)
  const [level, setLevel] = useState<string>('')
  const [filter, setFilter] = useState<string>('')
  const [logLines, setLogLines] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [liveMode, setLiveMode] = useState(false)

  const wsRef = useRef<WebSocket | null>(null)
  const logContainerRef = useRef<HTMLDivElement | null>(null)

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const res = await api.getLogs({
        type: logType,
        lines,
        level,
        filter,
      })
      if (res.data.data) {
        setLogLines(res.data.data.lines)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!liveMode) {
      fetchLogs()
    }
  }, [logType, lines, level])

  // Live WebSocket tail
  useEffect(() => {
    if (liveMode) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const token = localStorage.getItem('token') || ''
      const wsUrl = `${protocol}//${window.location.host}/api/v1/logs/ws?type=${logType}&token=${token}`

      const ws = new WebSocket(wsUrl)
      ws.onmessage = (event) => {
        setLogLines((prev) => [...prev.slice(-4999), event.data])
      }
      wsRef.current = ws

      return () => {
        ws.close()
        wsRef.current = null
      }
    } else {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [liveMode, logType])

  // Scroll to bottom on updates
  useEffect(() => {
    if (liveMode && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logLines, liveMode])

  const handleCopy = () => {
    if (!logLines.length) return
    navigator.clipboard.writeText(logLines.join('\n'))
    message.success('日志已复制到剪贴板')
  }

  const handleDownload = () => {
    if (!logLines.length) return
    const blob = new Blob([logLines.join('\n')], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `clickhouse_${logType}_${Date.now()}.log`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space wrap>
            <FileTextOutlined />
            <Radio.Group value={logType} onChange={(e) => setLogType(e.target.value)} buttonStyle="solid">
              <Radio.Button value="server">主服务日志 (Server Log)</Radio.Button>
              <Radio.Button value="error">错误日志 (Error Log)</Radio.Button>
              <Radio.Button value="journal">Systemd Journal</Radio.Button>
            </Radio.Group>

            <Select
              allowClear
              placeholder="日志级别"
              style={{ width: 130 }}
              value={level || undefined}
              onChange={(val) => setLevel(val || '')}
              options={[
                { label: '全部级别', value: '' },
                { label: 'TRACE', value: 'trace' },
                { label: 'DEBUG', value: 'debug' },
                { label: 'INFO', value: 'information' },
                { label: 'WARNING', value: 'warning' },
                { label: 'ERROR', value: 'error' },
              ]}
            />

            <Select
              style={{ width: 110 }}
              value={lines}
              onChange={(val) => setLines(val)}
              options={[
                { label: '100 行', value: 100 },
                { label: '500 行', value: 500 },
                { label: '1000 行', value: 1000 },
                { label: '5000 行', value: 5000 },
              ]}
            />

            <Input
              placeholder="搜索文本关键词..."
              prefix={<SearchOutlined />}
              style={{ width: 220 }}
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              onPressEnter={fetchLogs}
            />

            <Button type="primary" icon={<SearchOutlined />} onClick={fetchLogs}>
              查询
            </Button>
          </Space>

          <Space>
            <Space>
              <ThunderboltOutlined style={{ color: liveMode ? '#52c41a' : '#888' }} />
              <span>实时 Tail:</span>
              <Switch checked={liveMode} onChange={(checked) => setLiveMode(checked)} />
            </Space>

            <Button icon={<CopyOutlined />} onClick={handleCopy}>
              复制
            </Button>
            <Button icon={<DownloadOutlined />} onClick={handleDownload}>
              下载
            </Button>
            <Button icon={<ClearOutlined />} onClick={() => setLogLines([])}>
              清空
            </Button>
            {!liveMode && (
              <Button icon={<ReloadOutlined />} onClick={fetchLogs} loading={loading}>
                刷新
              </Button>
            )}
          </Space>
        </div>
      }
    >
      <div
        ref={logContainerRef}
        style={{
          background: '#0c0d10',
          color: '#d1d5db',
          padding: '16px',
          borderRadius: '8px',
          fontFamily: 'Consolas, Monaco, "Courier New", monospace',
          fontSize: '12px',
          height: 600,
          overflowY: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
          border: '1px solid #262626',
        }}
      >
        {logLines.length > 0 ? (
          logLines.map((line, idx) => {
            let color = '#d1d5db'
            if (line.includes('<Error>')) color = '#ff7875'
            else if (line.includes('<Warning>')) color = '#ffc069'
            else if (line.includes('<Information>')) color = '#95de64'
            else if (line.includes('<Debug>')) color = '#85a5ff'
            return (
              <div key={idx} style={{ color, lineHeight: '1.6' }}>
                {line}
              </div>
            )
          })
        ) : (
          <div style={{ color: '#666', textAlign: 'center', marginTop: 100 }}>
            暂无匹配日志记录
          </div>
        )}
      </div>
    </Card>
  )
}
