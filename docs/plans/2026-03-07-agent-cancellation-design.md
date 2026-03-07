# Agent 执行取消设计

## 概述

为 agent 执行添加取消能力。调用方通过 task_id 取消正在运行的 agent，立即中断与 LLM 的通信，释放资源。支持单实例取消和 Redis Pub/Sub 分布式取消。

## 核心架构

### 领域层接口

**TaskRegistry** (`internal/domain/agent/executor/task_registry.go`)

```go
type TaskRegistry interface {
    Register(taskID string, cancelFunc context.CancelFunc)
    Unregister(taskID string)
    Cancel(taskID string) bool
}
```

- `Register`：执行开始时注册 taskID 与 cancelFunc
- `Unregister`：执行完成/失败时注销
- `Cancel`：取消指定任务，返回是否找到并取消

**CancelBroadcaster** (`internal/domain/agent/executor/cancel_broadcaster.go`)

```go
type CancelBroadcaster interface {
    Broadcast(ctx context.Context, taskID string) error
    Subscribe(ctx context.Context, registry TaskRegistry) error
}
```

- `Broadcast`：广播取消信号到所有实例
- `Subscribe`：订阅取消信号，收到后调用本地 TaskRegistry.Cancel()

### Context 传播链路

```
ExecutorOrchestrator.Execute()
    ├─ ctx, cancel := context.WithCancel(ctx)
    ├─ registry.Register(taskID, cancel)
    ├─ defer registry.Unregister(taskID)
    └─ goroutine: executeAgent(ctx, ...)
        └─ agent.Execute(ctx, ...)
            └─ ctx.Done() → 中断 HTTP 连接、停止 streaming
```

### 取消流程

```
POST /v1/chat/tasks/{task_id}/cancel
    ↓
CancelHandler
    ├─ TaskRegistry.Cancel(taskID) → 本地取消
    ├─ CancelBroadcaster.Broadcast(taskID) → 广播到所有实例
    └─ 返回 200

其他实例收到 Redis Pub/Sub 消息
    ↓
CancelBroadcaster.Subscribe 回调
    ↓
TaskRegistry.Cancel(taskID)
```

## SSE/Blocking 响应

### SSE 模式

每个事件 data 中携带 task_id。取消时发送 cancelled 事件后关闭流：

```
event: cancelled
data: {"task_id":"uuid-xxx","reason":"user_cancelled"}
```

### Blocking 模式

取消时返回 cancelled 状态的 JSON：

```json
{
  "task_id": "uuid-xxx",
  "status": "cancelled"
}
```

### 取消事件处理

1. `agent.Execute(ctx)` 因 ctx 取消返回 `context.Canceled`
2. `executeAgent()` 检测到此错误，调用 `agentExecutor.Cancel()`
3. `Cancel()` 发布 `EventTypeCancelled` 事件并关闭 subscriber channel
4. SSEAdapter/BlockingAdapter 收到事件后通知客户端

## API 端点

```
POST /v1/chat/tasks/{task_id}/cancel
Header: Authorization: Bearer <app_api_key>
Response: 200 {"result": "success"}
```

复用 AppApiKey 中间件认证。路由注册在 chat 路由组下。

## 基础设施层实现

### 内存注册表

`internal/infrastructure/agent/memory_task_registry.go`

- `sync.Map` 存储 `taskID → context.CancelFunc`
- Register 存入，Cancel 取出并调用，Unregister 删除

### Redis 取消广播

`internal/infrastructure/agent/redis_cancel_broadcaster.go`

- 固定 channel：`agent:task:cancel`
- `Broadcast()`：发布 taskID 到 channel
- `Subscribe()`：监听 channel，收到后调用 `registry.Cancel()`
- 应用启动时调用 Subscribe，关闭时通过 context 退出

### DI 注入

- MemoryTaskRegistry 单例注入
- RedisCancelBroadcaster 单例注入
- 注入到 ExecutorOrchestrator 和 CancelHandler
- 应用启动时调用 `broadcaster.Subscribe(ctx, registry)`

## 边界情况

| 场景 | 处理方式 |
|------|----------|
| 取消不存在的 task_id | 返回 200，幂等设计 |
| 取消已完成的任务 | Registry 已 Unregister，返回 200 |
| 重复取消 | cancelFunc 已调用并删除，后续无效，返回 200 |
| Redis 不可用 | 本地取消正常，Broadcast 记录警告日志 |
| 工具调用中取消 | context 传播中断工具调用 |

### 错误区分

`executeAgent()` 中：
- `context.Canceled` → `agentExecutor.Cancel()`
- 其他错误 → `agentExecutor.Fail(err)`
