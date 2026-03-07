# Agent Execution Cancellation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add task cancellation capability to agent execution, supporting local cancel via TaskRegistry and distributed cancel via Redis Pub/Sub.

**Architecture:** TaskRegistry (sync.Map) holds taskID→cancelFunc mappings for local cancellation. CancelBroadcaster uses Redis Pub/Sub to propagate cancel signals across instances. ExecutorOrchestrator wraps execution context with WithCancel, registers/unregisters tasks, and distinguishes context.Canceled from other errors.

**Tech Stack:** Go, Gin, Google Wire, Redis (github.com/redis/go-redis/v9), sync.Map

---

### Task 1: TaskRegistry Interface & MemoryTaskRegistry

**Files:**
- Create: `internal/domain/agent/executor/task_registry.go`
- Create: `internal/infrastructure/agent/memory_task_registry.go`

**Step 1: Create the TaskRegistry domain interface**

```go
// internal/domain/agent/executor/task_registry.go
package executor

import "context"

// TaskRegistry 任务注册表 - 管理 taskID 与 cancelFunc 的映射
type TaskRegistry interface {
	Register(taskID string, cancelFunc context.CancelFunc)
	Unregister(taskID string)
	Cancel(taskID string) bool
}
```

**Step 2: Create MemoryTaskRegistry implementation**

```go
// internal/infrastructure/agent/memory_task_registry.go
package agent

import (
	"context"
	"sync"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

// MemoryTaskRegistry 基于内存的任务注册表
type MemoryTaskRegistry struct {
	tasks sync.Map // taskID → context.CancelFunc
}

// NewMemoryTaskRegistry 创建内存任务注册表
func NewMemoryTaskRegistry() *MemoryTaskRegistry {
	return &MemoryTaskRegistry{}
}

func (r *MemoryTaskRegistry) Register(taskID string, cancelFunc context.CancelFunc) {
	r.tasks.Store(taskID, cancelFunc)
}

func (r *MemoryTaskRegistry) Unregister(taskID string) {
	r.tasks.Delete(taskID)
}

func (r *MemoryTaskRegistry) Cancel(taskID string) bool {
	value, ok := r.tasks.LoadAndDelete(taskID)
	if !ok {
		return false
	}
	cancelFunc := value.(context.CancelFunc)
	cancelFunc()
	return true
}

// 编译期接口检查
var _ executor.TaskRegistry = (*MemoryTaskRegistry)(nil)
```

**Step 3: Commit**

```bash
git add internal/domain/agent/executor/task_registry.go internal/infrastructure/agent/memory_task_registry.go
git commit -m "feat(agent): add TaskRegistry interface and MemoryTaskRegistry implementation"
```

---

### Task 2: CancelBroadcaster Interface & RedisCancelBroadcaster

**Files:**
- Create: `internal/domain/agent/executor/cancel_broadcaster.go`
- Create: `internal/infrastructure/agent/redis_cancel_broadcaster.go`

**Step 1: Create the CancelBroadcaster domain interface**

```go
// internal/domain/agent/executor/cancel_broadcaster.go
package executor

import "context"

// CancelBroadcaster 取消信号广播器 - 在分布式环境中广播取消信号
type CancelBroadcaster interface {
	Broadcast(ctx context.Context, taskID string) error
	Subscribe(ctx context.Context, registry TaskRegistry) error
}
```

**Step 2: Create RedisCancelBroadcaster implementation**

```go
// internal/infrastructure/agent/redis_cancel_broadcaster.go
package agent

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

const cancelChannel = "agent:task:cancel"

// RedisCancelBroadcaster 基于 Redis Pub/Sub 的取消信号广播器
type RedisCancelBroadcaster struct {
	client *redis.Client
}

// NewRedisCancelBroadcaster 创建 Redis 取消广播器
func NewRedisCancelBroadcaster(client *redis.Client) *RedisCancelBroadcaster {
	return &RedisCancelBroadcaster{client: client}
}

func (b *RedisCancelBroadcaster) Broadcast(ctx context.Context, taskID string) error {
	if err := b.client.Publish(ctx, cancelChannel, taskID).Err(); err != nil {
		logger.Warn(ctx, "[CancelBroadcaster] broadcast failed", logger.ErrorField(err))
		return err
	}
	return nil
}

func (b *RedisCancelBroadcaster) Subscribe(ctx context.Context, registry executor.TaskRegistry) error {
	pubsub := b.client.Subscribe(ctx, cancelChannel)

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				taskID := msg.Payload
				registry.Cancel(taskID)
			}
		}
	}()

	return nil
}

// 编译期接口检查
var _ executor.CancelBroadcaster = (*RedisCancelBroadcaster)(nil)
```

**Step 3: Commit**

