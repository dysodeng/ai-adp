# AI 引擎重设计 — 统一 Executor 架构

**日期**: 2026-03-07
**状态**: 已批准
**目标**: 重新设计 AI 引擎层，支持类似 Dify/扣子的多种 AI 应用类型

---

## 1. 背景与问题

现有设计将 AI 能力封装为底层组件（`LLMExecutor`、`Embedder`、`AgentExecutor`），缺少"AI 应用"的概念。问题：
1. 没有应用类型抽象（Agent/Chat/TextGeneration/ChatFlow/Workflow）
2. Agent 实现仅包装单一 ReAct，未利用 ADK 丰富能力
3. 启动时加载固定默认模型，无法支持每个应用配置不同模型
4. 缺少应用编排层（版本管理、配置、变量注入等）

---

## 2. 应用类型定义

| 类型 | 说明 | 一期 |
|---|---|---|
| `agent` | LLM + 工具调用的 ReAct 推理循环，支持工具和知识库 | Yes |
| `chat` | 纯多轮对话，保持会话历史，无工具/无知识库 | Yes |
| `text_completion` | 单次 LLM 调用，无对话上下文，用于翻译/摘要/写作 | Yes |
| `chat_flow` | 基于条件分支的多轮对话流 | 二期 |
| `workflow` | 确定性 DAG 工作流，单次触发执行 | 二期 |

---

## 3. 领域模型：App Bounded Context

### 3.1 层级关系

```
Tenant → App → AppVersion → (Conversation)
```

App 直接归属 Tenant，不经过 Workspace 中间层。

### 3.2 聚合根与实体

**App（聚合根）**
- `id`: UUID v7
- `tenantID`: UUID — 归属租户
- `name`: string — 应用名称
- `description`: string — 应用描述
- `type`: AppType — 应用类型枚举
- `icon`: string — 应用图标
- `createdAt`, `updatedAt`

**AppVersion（实体）**
- `id`: UUID v7
- `appID`: UUID
- `version`: int — 版本号，自增
- `status`: VersionStatus — Draft | Published | Archived
- `config`: AppConfig — JSON 序列化的应用配置
- `publishedAt`: *time.Time
- `createdAt`, `updatedAt`

### 3.3 值对象

**AppType**: `agent` | `chat` | `text_completion` | `chat_flow` | `workflow`

**VersionStatus**: `draft` | `published` | `archived`

**AppConfig（JSON 序列化）**:
- `modelID`: UUID — 引用 ModelConfig
- `systemPrompt`: string — 系统提示词，支持 `{{变量名}}` 占位符
- `temperature`: *float32 — 覆盖模型默认温度
- `maxTokens`: int — 覆盖模型默认 token 上限
- `tools`: []ToolConfig — Agent 专用：工具配置列表
- `openingStatement`: string — 开场白
- `suggestedQuestions`: []string — 推荐问题

**ToolConfig（JSON 序列化）**:
- `name`: string
- `description`: string
- `parameters`: JSON — JSON Schema 定义参数

### 3.4 业务规则

1. 每个 App 同时只能有一个 Draft 版本
2. 发布时 Draft → Published，之前的 Published → Archived
3. 只有 Published 版本可被执行
4. App 创建时自动创建第一个 Draft 版本

---

## 4. AI 引擎层：统一 Executor 架构

### 4.1 领域端口

```go
// domain/shared/port/app_executor.go

type AppEventType string
const (
    AppEventMessage    AppEventType = "message"     // LLM 输出文本片段（流式）
    AppEventToolCall   AppEventType = "tool_call"   // 工具调用开始
    AppEventToolResult AppEventType = "tool_result"  // 工具调用结果
    AppEventDone       AppEventType = "done"        // 执行完成
    AppEventError      AppEventType = "error"       // 执行出错
)

type AppEvent struct {
    Type    AppEventType
    Content string
    Error   error
}

type AppExecutorInput struct {
    Query     string            // 用户输入
    Variables map[string]string // 提示词变量注入
    History   []Message         // 对话历史（Chat/Agent 用）
    Stream    bool              // 是否流式输出
}

type AppResult struct {
    Content      string
    InputTokens  int
    OutputTokens int
}

// AppExecutor 统一的 AI 应用执行端口
type AppExecutor interface {
    Run(ctx context.Context, input *AppExecutorInput) (<-chan AppEvent, error)
    Execute(ctx context.Context, input *AppExecutorInput) (*AppResult, error)
}
```

### 4.2 ExecutorFactory

```go
// infrastructure/ai/engine/factory.go

type ExecutorFactory interface {
    Create(ctx context.Context, appType AppType, config *AppConfig, modelConfig *ModelConfig) (AppExecutor, error)
}
```

工厂根据 `appType` 分发创建对应 Executor。当前实现为每次请求动态创建，后续可根据需要加缓存池。

### 4.3 Executor 与 Eino ADK 映射

