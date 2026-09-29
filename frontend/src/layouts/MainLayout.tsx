import React, { useState, useEffect } from 'react'
import {
  Layout,
  Menu,
  theme,
  Badge,
  Alert,
  Spin,
  Button,
  Dropdown,
  Modal,
  Form,
  Input,
  message,
  Tooltip,
} from 'antd'
import type { MenuProps } from 'antd'
import { Button as HeroButton } from '@heroui/react'
import {
  DashboardOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  SearchOutlined,
  CodeOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  LineChartOutlined,
  AppstoreOutlined,
  SyncOutlined,
  ToolOutlined,
  HddOutlined,
  ClusterOutlined,
  FileTextOutlined,
  SaveOutlined,
  UserOutlined,
  SettingOutlined,
  EditOutlined,
  SecurityScanOutlined,
  SunOutlined,
  MoonOutlined,
  LogoutOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { api } from '../api/client'

const { Header, Sider, Content } = Layout

interface MainLayoutProps {
  currentKey: string
  onSelectKey: (key: string) => void
  isDark: boolean
  onToggleTheme: () => void
  children: React.ReactNode
}

export const MainLayout: React.FC<MainLayoutProps> = ({
  currentKey,
  onSelectKey,
  isDark,
  onToggleTheme,
  children,
}) => {
  const [collapsed, setCollapsed] = useState(false)
  const [compact, setCompact] = useState(false)
  const [userReady, setUserReady] = useState(false)
  const [userError, setUserError] = useState(false)
  const [username, setUsername] = useState(localStorage.getItem('username') || 'admin')
  const [mustChangePwd, setMustChangePwd] = useState(false)
  const [pwdModalOpen, setPwdModalOpen] = useState(false)
  const [form] = Form.useForm()

  const {
    token: { colorBgContainer },
  } = theme.useToken()

  useEffect(() => {
    api.getMe().then((res) => {
      if (res.data.data) {
        setUserReady(true)
        setUsername(res.data.data.username)
        if (res.data.data.must_change_password) {
          setMustChangePwd(true)
          setPwdModalOpen(true)
        }
      }
    }).catch(() => setUserError(true))
  }, [])

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    window.location.href = '/login'
  }

  const handleChangePassword = async (values: any) => {
    try {
      await api.changePassword(values)
      message.success('密码修改成功，请妥善保存')
      setMustChangePwd(false)
      setPwdModalOpen(false)
      form.resetFields()
    } catch {
      // handled by api interceptor
    }
  }

  const menuItems: MenuProps['items'] = [
    { key: 'dashboard', icon: <DashboardOutlined />, label: '首页' },
    { key: 'databases', icon: <DatabaseOutlined />, label: '数据库与表' },
    { key: 'data-browser', icon: <SearchOutlined />, label: '查看数据' },
    { key: 'sql-console', icon: <CodeOutlined />, label: 'SQL 查询' },
    { key: 'settings', icon: <SettingOutlined />, label: '连接与设置' },
    { type: 'divider' },
    {
      key: 'advanced', icon: <ToolOutlined />, label: '高级功能',
      children: [
        { key: 'monitoring', icon: <LineChartOutlined />, label: '实时监控' },
        { key: 'processes', icon: <ThunderboltOutlined />, label: '运行中的查询' },
        { key: 'slow-query', icon: <ClockCircleOutlined />, label: '慢查询' },
        { key: 'clickhouse-ops', icon: <CloudServerOutlined />, label: '本机服务管理' },
        { key: 'backup', icon: <SaveOutlined />, label: '备份与恢复' },
        { key: 'users', icon: <UserOutlined />, label: '用户与权限' },
        { key: 'logs', icon: <FileTextOutlined />, label: '服务日志' },
        { key: 'parts', icon: <AppstoreOutlined />, label: '数据分片 · Parts' },
        { key: 'merges', icon: <SyncOutlined />, label: '合并任务 · Merge' },
        { key: 'mutations', icon: <ToolOutlined />, label: '数据变更 · Mutation' },
        { key: 'disks', icon: <HddOutlined />, label: '磁盘与存储' },
        { key: 'clusters', icon: <ClusterOutlined />, label: '集群与字典' },
        { key: 'config-center', icon: <SettingOutlined />, label: '服务器参数' },
        { key: 'xml-editor', icon: <EditOutlined />, label: 'XML 配置文件' },
        { key: 'audit-log', icon: <SecurityScanOutlined />, label: '操作记录' },
      ],
    },
  ]

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'pwd',
      icon: <KeyOutlined />,
      label: '修改密码',
      onClick: () => setPwdModalOpen(true),
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
      onClick: handleLogout,
    },
  ]

  return (
    <Layout className="panel-layout" style={{ minHeight: '100vh' }}>
      <Sider
        theme={isDark ? 'dark' : 'light'}
        breakpoint="lg"
        collapsedWidth={compact ? 0 : 80}
        onBreakpoint={(broken) => { setCompact(broken); setCollapsed(broken) }}
        collapsible
        collapsed={collapsed}
        onCollapse={(value) => setCollapsed(value)}
        width={240}
        style={{
          borderRight: isDark ? '1px solid #393932' : '1px solid #e3e1d8',
          overflowY: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            padding: '0 16px',
            gap: 12,
            borderBottom: isDark ? '1px solid #393932' : '1px solid #e3e1d8',
          }}
        >
          <div
            style={{
              width: 34,
              height: 34,
              borderRadius: 8,
              background: '#e4bd55',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontWeight: 800,
              color: '#25251f',
              fontSize: 15,
              flexShrink: 0,
              letterSpacing: -1,
            }}
          >
            CH
          </div>
          {!collapsed && (
            <div style={{ overflow: 'hidden', whiteSpace: 'nowrap' }}>
              <div style={{ fontWeight: 700, fontSize: 15, letterSpacing: -0.2 }}>
                ClickHouse
              </div>
              <div style={{ fontSize: 11, opacity: 0.6 }}>Manager Panel</div>
            </div>
          )}
        </div>

        <Menu
          mode="inline"
          selectedKeys={[currentKey]}
          items={menuItems}
          onClick={({ key }) => { onSelectKey(key); if (compact) setCollapsed(true) }}
          style={{ borderRight: 0 }}
        />
      </Sider>

      {compact && !collapsed && <button className="nav-backdrop" aria-label="关闭导航" onClick={() => setCollapsed(true)} />}
      <Layout style={{ marginLeft: compact ? 0 : (collapsed ? 80 : 240), minWidth: 0, transition: 'margin-left 0.2s ease' }}>
        <Header
          className="panel-header"
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            borderBottom: isDark ? '1px solid #393932' : '1px solid #e3e1d8',
            boxShadow: isDark ? 'none' : '0 1px 2px 0 rgba(0, 0, 0, 0.03)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            position: 'sticky',
            top: 0,
            zIndex: 99,
            height: 64,
          }}
        >
          {/* Left: Status badges & indicators */}
          <div className="panel-status">
            {compact && <HeroButton variant="tertiary" isIconOnly aria-label={collapsed ? '打开导航' : '收起导航'} onPress={() => setCollapsed(!collapsed)}><AppstoreOutlined /></HeroButton>}
            <span style={{ fontWeight: 500 }}>数据库工作空间</span>
          </div>

          {/* Right: Theme switcher & user info */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Tooltip title={isDark ? '切换浅色模式' : '切换深色模式'}>
              <HeroButton variant="tertiary" isIconOnly aria-label={isDark ? '切换浅色模式' : '切换深色模式'} onPress={onToggleTheme}>
                {isDark ? <SunOutlined /> : <MoonOutlined />}
              </HeroButton>
            </Tooltip>

            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Button type="text" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Badge dot={mustChangePwd} status="warning">
                  <UserOutlined style={{ fontSize: 16 }} />
                </Badge>
                <span>{username}</span>
              </Button>
            </Dropdown>
          </div>
        </Header>

        <Content className="panel-content" style={{ minHeight: 'calc(100vh - 104px)' }}>
          {userError ? <Alert type="error" showIcon title="无法加载账号信息" action={<Button onClick={() => window.location.reload()}>重试</Button>} /> : !userReady ? <Spin /> : !mustChangePwd && children}
        </Content>
      </Layout>

      {/* Must change password modal on first login or manual change */}
      <Modal
        title={mustChangePwd ? '首次登录：请立即修改初始密码' : '修改管理员密码'}
        open={pwdModalOpen}
        closable={!mustChangePwd}
        mask={{ closable: !mustChangePwd }}
        onCancel={() => !mustChangePwd && setPwdModalOpen(false)}
        footer={null}
      >
        <p style={{ fontSize: 13, color: '#888', marginBottom: 16 }}>
          {mustChangePwd
            ? '为确保服务器 ClickHouse 数据库及管理面板安全，首次登录必须修改默认密码。'
            : '请输入当前密码以验证身份，并设置新的安全密码（长度不少于 8 位）。'}
        </p>
        <Form form={form} layout="vertical" onFinish={handleChangePassword}>
          <Form.Item
            name="old_password"
            label="当前原密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password placeholder="输入原密码" />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 8, message: '密码长度至少 8 位' },
            ]}
          >
            <Input.Password placeholder="设置新密码 (最少8位)" />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label="确认新密码"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的新密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            {!mustChangePwd && (
              <Button style={{ marginRight: 8 }} onClick={() => setPwdModalOpen(false)}>
                取消
              </Button>
            )}
            <Button type="primary" htmlType="submit">
              确认修改
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}
