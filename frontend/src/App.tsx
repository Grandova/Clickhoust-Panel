import React, { useState, useEffect, lazy, Suspense } from 'react'
import { ConfigProvider, Spin, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { MainLayout } from './layouts/MainLayout'
import { LoginPage } from './pages/Login/LoginPage'
const DashboardPage = lazy(() => import('./pages/Dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const ClickHouseOpsPage = lazy(() => import('./pages/ClickHouse/ClickHouseOpsPage').then((m) => ({ default: m.ClickHouseOpsPage })))
const DatabasesPage = lazy(() => import('./pages/Databases/DatabasesPage').then((m) => ({ default: m.DatabasesPage })))
const DataBrowserPage = lazy(() => import('./pages/DataBrowser/DataBrowserPage').then((m) => ({ default: m.DataBrowserPage })))
const SQLConsolePage = lazy(() => import('./pages/SQLConsole/SQLConsolePage').then((m) => ({ default: m.SQLConsolePage })))
const ProcessesPage = lazy(() => import('./pages/Processes/ProcessesPage').then((m) => ({ default: m.ProcessesPage })))
const SlowQueryPage = lazy(() => import('./pages/SlowQuery/SlowQueryPage').then((m) => ({ default: m.SlowQueryPage })))
const MonitoringPage = lazy(() => import('./pages/Monitoring/MonitoringPage').then((m) => ({ default: m.MonitoringPage })))
const PartsPage = lazy(() => import('./pages/Parts/PartsPage').then((m) => ({ default: m.PartsPage })))
const MergesPage = lazy(() => import('./pages/Merges/MergesPage').then((m) => ({ default: m.MergesPage })))
const MutationsPage = lazy(() => import('./pages/Mutations/MutationsPage').then((m) => ({ default: m.MutationsPage })))
const DisksPage = lazy(() => import('./pages/Disks/DisksPage').then((m) => ({ default: m.DisksPage })))
const ClustersPage = lazy(() => import('./pages/Clusters/ClustersPage').then((m) => ({ default: m.ClustersPage })))
const LogsPage = lazy(() => import('./pages/Logs/LogsPage').then((m) => ({ default: m.LogsPage })))
const BackupPage = lazy(() => import('./pages/Backup/BackupPage').then((m) => ({ default: m.BackupPage })))
const UsersPage = lazy(() => import('./pages/Users/UsersPage').then((m) => ({ default: m.UsersPage })))
const ConfigCenterPage = lazy(() => import('./pages/ConfigCenter/ConfigCenterPage').then((m) => ({ default: m.ConfigCenterPage })))
const AdvancedXMLEditorPage = lazy(() => import('./pages/ConfigCenter/AdvancedXMLEditorPage').then((m) => ({ default: m.AdvancedXMLEditorPage })))
const AuditLogPage = lazy(() => import('./pages/AuditLog/AuditLogPage').then((m) => ({ default: m.AuditLogPage })))
const SettingsPage = lazy(() => import('./pages/Settings/SettingsPage').then((m) => ({ default: m.SettingsPage })))

export const App: React.FC = () => {
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'))
  const [currentKey, setCurrentKey] = useState<string>('dashboard')
  const [isDark, setIsDark] = useState<boolean>(() => {
    const saved = localStorage.getItem('theme_mode')
    return saved === 'dark'
  })

  // Cross-page navigation state
  const [browserParams, setBrowserParams] = useState<{ db: string; table: string }>({ db: 'default', table: '' })
  const [consoleInitialSql, setConsoleInitialSql] = useState<string>('')

  useEffect(() => {
    const handleStorage = () => {
      setToken(localStorage.getItem('token'))
    }
    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [])

  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark)
    document.documentElement.dataset.theme = isDark ? 'dark' : 'light'
  }, [isDark])

  const toggleTheme = () => {
    setIsDark((prev) => {
      const next = !prev
      localStorage.setItem('theme_mode', next ? 'dark' : 'light')
      return next
    })
  }

  const themeConfig = {
    algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: isDark ? '#e4bd55' : '#876817',
      colorLink: isDark ? '#e4bd55' : '#876817',
      borderRadius: 8,
      controlHeight: 38,
      fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif',
      colorBgBase: isDark ? '#171715' : '#f5f4ef',
      colorBgContainer: isDark ? '#20201d' : '#ffffff',
      colorBorder: isDark ? '#393932' : '#e3e1d8',
      colorBorderSecondary: isDark ? '#20201d' : '#efeee7',
      colorText: isDark ? '#f5f4ef' : '#171715',
      colorTextSecondary: isDark ? '#a5a397' : '#79776b',
    },
    components: {
      Card: {
        headerBg: 'transparent',
        boxShadowTertiary: 'none',
      },
      Layout: {
        bodyBg: isDark ? '#171715' : '#f5f4ef',
        headerBg: isDark ? '#20201d' : '#ffffff',
        siderBg: isDark ? '#20201d' : '#ffffff',
      },
      Menu: {
        itemBg: 'transparent',
        itemSelectedBg: isDark ? '#d6b4491f' : '#eee9d5',
        itemSelectedColor: isDark ? '#e4bd55' : '#6f5510',
        itemColor: isDark ? '#a5a397' : '#646258',
        itemHoverBg: isDark ? '#ffffff08' : '#efeee7',
        itemHoverColor: isDark ? '#f5f4ef' : '#171715',
      },
      Table: {
        headerBg: isDark ? '#20201d' : '#f5f4ef',
        headerColor: isDark ? '#a5a397' : '#646258',
        rowHoverBg: isDark ? '#ffffff06' : '#f5f4ef',
      },
      Button: {
        borderRadius: 6,
      },
    },
  }

  if (!token) {
    return (
      <ConfigProvider locale={zhCN} theme={themeConfig}>
        <LoginPage
          isDark={isDark}
          onLoginSuccess={() => {
            setToken(localStorage.getItem('token'))
          }}
        />
      </ConfigProvider>
    )
  }

  const renderContent = () => {
    switch (currentKey) {
      case 'dashboard':
        return <DashboardPage onNavigate={(key) => setCurrentKey(key)} />
      case 'clickhouse-ops':
        return <ClickHouseOpsPage isDark={isDark} />
      case 'databases':
        return (
          <DatabasesPage
            isDark={isDark}
            onNavigateToBrowser={(db, table) => {
              setBrowserParams({ db, table })
              setCurrentKey('data-browser')
            }}
            onNavigateToSQL={(sql) => {
              setConsoleInitialSql(sql)
              setCurrentKey('sql-console')
            }}
          />
        )
      case 'data-browser':
        return <DataBrowserPage initialDb={browserParams.db} initialTable={browserParams.table} />
      case 'sql-console':
        return <SQLConsolePage initialSql={consoleInitialSql} isDark={isDark} />
      case 'processes':
        return <ProcessesPage isDark={isDark} />
      case 'slow-query':
        return <SlowQueryPage isDark={isDark} />
      case 'monitoring':
        return <MonitoringPage isDark={isDark} />
      case 'parts':
        return <PartsPage />
      case 'merges':
        return <MergesPage />
      case 'mutations':
        return <MutationsPage />
      case 'disks':
        return <DisksPage />
      case 'clusters':
        return <ClustersPage />
      case 'logs':
        return <LogsPage />
      case 'backup':
        return <BackupPage />
      case 'users':
        return <UsersPage />
      case 'config-center':
        return <ConfigCenterPage />
      case 'xml-editor':
        return <AdvancedXMLEditorPage isDark={isDark} />
      case 'audit-log':
        return <AuditLogPage />
      case 'settings':
        return <SettingsPage />
      default:
        return <DashboardPage onNavigate={(key) => setCurrentKey(key)} />
    }
  }

  return (
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <MainLayout
        currentKey={currentKey}
        onSelectKey={(k) => setCurrentKey(k)}
        isDark={isDark}
        onToggleTheme={toggleTheme}
      >
        <Suspense fallback={<div style={{ padding: 64, textAlign: 'center' }}><Spin /></div>}>
          {renderContent()}
        </Suspense>
      </MainLayout>
    </ConfigProvider>
  )
}

export default App
