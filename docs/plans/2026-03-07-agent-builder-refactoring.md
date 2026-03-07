# AgentBuilder 重构实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将业务逻辑从基础设施层（ExecutorFactory）迁移到领域层（AgentBuilder），实现清晰的职责分离，符合 DDD 原则。

**Architecture:** 在领域层创建 Agent 接口和 AgentBuilder 服务，负责所有业务逻辑（配置处理、工具加载、Agent 构建）。基础设施层的 Executor 变为纯技术实现（Eino 组件封装、事件转换），不做任何业务判断。

**Tech Stack:** Go 1.25, Eino v0.7.37 (ADK), Google Wire, GORM (PostgreSQL)

---

## Task 1: 创建 Agent 领域接口

**Files:**
- Create: `internal/domain/agent/agent/agent.go`
- Create: `internal/domain/agent/executor/executor.go`
- Create: `internal/domain/agent/model/config.go`
- Create: `internal/domain/agent/model/event.go`

**Step 1: 创建 Agent 接口**

在 `internal/domain/agent/agent/agent.go` 中定义 Agent 接口：

```go
package agent

import (
	"context"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
	appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// Agent 领域核心接口
type Agent interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetAppType() appModel.AppType

	// Execute 执行 Agent，将结果和事件填充到 agentExecutor 中
	Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error

	GetTools() []tool.Tool
	Validate(agentExecutor executor.AgentExecutor) error
}
```

**Step 2: 创建 AgentExecutor 接口**

在 `internal/domain/agent/executor/executor.go` 中定义 AgentExecutor 接口：

```go
package executor

import (
	"context"
	"time"
	"github.com/google/uuid"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// AgentExecutor Agent 执行器 - 封装单次 Agent 执行的完整生命周期
type AgentExecutor interface {
	// ========== 上下文信息 ==========
	Ctx() context.Context
	GetTaskID() uuid.UUID
	GetAppID() string
	GetAppType() appModel.AppType
	GetConversationID() uuid.UUID
	GetMessageID() uuid.UUID
	GetInput() model.ExecutionInput

	// ========== 生命周期管理 ==========
	Start()
	Complete(output *model.ExecutionOutput)
	Fail(err error)
	Cancel()
	Err() error

	// ========== 事件发布 ==========
	PublishChunk(content string)
	PublishThinking(content string)
	PublishToolCall(toolCall *model.ToolCall)
	PublishToolStart(toolCall *model.ToolCall)
	PublishToolResult(toolResult *model.ToolResult)
	PublishToolError(toolCallID, toolName, errMsg string)
	PublishMessage(message *model.Message)
	PublishTokenUsage(usage *model.TokenUsage)

	// ========== 事件订阅 ==========
	Subscribe() <-chan *model.Event
	AddSubscriber() <-chan *model.Event

	// ========== 状态查询 ==========
	GetStatus() model.ExecutionStatus
	IsRunning() bool
	IsCompleted() bool
	Duration() time.Duration
	GetOutput() *model.ExecutionOutput
}
```

**Step 3: 创建 Agent 配置模型**

在 `internal/domain/agent/model/config.go` 中定义配置结构：

```go
package model

import "github.com/dysodeng/ai-adp/internal/domain/agent/tool"

// Config Agent 配置
type Config struct {
	AgentID          string
	AgentName        string
	AgentDescription string
	Type             string
	IsStreaming      bool
	MaxIterations    int

	LLMConfig   *LLMConfig
	Prompt      *PromptConfig
	ToolsConfig *ToolsConfig
	CustomConfig map[string]interface{}
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Provider    string
	Model       string
	APIKey      string
	BaseURL     string
	Temperature *float64
	MaxTokens   *int
	TopP        *float64
}

// PromptConfig 提示词配置
type PromptConfig struct {
	SystemPrompt string
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Enabled bool
	Tools   []tool.Tool
}
```

**Step 4: 创建事件模型**

