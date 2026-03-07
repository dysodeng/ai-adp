# Chat API 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 参考旧项目架构，在当前项目中实现完整的 Chat API，支持 SSE 流式和 Blocking 阻塞两种响应模式。

**Architecture:** 采用分层架构：HTTP Controller → Application Service（ChatAppService）→ Orchestrator（ExecutorOrchestrator）→ Domain Agent。协议适配器（SSE/Blocking）订阅 AgentExecutor 事件流，将领域事件转换为客户端可消费的格式。整个流程遵循 DDD 原则，领域层不依赖 Eino 框架。

**Tech Stack:** Go 1.25, Gin, Google Wire, Eino ADK, GORM/PostgreSQL, OpenTelemetry

---

## 概述

需要新建/修改的文件清单：

**新建文件（共 10 个）：**
1. `internal/infrastructure/protocol/adapter.go` — 协议适配器接口
2. `internal/infrastructure/protocol/sse.go` — SSE 适配器
3. `internal/infrastructure/protocol/blocking.go` — 阻塞式适配器
4. `internal/application/chat/dto/command.go` — Chat 命令 DTO
5. `internal/application/chat/service/chat_app_service.go` — Chat 应用服务
6. `internal/application/chat/orchestrator/executor_orchestrator.go` — 执行编排器
7. `internal/interfaces/http/handler/chat_handler.go` — Chat HTTP Handler
8. `internal/interfaces/http/dto/request/chat_request.go` — HTTP 请求 DTO
9. `internal/interfaces/http/dto/response/api_response.go` — 统一 API 响应结构
10. `internal/di/chat_module.go` — Chat 模块 DI 配置

**修改文件（共 4 个）：**
1. `internal/infrastructure/server/http.go` — 注册 ChatHandler 路由
2. `internal/di/app.go` — 注入 ChatHandler
3. `internal/di/module.go` — 添加 ChatModuleSet
4. `internal/di/wire_gen.go` — 重新生成

---

### Task 1: 统一 API 响应结构

**Files:**
- Create: `internal/interfaces/http/dto/response/api_response.go`

**Step 1: 创建统一响应结构**

```go
// internal/interfaces/http/dto/response/api_response.go
package response

// Response 统一 API 响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	CodeSuccess             = 0
	CodeFail                = 400
	CodeInternalServerError = 500
)

func Success(data any) *Response {
	return &Response{Code: CodeSuccess, Message: "success", Data: data}
}

func Fail(message string, code int) *Response {
	return &Response{Code: code, Message: message}
}
```

**Step 2: 验证编译**

Run: `go build ./internal/interfaces/http/dto/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/interfaces/http/dto/response/api_response.go
git commit -m "feat(http): add unified API response structure"
```

---

### Task 2: 协议适配器接口与 SSE 实现

**Files:**
- Create: `internal/infrastructure/protocol/adapter.go`
- Create: `internal/infrastructure/protocol/sse.go`
- Create: `internal/infrastructure/protocol/blocking.go`

**Step 1: 创建协议适配器接口**

```go
// internal/infrastructure/protocol/adapter.go
package protocol

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

// Adapter 协议适配器接口
type Adapter interface {
	// HandleExecution 订阅 AgentExecutor 事件流并转发给客户端
	HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error
	// Close 关闭适配器
	Close() error
	// SendError 发送错误事件
	SendError(err error) error
}
```

**Step 2: 创建 SSE 适配器**

```go
// internal/infrastructure/protocol/sse.go
package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// SSEAdapter SSE 协议适配器
type SSEAdapter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	closed  bool
}

func NewSSEAdapter(w http.ResponseWriter) (*SSEAdapter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEAdapter{w: w, flusher: flusher}, nil
}

func (a *SSEAdapter) HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	eventChan := agentExecutor.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			if err := a.sendEvent(event); err != nil {
				return err
			}
		}
	}
}

func (a *SSEAdapter) sendEvent(event *model.Event) error {
	if a.closed {
		return fmt.Errorf("adapter is closed")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}
	sseData := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	if _, err = fmt.Fprint(a.w, sseData); err != nil {
		return fmt.Errorf("write sse data failed: %w", err)
	}
	a.flusher.Flush()
	return nil
}

func (a *SSEAdapter) SendError(err error) error {
	return a.sendEvent(&model.Event{
		Type:      model.EventTypeError,
		Timestamp: time.Now(),
		Data:      map[string]any{"error": err.Error()},
	})
}

func (a *SSEAdapter) Close() error {
	a.closed = true
	return nil
}
```

