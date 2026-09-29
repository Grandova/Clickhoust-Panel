# ClickHouse Manager

现代化、一体化 Web 可视化 ClickHouse 数据库运维与管理面板。

> **核心目标**：让任何不会手动编辑复杂 XML 配置文件的开发者与运维工程师，都能通过直观、现代化的 Web 界面完成 ClickHouse 的**一键安装、配置管理、服务启停、用户权限、可视化建表、数据浏览、SQL 查询、实时监控、慢查询追踪、后台 Merge/Mutation、系统日志与原生备份恢复**。

---

## 📸 核心特性

- 🖥️ **仪表盘 Dashboard**
  - ClickHouse 运行状态、版本（Server / Client）、进程 PID、连续运行时长。
  - 硬件指标实时采集：CPU 占用率、RAM 内存、磁盘空间、网络吞吐（RX/TX）。
  - 数据库资产统计：库数量、表数量、总记录行数、压缩后容量、未压缩容量、活跃连接数。
  - 实时性能图表：QPS、Insert/s、Select/s、Rows Read/s、Rows Write/s。

- 🚀 **一键安装 ClickHouse (Official Repos)**
  - 自动识别操作系统发行版（Debian / Ubuntu / Rocky Linux / AlmaLinux / CentOS）与 CPU 架构（x86_64 / aarch64）。
  - 自动导入 ClickHouse 官方 GPG 签名密钥，配置官方 APT / RPM 稳定版软件源。
  - 自动化安装 `clickhouse-server` 与 `clickhouse-client`，开放网络监听地址，配置管理员密码。
  - 安装进度与运维命令**通过 WebSocket 全程实时流式输出**。

- ⚙️ **服务生命周期管理 (Systemd)**
  - 启动、停止（双重二次确认）、重启（自动前置配置文件有效性验证，避免拉崩服务）、热重载。
  - 设置开机自启、取消开机自启、版本升级。
  - 直接查看 `systemd` 原生状态输出与运行日志。

- 🎛️ **可视化配置中心 & 高级 XML 编辑器**
  - **表单模式**：直观管理网络（`listen_host`、端口、连接数）、内存（`max_server_memory_usage` 等）、查询限制、缓存、MergeTree 并发以及日志级别。标注是否需要重启，支持关键词搜索。
  - **高级 XML 模式 (Monaco Editor)**：直接在线编辑 `config.xml`、`users.xml`、`config.d/*.xml`、`users.d/*.xml`。
  - **配置安全保障**：保存前强制 XML 语法检查；修改自动生成带时间戳的历史备份（`.bak`）；支持版本 Diff 对比与一键回滚。

- 🗄️ **数据库与数据表管理**
  - 数据库树形导航，创建、重命名、删除数据库（必须输入完整 `DROP <dbname>` 确认）。
  - 数据表详尽指标：引擎（MergeTree、ReplacingMergeTree、SummingMergeTree 等）、行数（无过滤极速读取 `system.tables` 统计）、压缩体积、Part 分区数、排序键（ORDER BY）、主键、TTL。
  - **字段结构管理 (Columns Drawer)**：在线查看表字段结构，支持添加新字段（Nullable、Codec 压缩算法、After 列、Comment 注释）、删除字段（双重二次确认保护）、修改字段注释。
  - **可视化建表生成器**：支持 ClickHouse 丰富类型（UInt/Int、Float、Decimal、String、DateTime、UUID、IPv4/6、Array、Map 等）与各种 MergeTree 变体引擎，支持引擎参数、ORDER BY、PRIMARY KEY、PARTITION BY、TTL，实时预览并执行 SQL。
  - **批量数据导入向导**：支持 CSV、CSVWithNames、TSV、TSVWithNames、JSONEachRow 格式；支持文件上传或文本直接粘贴；内置前 10 行数据结构与类型**即时预览**与总行数预估；一键调用 ClickHouse 原生 `FORMAT` 批量导入并汇报耗时与行数。
  - 表运维支持：TRUNCATE、OPTIMIZE FINAL、DETACH、ATTACH、查看与复制 CREATE TABLE DDL。

- 🌐 **集群拓扑、副本健康与外部字典 (Clusters & Dictionaries)**
  - **集群拓扑 (Clusters)**：实时读取 `system.clusters`，全面展示分布式集群、分片（Shard）、分片权重、副本（Replica）、节点地址、端口及本机/远程标签；提供单机模式智能向导。
  - **副本健康监控 (Replicas)**：深入监控 `system.replicas`，跟踪 ReplicatedMergeTree 副本 Leader 角色、只读异常状态（ZooKeeper 断开/只读）、复制延迟（`absolute_delay`）与插入/合并队列积压情况。
  - **外部字典管理 (Dictionaries)**：全面展示 `system.dictionaries` 外部字典列表，掌握加载状态（LOADED/FAILED）、内存占用、缓存条目数与数据源类型；支持**一键热重载字典 (`SYSTEM RELOAD DICTIONARY`)**。