在 `internal/domain/agent/model/event.go` 中定义事件结构：

```go
package model

import "time"

// EventType 事件类型
type EventType string

const (
	EventTypeChunk       EventType = "chunk"
	EventTypeThinking    EventType = "thinking"
	EventTypeToolCall    EventType = "tool_call"
	EventTypeToolStart   EventType = "tool_start"
	EventTypeToolResult  EventType = "tool_result"
	EventTypeToolError   EventType = "tool_error"
	EventTypeMessage     EventType = "message"
	EventTypeTokenUsage  EventType = "token_usage"
	EventTypeComplete    EventType = "complete"
	EventTypeError       EventType = "error"
)

// Event 领域事件
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// ExecutionInput 执行输入
type ExecutionInput struct {
	Query     string
	Variables map[string]interface{}
	History   []Message
}

// ExecutionOutput 执行输出
type ExecutionOutput struct {
	Message *Message
	Usage   *TokenUsage
}

// Message 消息
type Message struct {
	Role    string
	Content MessageContent
}

// MessageContent 消息内容
type MessageContent struct {
	Content string
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string
	ToolName string
	Input    map[string]interface{}
}

// ToolResult 工具结果
type ToolResult struct {
	ToolCallID string
	ToolName   string
	Output     string
}

// TokenUsage Token 使用量
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)
```

**Step 5: 提交**

```bash
git add internal/domain/agent/agent/agent.go \
        internal/domain/agent/executor/executor.go \
        internal/domain/agent/model/config.go \
        internal/domain/agent/model/event.go
git commit -m "feat(domain/agent): add Agent and AgentExecutor interfaces"
```

---

## Task 2: 扩展 App 模型以支持工具配置

**Files:**
- Modify: `internal/domain/app/model/app.go`

**说明**：App 需要包含工具配置（知识库、插件、MCP、内置工具等），AgentBuilder 将从 App 读取这些配置。

**Step 1: 在 App 结构中添加工具配置字段**

```go
// App AI 应用聚合根
type App struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	description string
	appType     valueobject.AppType
	icon        string

	// 工具配置（由工具领域处理）
	knowledgeList   []uuid.UUID  // 知识库 ID 列表
	toolList        []uuid.UUID  // 插件工具 ID 列表
	mcpServerList   []uuid.UUID  // MCP 服务器 ID 列表
	builtinToolList []uuid.UUID  // 内置工具 ID 列表
	appToolList     []uuid.UUID  // App-as-Tool ID 列表
}
```

**Step 2: 添加 Getter 方法**

```go
func (a *App) KnowledgeList() []uuid.UUID  { return a.knowledgeList }
func (a *App) ToolList() []uuid.UUID       { return a.toolList }
func (a *App) McpServerList() []uuid.UUID  { return a.mcpServerList }
func (a *App) BuiltinToolList() []uuid.UUID { return a.builtinToolList }
func (a *App) AppToolList() []uuid.UUID    { return a.appToolList }

// IsToolAgent 判断是否支持工具调用
func (a *App) IsToolAgent() bool {
	return a.appType == valueobject.AppTypeAgent
}
```

**Step 3: 更新构造函数和 Reconstitute**

暂时将工具配置字段初始化为空切片，后续实现工具管理时再完善。

**Step 4: 提交**

```bash
git add internal/domain/app/model/app.go
git commit -m "feat(domain/app): add tool configuration fields to App model"
```

---

## Task 3: 定义领域层 Tool 接口

**Files:**
- Create: `internal/domain/agent/tool/tool.go`

**说明**：在领域层定义 Tool 接口，不依赖任何基础设施技术（如 Eino）。

**Step 1: 创建领域 Tool 接口**

在 `internal/domain/agent/tool/tool.go` 中定义：

```go
package tool

import "context"

// Tool 领域层工具接口（不依赖 Eino）
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// InputSchema 输入参数 Schema（JSON Schema）
	InputSchema() map[string]interface{}

	// Invoke 执行工具
	Invoke(ctx context.Context, input map[string]interface{}) (string, error)
}
```