**Step 3: 创建 Blocking 适配器**

```go
// internal/infrastructure/protocol/blocking.go
package protocol

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// BlockingAdapter 阻塞式响应适配器
type BlockingAdapter struct {
	w http.ResponseWriter
}

func NewBlockingAdapter(w http.ResponseWriter) *BlockingAdapter {
	return &BlockingAdapter{w: w}
}

func (a *BlockingAdapter) HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	// 订阅事件流并等待结束
	eventChan := agentExecutor.Subscribe()
	for range eventChan {
		// 消费所有事件，等待 channel 关闭
	}

	a.w.Header().Set("Content-Type", "application/json")

	if err := agentExecutor.Err(); err != nil {
		resp := response.Fail(err.Error(), response.CodeFail)
		return json.NewEncoder(a.w).Encode(resp)
	}

	output := agentExecutor.GetOutput()
	if output == nil {
		resp := response.Fail("execution completed but no output produced", response.CodeInternalServerError)
		return json.NewEncoder(a.w).Encode(resp)
	}

	result := map[string]any{
		"conversation_id": agentExecutor.GetConversationID(),
		"message_id":      agentExecutor.GetMessageID(),
		"task_id":         agentExecutor.GetTaskID(),
		"status":          agentExecutor.GetStatus(),
		"duration":        agentExecutor.Duration().Milliseconds(),
		"output": map[string]any{
			"content": output.Message.Content.Content,
		},
	}

	resp := response.Success(result)
	return json.NewEncoder(a.w).Encode(resp)
}

func (a *BlockingAdapter) Close() error           { return nil }
func (a *BlockingAdapter) SendError(err error) error { return nil }
```

**Step 4: 验证编译**

Run: `go build ./internal/infrastructure/protocol/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/protocol/
git commit -m "feat(protocol): add SSE and Blocking protocol adapters"
```

---

### Task 3: Chat DTO 命令

**Files:**
- Create: `internal/application/chat/dto/command.go`

**Step 1: 创建 Chat 命令 DTO**

```go
// internal/application/chat/dto/command.go
package dto

// ResponseMode 响应模式
type ResponseMode string

const (
	ResponseModeStreaming ResponseMode = "streaming"
	ResponseModeBlocking ResponseMode = "blocking"
)

func (m ResponseMode) IsValid() bool {
	return m == ResponseModeStreaming || m == ResponseModeBlocking
}

// ChatCommand Chat 对话命令
type ChatCommand struct {
	ConversationID string         `json:"conversation_id"`
	Query          string         `json:"query"`
	Input          map[string]any `json:"input"`
	ResponseMode   ResponseMode   `json:"response_mode"`
}
```

**Step 2: 验证编译**

Run: `go build ./internal/application/chat/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/application/chat/dto/command.go
git commit -m "feat(chat): add ChatCommand DTO with ResponseMode"
```

---

### Task 4: ExecutorOrchestrator 执行编排器

**Files:**
- Create: `internal/application/chat/orchestrator/executor_orchestrator.go`

**Step 1: 创建执行编排器**