```bash
git add internal/domain/agent/executor/cancel_broadcaster.go internal/infrastructure/agent/redis_cancel_broadcaster.go
git commit -m "feat(agent): add CancelBroadcaster interface and RedisCancelBroadcaster implementation"
```

---

### Task 3: Integrate TaskRegistry into ExecutorOrchestrator

**Files:**
- Modify: `internal/application/chat/orchestrator/executor_orchestrator.go`

**Step 1: Add TaskRegistry dependency and context cancellation**

Modify `executorOrchestrator` to accept `TaskRegistry`, wrap context with `WithCancel`, and register/unregister tasks. In `executeAgent`, distinguish `context.Canceled` from other errors.

```go
// executor_orchestrator.go - full replacement

package orchestrator

import (
	"context"
	"errors"
	"fmt"

	domainagent "github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/service"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// ExecutorOrchestrator Agent 执行编排器
type ExecutorOrchestrator interface {
	// Execute 执行 Agent（非阻塞），通过 AgentExecutor 事件流获取结果
	Execute(ctx context.Context, app *appmodel.App, version *appmodel.AppVersion, agentExecutor executor.AgentExecutor, isStreaming bool) error
}

type executorOrchestrator struct {
	agentBuilder service.AgentBuilder
	agentFactory *adapter.AgentFactory
	taskRegistry executor.TaskRegistry
}

// NewExecutorOrchestrator 创建执行编排器
func NewExecutorOrchestrator(
	agentBuilder service.AgentBuilder,
	agentFactory *adapter.AgentFactory,
	taskRegistry executor.TaskRegistry,
) ExecutorOrchestrator {
	return &executorOrchestrator{
		agentBuilder: agentBuilder,
		agentFactory: agentFactory,
		taskRegistry: taskRegistry,
	}
}

func (o *executorOrchestrator) Execute(
	ctx context.Context,
	app *appmodel.App,
	version *appmodel.AppVersion,
	agentExecutor executor.AgentExecutor,
	isStreaming bool,
) error {
	logger.Info(ctx, "[orchestrator] Execute start",
		logger.AddField("appID", app.ID().String()),
		logger.AddField("modelID", version.Config().ModelID.String()),
		logger.AddField("isStreaming", isStreaming),
	)

	// 1. 构建 Agent 配置
	config, err := o.agentBuilder.BuildAgentConfig(ctx, app, version, agentExecutor.GetInput().Variables, isStreaming)
	if err != nil {
		return fmt.Errorf("build agent config failed: %w", err)
	}
	logger.Info(ctx, "[orchestrator] BuildAgentConfig done")

	// 2. 通过工厂创建 Agent，使用 version 中的 ModelID
	modelID := version.Config().ModelID.String()
	ag, err := o.agentFactory.CreateAgent(ctx, app.Type(), config, modelID)
	if err != nil {
		return fmt.Errorf("create agent failed: %w", err)
	}
	logger.Info(ctx, "[orchestrator] CreateAgent done")

	// 3. 创建可取消的 context 并注册到 TaskRegistry
	taskID := agentExecutor.GetTaskID().String()
	execCtx, cancel := context.WithCancel(ctx)
	o.taskRegistry.Register(taskID, cancel)

	// 4. 启动执行
	agentExecutor.Start()
	logger.Info(ctx, "[orchestrator] agentExecutor.Start() done, launching goroutine")

	// 5. 异步执行 Agent
	go func() {
		defer o.taskRegistry.Unregister(taskID)
		o.executeAgent(execCtx, ag, agentExecutor)
	}()

	// 6. 如果是阻塞模式，等待完成
	if !isStreaming {
		eventChan := agentExecutor.AddSubscriber()
		for range eventChan {
			// 消费事件直到 channel 关闭
		}
		if agentExecutor.Err() != nil {
			return agentExecutor.Err()
		}
	}

	logger.Info(ctx, "[orchestrator] Execute returning")
	return nil
}

func (o *executorOrchestrator) executeAgent(ctx context.Context, ag domainagent.Agent, agentExecutor executor.AgentExecutor) {
	logger.Info(ctx, "[orchestrator] executeAgent goroutine started")
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("agent execution panic: %v", r)
			logger.Error(ctx, "[orchestrator] agent execution panic", logger.ErrorField(err))
			agentExecutor.Fail(err)
		}
	}()

	logger.Info(ctx, "[orchestrator] calling ag.Execute ...")
	if err := ag.Execute(ctx, agentExecutor); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info(ctx, "[orchestrator] agent execution cancelled")
			agentExecutor.Cancel()
			return
		}
		logger.Error(ctx, "[orchestrator] ag.Execute failed", logger.ErrorField(err))
		agentExecutor.Fail(err)
		return
	}
	logger.Info(ctx, "[orchestrator] ag.Execute completed successfully")
}
```

**Step 2: Verify build**

