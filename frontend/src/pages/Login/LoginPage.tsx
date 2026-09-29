import React, { useState } from 'react'
import { Button, Card, Input, Label, Spinner, TextField } from '@heroui/react'
import { ArrowRightOutlined, DatabaseOutlined } from '@ant-design/icons'
import { api } from '../../api/client'

interface LoginPageProps {
  onLoginSuccess: () => void
  isDark?: boolean
}

export const LoginPage: React.FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const values = new FormData(event.currentTarget)
    setLoading(true)
    try {
      const res = await api.login({ username: String(values.get('username')), password: String(values.get('password')) })
      if (res.data.data) {
        localStorage.setItem('token', res.data.data.token)
        localStorage.setItem('username', res.data.data.username)
        onLoginSuccess()
      }
    } catch {
      // Errors are displayed by the API interceptor.
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="login-page">
      <div className="login-brand"><span className="brand-mark"><DatabaseOutlined /></span> ClickHouse <span>Manager</span></div>
      <section className="login-form-area">
        <Card className="login-card">
          <Card.Header>
            <span className="login-eyebrow">DATABASE CONSOLE / 01</span>
            <Card.Title className="login-title">登录控制台</Card.Title>
            <Card.Description>查询数据，管理实例，查看运行状态。</Card.Description>
          </Card.Header>
          <Card.Content>
            <form onSubmit={handleSubmit} className="login-form">
              <TextField name="username" isRequired defaultValue="admin">
                <Label>管理员账号</Label>
                <Input autoComplete="username" placeholder="输入管理员账号" />
              </TextField>
              <TextField name="password" type="password" isRequired>
                <Label>登录密码</Label>
                <Input autoComplete="current-password" placeholder="输入登录密码" />
              </TextField>
              <Button type="submit" size="lg" fullWidth isPending={loading}>
                {loading ? <Spinner size="sm" /> : null}
                {loading ? '正在登录' : '登录'}
                {!loading && <ArrowRightOutlined />}
              </Button>
            </form>
          </Card.Content>
          <Card.Footer className="login-note">首次登录后，请按照提示修改初始密码。</Card.Footer>
        </Card>
        <span className="login-footer">CLICKHOUSE MANAGER  /  数据库管理控制台</span>
      </section>
    </main>
  )
}
