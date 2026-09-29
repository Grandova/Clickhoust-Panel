import React, { useState, useEffect } from 'react'
import {
  Card,
  Tabs,
  Table,
  Tag,
  Space,
  Button,
  Row,
  Col,
  Statistic,
  Input,
  Select,
  Alert,
  Tooltip,
  Badge,
  message,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ClusterOutlined,
  CopyOutlined,
  BookOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  SearchOutlined,
  CrownOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { api } from '../../api/client'
import type { ClusterNodeInfo, ReplicaInfo, DictionaryInfo } from '../../types'
import { formatBytes } from '../../utils/format'

const { Text } = Typography

export const ClustersPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState('clusters')
  const [loading, setLoading] = useState(false)

  // Data states
  const [clusters, setClusters] = useState<ClusterNodeInfo[]>([])
  const [replicas, setReplicas] = useState<ReplicaInfo[]>([])
  const [dictionaries, setDictionaries] = useState<DictionaryInfo[]>([])

  // Filter states
  const [selectedCluster, setSelectedCluster] = useState<string>('ALL')
  const [replicaSearch, setReplicaSearch] = useState<string>('')
  const [dictSearch, setDictSearch] = useState<string>('')

  // Action states
  const [reloadingDict, setReloadingDict] = useState<string | null>(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [cRes, rRes, dRes] = await Promise.allSettled([
        api.getClusters(),
        api.getReplicas(),
        api.getDictionaries(),
      ])

      if (cRes.status === 'fulfilled' && cRes.value.data.data) {
        setClusters(cRes.value.data.data)
      }
      if (rRes.status === 'fulfilled' && rRes.value.data.data) {
        setReplicas(rRes.value.data.data)
      }
      if (dRes.status === 'fulfilled' && dRes.value.data.data) {
        setDictionaries(dRes.value.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleReloadDictionary = async (dict: DictionaryInfo) => {
    const dictKey = `${dict.database}.${dict.name}`
    setReloadingDict(dictKey)
    try {
      const res = await api.reloadDictionary({
        database: dict.database,
        name: dict.name,
      })
      message.success(res.data.message || `字典 [${dict.name}] 重载成功`)
      fetchData()
    } catch {
      // Handled by api interceptor
    } finally {
      setReloadingDict(null)
    }
  }

  // Clusters summary
  const uniqueClusters = Array.from(new Set(clusters.map((c) => c.cluster)))
  const filteredClusters =
    selectedCluster === 'ALL'
      ? clusters
      : clusters.filter((c) => c.cluster === selectedCluster)
  const localNodesCount = clusters.filter((c) => c.is_local).length
  const remoteNodesCount = clusters.filter((c) => !c.is_local).length

  // Replicas summary
  const filteredReplicas = replicas.filter(
    (r) =>
      r.database.toLowerCase().includes(replicaSearch.toLowerCase()) ||
      r.table.toLowerCase().includes(replicaSearch.toLowerCase())
  )
  const leaderReplicasCount = replicas.filter((r) => r.is_leader).length
  const readonlyReplicasCount = replicas.filter((r) => r.is_readonly).length
  const totalQueueSize = replicas.reduce((sum, r) => sum + r.queue_size, 0)
  const maxDelaySeconds =
    replicas.length > 0 ? Math.max(...replicas.map((r) => r.absolute_delay)) : 0

  // Dictionaries summary
  const filteredDictionaries = dictionaries.filter(
    (d) =>
      d.name.toLowerCase().includes(dictSearch.toLowerCase()) ||
      d.database.toLowerCase().includes(dictSearch.toLowerCase()) ||
      d.source.toLowerCase().includes(dictSearch.toLowerCase())
  )
  const loadedDictsCount = dictionaries.filter((d) => d.status === 'LOADED').length
  const failedDictsCount = dictionaries.filter((d) => d.status === 'FAILED').length
  const totalDictBytes = dictionaries.reduce((sum, d) => sum + d.bytes_allocated, 0)
  const totalDictElements = dictionaries.reduce((sum, d) => sum + d.element_count, 0)

  // Columns: Clusters
  const clusterColumns: ColumnsType<ClusterNodeInfo> = [
    {
      title: '集群标识 (Cluster)',
      dataIndex: 'cluster',
      key: 'cluster',
      render: (c: string) => <Tag color="blue">{c}</Tag>,
    },
    {
      title: '分片编号 (Shard)',
      dataIndex: 'shard_num',
      key: 'shard_num',
      sorter: (a, b) => a.shard_num - b.shard_num,
      render: (s: number, record) => (
        <span>
          分片 <strong>#{s}</strong>{' '}
          <Text type="secondary" style={{ fontSize: 12 }}>
            (权重: {record.shard_weight})
          </Text>
        </span>
      ),
    },
    {
      title: '副本编号 (Replica)',
      dataIndex: 'replica_num',
      key: 'replica_num',
      sorter: (a, b) => a.replica_num - b.replica_num,
      render: (r: number) => <span>副本 #{r}</span>,
    },
    {
      title: '节点主机 / IP',
      dataIndex: 'host_address',
      key: 'host_address',
      render: (addr: string, record) => (
        <Space orientation="vertical" size={2}>
          <Space>
            <code style={{ fontSize: 13, fontWeight: 600 }}>{addr || record.host_name}</code>
            <Tooltip title="复制地址">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => {
                  navigator.clipboard.writeText(`${addr || record.host_name}:${record.port}`)
                  message.success('已复制主机地址与端口')
                }}
              />
            </Tooltip>
          </Space>
          {record.host_name && record.host_name !== addr && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              主机名: {record.host_name}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: 'TCP 端口',
      dataIndex: 'port',
      key: 'port',
      render: (p: number) => <Tag>{p}</Tag>,
    },
    {
      title: '节点属性',
      dataIndex: 'is_local',
      key: 'is_local',
      render: (isLocal: boolean) =>
        isLocal ? (
          <Badge status="success" text={<strong style={{ color: '#52c41a' }}>当前本机节点</strong>} />
        ) : (
          <Badge status="processing" text={<span style={{ color: '#987b2e' }}>远程集群节点</span>} />
        ),
    },
    {
      title: '认证用户',
      dataIndex: 'user',
      key: 'user',
      render: (u: string) => <span>{u || 'default'}</span>,
    },
    {
      title: '默认库',
      dataIndex: 'default_database',
      key: 'default_database',
      render: (db: string) => <code>{db || 'default'}</code>,
    },
  ]

  // Columns: Replicas
  const replicaColumns: ColumnsType<ReplicaInfo> = [
    {
      title: '数据库与表名',
      key: 'table',
      render: (_, r) => (
        <Space>
          <code>{r.database}</code>
          <span>.</span>
          <strong>{r.table}</strong>
        </Space>
      ),
    },
    {
      title: '副本角色',
      dataIndex: 'is_leader',
      key: 'is_leader',
      render: (isLeader: boolean) =>
        isLeader ? (
          <Tag color="gold" icon={<CrownOutlined />}>
            Leader (主节点)
          </Tag>
        ) : (
          <Tag color="default">Follower (从副本)</Tag>
        ),
    },
    {
      title: '读写状态',
      dataIndex: 'is_readonly',
      key: 'is_readonly',
      render: (isReadonly: boolean) =>
        isReadonly ? (
          <Tag color="error" icon={<CloseCircleOutlined />}>
            只读状态 (ZooKeeper 异常)
          </Tag>
        ) : (
          <Tag color="success" icon={<CheckCircleOutlined />}>
            正常读写
          </Tag>
        ),
    },
    {
      title: '同步延迟 (Delay)',
      dataIndex: 'absolute_delay',
      key: 'absolute_delay',
      sorter: (a, b) => a.absolute_delay - b.absolute_delay,
      render: (delay: number) => {
        if (delay === 0) return <Tag color="success">0 秒 (已实时同步)</Tag>
        if (delay < 10) return <Tag color="warning">{delay} 秒延迟</Tag>
        return <Tag color="error">{delay} 秒延迟 (较大)</Tag>
      },
    },
    {
      title: '队列积压 (Queue)',
      dataIndex: 'queue_size',
      key: 'queue_size',
      sorter: (a, b) => a.queue_size - b.queue_size,
      render: (size: number, record) => (
        <Tooltip
          title={`插入积压: ${record.inserts_in_queue} | 合并积压: ${record.merges_in_queue}`}
        >
          <Tag color={size > 10 ? 'red' : size > 0 ? 'orange' : 'default'}>
            {size} 任务
          </Tag>
        </Tooltip>
      ),
    },
    {
      title: '日志指针 (Log Pointer)',
      dataIndex: 'log_pointer',
      key: 'log_pointer',
      render: (p: number) => <code>#{p}</code>,
    },
    {
      title: '最近队列同步时间',
      dataIndex: 'last_queue_update',
      key: 'last_queue_update',
      render: (t: string) => (
        <span style={{ fontSize: 12, opacity: 0.85 }}>{t || '-'}</span>
      ),
    },
  ]

  // Columns: Dictionaries
  const dictColumns: ColumnsType<DictionaryInfo> = [
    {
      title: '字典标识 (Name)',
      key: 'name',
      render: (_, d) => (
        <Space orientation="vertical" size={2}>
          <Space>
            <BookOutlined style={{ color: '#faad14' }} />
            <strong>{d.name}</strong>
          </Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            所属库: <code>{d.database}</code>
          </Text>
        </Space>
      ),
    },
    {
      title: '加载状态 (Status)',
      dataIndex: 'status',
      key: 'status',
      render: (status: string, record) => {
        if (status === 'LOADED') {
          return (
            <Tag color="success" icon={<CheckCircleOutlined />}>
              LOADED (已加载)
            </Tag>
          )
        }
        if (status === 'FAILED') {
          return (
            <Tooltip title={record.last_exception || '加载发生异常'}>
              <Tag color="error" icon={<CloseCircleOutlined />}>
                FAILED (失败)
              </Tag>
            </Tooltip>
          )
        }
        if (status === 'LOADING') {
          return (
            <Tag color="processing" icon={<SyncOutlined spin />}>
              LOADING (正在加载)
            </Tag>
          )
        }
        return <Tag>{status}</Tag>
      },
    },
    {
      title: '数据源类型 (Source)',
      dataIndex: 'source',
      key: 'source',
      render: (src: string, record) => (
        <Space>
          <Tag color="cyan">{record.type || 'Hashed'}</Tag>
          <code style={{ fontSize: 12 }}>{src || '-'}</code>
        </Space>
      ),
    },
    {
      title: '缓存条目数',
      dataIndex: 'element_count',
      key: 'element_count',
      sorter: (a, b) => a.element_count - b.element_count,
      render: (cnt: number) => cnt.toLocaleString(),
    },
    {
      title: '内存占用',
      dataIndex: 'bytes_allocated',
      key: 'bytes_allocated',
      sorter: (a, b) => a.bytes_allocated - b.bytes_allocated,
      render: (bytes: number) => formatBytes(bytes),
    },
    {
      title: '加载耗时/时间',
      dataIndex: 'loading_start_time',
      key: 'loading_start_time',
      render: (t: string) => <span style={{ fontSize: 12 }}>{t || '-'}</span>,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => {
        const isReloading = reloadingDict === `${record.database}.${record.name}`
        return (
          <Button
            size="small"
            type="primary"
            ghost
            icon={<ReloadOutlined spin={isReloading} />}
            loading={isReloading}
            onClick={() => handleReloadDictionary(record)}
          >
            重载字典
          </Button>
        )
      },
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Page Header */}
      <Card
        size="small"
        style={{ borderRadius: 8 }}
        title={
          <Space>
            <ClusterOutlined style={{ color: '#987b2e', fontSize: 18 }} />
            <span style={{ fontSize: 16, fontWeight: 600 }}>集群拓扑与外部字典</span>
          </Space>
        }
        extra={
          <Button
            icon={<ReloadOutlined spin={loading} />}
            onClick={fetchData}
            loading={loading}
          >
            刷新数据
          </Button>
        }
      >
        <Text type="secondary">
          集中查看与监控 ClickHouse 分布式集群拓扑、分片与副本分布，观测 ReplicatedMergeTree 副本同步延迟与健康状态，并支持外部字典 (system.dictionaries) 的实时查看与一键热重载 (SYSTEM RELOAD DICTIONARY)。
        </Text>
      </Card>

      {/* Main Tabs */}
      <Card size="small" style={{ borderRadius: 8 }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'clusters',
              label: (
                <Space>
                  <ClusterOutlined />
                  <span>集群拓扑 (Clusters)</span>
                  <Badge count={uniqueClusters.length} overflowCount={999} style={{ backgroundColor: '#987b2e' }} />
                </Space>
              ),
              children: (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 16, paddingTop: 8 }}>
                  {/* Cluster KPI cards */}
                  <Row gutter={16}>
                    <Col xs={24} sm={12} md={6}>
                      <Card size="small" style={{ borderRadius: 6, background: 'rgba(152, 123, 46, 0.05)' }}>
                        <Statistic
                          title="集群总数"
                          value={uniqueClusters.length}
                          valueStyle={{ color: '#987b2e', fontWeight: 'bold' }}
                          prefix={<ClusterOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card size="small" style={{ borderRadius: 6, background: 'rgba(82, 196, 26, 0.05)' }}>
                        <Statistic
                          title="节点总数"
                          value={clusters.length}
                          valueStyle={{ color: '#52c41a', fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="当前本机节点"
                          value={localNodesCount}
                          valueStyle={{ fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="远程集群节点"
                          value={remoteNodesCount}
                          valueStyle={{ fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* Empty Alert for Standalone Mode */}
                  {clusters.length === 0 && !loading && (
                    <Alert
                      type="info"
                      showIcon
                      message="当前实例运行于单机模式 (Standalone)"
                      description={
                        <div>
                          当前 ClickHouse 的 <code>system.clusters</code> 中未检测到配置的分布式集群。
                          若需搭建分布式集群环境，可在 <strong>参数配置中心</strong> 或 <strong>高级 XML 编辑器</strong> 中配置 <code>&lt;remote_servers&gt;</code> 标签，添加分片 (shard) 与副本 (replica) 节点。
                        </div>
                      }
                    />
                  )}

                  {/* Filter Toolbar */}
                  {uniqueClusters.length > 0 && (
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Space>
                        <span>选择集群:</span>
                        <Select
                          value={selectedCluster}
                          onChange={setSelectedCluster}
                          style={{ width: 220 }}
                        >
                          <Select.Option value="ALL">全部集群 ({clusters.length} 节点)</Select.Option>
                          {uniqueClusters.map((c) => (
                            <Select.Option key={c} value={c}>
                              {c} ({clusters.filter((node) => node.cluster === c).length} 节点)
                            </Select.Option>
                          ))}
                        </Select>
                      </Space>
                      <Text type="secondary">共 {filteredClusters.length} 个节点</Text>
                    </div>
                  )}

                  {/* Table */}
                  <Table
                    rowKey={(r) => `${r.cluster}-${r.shard_num}-${r.replica_num}-${r.host_address}-${r.port}`}
                    columns={clusterColumns}
                    dataSource={filteredClusters}
                    loading={loading}
                    pagination={{ pageSize: 15, showSizeChanger: true }}
                    size="middle"
                  />
                </div>
              ),
            },
            {
              key: 'replicas',
              label: (
                <Space>
                  <CopyOutlined />
                  <span>副本健康 (Replicas)</span>
                  <Badge count={replicas.length} overflowCount={999} style={{ backgroundColor: '#52c41a' }} />
                </Space>
              ),
              children: (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 16, paddingTop: 8 }}>
                  {/* Replicas KPI cards */}
                  <Row gutter={16}>
                    <Col xs={24} sm={12} md={4}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="副本表总数"
                          value={replicas.length}
                          valueStyle={{ fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="Leader 主副本"
                          value={leaderReplicasCount}
                          valueStyle={{ color: '#faad14', fontWeight: 'bold' }}
                          prefix={<CrownOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card
                        size="small"
                        style={{
                          borderRadius: 6,
                          background: readonlyReplicasCount > 0 ? 'rgba(255, 77, 79, 0.08)' : undefined,
                        }}
                      >
                        <Statistic
                          title="只读异常表"
                          value={readonlyReplicasCount}
                          valueStyle={{
                            color: readonlyReplicasCount > 0 ? '#ff4d4f' : '#52c41a',
                            fontWeight: 'bold',
                          }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="同步队列积压"
                          value={totalQueueSize}
                          valueStyle={{
                            color: totalQueueSize > 20 ? '#faad14' : '#52c41a',
                            fontWeight: 'bold',
                          }}
                          suffix="任务"
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="最大同步延迟"
                          value={maxDelaySeconds}
                          valueStyle={{
                            color: maxDelaySeconds > 10 ? '#ff4d4f' : '#52c41a',
                            fontWeight: 'bold',
                          }}
                          suffix="秒"
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* Empty Alert for Replicas */}
                  {replicas.length === 0 && !loading && (
                    <Alert
                      type="info"
                      showIcon
                      message="未检测到 ReplicatedMergeTree 副本表"
                      description="当前数据库中暂无使用 ReplicatedMergeTree 引擎的表。创建具备多副本高可用冗余的表时，可在引擎中指定 ReplicatedMergeTree('/clickhouse/tables/{shard}/table_name', '{replica}')。"
                    />
                  )}

                  {/* Filter Toolbar */}
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Input
                      placeholder="搜索数据库名或表名..."
                      prefix={<SearchOutlined />}
                      value={replicaSearch}
                      onChange={(e) => setReplicaSearch(e.target.value)}
                      style={{ width: 260 }}
                      allowClear
                    />
                    <Text type="secondary">共 {filteredReplicas.length} 个副本表</Text>
                  </div>

                  {/* Table */}
                  <Table
                    rowKey={(r) => `${r.database}.${r.table}`}
                    columns={replicaColumns}
                    dataSource={filteredReplicas}
                    loading={loading}
                    pagination={{ pageSize: 15, showSizeChanger: true }}
                    size="middle"
                  />
                </div>
              ),
            },
            {
              key: 'dictionaries',
              label: (
                <Space>
                  <BookOutlined />
                  <span>外部字典 (Dictionaries)</span>
                  <Badge count={dictionaries.length} overflowCount={999} style={{ backgroundColor: '#7a8c67' }} />
                </Space>
              ),
              children: (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 16, paddingTop: 8 }}>
                  {/* Dictionaries KPI cards */}
                  <Row gutter={16}>
                    <Col xs={24} sm={12} md={4}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="字典总数"
                          value={dictionaries.length}
                          valueStyle={{ fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="已加载成功"
                          value={loadedDictsCount}
                          valueStyle={{ color: '#52c41a', fontWeight: 'bold' }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card
                        size="small"
                        style={{
                          borderRadius: 6,
                          background: failedDictsCount > 0 ? 'rgba(255, 77, 79, 0.08)' : undefined,
                        }}
                      >
                        <Statistic
                          title="加载失败异常"
                          value={failedDictsCount}
                          valueStyle={{
                            color: failedDictsCount > 0 ? '#ff4d4f' : '#52c41a',
                            fontWeight: 'bold',
                          }}
                          prefix={<ExclamationCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="缓存元素总数"
                          value={totalDictElements.toLocaleString()}
                          valueStyle={{ fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={5}>
                      <Card size="small" style={{ borderRadius: 6 }}>
                        <Statistic
                          title="总内存占用"
                          value={formatBytes(totalDictBytes)}
                          valueStyle={{ color: '#987b2e', fontWeight: 'bold' }}
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* Empty Alert for Dictionaries */}
                  {dictionaries.length === 0 && !loading && (
                    <Alert
                      type="info"
                      showIcon
                      message="未检测到外部字典 (External Dictionaries)"
                      description="外部字典是 ClickHouse 极为强大的特性，能够将外部数据源 (MySQL、PostgreSQL、ClickHouse、HTTP、本地文件等) 映射并在内存中维护高效索引，在 SQL 查询中实现极速的 dictGet() 与关联。"
                    />
                  )}

                  {/* Filter Toolbar */}
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Input
                      placeholder="搜索字典名称、数据库或数据源..."
                      prefix={<SearchOutlined />}
                      value={dictSearch}
                      onChange={(e) => setDictSearch(e.target.value)}
                      style={{ width: 280 }}
                      allowClear
                    />
                    <Text type="secondary">共 {filteredDictionaries.length} 个字典</Text>
                  </div>

                  {/* Table */}
                  <Table
                    rowKey={(d) => `${d.database}.${d.name}`}
                    columns={dictColumns}
                    dataSource={filteredDictionaries}
                    loading={loading}
                    pagination={{ pageSize: 15, showSizeChanger: true }}
                    size="middle"
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  )
}