Run: `go build ./internal/application/chat/orchestrator/...`
Expected: Build will fail because DI not yet updated (NewExecutorOrchestrator signature changed). This is expected — DI fix is in Task 5.

**Step 3: Commit**

```bash
git add internal/application/chat/orchestrator/executor_orchestrator.go
git commit -m "feat(orchestrator): integrate TaskRegistry for cancellation context management"
```

---

### Task 4: Cancel API Handler

**Files:**
- Create: `internal/interfaces/http/handler/cancel_handler.go`
- Modify: `internal/interfaces/http/router.go`

**Step 1: Create CancelHandler**

```go
// internal/interfaces/http/handler/cancel_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// CancelHandler 任务取消 Handler
type CancelHandler struct {
	taskRegistry      executor.TaskRegistry
	cancelBroadcaster executor.CancelBroadcaster
}

// NewCancelHandler 创建 Cancel Handler
func NewCancelHandler(
	taskRegistry executor.TaskRegistry,
	cancelBroadcaster executor.CancelBroadcaster,
) *CancelHandler {
	return &CancelHandler{
		taskRegistry:      taskRegistry,
		cancelBroadcaster: cancelBroadcaster,
	}
}

// Cancel 取消指定任务
func (h *CancelHandler) Cancel(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusOK, response.Fail(ctx, "task_id is required", response.CodeFail))
		return
	}

	// 本地取消
	h.taskRegistry.Cancel(taskID)

	// 广播到其他实例
	if err := h.cancelBroadcaster.Broadcast(ctx, taskID); err != nil {
		logger.Warn(ctx, "[CancelHandler] broadcast cancel failed", logger.ErrorField(err))
	}

	ctx.JSON(http.StatusOK, response.Success(ctx, map[string]string{"result": "success"}))
}
```

**Step 2: Add CancelHandler to Router and register cancel route**

Modify `internal/interfaces/http/router.go`:

- Add `cancelHandler *handler.CancelHandler` field to Router struct
- Add it to NewRouter parameters
- Register route: `chats.POST("/tasks/:task_id/cancel", r.cancelHandler.Cancel)`

Updated router:

```go
package http

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// Router HTTP 路由管理器，集中管理所有路由注册
type Router struct {
	tenantHandler *handler.TenantHandler
	chatHandler   *handler.ChatHandler
	cancelHandler *handler.CancelHandler
}

// NewRouter 创建路由管理器
func NewRouter(tenantHandler *handler.TenantHandler, chatHandler *handler.ChatHandler, cancelHandler *handler.CancelHandler) *Router {
	return &Router{
		tenantHandler: tenantHandler,
		chatHandler:   chatHandler,
		cancelHandler: cancelHandler,
	}
}

// Setup 配置 Gin Engine 的中间件和路由
func (r *Router) Setup(engine *gin.Engine, appName string) {
	engine.Use(
		middleware.Recovery(),
		middleware.Tracing(appName),
		middleware.Logger(),
		middleware.RequestID(),
	)

	engine.GET("/health", handler.HealthCheck)

	v1 := engine.Group("/v1")
	r.registerV1Routes(v1)
}

// registerV1Routes 注册 v1 版本的所有路由
func (r *Router) registerV1Routes(v1 *gin.RouterGroup) {
	// Tenant 路由
	tenants := v1.Group("/tenants")
	{
		tenants.POST("", r.tenantHandler.Create)
		tenants.GET("", r.tenantHandler.List)
		tenants.GET("/:id", r.tenantHandler.GetByID)
		tenants.DELETE("/:id", r.tenantHandler.Delete)
	}

	// Chat 路由
	chats := v1.Group("/chat", middleware.AppApiKey)
	{
		chats.POST("/send-messages", r.chatHandler.Chat)
		chats.POST("/tasks/:task_id/cancel", r.cancelHandler.Cancel)
	}
}
```

**Step 3: Commit**

```bash
git add internal/interfaces/http/handler/cancel_handler.go internal/interfaces/http/router.go
git commit -m "feat(api): add cancel task endpoint POST /v1/chat/tasks/:task_id/cancel"
```

---

### Task 5: DI Wiring

**Files:**
- Modify: `internal/di/infrastructure.go`
- Modify: `internal/di/chat_module.go`
- Modify: `internal/di/app.go`
- Regenerate: `internal/di/wire_gen.go`

**Step 1: Add MemoryTaskRegistry and RedisCancelBroadcaster to InfrastructureSet**

Add to `internal/di/infrastructure.go` in `InfrastructureSet`:

```go
// 取消能力组件
infraagent.NewMemoryTaskRegistry,
wire.Bind(new(executor.TaskRegistry), new(*infraagent.MemoryTaskRegistry)),
infraagent.NewRedisCancelBroadcaster,
wire.Bind(new(executor.CancelBroadcaster), new(*infraagent.RedisCancelBroadcaster)),
```