**Step 2: 提交**

```bash
git add internal/domain/agent/tool/tool.go
git commit -m "feat(domain/agent): add domain Tool interface"
```

---

## Task 4: 定义工具服务接口（待工具领域实现）

**Files:**
- Create: `internal/domain/shared/port/tool_service.go`

**说明**：定义工具服务接口，返回领域层的 Tool 接口，不依赖 Eino。

**Step 1: 创建 ToolService 接口**

在 `internal/domain/shared/port/tool_service.go` 中定义：

```go
package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
)

// ToolService 工具服务接口
// 负责将 App 的工具配置转换为领域层的 Tool 实例
type ToolService interface {
	// LoadTools 根据 App 的工具配置加载所有工具
	// 返回工具列表和清理函数（用于 MCP 连接清理）
	LoadTools(ctx context.Context, config *ToolLoadConfig) ([]tool.Tool, func(), error)
}

// ToolLoadConfig 工具加载配置
type ToolLoadConfig struct {
	TenantID        uuid.UUID  // 租户 ID
	KnowledgeList   []uuid.UUID
	ToolList        []uuid.UUID
	McpServerList   []uuid.UUID
	BuiltinToolList []uuid.UUID
	AppToolList     []uuid.UUID
	IsToolAgent     bool  // 是否支持工具调用
}
```

**Step 2: 创建 Mock 实现（临时）**

在同一文件中添加 mock 实现：

```go
// MockToolService 临时 mock 实现，工具领域实现后删除
type MockToolService struct{}

func NewMockToolService() ToolService {
	return &MockToolService{}
}

func (m *MockToolService) LoadTools(ctx context.Context, config *ToolLoadConfig) ([]tool.Tool, func(), error) {
	// 暂时返回空工具列表
	return []tool.Tool{}, func() {}, nil
}
```

**Step 3: 提交**

```bash
git add internal/domain/shared/port/tool_service.go
git commit -m "feat(port): add ToolService interface with mock implementation"
```

---

## Task 4: 创建 AgentBuilder 领域服务接口

**Files:**
- Create: `internal/domain/agent/service/agent_builder.go`

**Step 1: 创建 AgentBuilder 接口**

在 `internal/domain/agent/service/agent_builder.go` 中定义 AgentBuilder 接口：

```go
package service

import (
	"context"
	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// AgentBuilder Agent 构建器 - 领域服务
// 负责根据 App 配置构建完整的 Agent
type AgentBuilder interface {
	// BuildAgent 根据 App 和输入构建 Agent
	// 处理所有业务逻辑：提示词、工具、模型配置
	BuildAgent(
		ctx context.Context,
		app *model.App,
		input map[string]interface{},
		isStreaming bool,
	) (agent.Agent, error)
}
```

**Step 2: 提交**

```bash
git add internal/domain/agent/service/agent_builder.go
git commit -m "feat(domain/agent): add AgentBuilder service interface"
```

---

## Task 5: 实现 AgentBuilder 领域服务

**Files:**
- Create: `internal/domain/agent/service/agent_builder_impl.go`

**说明**：实现 AgentBuilder，负责所有业务逻辑：根据 AppType 决定创建哪种 Agent，加载工具，处理配置。

**Step 1: 创建 AgentBuilder 实现**

在 `internal/domain/agent/service/agent_builder_impl.go` 中：

