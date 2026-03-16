# SSE 断线重连与续接设计

## 概述

为 AI-ADP 平台的 SSE 流式推送增加断线重连和断点续接能力。客户端因网络问题断开后，能在一定时间窗口内（默认 60s）从断点继续接收事件，超时后回退为重新生成。

## 需求

1. **混合策略**：60s 内支持断点续接，超时后回退为重新生成
2. **存储方案**：使用 Redis Streams 缓存事件
3. **客户端兼容**：同时支持浏览器 EventSource 和自定义客户端（如移动端 SDK）
4. **功能开关**：通过请求参数 `enable_sse_resume`（默认 `false`）控制，对现有调用方无影响
5. **复用现有端点**：在 `POST /v1/chat/send-messages` 上扩展重连参数；新增 `GET /v1/chat/tasks/:task_id/stream` 端点供浏览器 EventSource 自动重连使用

## 架构设计

### 1. 事件存储层 — Redis Stream

每个 task 对应一个 Redis Stream，key 格式：`sse:events:{task_id}`。

**写入：**
- 在 `AgentExecutor.publishEvent()` 中，广播事件后同步写入 Redis Stream（`XADD`）
- 使用 Redis 自动生成的 Stream ID（`*`），格式为 `{毫秒时间戳}-{序号}`
- 每条事件存储字段：`type`（事件类型）、`data`（JSON 序列化的事件数据）
- 仅当 `enable_sse_resume=true` 时才写入
- `XADD` 返回的 Stream ID 会被设置到 `model.Event.StreamID` 字段（新增字段），供 SSE 输出 `id:` 使用

**过期：**
- 使用 `EXPIRE` 对 Stream key 设置 60s TTL
- task 完成/取消/失败后，通过 EventStore 接口缩短 TTL 至 30s（在 infrastructure 层操作，不破坏 domain 层的 Redis 无关性）
- 无需 MAXLEN 限制（单次 AI 响应事件量有限），但 `XRANGE` 读取时使用 `COUNT 10000` 作为安全上限

**容错：**
- Redis 写入失败不影响正常事件推送（降级为不可续接，记录 warn 日志）

### 2. Executor 注册表 — ExecutorRegistry

**问题：** 现有 `TaskRegistry` 仅存储 `context.CancelFunc`，重连时无法获取 `AgentExecutor` 引用来订阅实时事件。

**方案：** 新增 `ExecutorRegistry` 接口和内存实现，用于映射 `task_id -> AgentExecutor`。

```go
// internal/domain/agent/executor/executor_registry.go
type ExecutorRegistry interface {
    Register(taskID string, executor AgentExecutor)
    Get(taskID string) (AgentExecutor, bool)
    Unregister(taskID string)
}
```

- 在 `ExecutorOrchestrator` 中，创建 AgentExecutor 后立即注册
- task 完成/取消/失败后，延迟 30s 注销（给重连留出时间窗口）
- 内存实现基于 `sync.Map`，与现有 `TaskRegistry` 模式一致

**多实例部署限制：**
- `ExecutorRegistry` 是内存级别的，重连请求必须命中与原始请求相同的实例才能获取到实时事件流
- 如果重连命中不同实例：仍可从 Redis Stream 回放已缓存的事件，但无法续接实时流。此时行为降级为"回放已有事件后关闭连接"，客户端可选择重新发起完整请求
- 建议在负载均衡层配置 sticky session（基于 task_id 或 client IP）以获得最佳体验

### 3. SSE 协议增强

**当前格式：**
```
event: chunk
data: {"type":"chunk","task_id":"...","data":"..."}

```

**增强后格式（enable_sse_resume=true 时）：**
```
id: 1710000000000-0
event: chunk
data: {"type":"chunk","task_id":"...","data":"..."}
retry: 3000

```

- `id:`：Redis Stream 自动生成的 ID（来自 `Event.StreamID` 字段）
- `retry:`：仅在首条事件中发送，告知浏览器 EventSource 重连间隔 3000ms
- `enable_sse_resume=false` 时，格式与当前完全一致

**新增端点（浏览器 EventSource 自动重连）：**

```
GET /v1/chat/tasks/:task_id/stream?last_event_id=xxx
```

浏览器原生 `EventSource` 只支持 `GET` 请求，无法发送 `POST` 请求体。因此需要一个 `GET` 端点供 EventSource 自动重连使用。客户端首次通过 `POST /v1/chat/send-messages` 创建任务后，可使用返回的 `task_id` 构造 EventSource URL。

自定义客户端（如移动端 SDK）可以选择：
- 使用 `POST /v1/chat/send-messages`（带 `task_id` + `last_event_id` 参数）重连
- 使用 `GET /v1/chat/tasks/:task_id/stream`（查询参数）重连

服务端识别 `Last-Event-ID`（Header，浏览器自动携带）和 `last_event_id`（查询参数），优先读取 Header。

**GET 端点鉴权：**
- 浏览器 `EventSource` 无法方便地设置自定义 Header，因此 GET 端点通过查询参数 `?api_key=xxx` 传递鉴权信息
- 复用现有的 API Key 中间件，增加对查询参数的读取支持