| App Type | Executor | Eino ADK 组件 |
|---|---|---|
| `text_completion` | TextGenExecutor | `adk.ChatModelAgent`（无工具）+ `adk.Runner` |
| `chat` | ChatExecutor | `adk.ChatModelAgent`（无工具）+ `adk.Runner` |
| `agent` | AgentExecutor | `adk.ChatModelAgent`（带工具）+ `adk.Runner` |
| `chat_flow`（二期）| ChatFlowExecutor | `adk.SequentialAgent` + 条件分支 |
| `workflow`（二期）| WorkflowExecutor | `adk.SequentialAgent/ParallelAgent` |

**核心原理**：
- 所有应用类型统一使用 `adk.NewChatModelAgent` 作为基础
- TextCompletion/Chat：创建不带工具的 ChatModelAgent，本质是简单的 LLM Chain
- Agent：创建带工具的 ChatModelAgent，启用 ReAct 工具调用循环
- 所有 Executor 通过 `adk.NewRunner` 管理执行生命周期、流式输出和检查点
- 事件适配层（`event_adapter.go`）统一转换 ADK 的 `AgentEvent` 到领域的 `AppEvent`

### 4.4 提示词变量注入

- `AppConfig.systemPrompt` 中使用 `{{变量名}}` 占位符
- 执行时从 `AppExecutorInput.Variables` 中取值替换
- 示例：`"你是一个{{language}}翻译专家"` + `{"language": "英语"}` → `"你是一个英语翻译专家"`

### 4.5 三种交互模式

- **非流式（HTTP）**: 调用 `Execute()` → 返回完整 JSON 响应
- **流式（SSE）**: 调用 `Run()` → 消费 channel → 逐事件推送 SSE
- **WebSocket**: 调用 `Run()` → 消费 channel → 逐事件推送 WS 消息

---

## 5. 数据流

```
HTTP/WS Handler
    ↓
Application Service
    ├── 1. 加载 App + Published AppVersion
    ├── 2. 从 AppConfig 获取 modelID
    ├── 3. 通过 ModelConfigRepository 加载 ModelConfig
    ├── 4. ExecutorFactory.Create(appType, appConfig, modelConfig)
    ├── 5. executor.Run(input) 或 executor.Execute(input)
    └── 6. 消费 AppEvent channel → 流式响应 / 返回 AppResult
```

---

## 6. 变更范围

### 6.1 删除的文件

```
internal/domain/shared/port/llm_executor.go
internal/domain/shared/port/agent_executor.go
internal/domain/shared/port/agent_executor_test.go
internal/infrastructure/ai/llm/adapter.go
internal/infrastructure/ai/llm/adapter_test.go
internal/infrastructure/ai/llm/factory.go
internal/infrastructure/ai/agent/agent.go
internal/infrastructure/ai/agent/agent_test.go
internal/infrastructure/ai/provider.go
```

### 6.2 新增的文件

```
# App 领域层
internal/domain/app/model/app.go
internal/domain/app/model/app_version.go
internal/domain/app/valueobject/app_type.go
internal/domain/app/valueobject/version_status.go
internal/domain/app/valueobject/app_config.go
internal/domain/app/repository/app_repo.go
internal/domain/app/errors/errors.go

# AI 执行端口
internal/domain/shared/port/app_executor.go

# AI 引擎层
internal/infrastructure/ai/engine/factory.go
internal/infrastructure/ai/engine/chat_executor.go
internal/infrastructure/ai/engine/text_gen_executor.go
internal/infrastructure/ai/engine/agent_executor.go
internal/infrastructure/ai/engine/model_factory.go
internal/infrastructure/ai/engine/prompt.go

# 持久化层
internal/infrastructure/persistence/entity/app.go
internal/infrastructure/persistence/entity/app_version.go
internal/infrastructure/persistence/repository/app/app_repo_impl.go

# DI
internal/di/modules/app.go
```

### 6.3 修改的文件

```
internal/di/infrastructure.go   # 移除 ai.NewComponents，添加 engine.NewExecutorFactory
internal/di/app.go              # 移除 AIComponents 字段
internal/di/module.go           # 添加 AppModuleSet
internal/infrastructure/persistence/migration/migrate.go  # 添加 App/AppVersion 迁移
```

### 6.4 保留不变的文件

```
internal/infrastructure/ai/embedding/   # Embedding（知识库用）
internal/domain/model/                  # ModelConfig 领域模型
internal/domain/shared/port/embedder.go # Embedder 端口
```

---

## 7. 技术决策记录

1. **App 归属 Tenant**（非 Workspace），简化层级
2. **版本管理**：Draft → Published → Archived 生命周期
3. **配置存储**：JSON 字段存 DB，灵活且易版本化
4. **每次请求动态创建 Executor**：设计工厂接口，当前简单实现，后续可加缓存
5. **Eino ADK 深度集成**：Agent 用 ChatModelAgent + Runner，Chat/TextGen 直接用 ChatModel
6. **统一事件流**：所有应用类型返回相同的 `AppEvent` 流，上层统一处理
