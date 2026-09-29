import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Tag,
  Space,
  Modal,
  Input,
  Checkbox,
  message,
  Descriptions,
  Typography,
  Alert,
  Divider,
} from 'antd'
import {
  CaretRightOutlined,
  PauseOutlined,
  ReloadOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  DeleteOutlined,
  ArrowUpOutlined,
  SafetyCertificateOutlined,
  ConsoleSqlOutlined,
  ClearOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { ServiceStatus, OSInfo } from '../../types'

const { Text, Title } = Typography

export const ClickHouseOpsPage: React.FC<{ isDark?: boolean }> = ({ isDark = false }) => {
  const [status, setStatus] = useState<ServiceStatus | null>(null)
  const [osInfo, setOsInfo] = useState<OSInfo | null>(null)
  const [installerName, setInstallerName] = useState<string>('')
  const [isInstalling, setIsInstalling] = useState(false)
  const [loading, setLoading] = useState(false)

  // Install modal & state
  const [installModalOpen, setInstallModalOpen] = useState(false)
  const [adminPassword, setAdminPassword] = useState('')
  const [installLogs, setInstallLogs] = useState<string[]>([])
  const wsRef = useRef<WebSocket | null>(null)
  const terminalEndRef = useRef<HTMLDivElement | null>(null)

  // Dangerous action modals
  const [stopModalOpen, setStopModalOpen] = useState(false)
  const [stopConfirmInput, setStopConfirmInput] = useState('')

  const [uninstallModalOpen, setUninstallModalOpen] = useState(false)
  const [uninstallConfirmInput, setUninstallConfirmInput] = useState('')
  const [purgeData, setPurgeData] = useState(false)

  const fetchStatus = async () => {
    try {
      const res = await api.getServiceStatus()
      if (res.data.data) {
        setStatus(res.data.data)
      }
      const instRes = await api.getInstallerInfo()
      if (instRes.data.data) {
        setOsInfo(instRes.data.data.os_info)
        setInstallerName(instRes.data.data.installer)
        setIsInstalling(instRes.data.data.is_installing)
      }
    } catch {
      // Ignored
    }
  }

  useEffect(() => {
    fetchStatus()
    const timer = setInterval(fetchStatus, 4000)
    return () => clearInterval(timer)
  }, [])

  // Auto-scroll terminal logs
  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [installLogs])

  const connectInstallWS = () => {
    if (wsRef.current) {
      wsRef.current.close()
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = localStorage.getItem('token') || ''
    const wsUrl = `${protocol}//${window.location.host}/api/v1/installer/logs/ws?token=${token}`

    const ws = new WebSocket(wsUrl)
    ws.onmessage = (event) => {
      setInstallLogs((prev) => [...prev, event.data])
    }
    ws.onclose = () => {
      setIsInstalling(false)
      fetchStatus()
    }
    wsRef.current = ws
  }

  const handleStart = async () => {
    setLoading(true)
    try {
      await api.startService()
      message.success('ClickHouse 服务已启动')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleStop = async () => {
    if (stopConfirmInput !== 'STOP') {
      message.error('请输入 STOP 确认停止服务')
      return
    }
    setLoading(true)
    try {
      await api.stopService(stopConfirmInput)
      message.success('ClickHouse 服务已停止')
      setStopModalOpen(false)
      setStopConfirmInput('')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleRestart = async () => {
    setLoading(true)
    try {
      await api.restartService()
      message.success('ClickHouse 服务已重启')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleReload = async () => {
    setLoading(true)
    try {
      await api.reloadService()
      message.success('ClickHouse 服务配置已热重载')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleEnable = async () => {
    setLoading(true)
    try {
      await api.enableService()
      message.success('已开启开机自启')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleDisable = async () => {
    setLoading(true)
    try {
      await api.disableService()
      message.success('已取消开机自启')
      fetchStatus()
    } finally {
      setLoading(false)
    }
  }

  const handleValidateConfig = async () => {
    setLoading(true)
    try {
      const res = await api.validateServiceConfig()
      Modal.info({
        title: '配置有效性验证结果',
        content: <pre style={{ maxHeight: 300, overflow: 'auto' }}>{res.data.data?.output || '配置文件语法正确，未发现致命错误。'}</pre>,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleTruncateSystemLogs = async () => {
    Modal.confirm({
      title: '确认清理系统日志表数据？',
      content: '此操作将清空 system.asynchronous_metric_log、system.metric_log、system.trace_log 等内部监控历史数据，立即释放磁盘空间，不会影响任何业务数据与表结构。',
      okText: '立即清理',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setLoading(true)
        try {
          const res = await api.truncateSystemLogs()
          message.success(res.data.message || '系统日志表已成功清空！')
          fetchStatus()
        } finally {
          setLoading(false)
        }
      },
    })
  }

  const handleInstall = async () => {
    setInstallLogs([])
    connectInstallWS()
    try {
      await api.installClickHouse(adminPassword)
      setIsInstalling(true)
      setInstallModalOpen(false)
      message.info('安装任务已启动，正在实时流式输出日志...')
    } catch {
      // Handled
    }
  }

  const handleUninstall = async () => {
    if (uninstallConfirmInput !== 'UNINSTALL') {
      message.error('请输入 UNINSTALL 确认卸载')
      return
    }
    connectInstallWS()
    try {
      await api.uninstallClickHouse(uninstallConfirmInput, purgeData)
      setUninstallModalOpen(false)
      setUninstallConfirmInput('')
      message.info('卸载任务已提交后台执行')
    } catch {
      // Handled
    }
  }

  const handleUpgrade = async () => {
    connectInstallWS()
    try {
      await api.upgradeClickHouse()
      message.info('升级任务已提交后台执行')
    } catch {
      // Handled
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Installation Banner / Status Card */}
      <Card
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>ClickHouse 服务控制台</span>
            <Space>
              {status?.installed ? (
                status.active ? (
                  <Tag color="success">Active (Running)</Tag>
                ) : (
                  <Tag color="error">Inactive (Stopped)</Tag>
                )
              ) : (
                <Tag color="warning">Not Installed</Tag>
              )}
              {status?.enabled && <Tag color="blue">Enabled on boot</Tag>}
            </Space>
          </div>
        }
      >
        <Row gutter={[24, 24]}>
          <Col xs={24} md={14}>
            <Descriptions bordered size="small" column={{ xs: 1, sm: 2 }}>
              <Descriptions.Item label="操作系统">
                {osInfo?.pretty_name || 'Linux'} ({osInfo?.arch || 'x86_64'})
              </Descriptions.Item>
              <Descriptions.Item label="推荐包管理器">
                {osInfo?.pkg_type ? osInfo.pkg_type.toUpperCase() : 'APT / DNF'}
              </Descriptions.Item>
              <Descriptions.Item label="Server 版本">
                {status?.server_version || '未安装'}
              </Descriptions.Item>
              <Descriptions.Item label="Client 版本">
                {status?.client_version || '未安装'}
              </Descriptions.Item>
              <Descriptions.Item label="进程 PID">
                {status?.pid || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="运行时长">
                {status?.uptime || '-'}
              </Descriptions.Item>
            </Descriptions>

            <Divider style={{ margin: '16px 0' }} />

            {/* Action buttons */}
            <Space wrap size={[12, 12]}>
              {!status?.installed ? (
                <Button
                  type="primary"
                  icon={<DownloadOutlined />}
                  size="large"
                  onClick={() => setInstallModalOpen(true)}
                  disabled={isInstalling}
                >
                  一键安装 ClickHouse
                </Button>
              ) : (
                <>
                  {!status.active ? (
                    <Button
                      type="primary"
                      icon={<CaretRightOutlined />}
                      onClick={handleStart}
                      loading={loading}
                    >
                      启动服务
                    </Button>
                  ) : (
                    <Button
                      danger
                      icon={<PauseOutlined />}
                      onClick={() => setStopModalOpen(true)}
                      loading={loading}
                    >
                      停止服务
                    </Button>
                  )}

                  <Button
                    icon={<ReloadOutlined />}
                    onClick={handleRestart}
                    loading={loading}
                  >
                    重启服务
                  </Button>

                  <Button
                    icon={<SyncOutlined />}
                    onClick={handleReload}
                    loading={loading}
                  >
                    热重载配置
                  </Button>

                  {status.enabled ? (
                    <Button onClick={handleDisable} loading={loading}>
                      取消开机自启
                    </Button>
                  ) : (
                    <Button type="dashed" onClick={handleEnable} loading={loading}>
                      设为开机自启
                    </Button>
                  )}

                  <Button
                    icon={<SafetyCertificateOutlined />}
                    onClick={handleValidateConfig}
                    loading={loading}
                  >
                    校验配置
                  </Button>

                  <Button
                    icon={<ClearOutlined />}
                    onClick={handleTruncateSystemLogs}
                    loading={loading}
                  >
                    清理系统日志
                  </Button>

                  <Button
                    icon={<ArrowUpOutlined />}
                    onClick={handleUpgrade}
                    loading={loading}
                  >
                    检查升级
                  </Button>

                  <Button
                    danger
                    type="text"
                    icon={<DeleteOutlined />}
                    onClick={() => setUninstallModalOpen(true)}
                  >
                    卸载 ClickHouse
                  </Button>
                </>
              )}
            </Space>
          </Col>

          {/* Systemd detail output */}
          <Col xs={24} md={10}>
            <div style={{ fontWeight: 600, marginBottom: 8, fontSize: 13 }}>
              systemd 服务状态输出:
            </div>
            <div
              style={{
                background: isDark ? '#141414' : '#f8fafc',
                color: isDark ? '#52c41a' : '#15803d',
                padding: '12px',
                borderRadius: '8px',
                fontFamily: 'Consolas, Monaco, monospace',
                fontSize: '12px',
                height: 200,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
                border: isDark ? '1px solid #303030' : '1px solid #e2e8f0',
              }}
            >
              {status?.status_text || '暂无 systemd 详细状态输出'}
            </div>
          </Col>
        </Row>
      </Card>

      {/* Real-time Installation & Task Terminal Log */}
      {(installLogs.length > 0 || isInstalling) && (
        <Card
          title="实时安装 / 运维执行日志"
          extra={
            <Button size="small" onClick={() => setInstallLogs([])}>
              清空输出
            </Button>
          }
        >
          <div style={{ borderRadius: 8, overflow: 'hidden', border: isDark ? '1px solid #334155' : '1px solid #cbd5e1' }}>
            <div style={{ background: '#1e293b', padding: '8px 12px', display: 'flex', alignItems: 'center', gap: 6 }}>
              <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#ef4444' }} />
              <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#f59e0b' }} />
              <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#10b981' }} />
              <span style={{ color: '#94a3b8', fontSize: 11, marginLeft: 8, fontFamily: 'monospace' }}>
                terminal output - install stream
              </span>
            </div>
            <div
              style={{
                background: '#0f172a',
                color: '#4ade80',
                padding: '14px',
                fontFamily: 'Consolas, Monaco, "Courier New", monospace',
                fontSize: '12px',
                maxHeight: 380,
                overflowY: 'auto',
                whiteSpace: 'pre-wrap',
              }}
            >
              {installLogs.map((log, idx) => (
                <div key={idx} style={{ lineHeight: '1.6' }}>
                  {log}
                </div>
              ))}
              <div ref={terminalEndRef} />
            </div>
          </div>
        </Card>
      )}

      {/* Modal: One-click Install */}
      <Modal
        title="一键安装 ClickHouse 数据库"
        open={installModalOpen}
        onCancel={() => setInstallModalOpen(false)}
        onOk={handleInstall}
        okText="开始自动安装"
      >
        <Alert
          type="info"
          showIcon
          message="CPU 指令集与虚拟化环境须知"
          description="ClickHouse 预编译二进制强制要求 CPU 支持 SSE 4.2 向量指令集。如果您使用虚拟机（如 PVE / KVM / OpenStack / VMware 等），请务必在宿主机将该虚拟机的 CPU 类型设置为 'host'（CPU 主机穿透模式），避免因虚拟 CPU 指令缺失导致崩溃（Illegal instruction）。"
          style={{ marginBottom: 16 }}
        />
        <p style={{ fontSize: 13, color: '#888' }}>
          安装程序将自动检测系统（{osInfo?.pretty_name || 'Linux'} {osInfo?.arch}）、导入 ClickHouse 官方 GPG Key、添加稳定版官方软件源、安装 clickhouse-server 与 clickhouse-client、开放 0.0.0.0 监听，并初始化 systemd 守护进程。
        </p>
        <div style={{ marginTop: 16 }}>
          <div style={{ marginBottom: 6, fontWeight: 500 }}>管理员 default 密码（可选）:</div>
          <Input.Password
            placeholder="留空表示初始无密码 (安装后可随时在面板中设置)"
            value={adminPassword}
            onChange={(e) => setAdminPassword(e.target.value)}
          />
        </div>
      </Modal>

      {/* Modal: Stop Service Confirmation */}
      <Modal
        title="危险操作：停止 ClickHouse 服务确认"
        open={stopModalOpen}
        onCancel={() => {
          setStopModalOpen(false)
          setStopConfirmInput('')
        }}
        onOk={handleStop}
        okButtonProps={{ danger: true, disabled: stopConfirmInput !== 'STOP' }}
        okText="强制停止"
      >
        <p style={{ color: '#ff4d4f' }}>
          停止 ClickHouse 服务将导致所有客户端连接中断并拒绝所有新查询！
        </p>
        <p>请输入 <strong>STOP</strong> 确认停止操作：</p>
        <Input
          placeholder="输入 STOP"
          value={stopConfirmInput}
          onChange={(e) => setStopConfirmInput(e.target.value)}
        />
      </Modal>

      {/* Modal: Uninstall ClickHouse Confirmation */}
      <Modal
        title="高危操作：卸载 ClickHouse 确认"
        open={uninstallModalOpen}
        onCancel={() => {
          setUninstallModalOpen(false)
          setUninstallConfirmInput('')
        }}
        onOk={handleUninstall}
        okButtonProps={{ danger: true, disabled: uninstallConfirmInput !== 'UNINSTALL' }}
        okText="确认彻底卸载"
      >
        <Alert
          type="error"
          message="卸载警告"
          description="卸载将停止 ClickHouse 服务并删除 clickhouse-server 核心软件包！"
          style={{ marginBottom: 16 }}
        />
        <div style={{ marginBottom: 12 }}>
          <Checkbox checked={purgeData} onChange={(e) => setPurgeData(e.target.checked)}>
            同时彻底清除数据目录 (/var/lib/clickhouse) 与日志文件 (不可恢复)
          </Checkbox>
        </div>
        <p>请输入 <strong>UNINSTALL</strong> 确认卸载：</p>
        <Input
          placeholder="输入 UNINSTALL"
          value={uninstallConfirmInput}
          onChange={(e) => setUninstallConfirmInput(e.target.value)}
        />
      </Modal>
    </div>
  )
}