### 4. 重连续接流程

```
客户端断开 → 网络恢复 → 客户端重连(带 task_id + last_event_id)
                              ↓
                    服务端收到请求，解析参数
                              ↓
                    查询 Redis Stream: XRANGE sse:events:{task_id} ({last_id} + COUNT 10000
                              ↓
               ┌─── Stream 存在且有后续事件 ───┐
               ↓                               ↓
        回放缓存的事件                    Stream 不存在或已过期
        （跳过 last_id 本身）                    ↓
               ↓                        返回 expired 事件后关闭
        通过 ExecutorRegistry                客户端决定是否
        查找 AgentExecutor                   重新发起请求
               ↓
     ┌── 找到且仍在执行 ──┐
     ↓                    ↓
  订阅实时事件          未找到(不同实例/已结束)
  继续推送              仅回放缓存事件后关闭
```

**回放与实时的无缝衔接（关键序列）：**

```
1. 通过 ExecutorRegistry.Get(taskID) 获取 AgentExecutor
2. 如果找到且任务仍在执行中：
   a. 先调用 executor.Subscribe() 订阅实时事件通道（此时新事件开始缓冲到 channel 中）
   b. 从 Redis Stream XRANGE 读取 last_id 之后的所有缓存事件
   c. 发送所有缓存事件给客户端，记录最后发送的 StreamID（lastSentID）
   d. 从实时通道中读取事件：
      - 如果事件的 StreamID <= lastSentID，跳过（去重）
      - 否则正常发送
   e. 持续推送直到任务完成或客户端断开
3. 如果未找到 AgentExecutor：
   a. 从 Redis Stream 回放所有缓存事件
   b. 发送完毕后关闭连接
```

**关键点：先订阅再回放**，确保不丢失订阅前发布的事件。通过 `StreamID` 字段去重。

**重连时 Subscribe() 的合成 start 事件处理：**
- 现有 `Subscribe()` 在订阅者加入时会注入合成的 `start` 事件，该事件无 `StreamID`
- 重连场景下，去重逻辑遇到 `StreamID` 为空的事件时直接跳过（客户端已经收到过 start 事件）
- 实现上在 `HandleReconnection` 的实时事件消费循环中加一个判断：`if event.StreamID == "" { continue }`

**实时通道事件丢弃的处理：**
- 现有 `broadcastEvent` 在订阅者 channel 满时会静默丢弃事件
- 重连场景下这个问题更为突出，因为丢弃的事件已写入 Redis Stream 但不在实时通道中
- 缓解措施：重连时创建的订阅 channel 使用更大的缓冲区（如 200），减少丢弃概率
- 已知限制：极端情况下仍可能丢失事件，但考虑到 AI 流式响应的速率，实际发生概率极低

### 5. EventStore 注入方式

`EventStore` 通过构造函数的 Option 模式注入 `AgentExecutor`，而非在 `AgentExecutor` 接口上增加 `SetEventStore` 方法。这避免了 domain 层接口对 infrastructure 层的依赖：

```go
// internal/domain/agent/executor/executor_impl.go
type Option func(*agentExecutorImpl)

func WithEventStore(store EventStore) Option {
    return func(e *agentExecutorImpl) {
        e.eventStore = store
    }
}

func NewAgentExecutor(metadata *model.ExecutionMetadata, opts ...Option) AgentExecutor {
    // ...
}
```

`EventStore` 接口定义在 domain 层，实现在 infrastructure 层，符合 DDD 依赖倒置原则。

### 6. Event 模型扩展

在 `model.Event` 结构体中新增 `StreamID` 字段：

```go
type Event struct {
    Type      EventType   `json:"type"`
    TaskID    string      `json:"task_id"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data,omitempty"`
    StreamID  string      `json:"-"` // Redis Stream ID，不序列化到 JSON，仅供 SSE 输出使用
}
```

新增 `expired` 事件类型：

```go
const EventTypeExpired EventType = "expired"
```

expired 事件 SSE 格式：
```
event: expired
data: {"type":"expired","task_id":"...","timestamp":"...","data":{"message":"event stream expired, please retry with conversation_id"}}

```

### 7. 请求参数扩展

**`POST /v1/chat/send-messages` 请求体新增字段：**

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enable_sse_resume` | bool | false | 是否启用 SSE 断线续接能力 |
| `task_id` | string | "" | 重连时传入，要续接的任务 ID |
| `last_event_id` | string | "" | 重连时传入，最后收到的事件 ID |

**验证规则调整：**
- 当 `task_id` + `last_event_id` 非空时（重连请求），`query` 字段不再 required
- Handler 层需在 Gin binding 前判断请求类型，重连请求使用独立的 DTO 或条件校验

**请求场景：**

1. **正常首次请求**：`enable_sse_resume=true`，不传 `task_id` 和 `last_event_id` → 新建任务，启用事件缓存
2. **重连请求（自定义客户端）**：传 `task_id` + `last_event_id` → 走重连续接逻辑（隐含 `enable_sse_resume=true`）
3. **重连请求（浏览器 EventSource）**：`GET /v1/chat/tasks/:task_id/stream` + Header `Last-Event-ID` → 走重连续接逻辑
4. **传统请求**：`enable_sse_resume=false`（默认）→ 行为与当前完全一致