```go
package service

import (
	"context"
	"fmt"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

type agentBuilderImpl struct {
	toolService port.ToolService
	// 后续添加：llmService, promptService 等
}

func NewAgentBuilder(toolService port.ToolService) AgentBuilder {
	return &agentBuilderImpl{
		toolService: toolService,
	}
}

func (b *agentBuilderImpl) BuildAgent(
	ctx context.Context,
	app *appModel.App,
	input map[string]interface{},
	isStreaming bool,
) (agent.Agent, error) {
	// 1. 加载工具
	tools, cleanup, err := b.loadTools(ctx, app)
	if err != nil {
		return nil, fmt.Errorf("failed to load tools: %w", err)
	}

	// 2. 构建 Agent 配置
	config := model.Config{
		AgentID:          app.ID().String(),
		AgentName:        app.Name(),
		AgentDescription: app.Description(),
		Type:             app.Type().String(),
		IsStreaming:      isStreaming,
		ToolsConfig: &model.ToolsConfig{
			Enabled: len(tools) > 0,
			Tools:   tools,
		},
		CustomConfig: map[string]interface{}{
			"cleanup": cleanup,
		},
	}

	// 3. 根据 AppType 创建对应的 Agent（业务判断在领域层）
	return b.createAgentByType(ctx, app.Type(), config)
}

func (b *agentBuilderImpl) loadTools(
	ctx context.Context,
	app *appModel.App,
) ([]tool.BaseTool, func(), error) {
	toolConfig := &port.ToolLoadConfig{
		TenantID:        app.TenantID(),
		KnowledgeList:   app.KnowledgeList(),
		ToolList:        app.ToolList(),
		McpServerList:   app.McpServerList(),
		BuiltinToolList: app.BuiltinToolList(),
		AppToolList:     app.AppToolList(),
		IsToolAgent:     app.IsToolAgent(),
	}

	return b.toolService.LoadTools(ctx, toolConfig)
}

func (b *agentBuilderImpl) createAgentByType(
	ctx context.Context,
	appType appModel.AppType,
	config model.Config,
) (agent.Agent, error) {
	// 业务判断：根据 AppType 决定创建哪种 Agent
	switch appType {
	case appModel.AppTypeTextCompletion:
		// 创建文本生成 Agent（基础设施层实现）
		return NewTextCompletionAgent(ctx, config)

	case appModel.AppTypeChat:
		// 创建对话 Agent（基础设施层实现）
		return NewChatAgent(ctx, config)

	case appModel.AppTypeAgent:
		// 创建 ReAct Agent（基础设施层实现）
		return NewReActAgent(ctx, config)

	default:
		return nil, fmt.Errorf("unsupported app type: %s", appType)
	}
}
```

**Step 2: 提交**

```bash
git add internal/domain/agent/service/agent_builder_impl.go
git commit -m "feat(domain/agent): implement AgentBuilder service"
```

---

## Task 6: 创建基础设施层 Eino Adapter

**Files:**
- Create: `internal/infrastructure/ai/adapter/text_completion_agent.go`
- Create: `internal/infrastructure/ai/adapter/chat_agent.go`
- Create: `internal/infrastructure/ai/adapter/react_agent.go`
- Create: `internal/infrastructure/ai/adapter/event_converter.go`

**说明**：基础设施层的 Agent 实现，纯技术实现，封装 Eino ADK，不做业务判断。

**Step 1: 创建 TextCompletionAgent 适配器**

在 `internal/infrastructure/ai/adapter/text_completion_agent.go` 中：

```go
package adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
)

// TextCompletionAgent 文本生成 Agent 适配器（纯技术实现）
type TextCompletionAgent struct {
	adkAgent adk.Agent
	config   model.Config
}

func NewTextCompletionAgent(ctx context.Context, config model.Config) (agent.Agent, error) {
	// 创建 ChatModel（技术实现）
	chatModel, err := createChatModel(ctx, config.LLMConfig)
	if err != nil {
		return nil, err
	}

	// 创建 ADK Agent（不配置工具）
	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        config.AgentName,
		Description: config.AgentDescription,
		Instruction: config.Prompt.SystemPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("text_completion_agent: %w", err)
	}

	return &TextCompletionAgent{
		adkAgent: adkAgent,
		config:   config,
	}, nil
}

func (a *TextCompletionAgent) Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	// 执行 ADK Agent 并转换事件
	input := agentExecutor.GetInput()
	iter := a.adkAgent.Query(ctx, input.Query)

	for {
		event, ok := iter.Next()
		if !ok {
			agentExecutor.Complete(&model.ExecutionOutput{})
			return nil
		}
		if event != nil {
			convertAndPublishEvent(event, agentExecutor)
		}
	}
}

func (a *TextCompletionAgent) GetID() string          { return a.config.AgentID }
func (a *TextCompletionAgent) GetName() string        { return a.config.AgentName }
func (a *TextCompletionAgent) GetDescription() string { return a.config.AgentDescription }
func (a *TextCompletionAgent) GetAppType() appModel.AppType {
	return appModel.AppType(a.config.Type)
}
func (a *TextCompletionAgent) GetTools() []tool.Tool { return a.config.ToolsConfig.Tools }
func (a *TextCompletionAgent) Validate(executor executor.AgentExecutor) error {
	return nil
}
```