```go
// internal/application/chat/orchestrator/executor_orchestrator.go
package orchestrator

import (
	"context"
	"fmt"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/service"
	"github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// ExecutorOrchestrator Agent 执行编排器
type ExecutorOrchestrator interface {
	// Execute 执行 Agent（非阻塞），通过 AgentExecutor 事件流获取结果
	Execute(ctx context.Context, app *model.App, agentExecutor executor.AgentExecutor, isStreaming bool) error
}

type executorOrchestrator struct {
	agentBuilder service.AgentBuilder
	agentFactory *adapter.AgentFactory
}

func NewExecutorOrchestrator(
	agentBuilder service.AgentBuilder,
	agentFactory *adapter.AgentFactory,
) ExecutorOrchestrator {
	return &executorOrchestrator{
		agentBuilder: agentBuilder,
		agentFactory: agentFactory,
	}
}

func (o *executorOrchestrator) Execute(
	ctx context.Context,
	app *model.App,
	agentExecutor executor.AgentExecutor,
	isStreaming bool,
) error {
	// 1. 构建 Agent 配置
	config, err := o.agentBuilder.BuildAgentConfig(ctx, app, agentExecutor.GetInput().Variables, isStreaming)
	if err != nil {
		return fmt.Errorf("build agent config failed: %w", err)
	}

	// 2. 通过工厂创建 Agent
	ag, err := o.agentFactory.CreateAgent(ctx, app.Type(), config, config.AgentID)
	if err != nil {
		return fmt.Errorf("create agent failed: %w", err)
	}

	// 3. 启动执行
	agentExecutor.Start()

	// 4. 异步执行 Agent
	go o.executeAgent(ctx, ag, agentExecutor)

	// 5. 如果是阻塞模式，等待完成
	if !isStreaming {
		eventChan := agentExecutor.AddSubscriber()
		for range eventChan {
			// 消费事件直到 channel 关闭
		}
		if agentExecutor.Err() != nil {
			return agentExecutor.Err()
		}
	}

	return nil
}

func (o *executorOrchestrator) executeAgent(ctx context.Context, ag agent.Agent, agentExecutor executor.AgentExecutor) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("agent execution panic: %v", r)
			logger.Error(ctx, "agent execution panic", logger.ErrorField(err))
			agentExecutor.Fail(err)
		}
	}()

	if err := ag.Execute(ctx, agentExecutor); err != nil {
		logger.Error(ctx, "agent execution failed", logger.ErrorField(err))
		// Agent.Execute 内部已调用 agentExecutor.Fail(err)
		// 这里不重复调用，只记录日志
		return
	}
}
```

**注意：** 此处 `agentFactory.CreateAgent` 的 `modelID` 参数需要从 App 的 AppConfig 中获取。当前 AgentBuilder.BuildAgentConfig 返回的 config.AgentID 是 appID，需要调整。在 Task 5 中会处理模型 ID 的传递问题。

**Step 2: 验证编译**

Run: `go build ./internal/application/chat/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/application/chat/orchestrator/executor_orchestrator.go
git commit -m "feat(chat): add ExecutorOrchestrator for async agent execution"
```

---

### Task 5: ChatAppService 应用服务

**Files:**
- Create: `internal/application/chat/service/chat_app_service.go`

**Step 1: 创建 Chat 应用服务**

```go
// internal/application/chat/service/chat_app_service.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	"github.com/dysodeng/ai-adp/internal/application/chat/orchestrator"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/repository"
)

// ChatAppService Chat 应用服务接口
type ChatAppService interface {
	Chat(ctx context.Context, appID string, cmd chatdto.ChatCommand) (executor.AgentExecutor, error)
}

type chatAppService struct {
	orchestrator  orchestrator.ExecutorOrchestrator
	appRepository repository.AppRepository
}

func NewChatAppService(
	orch orchestrator.ExecutorOrchestrator,
	appRepository repository.AppRepository,
) ChatAppService {
	return &chatAppService{
		orchestrator:  orch,
		appRepository: appRepository,
	}
}

func (svc *chatAppService) Chat(
	ctx context.Context,
	appID string,
	cmd chatdto.ChatCommand,
) (executor.AgentExecutor, error) {
	// 1. 解析 conversation ID
	var conversationID uuid.UUID
	if cmd.ConversationID == "" {
		conversationID, _ = uuid.NewV7()
	} else {
		var err error
		conversationID, err = uuid.Parse(cmd.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("invalid conversation ID: %w", err)
		}
	}

	// 2. 验证 ResponseMode
	if !cmd.ResponseMode.IsValid() {
		return nil, fmt.Errorf("invalid response mode: %s", cmd.ResponseMode)
	}

	// 3. 加载 App
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	app, err := svc.appRepository.FindAppByID(ctx, appUUID)
	if err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	// 4. 构建 ExecutionInput
	isStream := cmd.ResponseMode == chatdto.ResponseModeStreaming
	input := model.ExecutionInput{
		Query:     cmd.Query,
		Variables: cmd.Input,
	}

	// 5. 创建 AgentExecutor
	messageID := uuid.New()
	taskID := uuid.New()
	agentExecutor := executor.NewAgentExecutor(
		ctx,
		taskID,
		appID,
		app.Type(),
		conversationID,
		messageID,
		input,
	)

	// 6. 通过编排器执行
	if err = svc.orchestrator.Execute(ctx, app, agentExecutor, isStream); err != nil {
		return nil, err
	}

	return agentExecutor, nil
}
```

