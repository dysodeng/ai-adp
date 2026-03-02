# AI Development Platform (ADP) — 底层架构设计

**日期**: 2026-03-02
**状态**: 已批准
**模块**: github.com/dysodeng/ai-adp

---

## 1. 项目定位

全栈 AI 开发平台，提供：

- **统一模型网关**：接入 OpenAI、Claude、国内大模型（通义千问、深度求索、百度文心）及本地私有化部署（Ollama）
- **AI 应用开发套件**：LLM 对话、Prompt 管理、AI Agent、知识库 / RAG
- **平台管控中台**：多租户 / 工作空间、API Key 管理、用量与费用统计、模型配置管理

AI 执行引擎选用 **Eino**（github.com/cloudwego/eino）。

---

## 2. 架构方案：模块化 DDD + AI 能力分域

严格遵循 DDD 四层架构，AI 能力按 Bounded Context 拆分。参考 [dysodeng/app](https://github.com/dysodeng/app) 架构风格。

### 2.1 架构分层原则

| 层 | 职责 | 依赖规则 |
|---|---|---|
| **domain** | 纯业务逻辑，聚合根、值对象、仓储接口 | 不依赖任何外部框架 |
| **application** | 用例编排，协调 domain 与基础设施 | 只 import domain |
| **infrastructure** | 持久化实现、Eino AI 引擎、缓存、搜索 | 实现 domain 定义的接口 |
| **interfaces** | HTTP/WebSocket 协议适配 | 调用 application 层 |

---

## 3. 目录结构

```
ai-adp/
├── main.go
├── cmd/app/
├── api/                           # protobuf 定义（预留 gRPC）
├── configs/                       # YAML 配置
├── docs/plans/                    # 设计文档
├── scripts/
├── var/
│
└── internal/
    ├── domain/
    │   ├── shared/                # 共享值对象、领域错误基类
    │   │   └── port/
    │   │       └── llm_executor.go  # LLM 执行端口接口（由 domain 层定义）
    │   ├── tenant/
    │   ├── model/
    │   ├── prompt/
    │   ├── conversation/
    │   ├── agent/
    │   ├── knowledge/
    │   └── usage/
    │
    ├── application/
    │   ├── tenant/
    │   ├── model/
    │   ├── prompt/
    │   ├── conversation/
    │   ├── agent/
    │   ├── knowledge/
    │   └── usage/
    │
    ├── infrastructure/
    │   ├── ai/                    # Eino 封装（仅此处依赖 Eino）
    │   │   ├── provider/          # LLM 提供商适配
    │   │   ├── chain/             # Chat/RAG Chain 封装
    │   │   └── embedding/         # Embedding 服务
    │   ├── persistence/
    │   │   ├── entity/            # GORM 实体（PostgreSQL，主键 UUID v7）
    │   │   ├── repository/        # 仓储接口实现
    │   │   ├── vector/            # 向量数据库适配（Milvus/Qdrant）
    │   │   └── transactions/
    │   ├── cache/                 # Redis
    │   ├── search/                # Elasticsearch
    │   ├── config/
    │   ├── event/                 # 事件总线
    │   └── server/
    │
    ├── interfaces/
    │   └── http/
    │       ├── middleware/        # Auth、租户解析、限流
    │       └── handler/           # 各 context handler（含 SSE 响应模式）
    │
    └── di/                        # Google Wire 依赖注入
        ├── wire.go
        ├── wire_gen.go
        ├── app.go
        ├── infrastructure.go
        ├── module.go
        └── modules/
```

---

## 4. Bounded Contexts

### 4.1 一期核心上下文

| Context | 聚合根 / 核心实体 | 关键值对象 |
|---|---|---|
| `tenant` | Tenant（租户）、Workspace（工作空间） | TenantStatus、Plan |
| `model` | ModelProvider（提供商）、ModelConfig（模型配置） | ProviderType、ModelCapability |
| `prompt` | PromptTemplate（模板聚合根） | PromptVersion、VariableSchema |
| `conversation` | Conversation（会话）、Message（消息实体） | Role、MessageStatus |
| `agent` | AgentDefinition（预留骨架） | AgentType 枚举 |
| `knowledge` | KnowledgeBase（知识库）、Document（文档） | ChunkStrategy、IndexStatus |
| `usage` | UsageRecord（用量记录）、Quota（配额） | TokenCount、CostEstimate |

### 4.2 Agent 类型规划（后续迭代）

`agent` context 一期仅定义 `AgentType` 枚举和基础聚合根骨架，后续按序实现：

| 类型 | 说明 |
|---|---|
| `ReAct` | 推理 + 行动循环，Tool Call 驱动 |
| `Chat` | 多轮对话 Agent，支持工具调用 |
| `TextGeneration` | 单次文本生成，无对话状态 |
| `ChatFlow` | 带条件分支的对话流 |
| `Workflow` | 确定性 DAG 工作流 |
| `MultiAgent` | 多 Agent 协作编排 |

### 4.3 各 Context 内部标准结构

以 `conversation` 为例：

```
domain/conversation/
├── model/
│   ├── conversation.go        # 聚合根
│   └── message.go             # 实体
├── valueobject/
│   ├── role.go                # User / Assistant / System
│   └── message_status.go      # Pending / Streaming / Done / Failed
├── repository/
│   └── conversation_repo.go   # 仓储接口
├── service/
│   └── conversation_service.go
└── errors/
    └── errors.go

application/conversation/
├── dto/
├── service/
│   └── conversation_app_service.go
├── event/
└── decorator/

infrastructure/persistence/repository/conversation/
├── conversation_repo_impl.go
└── query.go
```

---

## 5. Eino 封装策略

Eino 仅存在于 `infrastructure/ai/`，domain 层通过接口解耦：

```
domain/shared/port/llm_executor.go
  └─ interface LLMExecutor { Execute, Stream }

infrastructure/ai/
├── provider/
│   ├── factory.go          # 根据 ModelConfig 创建对应 provider
│   ├── openai.go           # Eino OpenAI ChatModel
│   ├── anthropic.go        # Eino Claude
│   ├── ollama.go           # 本地模型
│   └── domestic/           # 国内大模型（通义、DeepSeek、文心）
├── chain/
│   ├── chat_chain.go       # 对话 Chain（含 Prompt 注入）
│   ├── rag_chain.go        # RAG Chain（Retriever + ChatModel）
│   └── stream.go           # 流式输出统一封装（SSE/WebSocket）
└── embedding/
    └── embedding_svc.go    # 文档切片 → 向量
```

---

## 6. 数据流

```
HTTP Request (Gin)
        ↓
interfaces/http/handler/        # DTO 转换、参数校验
        ↓
application/*/service/          # 用例编排
        ↓                               ↓
domain/*/service/ + model/      infrastructure/ai/
（纯业务逻辑）                    （Eino 执行：Chat/RAG/Stream）
        ↓
infrastructure/persistence/     infrastructure/cache/
（PostgreSQL + UUID v7）         （Redis：会话缓存、限流）
        ↓
infrastructure/persistence/vector/   infrastructure/search/
（向量数据库：知识库检索）              （ES：全文搜索）
```

---

## 7. 存储选型

| 存储 | 用途 |
|---|---|
| PostgreSQL | 主关系数据库，主键统一使用 UUID v7 |
| Redis | 会话缓存、API Key 限流、短期状态 |
| 向量数据库（Milvus/Qdrant） | RAG 知识库向量存储与检索 |
| Elasticsearch | 知识库文档全文搜索 |

---

## 8. 接口层

- **HTTP REST**（Gin）：主要对外 API，含 SSE 响应模式（流式输出直接在 handler 中处理）
- **WebSocket**：实时双向通信，用于流式对话场景

---

## 9. 依赖注入

使用 **Google Wire** 编译期 DI，结构与 `dysodeng/app` 保持一致：
- `infrastructure.go`：基础设施组件（DB、Redis、Vector DB、ES、AI Engine）
- `module.go`：各 context 服务聚合
- `modules/`：每个 bounded context 独立 Wire Set

---

## 10. 技术栈汇总

| 分类 | 选型 |
|---|---|
| 语言 / 运行时 | Go 1.25 |
| AI 执行引擎 | Eino (cloudwego/eino) |
| Web 框架 | Gin |
| ORM | GORM |
| 主数据库 | PostgreSQL（UUID v7 主键） |
| 缓存 | Redis |
| 向量数据库 | Milvus / Qdrant（可配置） |
| 全文搜索 | Elasticsearch |
| 依赖注入 | Google Wire |
| 配置管理 | Viper |
| 日志 | Zap |
| 可观测性 | OpenTelemetry |