**Step 2: 创建 Tool 适配器（领域 Tool → Eino BaseTool）**

在 `internal/infrastructure/ai/adapter/tool_adapter.go` 中：

```go
package adapter

import (
	"context"
	"github.com/cloudwego/eino/components/tool"
	domainTool "github.com/dysodeng/ai-adp/internal/domain/agent/tool"
)

// ToolAdapter 将领域层 Tool 适配为 Eino BaseTool
type ToolAdapter struct {
	domainTool domainTool.Tool
}

func NewToolAdapter(t domainTool.Tool) tool.BaseTool {
	return &ToolAdapter{domainTool: t}
}

func (a *ToolAdapter) Info(ctx context.Context) (*tool.Info, error) {
	return &tool.Info{
		Name:        a.domainTool.Name(),
		Desc:        a.domainTool.Description(),
		ParamsOneOf: a.domainTool.InputSchema(),
	}, nil
}

func (a *ToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 将 JSON 字符串解析为 map
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	return a.domainTool.Invoke(ctx, input)
}

// ConvertDomainToolsToEino 批量转换领域 Tool 为 Eino BaseTool
func ConvertDomainToolsToEino(domainTools []domainTool.Tool) []tool.BaseTool {
	einoTools := make([]tool.BaseTool, 0, len(domainTools))
	for _, t := range domainTools {
		einoTools = append(einoTools, NewToolAdapter(t))
	}
	return einoTools
}
```

**Step 3: 创建 ChatAgent 和 ReActAgent 适配器**

类似地创建 `chat_agent.go` 和 `react_agent.go`。

ReActAgent 示例（需要配置工具）：

```go
func NewReActAgent(ctx context.Context, config model.Config) (agent.Agent, error) {
	chatModel, err := createChatModel(ctx, config.LLMConfig)
	if err != nil {
		return nil, err
	}

	// 将领域 Tool 转换为 Eino BaseTool
	einoTools := ConvertDomainToolsToEino(config.ToolsConfig.Tools)

	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        config.AgentName,
		Description: config.AgentDescription,
		Instruction: config.Prompt.SystemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: einoTools,  // 使用转换后的 Eino 工具
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("react_agent: %w", err)
	}

	return &ReActAgent{
		adkAgent: adkAgent,
		config:   config,
	}, nil
}
```

**Step 4: 创建事件转换器**

在 `internal/infrastructure/ai/adapter/event_converter.go` 中：

```go
package adapter

import (
	"github.com/cloudwego/eino/adk"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// convertAndPublishEvent 转换 Eino 事件为领域事件并发布
func convertAndPublishEvent(event *adk.AgentEvent, executor executor.AgentExecutor) {
	if event.Err != nil {
		executor.PublishToolError("", "", event.Err.Error())
		return
	}

	if event.Output != nil {
		msg := event.Output.Message()
		if msg != nil && msg.Content != "" {
			executor.PublishChunk(msg.Content)
		}
	}

	if event.Action != nil {
		executor.PublishToolCall(&model.ToolCall{
			ToolName: event.Action.String(),
		})
	}
}

// createChatModel 创建 ChatModel（技术实现）
func createChatModel(ctx context.Context, config *model.LLMConfig) (einomodel.ToolCallingChatModel, error) {
	// 使用现有的 llm factory
	llmProtocol := llm.Protocol(config.Provider)
	llmFactory := llm.NewDefaultFactory()

	return llmFactory.CreateChatModel(ctx, llmProtocol, &llm.Config{
		BaseURL:     config.BaseURL,
		APIKey:      config.APIKey,
		Model:       config.Model,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		TopP:        config.TopP,
	})
}
```