Add the necessary imports:

```go
"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
infraagent "github.com/dysodeng/ai-adp/internal/infrastructure/agent"
```

**Step 2: Add CancelHandler to ChatModuleSet**

Modify `internal/di/chat_module.go`:

```go
var ChatModuleSet = wire.NewSet(
	chatorch.NewExecutorOrchestrator,
	chatservice.NewChatAppService,
	handler.NewChatHandler,
	handler.NewCancelHandler,
)
```

**Step 3: Add CancelBroadcaster subscription to App lifecycle**

Modify `internal/di/app.go` — add `CancelBroadcaster` and `TaskRegistry` to `App` struct so the application can start subscription on launch:

```go
package di

import (
	"context"

	"go.uber.org/zap"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	agentservice "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	appdomainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// App 持有所有服务实例和清理函数，由 cmd/app 驱动生命周期
type App struct {
	HTTPServer        server.Server
	tracerShutdown    telemetry.ShutdownFunc
	cancelBroadcaster executor.CancelBroadcaster
	taskRegistry      executor.TaskRegistry
}

// NewApp 构造 App。_ *zap.Logger 确保 Wire 在构建 App 前初始化全局 logger（顺序依赖）。
func NewApp(
	httpServer *server.HTTPServer,
	_ *engine.ExecutorFactory,
	_ appdomainrepo.AppRepository,
	_ *zap.Logger,
	tracerShutdown telemetry.ShutdownFunc,
	// 新架构组件
	_ port.ToolService,
	_ agentservice.AgentBuilder,
	_ *adapter.AgentFactory,
	// 取消能力组件
	cancelBroadcaster executor.CancelBroadcaster,
	taskRegistry executor.TaskRegistry,
) *App {
	return &App{
		HTTPServer:        httpServer,
		tracerShutdown:    tracerShutdown,
		cancelBroadcaster: cancelBroadcaster,
		taskRegistry:      taskRegistry,
	}
}

// StartCancelSubscriber 启动取消信号订阅
func (a *App) StartCancelSubscriber(ctx context.Context) error {
	return a.cancelBroadcaster.Subscribe(ctx, a.taskRegistry)
}

// Stop 释放应用资源，在所有 Server 停止后调用
func (a *App) Stop(ctx context.Context) error {
	// 刷新日志缓冲，防止文件输出丢失最后几行
	_ = logger.ZapLogger().Sync()
	return a.tracerShutdown(ctx)
}
```

**Step 4: Call StartCancelSubscriber in application startup**

Modify `internal/cmd/app/app.go` — in `serve()` method, after server registration, call `StartCancelSubscriber`:

Add after `a.registerServer(...)` block in `serve()`:

```go
// 启动取消信号订阅
if err := a.mainApp.StartCancelSubscriber(a.ctx); err != nil {
    logger.Error(a.ctx, "failed to start cancel subscriber", logger.ErrorField(err))
}
```

**Step 5: Regenerate Wire code**

Run: `cd internal/di && go run github.com/google/wire/cmd/wire`
Expected: `wire_gen.go` regenerated successfully with the new dependencies.

**Step 6: Verify build**

Run: `go build ./...`
Expected: PASS — all packages compile.

**Step 7: Commit**

```bash
git add internal/di/ cmd/app/app.go
git commit -m "feat(di): wire TaskRegistry, CancelBroadcaster, and CancelHandler; start subscriber on boot"
```

---

### Task 6: Update Cancel() to publish EventTypeCancelled

**Files:**
- Modify: `internal/domain/agent/executor/executor_impl.go`

**Step 1: Update Cancel() method to broadcast a cancelled event before closing subscribers**

The current `Cancel()` just sets status and closes channels. Per the design doc, it should also publish an `EventTypeCancelled` event so SSE/blocking adapters can notify clients.

Change in `executor_impl.go` `Cancel()` method:

```go
func (e *agentExecutorImpl) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = model.ExecutionStatusCancelled
	e.endTime = time.Now()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeCancelled,
		Timestamp: time.Now(),
		Data: map[string]string{
			"task_id": e.taskID.String(),
			"reason":  "user_cancelled",
		},
	})
	e.closeAllSubscribers()
}
```

**Step 2: Verify build**

Run: `go build ./internal/domain/agent/executor/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/domain/agent/executor/executor_impl.go
git commit -m "feat(executor): publish EventTypeCancelled event on cancel"
```

---

### Task 7: Verify Full Build & Manual Test

**Step 1: Full build verification**

Run: `go build ./...`
Expected: PASS

**Step 2: Run any existing tests**

Run: `go test ./...`
Expected: PASS (or skip if no tests exist yet)

**Step 3: Final commit (if any fixups needed)**

```bash
git add -A
git commit -m "fix: resolve any build issues from cancellation implementation"
```