- 🔍 **数据浏览器 (Data Browser)**
  - 类似 RedisInsight / phpMyAdmin 的交互式网格。
  - 安全机制：默认禁止无限制 `SELECT *`，内置 100/500/1000/5000 行安全分页保护。
  - 支持自定义 WHERE 条件过滤、多列排序、一键导出 CSV / TSV / JSON。

- 💻 **专业级 SQL 控制台 (Monaco Editor)**
  - ClickHouse SQL 语法高亮与全量关键字、系统表、内置函数智能自动补全。
  - **SQL 格式化器**：一键美化复杂 ClickHouse SQL。
  - **EXPLAIN 执行计划分析**：支持 `EXPLAIN PLAN`、`EXPLAIN PIPELINE`、`EXPLAIN SYNTAX`、`EXPLAIN AST` 四维执行计划解析。
  - 快捷键 `Ctrl + Enter` 执行选中文本或整段 SQL。
  - 详细性能指标：执行耗时（毫秒）、扫描行数、扫描字节、返回行数、Query ID。
  - **执行历史与本地持久化**：自动持久化保存执行历史；支持历史列表加载到当前标签、**在新标签页打开**、一键复制 SQL、单条删除与**一键清空全部历史**。
  - 多 Tab 标签页、常用 SQL 收藏夹、表格视图与 JSON 视图切换，CSV / TSV / JSON 导出。

- ⚡ **运维监控与性能排查**
  - **运行中查询**：实时捕获 `system.processes`，支持一键 Kill 阻塞慢查询。
  - **慢查询日志**：分析 `system.query_log`，支持自定义阈值（500ms、1s、3s、5s、10s），提供慢查询详情抽屉。
  - **实时监控中心**：ECharts 高清图表追踪 QPS、并发连接、CPU/RAM 占用率、网络吞吐。
  - **Parts 分区管理**：读取 `system.parts`，查看分区数据分布、Active 状态、Marks 与修改时刻。
  - **Merge 进度监控**：读取 `system.merges`，带实时百分比进度条与数据读写速度。
  - **Mutation 异步任务**：监控 ALTER UPDATE/DELETE 任务进度，排查失败原因，支持 Kill Mutation。
  - **磁盘与存储策略**：实时监控 `system.disks`，超过 80% 黄色告警，超过 90% 红色紧急告警。

- 📋 **日志中心 (Log Center)**
  - 涵盖 ClickHouse Server 主日志、Error 错误日志以及系统 Systemd Journal 日志。
  - 支持实时 WebSocket Tail 流式跟踪、级别过滤、关键词搜索、复制与下载。

- 💾 **原生备份与恢复 (BACKUP / RESTORE)**
  - 基于 ClickHouse 原生高效备份引擎，支持备份单表、整库或全实例。
  - 历史备份管理与一键恢复。

- 🛡️ **安全与操作审计**
  - ClickHouse 用户管理（Web 端配置用户、密码、允许访问主机 IP 网段、Profile、Quota、GRANT/REVOKE 权限）。
  - **防锁定 IP 白名单与安全限制**：支持配置面板访问白名单；**内置防自我锁定拦截保护**（防止管理员失误将自身 IP 关在门外）；展示当前客户端真实 IP 与一键快速添加。
  - **暴力破解防护与一键解封**：自动识别暴力密码猜测行为并锁定 IP；管理员可在设置面板实时查看被锁定的 IP 列表并一键解封。
  - 面板内置安全机制：JWT 鉴权、无状态连接池防污染隔离、首次登录强制改密、操作审计日志（记录所有高危指令、执行者、IP 与结果）。
  - **严格杜绝 Web Shell 后门**：底层命令统一经过白名单 Command Executor 执行，避免任意远程代码执行漏洞。

---

## 🛠️ 技术架构

```
ClickHouse Manager
├── frontend/ (React 19 + TypeScript + Vite + HeroUI v3 + Ant Design v6 + Monaco Editor + ECharts)
│   ├── src/
│   │   ├── api/          # 统一 Axios 请求封装与拦截器
│   │   ├── layouts/      # 响应式侧边栏布局，主题切换与顶部指标
│   │   ├── pages/        # 各业务模块页面
│   │   ├── types/        # 统一 TypeScript 类型契约
│   │   └── utils/        # 格式化与通用工具
└── backend/ (Go + Gin + SQLite + pure-Go ClickHouse Client)
    ├── cmd/server/       # 主服务入口，支持单个独立静态二进制
    ├── internal/
    │   ├── api/          # RESTful 控制器与 WebSocket 处理器
    │   ├── auth/         # JWT 与管理员密码哈希鉴权
    │   ├── clickhouse/   # ClickHouse Service 抽象层与驱动池
    │   ├── installer/    # APT / DNF 自动化安装器
    │   ├── config/       # XML 配置解析器、备份与回滚引擎
    │   ├── database/     # 面板自用 SQLite 存储（无需 CGO）
    │   ├── logs/         # 日志读取与流式 Tailer
    │   ├── system/       # 统一安全命令执行器与指标采集器
    │   └── audit/        # 操作审计记录器
    └── web/              # 单一二进制内置的前端静态资产 (embed.FS)
```

---

## 🚀 部署与更新指南

### 方式一：Linux 服务器一键安装（推荐）

