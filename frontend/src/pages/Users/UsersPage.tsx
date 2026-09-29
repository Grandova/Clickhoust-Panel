import React, { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, Select, Space, Tag, Alert, Radio, Switch, Popconfirm, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  UserOutlined,
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
  KeyOutlined,
  SafetyCertificateOutlined,
  CrownOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { ClickHouseUser, DatabaseInfo } from '../../types'

const privilegeOptions = [
  { label: '全部权限 (ALL)', value: 'ALL' },
  { label: '读取数据 (SELECT)', value: 'SELECT' },
  { label: '写入数据 (INSERT)', value: 'INSERT' },
  { label: '建库与建表 (CREATE)', value: 'CREATE' },
  { label: '修改表与数据 (ALTER)', value: 'ALTER' },
  { label: '删除库与表 (DROP)', value: 'DROP' },
  { label: '清空表 (TRUNCATE)', value: 'TRUNCATE' },
  { label: '优化与合并 (OPTIMIZE)', value: 'OPTIMIZE' },
  { label: '查看对象 (SHOW)', value: 'SHOW' },
  { label: '系统管理指令 (SYSTEM)', value: 'SYSTEM' },
  { label: '账号权限管理 (ACCESS MANAGEMENT)', value: 'ACCESS MANAGEMENT' },
]

export const UsersPage: React.FC = () => {
  const [users, setUsers] = useState<ClickHouseUser[]>([])
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [connectionUser, setConnectionUser] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [createForm] = Form.useForm()
  const [alterModalOpen, setAlterModalOpen] = useState(false)
  const [alterForm] = Form.useForm()
  const [userToAlter, setUserToAlter] = useState<ClickHouseUser | null>(null)
  const [grantModalOpen, setGrantModalOpen] = useState(false)
  const [grantForm] = Form.useForm()
  const [userToGrant, setUserToGrant] = useState('')
  const [dropModalOpen, setDropModalOpen] = useState(false)
  const [userToDrop, setUserToDrop] = useState('')
  const [dropConfirmInput, setDropConfirmInput] = useState('')

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await api.listUsers()
      setUsers(res.data.data || [])
      const dbRes = await api.listDatabases()
      setDatabases(dbRes.data.data || [])
    } catch {
      // The API interceptor displays the error.
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUsers()
    api.getConnectionSettings().then((res) => setConnectionUser(res.data.data?.user || '')).catch(() => {})
  }, [])

  const handleCreateUser = async (values: any) => {
    setSaving(true)
    try {
      await api.createUser({
        username: values.username,
        password: values.password,
        allowed_hosts: values.allowed_hosts ? values.allowed_hosts.split(',').map((s: string) => s.trim()).filter(Boolean) : [],
        default_database: values.default_database || 'default',
        all_privileges: values.privilege_type === 'admin',
      })
      setCreateModalOpen(false)
      createForm.resetFields()

      try {
        if (values.privilege_type === 'admin') {
          await api.grantPrivileges({
            username: values.username,
            privileges: ['ALL'],
            database: '*',
            table: '*',
            with_grant_option: true,
          })
          message.success(`用户 ${values.username} 已成功创建并授予最高管理员全部权限 (ALL ON *.*)`)
        } else {
          await api.grantPrivileges({
            username: values.username,
            privileges: values.privileges && values.privileges.length > 0 ? values.privileges : ['SELECT', 'INSERT', 'CREATE', 'SHOW'],
            database: values.default_database || '*',
            table: '*',
            with_grant_option: !!values.with_grant_option,
          })
          message.success(`用户 ${values.username} 已成功创建并授权`)
        }
      } catch {
        message.warning('用户已创建，但授权未完全成功。已打开权限管理窗口供重试。')
        setUserToGrant(values.username)
        grantForm.resetFields()
        grantForm.setFieldsValue({
          database: values.default_database || '*',
          privileges: values.privilege_type === 'admin' ? ['ALL'] : values.privileges,
          with_grant_option: values.privilege_type === 'admin' || !!values.with_grant_option,
          action: 'grant',
        })
        setGrantModalOpen(true)
      }
      fetchUsers()
    } catch {
      // The API interceptor displays the error.
    } finally {
      setSaving(false)
    }
  }

  const handleAlterUser = async (values: any) => {
    if (!userToAlter) return
    setSaving(true)
    try {
      const res = await api.alterUser({ username: userToAlter.name, new_password: values.new_password })
      message.success(res.data?.message || `用户 ${userToAlter.name} 密码已修改`)
      setAlterModalOpen(false)
      alterForm.resetFields()
      setUserToAlter(null)
      fetchUsers()
    } catch {
      // The API interceptor displays the error.
    } finally {
      setSaving(false)
    }
  }

  const handleQuickGrantAll = async (username: string) => {
    setSaving(true)
    try {
      await api.grantPrivileges({
        username,
        privileges: ['ALL'],
        database: '*',
        table: '*',
        with_grant_option: true,
      })
      message.success(`已为用户 ${username} 授予全部权限 (ALL ON *.* WITH GRANT OPTION)`)
      fetchUsers()
    } catch {
      // The API interceptor displays the error.
    } finally {
      setSaving(false)
    }
  }

  const handleGrantPrivileges = async (values: any) => {
    if (!userToGrant) return
    setSaving(true)
    try {
      await (values.action === 'revoke' ? api.revokePrivileges : api.grantPrivileges)({
        username: userToGrant,
        privileges: values.privileges,
        database: values.database,
        table: values.table || '*',
        with_grant_option: values.action === 'grant' && !!values.with_grant_option,
      })
      message.success(values.action === 'revoke' ? '所选权限已撤销' : '权限已成功授予')
      setGrantModalOpen(false)
      grantForm.resetFields()
      setUserToGrant('')
      fetchUsers()
    } catch {
      // The API interceptor displays the error.
    } finally {
      setSaving(false)
    }
  }

  const handleDropUser = async () => {
    if (dropConfirmInput !== `DROP ${userToDrop}`) return
    setSaving(true)
    try {
      await api.dropUser(userToDrop, dropConfirmInput)
      message.success(`用户 ${userToDrop} 已删除`)
      setDropModalOpen(false)
      setDropConfirmInput('')
      fetchUsers()
    } catch {
      // The API interceptor displays the error.
    } finally {
      setSaving(false)
    }
  }

  const columns: ColumnsType<ClickHouseUser> = [
    {
      title: '用户名', dataIndex: 'name', key: 'name',
      render: (name: string, record: ClickHouseUser) => (
        <Space wrap>
          <UserOutlined />
          <strong>{name}</strong>
          {name === connectionUser && <Tag color="green">当前连接</Tag>}
          {record.storage === 'users_xml' || name === 'default' ? (
            <Tag color="blue">XML 配置</Tag>
          ) : (
            <Tag color="cyan">SQL 管理</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '允许连接的 IP', dataIndex: 'host_ip', key: 'host_ip',
      render: (ips: string[]) => ips?.length ? ips.map((ip) => <Tag key={ip}>{ip === '::/0' || ip === '0.0.0.0/0' ? '任意 IP' : ip}</Tag>) : '未配置 IP 规则',
    },
    {
      title: '默认数据库', dataIndex: 'default_database', key: 'default_database',
      render: (db: string) => db || 'default',
    },
    {
      title: '已有授权', dataIndex: 'grants', key: 'grants',
      render: (grants: string[]) => grants?.length ? (
        <div style={{ maxWidth: 540, overflowWrap: 'anywhere' }}>
          {grants.map((grant) => {
            const isAllGlobal = grant.includes('ALL ON *.*')
            const hasGrantOption = grant.includes('WITH GRANT OPTION')
            if (isAllGlobal) {
              return (
                <div key={grant} style={{ marginBottom: 4 }}>
                  <Tag color="gold" style={{ fontWeight: 600 }}>全部权限 (超级管理员)</Tag>
                  {hasGrantOption && <Tag color="purple">包含转授权</Tag>}
                </div>
              )
            }
            const match = grant.match(/^GRANT (.+) ON (.+) TO ([^ ]+)(.*)$/)
            return (
              <div key={grant} style={{ marginBottom: 4 }}>
                {match ? (
                  <>
                    <Tag>{match[2] === '*.*' ? '所有数据库' : match[2]}</Tag>
                    {match[1].split(', ').map((p) => privilegeOptions.find((o) => o.value === p)?.label || p).join('、')}
                    {hasGrantOption && <Tag color="purple">转授权</Tag>}
                  </>
                ) : (
                  grant
                )}
              </div>
            )
          })}
          <details><summary style={{ cursor: 'pointer', color: '#888' }}>完整授权 SQL</summary>{grants.map((grant) => <div key={grant}><code>{grant}</code></div>)}</details>
        </div>
      ) : '暂无授权',
    },
    {
      title: '操作', key: 'actions',
      render: (_, record) => (
        <Space size="small">
          <Button size="small" icon={<KeyOutlined />} onClick={() => {
            setUserToAlter(record)
            alterForm.resetFields()
            setAlterModalOpen(true)
          }}>改密码</Button>
          <Button size="small" icon={<SafetyCertificateOutlined />} onClick={() => {
            setUserToGrant(record.name)
            grantForm.resetFields()
            grantForm.setFieldsValue({
              database: record.default_database || '*',
              privileges: ['ALL'],
              with_grant_option: true,
              action: 'grant',
            })
            setGrantModalOpen(true)
          }}>权限</Button>
          <Popconfirm
            title="一键授予全部权限"
            description={`确定为用户【${record.name}】授予全局全部权限 (GRANT ALL ON *.* WITH GRANT OPTION) 吗？`}
            onConfirm={() => handleQuickGrantAll(record.name)}
            okText="确定授权"
            cancelText="取消"
          >
            <Button size="small" icon={<CrownOutlined />} style={{ color: '#d48806', borderColor: '#ffe58f' }}>赋全权</Button>
          </Popconfirm>
          {record.name !== 'default' && (
            <Button size="small" danger type="text" aria-label={`删除 ${record.name}`} icon={<DeleteOutlined />} onClick={() => {
              setUserToDrop(record.name)
              setDropConfirmInput('')
              setDropModalOpen(true)
            }} />
          )}
        </Space>
      ),
    },
  ]

  return (
    <Card title={
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
        <Space><UserOutlined /><span>数据库用户</span><Tag>{users.length} 个账号</Tag></Space>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>新建用户</Button>
          <Button icon={<ReloadOutlined />} onClick={fetchUsers}>刷新</Button>
        </Space>
      </div>
    }>
      <p style={{ color: '#777', marginTop: 0 }}>当前连接账号：{connectionUser || '—'}。支持在线修改 default 等配置文件的用户密码及分配超级管理员全部权限。</p>
      <Table dataSource={users} columns={columns} rowKey="name" loading={loading} pagination={{ pageSize: 20 }} size="middle" scroll={{ x: 900 }} />

      <Modal title="新建数据库用户" open={createModalOpen} confirmLoading={saving} okText="创建并授权" onCancel={() => setCreateModalOpen(false)} onOk={() => createForm.submit()} width={560}>
        <Form form={createForm} layout="vertical" onFinish={handleCreateUser} initialValues={{ privilege_type: 'admin', default_database: 'default' }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}><Input placeholder="例如 analytics_admin" /></Form.Item>
          <Form.Item name="password" label="连接密码"><Input.Password autoComplete="new-password" placeholder="留空为无密码" /></Form.Item>
          <Form.Item name="privilege_type" label="权限分配">
            <Radio.Group>
              <Radio.Button value="admin"><CrownOutlined /> 超级管理员 (全部权限 ALL)</Radio.Button>
              <Radio.Button value="custom">普通用户 (自定义权限)</Radio.Button>
            </Radio.Group>
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.privilege_type !== cur.privilege_type}>
            {({ getFieldValue }) => getFieldValue('privilege_type') === 'admin' ? (
              <Alert
                type="info"
                showIcon
                title="超级管理员权限说明"
                description="将自动授予最高权限 GRANT ALL ON *.* WITH GRANT OPTION，允许访问全部数据库、创建/删除库表、管理服务与所有账号权限。"
                style={{ marginBottom: 16 }}
              />
            ) : (
              <>
                <Form.Item name="default_database" label="可访问数据库" rules={[{ required: true, message: '请选择数据库' }]}>
                  <Select options={[{ label: '所有数据库 (*.*)', value: '*' }, ...databases.map((d) => ({ label: d.name, value: d.name }))]} />
                </Form.Item>
                <Form.Item name="privileges" label="数据库权限" initialValue={['SELECT', 'INSERT', 'CREATE', 'SHOW']} rules={[{ required: true, message: '请选择权限' }]}>
                  <Select mode="multiple" options={privilegeOptions} />
                </Form.Item>
                <Form.Item name="with_grant_option" valuePropName="checked" label="转授权选项">
                  <Switch checkedChildren="允许转授权" unCheckedChildren="禁止转授权" />
                </Form.Item>
              </>
            )}
          </Form.Item>
          <details><summary style={{ cursor: 'pointer', marginBottom: 12 }}>连接 IP 限制</summary>
            <Form.Item name="allowed_hosts" label="允许连接的 IP（逗号分隔）"><Input placeholder="留空允许任意 IP，例如 127.0.0.1, 192.168.1.0/24" /></Form.Item>
          </details>
        </Form>
      </Modal>

      <Modal title={`修改密码 · ${userToAlter?.name}`} open={alterModalOpen} confirmLoading={saving} okText="保存密码" onCancel={() => setAlterModalOpen(false)} onOk={() => alterForm.submit()}>
        {(userToAlter?.storage === 'users_xml' || userToAlter?.name === 'default') && (
          <Alert
            type="info"
            showIcon
            title="配置文件用户在线更新"
            description="该用户存储于 ClickHouse 配置文件 (users_xml)，修改密码后系统将自动生成 users.d 覆盖配置并热重载生效，无需手动编辑 XML。"
            style={{ marginBottom: 16 }}
          />
        )}
        {userToAlter?.name === connectionUser && (
          <Alert
            type="success"
            showIcon
            title="当前连接账号自动同步"
            description="修改后，面板将自动同步更新当前连接密码，确保后台监控与数据交互不受任何中断。"
            style={{ marginBottom: 16 }}
          />
        )}
        <Form form={alterForm} layout="vertical" onFinish={handleAlterUser}>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }]}>
            <Input.Password autoComplete="new-password" placeholder="输入新的数据库连接密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`数据库权限 · ${userToGrant}`} open={grantModalOpen} confirmLoading={saving} okText="执行" onCancel={() => setGrantModalOpen(false)} onOk={() => grantForm.submit()} width={600}>
        <Alert type="info" title="快捷权限配置" description="点击下方按钮可一键填充常用权限方案，也可在下方自定义组合。" style={{ marginBottom: 16 }} />
        <Form form={grantForm} layout="vertical" onFinish={handleGrantPrivileges} initialValues={{ action: 'grant', database: '*', privileges: ['ALL'], with_grant_option: true }}>
          <Form.Item name="action" label="操作"><Radio.Group options={[{ label: '添加权限', value: 'grant' }, { label: '撤销权限', value: 'revoke' }]} /></Form.Item>
          <Space wrap style={{ marginBottom: 16 }}>
            <Button type="dashed" icon={<CrownOutlined />} onClick={() => {
              grantForm.setFieldsValue({ database: '*', privileges: ['ALL'], with_grant_option: true, table: '' })
            }}>授予全部权限 (超级管理员)</Button>
            <Button onClick={() => grantForm.setFieldsValue({ privileges: ['SELECT', 'INSERT', 'CREATE', 'ALTER', 'DROP', 'TRUNCATE', 'OPTIMIZE', 'SHOW'] })}>建表与维护</Button>
            <Button onClick={() => grantForm.setFieldsValue({ privileges: ['SELECT', 'INSERT', 'SHOW'] })}>读写数据</Button>
            <Button onClick={() => grantForm.setFieldsValue({ privileges: ['SELECT', 'SHOW'] })}>只读查询</Button>
          </Space>
          <Form.Item name="database" label="授权数据库范围" rules={[{ required: true, message: '请选择数据库范围' }]}>
            <Select options={[{ label: '所有数据库 (*.*)', value: '*' }, ...databases.map((d) => ({ label: d.name, value: d.name }))]} />
          </Form.Item>
          <Form.Item name="privileges" label="权限" rules={[{ required: true, message: '请选择权限' }]}><Select mode="multiple" options={privilegeOptions} /></Form.Item>
          <Form.Item name="with_grant_option" valuePropName="checked" label="授权选项 (WITH GRANT OPTION)">
            <Switch checkedChildren="开启转授权" unCheckedChildren="关闭" />
          </Form.Item>
          <details><summary style={{ cursor: 'pointer', marginBottom: 12 }}>仅针对某张表（当数据库非所有数据库时生效）</summary>
            <Form.Item name="table" label="表名"><Input placeholder="留空表示该数据库的所有表" /></Form.Item>
          </details>
        </Form>
      </Modal>

      <Modal title={`删除用户 · ${userToDrop}`} open={dropModalOpen} confirmLoading={saving} onCancel={() => { setDropModalOpen(false); setDropConfirmInput('') }} onOk={handleDropUser} okButtonProps={{ danger: true, disabled: dropConfirmInput !== `DROP ${userToDrop}` }} okText="确认删除">
        <p>删除后该用户将无法连接数据库。请输入 <strong>DROP {userToDrop}</strong> 确认：</p>
        <Input value={dropConfirmInput} onChange={(e) => setDropConfirmInput(e.target.value)} />
      </Modal>
    </Card>
  )
}
