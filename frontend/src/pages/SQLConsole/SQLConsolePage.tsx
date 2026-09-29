import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  Tabs,
  Select,
  Table,
  Tag,
  Tooltip,
  Dropdown,
  message,
  Modal,
  Input,
  Drawer,
  Typography,
  Popconfirm,
  Empty,
  Alert,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  CaretRightOutlined,
  FormatPainterOutlined,
  StarOutlined,
  HistoryOutlined,
  DownloadOutlined,
  PlusOutlined,
  CloseOutlined,
  CopyOutlined,
  DeleteOutlined,
  DatabaseOutlined,
  ApartmentOutlined,
} from '@ant-design/icons'
import Editor, { useMonaco } from '../../editor'
import { api } from '../../api/client'
import type { DatabaseInfo, QueryResult } from '../../types'
import { formatBytes, formatNumber, formatDurationMs } from '../../utils/format'

const { Text } = Typography

interface SQLTab {
  key: string
  title: string
  sql: string
  result?: QueryResult
  loading?: boolean
  error?: string
}

interface SQLConsolePageProps {
  initialSql?: string
  isDark?: boolean
}

export const SQLConsolePage: React.FC<SQLConsolePageProps> = ({ initialSql, isDark = false }) => {
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [selectedDb, setSelectedDb] = useState<string>('default')

  // Tabs state
  const [tabs, setTabs] = useState<SQLTab[]>([
    {
      key: '1',
      title: '查询 1',
      sql: initialSql || 'SELECT version(), timezone(), uptime()',
    },
  ])
  const [activeTabKey, setActiveTabKey] = useState('1')

  // Favorites & History drawer
  const [favoritesOpen, setFavoritesOpen] = useState(false)
  const [favorites, setFavorites] = useState<any[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyList, setHistoryList] = useState<{ sql: string; time: string; elapsed: number }[]>(() => {
    try {
      const saved = localStorage.getItem('clickhouse_manager_query_history')
      return saved ? JSON.parse(saved) : []
    } catch {
      return []
    }
  })

  // Save favorite modal
  const [saveFavOpen, setSaveFavOpen] = useState(false)
  const [favTitle, setFavTitle] = useState('')

  // Result display mode: table | json
  const [resultMode, setResultMode] = useState<'table' | 'json'>('table')

  const editorRef = useRef<any>(null)
  const monaco = useMonaco()

  // Fetch databases
  useEffect(() => {
    api.listDatabases().then((res) => {
      if (res.data.data) {
        setDatabases(res.data.data)
      }
    }).catch(() => {})
  }, [])

  // Setup Monaco completion items for ClickHouse SQL
  useEffect(() => {
    if (!monaco) return

    const clickhouseKeywords = [
      'SELECT', 'FROM', 'WHERE', 'GROUP BY', 'ORDER BY', 'HAVING', 'LIMIT', 'OFFSET',
      'UNION ALL', 'JOIN', 'LEFT JOIN', 'INNER JOIN', 'GLOBAL', 'ARRAY JOIN',
      'PREWHERE', 'SAMPLE', 'SETTINGS', 'FORMAT', 'INSERT INTO', 'VALUES',
      'CREATE TABLE', 'DROP TABLE', 'ALTER TABLE', 'TRUNCATE TABLE', 'OPTIMIZE TABLE', 'FINAL',
      'ENGINE', 'MergeTree', 'ReplacingMergeTree', 'SummingMergeTree', 'AggregatingMergeTree',
      'SHOW TABLES', 'SHOW DATABASES', 'SHOW CREATE TABLE', 'DESCRIBE', 'EXPLAIN',
      'count()', 'uniq()', 'uniqExact()', 'sum()', 'avg()', 'min()', 'max()',
      'now()', 'today()', 'yesterday()', 'toDateTime()', 'toDate()', 'toYYYYMM()',
      'toString()', 'toInt32()', 'toUInt64()', 'arrayJoin()', 'length()',
    ]

    const disposable = monaco.languages.registerCompletionItemProvider('sql', {
      provideCompletionItems: (model, position) => {
        const word = model.getWordUntilPosition(position)
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        }
        const suggestions = clickhouseKeywords.map((kw) => ({
          label: kw,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: kw,
          range,
        }))
        return { suggestions }
      },
    })

    return () => disposable.dispose()
  }, [monaco])

  const activeTab = tabs.find((t) => t.key === activeTabKey) || tabs[0]

  const updateActiveTabSQL = (val: string) => {
    setTabs(
      tabs.map((t) => (t.key === activeTabKey ? { ...t, sql: val } : t))
    )
  }

  const handleExecute = async () => {
    if (!activeTab || !activeTab.sql || activeTab.loading) return

    let sqlToExecute = activeTab.sql
    // If text is selected in editor, execute only the selected text
    if (editorRef.current) {
      const selection = editorRef.current.getSelection()
      if (selection && !selection.isEmpty()) {
        const model = editorRef.current.getModel()
        const selectedText = model.getValueInRange(selection)
        if (selectedText.trim()) {
          sqlToExecute = selectedText.trim()
        }
      }
    }

    setTabs((prev) => prev.map((t) => (t.key === activeTabKey ? { ...t, loading: true, result: undefined, error: undefined } : t)))

    try {
      const res = await api.executeQuery(sqlToExecute, 1000, selectedDb)
      if (res.data.data) {
        const result = res.data.data
        setTabs(
          (prev) => prev.map((t) =>
            t.key === activeTabKey ? { ...t, result, loading: false } : t
          )
        )
        // Record in local history
        const newHistItem = {
          sql: sqlToExecute,
          time: new Date().toLocaleTimeString(),
          elapsed: result.elapsed_ms,
        }
        setHistoryList((prev) => {
          const updated = [newHistItem, ...prev.filter((p) => p.sql !== sqlToExecute).slice(0, 49)]
          try {
            localStorage.setItem('clickhouse_manager_query_history', JSON.stringify(updated))
          } catch {
            // Ignore quota issues
          }
          return updated
        })
      }
    } catch (error) {
      setTabs((prev) => prev.map((t) => (t.key === activeTabKey ? { ...t, loading: false, error: error instanceof Error ? error.message : '查询失败' } : t)))
    }
  }

  const executeRef = useRef(handleExecute)
  useEffect(() => { executeRef.current = handleExecute })

  const handleAddTab = () => {
    const newKey = String(Date.now())
    const newTab: SQLTab = {
      key: newKey,
      title: `查询 ${tabs.length + 1}`,
      sql: 'SELECT * FROM system.metrics LIMIT 20',
    }
    setTabs([...tabs, newTab])
    setActiveTabKey(newKey)
  }

  const handleRemoveTab = (targetKey: string) => {
    if (tabs.length === 1) return
    const nextTabs = tabs.filter((t) => t.key !== targetKey)
    setTabs(nextTabs)
    if (activeTabKey === targetKey) {
      setActiveTabKey(nextTabs[nextTabs.length - 1].key)
    }
  }

  const loadFavorites = async () => {
    try {
      const res = await api.listFavorites()
      if (res.data.data) {
        setFavorites(res.data.data)
      }
    } catch {
      // Ignored
    }
  }

  const handleSaveFavorite = async () => {
    if (!favTitle.trim()) {
      message.error('请输入收藏名称')
      return
    }
    try {
      await api.saveFavorite({
        title: favTitle,
        sql_text: activeTab.sql,
        database: selectedDb,
      })
      message.success('已加入收藏夹')
      setSaveFavOpen(false)
      setFavTitle('')
    } catch {
      // Handled
    }
  }

  const handleExportCSV = () => {
    const result = activeTab?.result
    if (!result || !result.rows.length) {
      message.warning('无数据可导出')
      return
    }
    const headers = result.columns.map((c) => `"${c.name.replace(/"/g, '""')}"`).join(',')
    const rows = result.rows.map((row) =>
      result.columns
        .map((c) => {
          const val = row[c.name]
          if (val === null || val === undefined) return ''
          const str = String(val).replace(/"/g, '""')
          return `"${str}"`
        })
        .join(',')
    )
    const csvContent = 'data:text/csv;charset=utf-8,' + encodeURIComponent('\uFEFF' + [headers, ...rows].join('\n'))
    const encodedUri = csvContent
    const link = document.createElement('a')
    link.setAttribute('href', encodedUri)
    link.setAttribute('download', `clickhouse_query_export_${Date.now()}.csv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleExportJSON = () => {
    const result = activeTab?.result
    if (!result || !result.rows.length) {
      message.warning('无数据可导出')
      return
    }
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(result.rows, null, 2))
    const link = document.createElement('a')
    link.setAttribute('href', dataStr)
    link.setAttribute('download', `clickhouse_query_export_${Date.now()}.json`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleExportTSV = () => {
    const result = activeTab?.result
    if (!result || !result.rows.length) {
      message.warning('无数据可导出')
      return
    }
    const headers = result.columns.map((c) => c.name).join('\t')
    const rows = result.rows.map((row) =>
      result.columns
        .map((c) => {
          const val = row[c.name]
          if (val === null || val === undefined) return '\\N'
          return String(val).replace(/\t/g, ' ').replace(/\n/g, '\\n')
        })
        .join('\t')
    )
    const tsvContent = 'data:text/tab-separated-values;charset=utf-8,' + encodeURIComponent('\uFEFF' + [headers, ...rows].join('\n'))
    const encodedUri = tsvContent
    const link = document.createElement('a')
    link.setAttribute('href', encodedUri)
    link.setAttribute('download', `clickhouse_query_export_${Date.now()}.tsv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleFormatSQL = () => {
    if (!activeTab || !activeTab.sql || activeTab.loading) return
    const raw = activeTab.sql.trim()
    if (!raw) return

    // Standardize and indent major ClickHouse clauses
    let formatted = raw
      .replace(/\s+/g, ' ')
      .replace(/\b(SELECT|FROM|PREWHERE|WHERE|GROUP BY|HAVING|ORDER BY|LIMIT|OFFSET|UNION ALL|UNION DISTINCT|SETTINGS|FORMAT)\b/gi, '\n$1\n  ')
      .replace(/,\s*/g, ',\n  ')
      .replace(/\(\s+/g, '(')
      .replace(/\s+\)/g, ')')
      .trim()

    // Clean up excessive blank lines
    formatted = formatted.split('\n').map((line) => line.trimEnd()).join('\n').replace(/\n{3,}/g, '\n\n')

    updateActiveTabSQL(formatted)
    if (editorRef.current) {
      editorRef.current.setValue(formatted)
    }
    message.success('SQL 已格式化')
  }

  const handleExplain = async (type: 'PLAN' | 'PIPELINE' | 'AST' | 'SYNTAX') => {
    if (!activeTab || !activeTab.sql || activeTab.loading) return
    let baseSql = activeTab.sql.trim()
    baseSql = baseSql.replace(/^EXPLAIN\s+(PLAN\s+|PIPELINE\s+|AST\s+|SYNTAX\s+)?/i, '')
    const explainSql = `EXPLAIN ${type} ${baseSql}`

    setTabs((prev) => prev.map((t) => (t.key === activeTabKey ? { ...t, loading: true, result: undefined, error: undefined } : t)))
    try {
      const res = await api.executeQuery(explainSql, 500, selectedDb)
      if (res.data.data) {
        setTabs(
          (prev) => prev.map((t) =>
            t.key === activeTabKey ? { ...t, result: res.data.data, loading: false } : t
          )
        )
      }
    } catch (error) {
      setTabs((prev) => prev.map((t) => (t.key === activeTabKey ? { ...t, loading: false, error: error instanceof Error ? error.message : '查询失败' } : t)))
    }
  }

  const dynamicColumns: ColumnsType<Record<string, any>> =
    activeTab?.result?.columns.map((col) => ({
      title: (
        <div>
          <div>{col.name}</div>
          <div style={{ fontSize: 10, opacity: 0.6 }}>{col.type}</div>
        </div>
      ),
      dataIndex: ['row', col.name],
      key: col.name,
      ellipsis: true,
      render: (val: any) => {
        if (val === null || val === undefined) {
          return <span style={{ color: '#888', fontStyle: 'italic' }}>NULL</span>
        }
        if (typeof val === 'object') {
          return <span style={{ fontFamily: 'monospace' }}>{JSON.stringify(val)}</span>
        }
        return <span style={{ fontFamily: 'monospace' }}>{String(val)}</span>
      },
    })) || []

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Top Toolbar */}
      <Card size="small" style={{ borderRadius: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space>
            <Select
              style={{ width: 160 }}
              value={selectedDb}
              onChange={(val) => setSelectedDb(val)}
              options={databases.map((d) => ({
                label: (
                  <Space>
                    <DatabaseOutlined />
                    <span>{d.name}</span>
                  </Space>
                ),
                value: d.name,
              }))}
            />

            <Button
              type="primary"
              icon={<CaretRightOutlined />}
              onClick={handleExecute}
              loading={activeTab?.loading}
            >
              执行查询 (Ctrl+Enter)
            </Button>

            <Dropdown
              menu={{
                items: [
                  { key: 'plan', label: 'EXPLAIN PLAN (执行计划)', onClick: () => handleExplain('PLAN') },
                  { key: 'pipeline', label: 'EXPLAIN PIPELINE (执行管道)', onClick: () => handleExplain('PIPELINE') },
                  { key: 'syntax', label: 'EXPLAIN SYNTAX (优化语法)', onClick: () => handleExplain('SYNTAX') },
                  { key: 'ast', label: 'EXPLAIN AST (语法树)', onClick: () => handleExplain('AST') },
                ],
              }}
            >
              <Button icon={<ApartmentOutlined />} loading={activeTab?.loading}>
                执行计划 (EXPLAIN)
              </Button>
            </Dropdown>

            <Button
              icon={<FormatPainterOutlined />}
              onClick={handleFormatSQL}
            >
              格式化
            </Button>

            <Button
              icon={<StarOutlined />}
              onClick={() => {
                setFavTitle(activeTab.title)
                setSaveFavOpen(true)
              }}
            >
              收藏 SQL
            </Button>

            <Button
              icon={<HistoryOutlined />}
              onClick={() => setHistoryOpen(true)}
            >
              执行历史 ({historyList.length})
            </Button>

            <Button
              icon={<StarOutlined />}
              onClick={() => {
                loadFavorites()
                setFavoritesOpen(true)
              }}
            >
              常用收藏
            </Button>
          </Space>

          <Space>
            {activeTab?.result && (
              <Dropdown
                menu={{
                  items: [
                    { key: 'csv', label: '导出 CSV (.csv)', onClick: handleExportCSV },
                    { key: 'tsv', label: '导出 TSV (.tsv)', onClick: handleExportTSV },
                    { key: 'json', label: '导出 JSON (.json)', onClick: handleExportJSON },
                  ],
                }}
              >
                <Button icon={<DownloadOutlined />}>导出结果</Button>
              </Dropdown>
            )}
          </Space>
        </div>
      </Card>

      {/* Editor & Tabs */}
      <Card size="small" style={{ borderRadius: 10, padding: 0 }}>
        <div style={{ padding: '4px 12px', borderBottom: '1px solid rgba(140, 140, 140, 0.2)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Tabs
            type="editable-card"
            activeKey={activeTabKey}
            onChange={(k) => setActiveTabKey(k)}
            onEdit={(targetKey, action) => {
              if (action === 'add') handleAddTab()
              else if (typeof targetKey === 'string') handleRemoveTab(targetKey)
            }}
            items={tabs.map((t) => ({
              key: t.key,
              label: t.title,
              closable: tabs.length > 1,
            }))}
            style={{ marginBottom: -16 }}
          />
        </div>

        <div style={{ height: 240 }}>
          <Editor
            height="100%"
            path={`query-${activeTabKey}.sql`}
            language="sql"
            theme={isDark ? 'vs-dark' : 'vs'}
            value={activeTab.sql}
            onChange={(val) => updateActiveTabSQL(val || '')}
            onMount={(editor, m) => {
              editorRef.current = editor
              // Bind Ctrl+Enter shortcut to execute
              editor.addCommand(m.KeyMod.CtrlCmd | m.KeyCode.Enter, () => {
                void executeRef.current()
              })
            }}
            options={{
              minimap: { enabled: false },
              fontSize: 14,
              wordWrap: 'on',
              lineNumbers: 'on',
              scrollBeyondLastLine: false,
              automaticLayout: true,
            }}
          />
        </div>
      </Card>

      {activeTab?.error && <Alert type="error" showIcon title="查询失败" description={activeTab.error} />}
      {/* Result Section */}
      {activeTab?.result && (
        <Card
          size="small"
          style={{ borderRadius: 10 }}
          title={
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Space split={<span>·</span>} style={{ fontSize: 13 }}>
                <span>执行耗时: <strong>{formatDurationMs(activeTab.result.elapsed_ms)}</strong></span>
                <span>返回行数: <strong>{activeTab.result.total_rows}</strong></span>
                <span>Query ID: <code style={{ fontSize: 11 }}>{activeTab.result.query_id}</code></span>
              </Space>
              <Space>
                <Button
                  size="small"
                  type={resultMode === 'table' ? 'primary' : 'default'}
                  onClick={() => setResultMode('table')}
                >
                  表格视图
                </Button>
                <Button
                  size="small"
                  type={resultMode === 'json' ? 'primary' : 'default'}
                  onClick={() => setResultMode('json')}
                >
                  JSON 视图
                </Button>
              </Space>
            </div>
          }
        >
          {resultMode === 'table' ? (
            <Table
              dataSource={activeTab.result.rows.map((row, key) => ({ key, row }))}
              columns={dynamicColumns}
              rowKey="key"
              pagination={{ pageSize: 50, showSizeChanger: true }}
              scroll={{ x: 'max-content', y: 400 }}
              size="small"
            />
          ) : (
            <pre
              style={{
                background: isDark ? '#141414' : '#f8fafc',
                color: isDark ? '#faad14' : '#0f172a',
                border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
                padding: 16,
                borderRadius: 8,
                maxHeight: 450,
                overflow: 'auto',
                fontFamily: 'Consolas, Monaco, monospace',
                fontSize: 13,
              }}
            >
              {JSON.stringify(activeTab.result.rows, null, 2)}
            </pre>
          )}
        </Card>
      )}

      {/* Modal: Save Favorite */}
      <Modal
        title="收藏当前 SQL 语句"
        open={saveFavOpen}
        onCancel={() => setSaveFavOpen(false)}
        onOk={handleSaveFavorite}
      >
        <div style={{ marginBottom: 12 }}>
          <div style={{ marginBottom: 6 }}>收藏名称:</div>
          <Input
            placeholder="例如: 每日活跃用户统计"
            value={favTitle}
            onChange={(e) => setFavTitle(e.target.value)}
          />
        </div>
        <div>
          <div style={{ marginBottom: 6 }}>SQL 语句:</div>
          <pre style={{ background: isDark ? '#141414' : '#f8fafc', color: isDark ? '#52c41a' : '#15803d', border: isDark ? '1px solid #303030' : '1px solid #e2e8f0', padding: 8, borderRadius: 6, maxHeight: 160, overflow: 'auto' }}>
            {activeTab.sql}
          </pre>
        </div>
      </Modal>

      {/* Drawer: History */}
      <Drawer
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%', paddingRight: 24 }}>
            <span>SQL 执行历史 ({historyList.length})</span>
            {historyList.length > 0 && (
              <Popconfirm
                title="清空历史记录"
                description="确定要清空所有本地保存的 SQL 执行历史吗？"
                onConfirm={() => {
                  setHistoryList([])
                  localStorage.removeItem('clickhouse_manager_query_history')
                  message.success('历史记录已清空')
                }}
              >
                <Button size="small" danger type="link" icon={<DeleteOutlined />}>
                  清空历史
                </Button>
              </Popconfirm>
            )}
          </div>
        }
        placement="right"
        width={500}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      >
        {historyList.length === 0 ? (
          <Empty description="暂无 SQL 执行历史记录" style={{ marginTop: 60 }} />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {historyList.map((item, idx) => (
              <Card key={idx} size="small" style={{ borderRadius: 6 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, fontSize: 12, opacity: 0.8 }}>
                  <span>{item.time}</span>
                  <Tag color="blue">{formatDurationMs(item.elapsed)}</Tag>
                </div>
                <pre
                  style={{
                    background: isDark ? '#141414' : '#f8fafc',
                    color: isDark ? '#ccc' : '#334155',
                    border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
                    padding: 8,
                    borderRadius: 4,
                    fontSize: 12,
                    maxHeight: 120,
                    overflow: 'auto',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                  }}
                >
                  {item.sql}
                </pre>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8 }}>
                  <Space size={4}>
                    <Button
                      size="small"
                      type="link"
                      style={{ padding: 0 }}
                      onClick={() => {
                        updateActiveTabSQL(item.sql)
                        setHistoryOpen(false)
                        message.success('已加载到当前编辑器')
                      }}
                    >
                      加载到当前标签
                    </Button>
                    <span style={{ color: '#666' }}>|</span>
                    <Button
                      size="small"
                      type="link"
                      style={{ padding: 0 }}
                      onClick={() => {
                        const newKey = String(Date.now())
                        const newTab: SQLTab = {
                          key: newKey,
                          title: `查询 ${tabs.length + 1}`,
                          sql: item.sql,
                        }
                        setTabs([...tabs, newTab])
                        setActiveTabKey(newKey)
                        setHistoryOpen(false)
                        message.success('已在新标签页中打开')
                      }}
                    >
                      新标签打开
                    </Button>
                  </Space>
                  <Space size={4}>
                    <Button
                      size="small"
                      type="text"
                      icon={<CopyOutlined />}
                      onClick={() => {
                        navigator.clipboard.writeText(item.sql)
                        message.success('SQL 已复制到剪贴板')
                      }}
                    >
                      复制
                    </Button>
                    <Button
                      size="small"
                      type="text"
                      danger
                      icon={<CloseOutlined />}
                      onClick={() => {
                        setHistoryList((prev) => {
                          const updated = prev.filter((_, i) => i !== idx)
                          try {
                            localStorage.setItem('clickhouse_manager_query_history', JSON.stringify(updated))
                          } catch {
                            // ignore
                          }
                          return updated
                        })
                      }}
                    />
                  </Space>
                </div>
              </Card>
            ))}
          </div>
        )}
      </Drawer>

      {/* Drawer: Favorites */}
      <Drawer
        title="SQL 收藏夹"
        placement="right"
        width={480}
        open={favoritesOpen}
        onClose={() => setFavoritesOpen(false)}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {favorites.map((fav) => (
            <Card
              key={fav.id}
              size="small"
              title={fav.title}
              extra={
                <Button
                  size="small"
                  danger
                  type="text"
                  onClick={async () => {
                    await api.deleteFavorite(fav.id)
                    loadFavorites()
                  }}
                >
                  删除
                </Button>
              }
            >
              <pre
                style={{
                  background: isDark ? '#141414' : '#f8fafc',
                  color: isDark ? '#faad14' : '#0f172a',
                  border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
                  padding: 8,
                  borderRadius: 4,
                  fontSize: 12,
                  maxHeight: 120,
                  overflow: 'auto',
                }}
              >
                {fav.sql_text}
              </pre>
              <Button
                size="small"
                type="link"
                style={{ padding: 0 }}
                onClick={() => {
                  updateActiveTabSQL(fav.sql_text)
                  setFavoritesOpen(false)
                }}
              >
                加载到当前编辑器
              </Button>
            </Card>
          ))}
        </div>
      </Drawer>
    </div>
  )
}
