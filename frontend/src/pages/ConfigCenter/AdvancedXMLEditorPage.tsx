import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Row,
  Col,
  Select,
  Button,
  Space,
  Tag,
  Modal,
  Drawer,
  Table,
  message,
  Alert,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  EditOutlined,
  SaveOutlined,
  HistoryOutlined,
  CheckCircleOutlined,
  ReloadOutlined,
  UndoOutlined,
  FileTextOutlined,
} from '@ant-design/icons'
import Editor, { DiffEditor } from '../../editor'
import { api } from '../../api/client'
import type { ConfigFileInfo, ConfigBackup } from '../../types'
import { formatDateTime } from '../../utils/format'

export const AdvancedXMLEditorPage: React.FC<{ isDark?: boolean }> = ({ isDark = false }) => {
  const [files, setFiles] = useState<ConfigFileInfo[]>([])
  const [selectedFile, setSelectedFile] = useState<string>('config.xml')
  const [content, setContent] = useState<string>('')
  const [originalContent, setOriginalContent] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  // History & Diff
  const [historyOpen, setHistoryOpen] = useState(false)
  const [backups, setBackups] = useState<ConfigBackup[]>([])
  const [diffOpen, setDiffOpen] = useState(false)
  const [diffOriginal, setDiffOriginal] = useState('')

  const fetchFiles = async () => {
    try {
      const res = await api.listConfigFiles()
      if (res.data.data) {
        setFiles(res.data.data)
        if (!res.data.data.some((f) => f.relative_path === selectedFile) && res.data.data.length > 0) {
          setSelectedFile(res.data.data[0].relative_path)
        }
      }
    } catch {
      // Handled
    }
  }

  const loadFileContent = async (path: string) => {
    if (!path) return
    setLoading(true)
    try {
      const res = await api.readConfigFile(path)
      if (res.data.data) {
        setContent(res.data.data.content)
        setOriginalContent(res.data.data.content)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchFiles()
  }, [])

  useEffect(() => {
    if (selectedFile) {
      loadFileContent(selectedFile)
    }
  }, [selectedFile])

  const handleValidate = async () => {
    try {
      const res = await api.validateXML(content)
      if (res.data.data?.valid) {
        message.success('XML 格式与标签闭合验证通过！')
      } else {
        message.error(`XML 语法错误: ${res.data.data?.message}`)
      }
    } catch {
      // Handled
    }
  }

  const handleSave = async () => {
    // Validate first
    const valRes = await api.validateXML(content)
    if (!valRes.data.data?.valid) {
      message.error(`XML 语法检查未通过，已阻止保存: ${valRes.data.data?.message}`)
      return
    }

    setSaving(true)
    try {
      await api.saveConfigFile(selectedFile, content, 'Monaco Web Editor Manual Save')
      message.success('配置文件已安全保存，历史备份已自动归档！')
      setOriginalContent(content)
    } finally {
      setSaving(false)
    }
  }

  const loadBackups = async () => {
    try {
      const res = await api.listConfigBackups(selectedFile)
      if (res.data.data) {
        setBackups(res.data.data)
      }
    } catch {
      // Handled
    }
  }

  const handleRollback = async (backupId: number) => {
    try {
      await api.rollbackConfig(backupId)
      message.success('已成功回滚至所选历史版本！')
      loadFileContent(selectedFile)
      setHistoryOpen(false)
    } catch {
      // Handled
    }
  }

  const isModified = content !== originalContent

  const backupColumns: ColumnsType<ConfigBackup> = [
    {
      title: '备份版本 ID',
      dataIndex: 'id',
      key: 'id',
      render: (id: number) => `#${id}`,
    },
    {
      title: '修改原因 / 备注',
      dataIndex: 'reason',
      key: 'reason',
    },
    {
      title: '操作人',
      dataIndex: 'created_by',
      key: 'created_by',
      render: (u: string) => <Tag color="blue">{u}</Tag>,
    },
    {
      title: '归档时刻',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (t: string) => formatDateTime(t),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, r) => (
        <Space size="small">
          <Button
            size="small"
            onClick={() => {
              setDiffOriginal(r.content)
              setDiffOpen(true)
            }}
          >
            与当前对比
          </Button>
          <Button
            size="small"
            type="primary"
            icon={<UndoOutlined />}
            onClick={() => handleRollback(r.id)}
          >
            还原此版本
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space>
            <EditOutlined />
            <span>高级 XML 配置编辑器 (Monaco Editor)</span>
            <Select
              style={{ width: 260 }}
              value={selectedFile}
              onChange={(val) => setSelectedFile(val)}
              options={files.map((f) => ({
                label: (
                  <Space>
                    <FileTextOutlined />
                    <span>{f.relative_path}</span>
                  </Space>
                ),
                value: f.relative_path,
              }))}
            />
            {isModified && <Tag color="warning">未保存修改</Tag>}
          </Space>

          <Space>
            <Button
              icon={<CheckCircleOutlined />}
              onClick={handleValidate}
            >
              语法检查
            </Button>

            <Button
              icon={<HistoryOutlined />}
              onClick={() => {
                loadBackups()
                setHistoryOpen(true)
              }}
            >
              历史备份与回滚
            </Button>

            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={handleSave}
              loading={saving}
            >
              保存修改 (自动备份)
            </Button>
          </Space>
        </div>
      }
    >
      <div style={{ height: 620, border: isDark ? '1px solid #334155' : '1px solid #e2e8f0', borderRadius: 6, overflow: 'hidden' }}>
        <Editor
          height="100%"
          language="xml"
          theme={isDark ? 'vs-dark' : 'vs'}
          value={content}
          onChange={(val) => setContent(val || '')}
          options={{
            minimap: { enabled: true },
            fontSize: 14,
            wordWrap: 'on',
            lineNumbers: 'on',
            automaticLayout: true,
            formatOnPaste: true,
          }}
        />
      </div>

      {/* Drawer: History Backups */}
      <Drawer
        title={`配置文件历史版本备份: ${selectedFile}`}
        placement="right"
        width={720}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      >
        <Table
          dataSource={backups}
          columns={backupColumns}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          size="middle"
        />
      </Drawer>

      {/* Modal: Diff View */}
      <Modal
        title="版本 Diff 对比 (左: 历史版本, 右: 当前编辑版本)"
        open={diffOpen}
        onCancel={() => setDiffOpen(false)}
        width={1000}
        footer={[
          <Button key="close" onClick={() => setDiffOpen(false)}>
            关闭对比
          </Button>,
        ]}
      >
        <div style={{ height: 480, border: isDark ? '1px solid #334155' : '1px solid #e2e8f0', borderRadius: 6, overflow: 'hidden' }}>
          <DiffEditor
            height="100%"
            language="xml"
            theme={isDark ? 'vs-dark' : 'vs'}
            original={diffOriginal}
            modified={content}
            options={{
              readOnly: true,
              automaticLayout: true,
            }}
          />
        </div>
      </Modal>
    </Card>
  )
}