**Step 2: 验证编译**

Run: `go build ./internal/application/chat/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/application/chat/service/chat_app_service.go
git commit -m "feat(chat): add ChatAppService with conversation management"
```

---

### Task 6: HTTP 请求 DTO 和 Chat Handler

**Files:**
- Create: `internal/interfaces/http/dto/request/chat_request.go`
- Create: `internal/interfaces/http/handler/chat_handler.go`

**Step 1: 创建 HTTP 请求 DTO**

```go
// internal/interfaces/http/dto/request/chat_request.go
package request

// ChatRequest Chat 对话请求
type ChatRequest struct {
	ConversationID string         `json:"conversation_id"`
	Query          string         `json:"query" binding:"required"`
	Input          map[string]any `json:"input"`
	ResponseMode   string         `json:"response_mode"`
}
```

**Step 2: 创建 Chat Handler**

```go
// internal/interfaces/http/handler/chat_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/infrastructure/protocol"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/request"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// ChatHandler Chat 对话 Handler
type ChatHandler struct {
	chatService chatservice.ChatAppService
}

func NewChatHandler(chatService chatservice.ChatAppService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// Chat Agent 对话接口，支持 SSE 流式和阻塞式响应
func (h *ChatHandler) Chat(c *gin.Context) {
	var req request.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(err.Error(), response.CodeFail))
		return
	}

	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.Fail("app_id is required", response.CodeFail))
		return
	}

	// 默认 streaming 模式
	if req.ResponseMode == "" {
		req.ResponseMode = "streaming"
	}
	responseMode := chatdto.ResponseMode(req.ResponseMode)
	if !responseMode.IsValid() {
		c.JSON(http.StatusBadRequest, response.Fail("invalid response_mode", response.CodeFail))
		return
	}

	// 创建协议适配器
	var adapter protocol.Adapter
	if responseMode == chatdto.ResponseModeBlocking {
		adapter = protocol.NewBlockingAdapter(c.Writer)
	} else {
		sseAdapter, err := protocol.NewSSEAdapter(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.Fail("streaming not supported", response.CodeInternalServerError))
			return
		}
		adapter = sseAdapter
	}
	defer func() { _ = adapter.Close() }()

	// 调用应用服务
	agentExecutor, err := h.chatService.Chat(
		c.Request.Context(),
		appID,
		chatdto.ChatCommand{
			ConversationID: req.ConversationID,
			Query:          req.Query,
			Input:          req.Input,
			ResponseMode:   responseMode,
		},
	)
	if err != nil {
		if responseMode == chatdto.ResponseModeBlocking {
			c.JSON(http.StatusOK, response.Fail(err.Error(), response.CodeFail))
		} else {
			_ = adapter.SendError(err)
		}
		return
	}

	// 使用适配器订阅并转发事件
	_ = adapter.HandleExecution(c.Request.Context(), agentExecutor)
}

// RegisterRoutes 注册 Chat 路由
func (h *ChatHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/apps/:app_id/chat", h.Chat)
}
```

**Step 3: 验证编译**

Run: `go build ./internal/interfaces/http/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/interfaces/http/dto/request/chat_request.go internal/interfaces/http/handler/chat_handler.go
git commit -m "feat(http): add ChatHandler with SSE/Blocking protocol support"
```

---

### Task 7: DI 接线与路由注册

**Files:**
- Create: `internal/di/chat_module.go`
- Modify: `internal/di/module.go` — 添加 ChatModuleSet
- Modify: `internal/di/app.go` — 注入 ChatHandler
- Modify: `internal/infrastructure/server/http.go` — 注册 ChatHandler 路由
- Modify: `internal/di/wire_gen.go` — 手动更新（网络不可用时）

**Step 1: 创建 Chat 模块 DI**