**Step 4: 提交**

```bash
git add internal/infrastructure/ai/adapter/
git commit -m "feat(infrastructure/ai): add Eino Agent adapters"
```

---

## Task 7: 更新 DI 配置

**Files:**
- Modify: `cmd/api/wire.go`
- Modify: `cmd/api/wire_gen.go`（运行 wire 后自动生成）

**Step 1: 在 wire.go 中添加 AgentBuilder**

```go
// 添加 AgentBuilder 到 provider set
var agentSet = wire.NewSet(
	port.NewMockToolService,
	service.NewAgentBuilder,
)

// 在 InitializeApp 中注入
func InitializeApp() (*App, error) {
	wire.Build(
		// ... 其他 sets
		agentSet,
		// ...
	)
	return &App{}, nil
}
```

**Step 2: 运行 wire 生成代码**

```bash
cd cmd/api && wire
```

**Step 3: 提交**

```bash
git add cmd/api/wire.go cmd/api/wire_gen.go
git commit -m "feat(di): wire AgentBuilder into DI container"
```

---

## Task 8: 删除旧的 ExecutorFactory

**Files:**
- Delete: `internal/infrastructure/ai/engine/factory.go`
- Delete: `internal/infrastructure/ai/engine/text_completion_executor.go`
- Delete: `internal/infrastructure/ai/engine/chat_executor.go`
- Delete: `internal/infrastructure/ai/engine/agent_executor.go`
- Delete: `internal/infrastructure/ai/engine/helper.go`（如果存在）

**Step 1: 删除旧文件**

```bash
rm internal/infrastructure/ai/engine/factory.go
rm internal/infrastructure/ai/engine/text_completion_executor.go
rm internal/infrastructure/ai/engine/chat_executor.go
rm internal/infrastructure/ai/engine/agent_executor.go
```

**Step 2: 提交**

```bash
git add -A
git commit -m "refactor(infrastructure/ai): remove old ExecutorFactory and executors"
```

---

## Task 9: 更新应用层使用 AgentBuilder

**Files:**
- Modify: 应用服务层文件（具体路径待确认）

**说明**：将应用层从使用 ExecutorFactory 改为使用 AgentBuilder。

**Step 1: 查找使用 ExecutorFactory 的地方**

```bash
grep -r "ExecutorFactory" internal/application/
```

**Step 2: 替换为 AgentBuilder**

旧代码：
```go
executor, err := executorFactory.Create(ctx, appType, config, chatModel, tools)
result, err := executor.Execute(ctx, input)
```

新代码：
```go
agent, err := agentBuilder.BuildAgent(ctx, app, input, false)
agentExecutor := executor.NewAgentExecutor(ctx, app, input)
err = agent.Execute(ctx, agentExecutor)
result := agentExecutor.GetOutput()
```

**Step 3: 提交**

```bash
git add internal/application/
git commit -m "refactor(application): use AgentBuilder instead of ExecutorFactory"
```

---

## 执行选择

计划已完成并保存到 `docs/plans/2026-03-07-agent-builder-refactoring.md`。

**两种执行方式：**

**1. Subagent-Driven（本会话）** - 我在本会话中逐任务派发子 agent，每个任务完成后进行代码审查，快速迭代

**2. Parallel Session（独立会话）** - 在新会话中使用 executing-plans 技能，批量执行并设置检查点

你选择哪种方式？