在 Ubuntu / Debian / Rocky / AlmaLinux / CentOS 服务器上以 root 运行：

```bash
curl -fsSL https://raw.githubusercontent.com/Grandova/Clickhoust-Panel/main/scripts/install.sh | bash
```

安装脚本将自动：
1. 识别 CPU 架构（x86_64 / aarch64）与操作系统版本；
2. 自动安装基础依赖，从 GitHub Release 下载最新的 Linux 纯静态二进制包；
3. 创建 `/opt/clickhouse-manager` 运行目录；
4. 配置并启动 `systemd` 服务守护进程；
5. 输出控制台访问地址、默认管理员账号与密码。

输出示例：
```
============================================================
        ClickHouse Manager successfully installed!         
============================================================

  Web Panel URL:   http://YOUR_SERVER_IP:8080
  Default Username: admin
  Default Password: admin123456
  (首次登录要求修改密码)

  服务管理指令:
    systemctl status clickhouse-manager
    systemctl restart clickhouse-manager
    journalctl -u clickhouse-manager -f

  一键升级指令:
    curl -fsSL https://raw.githubusercontent.com/Grandova/Clickhoust-Panel/main/scripts/update.sh | bash
============================================================
```

---

### 🔄 Linux 服务器一键升级

若已有安装环境，只需运行以下指令即可无缝热升级至最新版本（自动备份旧二进制、拉取最新 Release、重启 Systemd 并自动执行健康状态校验）：

```bash
curl -fsSL https://raw.githubusercontent.com/Grandova/Clickhoust-Panel/main/scripts/update.sh | bash
```

---

### 方式二：Docker Compose 一键部署

只需一条命令即可同时启动 ClickHouse Server 数据库和 ClickHouse Manager 管理面板：

```bash
docker compose up -d
```

- ClickHouse 端口：`8123` (HTTP), `9000` (Native TCP)
- 管理面板端口：`http://localhost:8080`
- 默认登录：`admin` / `admin123456`

---

### 方式三：从源码编译单一二进制文件

项目支持将前后端打包为一个独立运行的可执行文件，无需安装 Node.js 或 Nginx 即可在任意服务器独立运行。

#### 1. 编译前端
```bash
cd frontend
npm install
npm run build
cd ..
```

#### 2. 将前端产物同步至 Go embed 目录并编译后端
```bash
# 复制静态资源
mkdir -p backend/web/dist
cp -r frontend/dist/* backend/web/dist/

# 编译 Linux 纯静态二进制（无需 CGO，支持 amd64 / arm64）
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o clickhouse-manager ./cmd/server
```

> **提示**：也可以直接运行根目录的自动化构建脚本：
> ```bash
> chmod +x scripts/build.sh
> ./scripts/build.sh release
> ```

#### 3. 运行服务
```bash
./clickhouse-manager -port 8080 -db data/clickhouse-manager.db
```

在浏览器打开 `http://localhost:8080` 即可开始使用。

---

## 🔒 安全设计原则

1. **零 Shell 注入风险**：面板不提供任何可执行任意 Bash 命令的终端入口，所有底层系统操作（`systemctl`、软件源配置等）均严格通过预定义的统一 Command Executor 与命令白名单执行。
2. **操作二次确认**：对所有破坏性或高危操作（清空表、删除数据库、删除用户、强制终止查询、卸载 ClickHouse），强制要求在对话框中输入指定关键词确认。
3. **配置热备份**：无论是表单修改还是 Monaco XML 编辑器修改，保存前均执行语法校验并生成版本快照。配置错误时拒绝重启 ClickHouse，确保数据库高可用。
4. **全链路审计**：所有管理员登录与关键运维操作均记录于 SQLite 审计日志，包含操作时间、IP、对象、状态及错误信息。

## v1.1.0 更新与验证

界面接入 [HeroUI](https://heroui.com/)，保留现有数据库表格与编辑器。修复详情与升级注意事项见 [RELEASE_NOTES.md](RELEASE_NOTES.md)。

本地检查：`cd frontend && npm ci && npm run lint && npm run build`；后端检查：`cd backend && go vet ./... && go test ./...`。GitHub Actions 还会针对官方 ClickHouse 容器运行 Native/HTTP 集成测试，通过后才发布标签对应的 Linux 二进制。

## v1.2.0 简化与性能优化

常用入口为：首页、数据库与表、查看数据、SQL 查询、连接与设置。监控、备份、服务管理等能力在「高级功能」中。首次使用从首页的「连接数据库」开始；本机未安装 ClickHouse 不影响连接远程实例。

首页不加载图表库，SQL/XML 编辑器无需访问 CDN。隐藏页面停止采样；主机指标缓存 2 秒、ClickHouse 汇总缓存 30 秒、目录大小后台刷新间隔为 5 分钟。监控时间范围展示当前页面已采集的数据，不代表服务端历史。

前端回归测试：`cd frontend && npm ci && npm test`。Go 测试文件必须纳入版本控制，CI 会确认独立 ClickHouse 集成测试存在后再运行。超出 JavaScript 安全范围的整数及 NaN/Infinity 以字符串返回，DateTime64 保留小数秒。