```go
// internal/di/chat_module.go
package di

import (
	"github.com/google/wire"

	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	chatorch "github.com/dysodeng/ai-adp/internal/application/chat/orchestrator"
)

// ChatModuleSet wires the chat bounded context
var ChatModuleSet = wire.NewSet(
	chatorch.NewExecutorOrchestrator,
	wire.Bind(new(chatorch.ExecutorOrchestrator), new(*chatorch.executorOrchestrator)),
	chatservice.NewChatAppService,
	wire.Bind(new(chatservice.ChatAppService), new(*chatservice.chatAppService)),
)
```

**注意：** 由于 `executorOrchestrator` 和 `chatAppService` 是未导出类型，Wire 的 Bind 无法直接使用。需要让构造函数返回接口类型，Wire 会自动匹配。如果构造函数已经返回接口（`ExecutorOrchestrator`、`ChatAppService`），则不需要 `wire.Bind`。

实际实现应为：
```go
var ChatModuleSet = wire.NewSet(
	chatorch.NewExecutorOrchestrator,
	chatservice.NewChatAppService,
	handler.NewChatHandler,
)
```

**Step 2: 更新 module.go**

在 `ModulesSet` 中添加 `ChatModuleSet`：
```go
var ModulesSet = wire.NewSet(
	TenantModuleSet,
	AppModuleSet,
	ModelModuleSet,
	ChatModuleSet,  // 新增
)
```

**Step 3: 更新 app.go**

HTTPServer 需要接收 ChatHandler。更新 `NewApp` 签名不需要改变（ChatHandler 通过 HTTPServer 注入）。

更新 `NewHTTPServer` 调用，使其接收 ChatHandler：
```go
// internal/infrastructure/server/http.go
type HTTPServer struct {
	cfg           *config.Config
	mu            sync.Mutex
	server        *http.Server
	tenantHandler *handler.TenantHandler
	chatHandler   *handler.ChatHandler  // 新增
}

func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler, chatHandler *handler.ChatHandler) *HTTPServer {
	return &HTTPServer{cfg: cfg, tenantHandler: tenantHandler, chatHandler: chatHandler}
}
```

在 `Start()` 方法中注册路由：
```go
s.tenantHandler.RegisterRoutes(v1)
s.chatHandler.RegisterRoutes(v1)  // 新增
```

**Step 4: 手动更新 wire_gen.go**

在 `InitApp` 函数中添加：
```go
executorOrchestrator := chatorch.NewExecutorOrchestrator(agentBuilder, agentFactory)
chatAppService := chatservice.NewChatAppService(executorOrchestrator, appRepositoryImpl)
chatHandler := handler.NewChatHandler(chatAppService)
httpServer := server.NewHTTPServer(configConfig, tenantHandler, chatHandler)
```

**Step 5: 验证编译**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/di/chat_module.go internal/di/module.go internal/infrastructure/server/http.go internal/di/wire_gen.go
git commit -m "feat(di): wire Chat module with DI and register routes"
```

---

### Task 8: 端到端编译验证与清理

**Step 1: 全量编译**

Run: `go build ./...`
Expected: PASS (no errors)

**Step 2: 运行所有测试**

Run: `go test ./...`
Expected: All existing tests PASS

**Step 3: 运行 go vet**

Run: `go vet ./...`
Expected: PASS

**Step 4: 清理未使用的 import**

检查所有新文件是否有未使用的 import，如有则清理。

**Step 5: 最终 Commit（如有清理）**

```bash
git add -A
git commit -m "chore: cleanup unused imports and fix lint issues"
```

---

## API 接口说明

### POST /api/v1/apps/:app_id/chat

**Request Body:**
```json
{
  "conversation_id": "optional-uuid",
  "query": "你好",
  "input": {"key": "value"},
  "response_mode": "streaming"
}
```

**Streaming Response (SSE):**
```
event: chunk
data: {"type":"chunk","timestamp":"...","data":"你"}

event: chunk
data: {"type":"chunk","timestamp":"...","data":"好"}

event: complete
data: {"type":"complete","timestamp":"...","data":{...}}
```

**Blocking Response (JSON):**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "conversation_id": "uuid",
    "message_id": "uuid",
    "task_id": "uuid",
    "status": "completed",
    "duration": 1234,
    "output": {
      "content": "你好！有什么可以帮你的？"
    }
  }
}
```