**注意：** 当请求包含 `task_id` + `last_event_id` 时，即使未显式传 `enable_sse_resume=true`，也视为重连请求（意图明确）。

### 8. Adapter 接口扩展

现有 `Adapter` 接口保持不变，新增独立的 `ReconnectableAdapter` 接口：

```go
// 现有接口不变
type Adapter interface {
    HandleExecution(ctx context.Context, executor executor.AgentExecutor)
}

// 新增接口，仅 SSEAdapter 实现
type ReconnectableAdapter interface {
    Adapter
    HandleReconnection(ctx context.Context, cachedEvents []model.Event, liveExecutor executor.AgentExecutor) // liveExecutor 可为 nil
}
```

- `BlockingAdapter` 无需改动，不实现 `ReconnectableAdapter`
- Handler 层通过类型断言 `adapter.(ReconnectableAdapter)` 判断是否支持重连
- `cachedEvents`：从 Redis Stream 回放的事件列表
- `liveExecutor`：如果找到正在执行的 executor 则传入，否则为 nil（仅回放缓存后关闭）

## 代码改动范围

### 修改的文件

| 文件 | 改动说明 |
|------|----------|
| `internal/domain/agent/model/event.go` | Event 增加 `StreamID` 字段；新增 `EventTypeExpired` 常量 |
| `internal/domain/agent/executor/executor_impl.go` | 构造函数增加可选的 `EventStore` 参数（Option 模式）；`publishEvent` 增加 Redis Stream 写入；task 终态时通知缩短 TTL |
| `internal/infrastructure/protocol/sse.go` | SSE 事件格式增加 `id:` 和 `retry:` 字段；实现 `HandleReconnection` 方法 |
| `internal/infrastructure/protocol/adapter.go` | 新增 `ReconnectableAdapter` 接口（不修改现有 `Adapter` 接口） |
| `internal/interfaces/http/handler/chat_handler.go` | 处理重连参数，分流重连 vs 新建逻辑 |
| `internal/interfaces/http/dto/request/chat_request.go` | 请求 DTO 增加 `enable_sse_resume`、`task_id`、`last_event_id`；`query` 条件校验 |
| `internal/application/chat/service/chat_app_service.go` | 新增重连业务逻辑方法 |
| `internal/application/chat/orchestrator/executor_orchestrator.go` | 创建 executor 后注册到 ExecutorRegistry；task 终态后延迟注销 |
| `internal/interfaces/http/router/router.go` | 新增 `GET /v1/chat/tasks/:task_id/stream` 路由 |

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/domain/agent/executor/event_store.go` | EventStore 事件存储接口定义 |
| `internal/domain/agent/executor/executor_registry.go` | ExecutorRegistry 接口定义 |
| `internal/infrastructure/agent/stream/event_store.go` | Redis Stream 事件存储实现（XADD/XRANGE/TTL 管理） |
| `internal/infrastructure/agent/stream/executor_registry.go` | ExecutorRegistry 内存实现（sync.Map） |

### 不需要改动的文件

- `blocking.go` — 阻塞模式不涉及重连
- `cancel_handler.go` — 取消逻辑不变

## 错误处理

- **Redis Stream 写入失败**：记录 warn 日志，降级为不可续接，不中断正常推送
- **Stream 已过期**：发送 `expired` 事件（`event: expired`），携带提示信息，客户端可带 `conversation_id` 重新发起完整请求
- **task_id 无效或未找到**：返回 HTTP 404 或 `error` 事件
- **重连时 task 已完成**：从 Redis Stream 回放剩余事件后正常关闭连接
- **重连命中不同实例**：Redis Stream 回放可正常工作，但无法续接实时流，回放完毕后关闭连接

## 测试策略

- 单元测试：Redis Stream 事件存储的 XADD/XRANGE/EXPIRE 操作
- 单元测试：SSE 格式输出（有/无 id 字段）
- 单元测试：Event.StreamID 的赋值和去重逻辑
- 单元测试：ExecutorRegistry 的注册/获取/注销及延迟注销
- 集成测试：完整重连流程（首次连接 → 断开 → 重连 → 续接）
- 集成测试：过期后回退场景（expired 事件）
- 集成测试：enable_sse_resume=false 时的向后兼容性
- 集成测试：重连命中不同实例时的降级行为

## 验收标准

1. `enable_sse_resume=false` 时，所有现有行为完全不变
2. `enable_sse_resume=true` 时，SSE 事件携带 `id:` 字段
3. 客户端在断开后 60s 内重连，能从断点继续接收后续事件，无丢失无重复
4. 客户端在断开后 60s+ 重连，收到 `expired` 事件
5. 浏览器 EventSource 能通过 `GET /v1/chat/tasks/:task_id/stream` 自动重连
6. 自定义客户端能通过 `POST /v1/chat/send-messages`（带 `task_id` + `last_event_id`）手动重连
7. Redis 写入失败时不影响正常推送
