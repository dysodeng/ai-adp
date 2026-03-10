# AI-ADP (AI Application Development Platform)

基于 Go 语言的 AI 应用开发平台，采用领域驱动设计（DDD）架构，支持多租户、多模型接入和 Agent 智能交互。

## 技术栈

- **语言**: Go 1.25+
- **Web 框架**: Gin
- **依赖注入**: Google Wire
- **ORM**: GORM (PostgreSQL / MySQL)
- **缓存**: Redis (Standalone / Cluster / Sentinel)
- **AI 框架**: Eino (支持 OpenAI、Claude、Ark、Ollama)
- **可观测性**: OpenTelemetry (Tracing / Metrics / Logs)
- **日志**: Zap
- **配置**: Viper + YAML

## 项目结构

```
ai-adp/
├── cmd/app/                    # 应用入口
├── configs/                    # 配置文件
├── internal/
│   ├── domain/                 # 领域层
│   │   ├── agent/              # Agent 领域 (执行、工具、事件)
│   │   ├── tenant/             # 租户领域
│   │   ├── conversation/       # 会话领域
│   │   ├── model/              # 模型配置领域
│   │   ├── prompt/             # 提示词模板领域
│   │   ├── knowledge/          # 知识库领域
│   │   └── usage/              # 用量统计领域
│   ├── application/            # 应用服务层 (用例编排、DTO)
│   ├── infrastructure/         # 基础设施层
│   │   ├── config/             # 配置结构定义
│   │   ├── persistence/        # 数据持久化 (仓储实现、迁移)
│   │   ├── agent/              # Agent 基础设施 (适配器、取消机制)
│   │   ├── server/             # HTTP / Health 服务器
│   │   └── pkg/                # 第三方集成组件
│   │       ├── cache/          # 缓存 (Memory / Redis + 序列化)
│   │       ├── redis/          # Redis 客户端管理
│   │       ├── db/             # 数据库连接管理
│   │       ├── logger/         # 结构化日志
│   │       ├── serializer/     # 序列化器 (JSON / Msgpack)
│   │       └── telemetry/      # OpenTelemetry 集成
│   ├── di/                     # 依赖注入 (Wire)
│   └── interfaces/             # 接口层
│       └── http/               # HTTP Handler、路由、中间件
├── scripts/                    # 构建脚本
└── docs/                       # 文档
```

## 快速开始

### 环境要求

- Go 1.25+
- PostgreSQL 或 MySQL
- Redis

### 配置

```bash
cp .env.example .env
```

编辑 `.env` 或 `configs/app.yaml` 配置数据库、Redis 等连接信息。

### 运行

```bash
# 安装依赖
make tidy

# 生成 Wire 依赖注入代码
make wire

# 构建并运行
make run
```

### 常用命令

```bash
make build          # 构建二进制文件到 bin/ai-adp
make run            # 构建并运行
make test           # 运行测试
make test-cover     # 生成测试覆盖率报告
make wire           # 重新生成 Wire DI 代码
make lint           # 代码检查
make migrate        # 执行数据库迁移
make mockgen        # 生成 Mock 代码
make clean          # 清理构建产物
```

## API 接口

### 健康检查

```
GET /health
```

### 租户管理

```
POST   /v1/tenants          # 创建租户
GET    /v1/tenants           # 租户列表
GET    /v1/tenants/:id       # 获取租户
DELETE /v1/tenants/:id       # 删除租户
```

### 对话交互

```
POST /v1/chat/send-messages              # 发送消息 (支持 streaming / blocking)
POST /v1/chat/tasks/:task_id/cancel      # 取消运行中的任务
```

## 配置说明

配置文件 `configs/app.yaml` 主要包含以下模块：

| 模块 | 说明 |
|------|------|
| `app` | 应用名称、环境、调试模式 |
| `server` | HTTP / gRPC / WebSocket / Health 服务配置 |
| `security` | JWT 密钥配置 |
| `database` | 数据库连接及迁移配置 |
| `redis` | 三组 Redis 实例 (main / cache / mq)，支持单机、集群、哨兵模式 |
| `cache` | 缓存驱动 (memory / redis) 及序列化器 (json / msgpack) |
| `monitor` | OpenTelemetry 可观测性配置 |

## 架构设计

项目采用 DDD 四层架构 + 端口适配器模式：

- **领域层**: 核心业务逻辑，定义实体、值对象、仓储接口和领域服务
- **应用层**: 用例编排，协调领域对象完成业务流程
- **基础设施层**: 技术实现，包括数据库、缓存、AI 模型适配等
- **接口层**: 对外暴露 HTTP API，处理请求/响应转换

关键设计决策：
- 使用 Google Wire 实现编译期依赖注入
- Agent 执行支持取消机制 (Redis Pub/Sub 广播)
- 多模型抽象，通过 Eino 框架统一接入 OpenAI、Claude、Ark、Ollama
- 事件驱动的任务管理

## License

MIT
