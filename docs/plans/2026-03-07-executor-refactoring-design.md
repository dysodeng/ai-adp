# ExecutorFactory 重构设计 — 领域驱动的 Agent 架构

**日期**: 2026-03-07
**状态**: 待批准
**目标**: 将业务逻辑从基础设施层迁移到领域层，实现清晰的职责分离

---

## 1. 背景与问题

### 当前架构问题

```go
// infrastructure/ai/engine/factory.go
func (f *ExecutorFactory) Create(
    ctx context.Context,
    appType valueobject.AppType,
    config *valueobject.AppConfig,  // ❌ 业务配置在基础设施层处理
    chatModel einomodel.ToolCallingChatModel,
    tools []tool.BaseTool,
) (port.AppExecutor, error) {
    switch appType {  // ❌ 业务判断在基础设施层
    case valueobject.AppTypeAgent:
        return NewAgentExecutor(ctx, chatModel, config.SystemPrompt, tools)
    case valueobject.AppTypeChat:
        return NewChatExecutor(ctx, chatModel, config.SystemPrompt)
    case valueobject.AppTypeTextCompletion:
        return NewTextCompletionExecutor(ctx, chatModel, config.SystemPrompt)
    }
}
```

**核心问题**：
1. **业务逻辑泄漏到基础设施层**：`switch appType` 是业务判断，不应在基础设施层
2. **配置处理职责错位**：`SystemPrompt`、`Tools` 等业务配置在基础设施层处理
3. **违反 DDD 依赖方向**：基础设施层依赖领域层的值对象做业务判断
4. **难以测试**：业务逻辑和技术实现耦合，无法独立测试

### 期望架构

**领域层职责**：
- 根据 `App` 和 `AppVersion` 配置构建 Agent
- 处理提示词变量替换
- 加载工具（知识库、插件、MCP、内置工具）
- 根据 `AppType` 决定创建哪种 Agent

**基础设施层职责**：
- 纯技术实现：封装 Eino 组件
- 事件转换：Eino 事件 → 领域事件
- 不做任何业务判断

---

## 2. 设计方案

### 2.1 核心接口

#### Agent 接口（领域层）

```go
// domain/agent/agent.go
package agent

import (
    "context"
    "github.com/dysodeng/ai-adp/internal/domain/agent/executor"
    "github.com/dysodeng/ai-adp/internal/domain/agent/tool"
    "github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// Agent 领域核心接口
// 1. 由 AgentExecutor 管理流式输出
// 2. Execute 方法接收 AgentExecutor，Agent 负责填充事件
// 3. Agent 专注于"生成内容"，不关心"如何传输"
type Agent interface {
    GetID() string
    GetName() string
    GetDescription() string
    GetAppType() model.AppType

    // Execute 执行 Agent，将结果和事件填充到 agentExecutor 中
    // Agent 通过 agentExecutor.PublishChunk()、agentExecutor.PublishToolCall() 等方法发布事件
    // Agent 通过 agentExecutor.Complete() 或 agentExecutor.Fail() 结束执行
    Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error

    GetTools() []tool.Tool

    // Validate 参数验证
    Validate(agentExecutor executor.AgentExecutor) error
}
```

#### AgentExecutor 接口（领域层）

```go
// domain/agent/executor/executor.go
package executor

import (
    "context"
    "time"
    "github.com/google/uuid"
    "github.com/dysodeng/ai-adp/internal/domain/agent/model"
    "github.com/dysodeng/ai-adp/internal/domain/app/model"
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

#### AgentBuilder 接口（领域服务）

```go
// domain/agent/service/agent_builder.go
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
        input map[string]any,
        isStreaming bool,
    ) (agent.Agent, error)
}
```

### 2.2 领域层实现

#### AgentBuilder 实现

```go
// domain/agent/service/agent_builder_impl.go
package service

import (
    "context"
    "fmt"

    "github.com/dysodeng/ai-adp/internal/domain/agent/agent"
    "github.com/dysodeng/ai-adp/internal/domain/agent/agent/impl"
    "github.com/dysodeng/ai-adp/internal/domain/agent/model"
    "github.com/dysodeng/ai-adp/internal/domain/agent/tool"
    appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
    "github.com/dysodeng/ai-adp/internal/domain/app/repository"
    "github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

type agentBuilder struct {
    appRepository   repository.AppRepository
    llmService      port.LLMService
    toolOperator    tool.Operator
    checkpointStore agent.CheckPointStore
}

func NewAgentBuilder(
    appRepository repository.AppRepository,
    llmService port.LLMService,
    toolOperator tool.Operator,
    checkpointStore agent.CheckPointStore,
) AgentBuilder {
    return &agentBuilder{
        appRepository:   appRepository,
        llmService:      llmService,
        toolOperator:    toolOperator,
        checkpointStore: checkpointStore,
    }
}

func (b *agentBuilder) BuildAgent(
    ctx context.Context,
    app *appModel.App,
    input map[string]any,
    isStreaming bool,
) (agent.Agent, error) {
    // 1. 加载模型配置
    modelConfig, err := b.loadModelConfig(ctx, app.ModelInfo.ModelId)
    if err != nil {
        return nil, err
    }

    // 2. 处理提示词（变量替换）
    prompt, err := b.processPrompt(ctx, app.Prompt, input)
    if err != nil {
        return nil, err
    }

    // 3. 加载工具（知识库、插件、MCP、内置工具、App-as-Tool）
    tools, cleanup, err := b.loadTools(ctx, app)
    if err != nil {
        return nil, err
    }

    // 4. 构建 Agent 配置
    config := model.Config{
        AgentID:          app.ID.String(),
        AgentName:        app.Name,
        AgentDescription: app.Description,
        Type:             app.AppType.String(),
        IsStreaming:      isStreaming,
        LLMConfig: &model.LLMConfig{
            Provider:  modelConfig.ProviderType,
            Model:     modelConfig.Name,
            APIKey:    modelConfig.ProviderConfig.ApiKey,
            BaseURL:   modelConfig.ProviderConfig.BaseURL,
            MaxTokens: &modelConfig.ContextLength,
        },
        Prompt: &model.PromptConfig{
            SystemPrompt: prompt,
        },
        ToolsConfig: &model.ToolsConfig{
            Enabled: len(tools) > 0,
            Tools:   tools,
        },
        CustomConfig: map[string]interface{}{
            "mcp_cleanup": cleanup,
        },
    }

    // 5. 根据 AppType 创建对应的 Agent
    return b.createAgentByType(ctx, app.AppType, config)
}

func (b *agentBuilder) createAgentByType(
    ctx context.Context,
    appType appModel.AppType,
    config model.Config,
) (agent.Agent, error) {
    switch appType {
    case appModel.AppTypeReAct:
        config.MaxIterations = 5
        return impl.NewReActAgent(ctx, config, b.checkpointStore)

    case appModel.AppTypeChat:
        if config.ToolsConfig.Enabled {
            // 有工具（知识库）：使用 ReAct Agent
            config.MaxIterations = len(config.ToolsConfig.Tools) + 2
            return impl.NewReActAgent(ctx, config, b.checkpointStore)
        }
        // 无工具：纯对话
        return impl.NewChatAgent(ctx, config)

    case appModel.AppTypeTextCompletion:
        if config.ToolsConfig.Enabled {
            // 有工具（知识库）：使用 ReAct Agent
            config.MaxIterations = len(config.ToolsConfig.Tools) + 2
            return impl.NewReActAgent(ctx, config, b.checkpointStore)
        }
        // 无工具：纯文本生成
        return impl.NewTextCompletionAgent(ctx, config)

    case appModel.AppTypeChatFlow:
        return impl.NewChatFlowAgent(ctx, config)

    default:
        return nil, fmt.Errorf("unsupported app type: %s", appType)
    }
}

// loadModelConfig 加载模型配置
func (b *agentBuilder) loadModelConfig(ctx context.Context, modelID uuid.UUID) (*ModelConfig, error) {
    return b.llmService.ModelConfig(ctx, modelID.String())
}

// processPrompt 处理提示词（变量替换）
func (b *agentBuilder) processPrompt(
    ctx context.Context,
    prompt appModel.Prompt,
    input map[string]any,
) (string, error) {
    var promptVars []*VariableWithValue
    for _, variable := range prompt.Variables {
        var value any
        if input != nil {
            if v, ok := input[variable.Key]; ok {
                value = v
            }
        }
        promptVars = append(promptVars, &VariableWithValue{
            Key:      variable.Key,
            Name:     variable.Name,
            Required: variable.Required,
            DataType: variable.DataType.String(),
            Value:    value,
        })
    }
    return FormatPromptTemplate(ctx, prompt.Content, promptVars)
}

// loadTools 加载所有工具
func (b *agentBuilder) loadTools(
    ctx context.Context,
    app *appModel.App,
) ([]tool.Tool, func(), error) {
    var allTools []tool.Tool
    var cleanupFuncs []func()

    // 1. 知识库工具
    if len(app.KnowledgeList) > 0 {
        knowledgeTools, err := b.toolOperator.GetKnowledgeTools(ctx, app.WorkspaceId.String(), app.KnowledgeList)
        if err != nil {
            return nil, nil, err
        }
        allTools = append(allTools, knowledgeTools...)
    }

    // 2. 插件工具（仅支持工具调用的应用类型）
    if app.IsToolAgent() && len(app.ToolList) > 0 {
        pluginTools, err := b.toolOperator.GetPluginTools(ctx, app.WorkspaceId.String(), app.ToolList)
        if err != nil {
            return nil, nil, err
        }
        allTools = append(allTools, pluginTools...)
    }

    // 3. MCP 工具
    if app.IsToolAgent() && len(app.McpServerList) > 0 {
        mcpTools, cleanup, err := b.toolOperator.GetMCPTools(ctx, app.WorkspaceId.String(), app.McpServerList)
        if err != nil {
            return nil, nil, err
        }
        allTools = append(allTools, mcpTools...)
        if cleanup != nil {
            cleanupFuncs = append(cleanupFuncs, cleanup)
        }
    }

    // 4. 内置工具
    if app.IsToolAgent() && len(app.BuiltinToolList) > 0 {
        builtinTools, err := b.toolOperator.GetBuiltinTools(ctx, app.BuiltinToolList)
        if err != nil {
            return nil, nil, err
        }
        allTools = append(allTools, builtinTools...)
    }

    // 5. App-as-Tool
    if app.IsToolAgent() && len(app.AppToolList) > 0 {
        appTools, err := b.toolOperator.GetAppTools(ctx, app.WorkspaceId.String(), app.AppToolList)
        if err != nil {
            return nil, nil, err
        }
        allTools = append(allTools, appTools...)
    }

    // 合并所有 cleanup 函数
    cleanup := func() {
        for _, fn := range cleanupFuncs {
            if fn != nil {
                fn()
            }
        }
    }

    return allTools, cleanup, nil
}
```

### 2.3 基础设施层实现

#### Eino Adapter（纯技术实现）

```go
// infrastructure/ai/adapter/eino_react_adapter.go
package adapter

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"

    "github.com/dysodeng/ai-adp/internal/domain/agent/agent"
    "github.com/dysodeng/ai-adp/internal/domain/agent/executor"
    "github.com/dysodeng/ai-adp/internal/domain/agent/model"
    "github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
)

// EinoReActAgentAdapter Eino ReAct Agent 适配器
// 纯技术实现：封装 Eino ADK，不做业务判断
type EinoReActAgentAdapter struct {
    adkAgent        *adk.Agent
    config          model.Config
    checkpointStore agent.CheckPointStore
    toolErrCh       chan error
}

// NewEinoReActAgentAdapter 创建 Eino ReAct Agent 适配器
func NewEinoReActAgentAdapter(
    ctx context.Context,
    config model.Config,
    checkpointStore agent.CheckPointStore,
) (agent.Agent, error) {
    // 1. 创建 ChatModel
    llmProtocol := llm.Protocol(config.LLMConfig.Provider)
    llmFactory := llm.NewDefaultFactory()
    if !llmFactory.SupportProtocol(llmProtocol) {
        return nil, fmt.Errorf("unsupported LLM protocol: %s", llmProtocol)
    }

    chatModel, err := llmFactory.CreateChatModel(ctx, llmProtocol, &llm.Config{
        BaseURL:     config.LLMConfig.BaseURL,
        APIKey:      config.LLMConfig.APIKey,
        Model:       config.LLMConfig.Model,
        Temperature: config.LLMConfig.Temperature,
        MaxTokens:   config.LLMConfig.MaxTokens,
        TopP:        config.LLMConfig.TopP,
    })
    if err != nil {
        return nil, err
    }

    // 2. 转换工具
    var tools []tool.BaseTool
    if config.ToolsConfig.Enabled && len(config.ToolsConfig.Tools) > 0 {
        tools = make([]tool.BaseTool, 0, len(config.ToolsConfig.Tools))
        for _, t := range config.ToolsConfig.Tools {
            tools = append(tools, t)
        }
    }

    // 3. 创建工具错误通道
    toolErrCh := make(chan error, 10)

    // 4. 工具错误捕获中间件
    toolErrorCatchMiddleware := compose.ToolMiddleware{
        Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
            return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
                output, err := next(ctx, input)
                if err != nil {
                    select {
                    case toolErrCh <- fmt.Errorf("tool [%s] execution failed: %w", input.Name, err):
                    default:
                    }
                    return &compose.ToolOutput{
                        Result: fmt.Sprintf("Error: %s", err.Error()),
                    }, nil
                }
                return output, nil
            }
        },
    }

    // 5. 创建 ADK Agent
    adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          config.AgentName,
        Description:   config.AgentDescription,
        Instruction:   config.Prompt.SystemPrompt,
        Model:         chatModel,
        MaxIterations: config.MaxIterations,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools:               tools,
                ToolCallMiddlewares: []compose.ToolMiddleware{toolErrorCatchMiddleware},
            },
        },
    })
    if err != nil {
        return nil, err
    }

    return &EinoReActAgentAdapter{
        adkAgent:        adkAgent,
        config:          config,
        checkpointStore: checkpointStore,
        toolErrCh:       toolErrCh,
    }, nil
}

// Execute 执行 Agent（纯技术实现）
func (a *EinoReActAgentAdapter) Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error {
    // 1. 获取输入
    input := agentExecutor.GetInput()

    // 2. 构建 ADK 运行选项
    var opts []adk.AgentRunOption
    if len(input.History) > 0 {
        msgs := make([]adk.Message, 0, len(input.History))
        for _, m := range input.History {
            msgs = append(msgs, adk.Message{Role: m.Role, Content: m.Content})
        }
        opts = append(opts, adk.WithHistoryModifier(func(_ context.Context, _ []adk.Message) []adk.Message {
            return msgs
        }))
    }

    // 3. 执行 ADK Agent
    iter := a.adkAgent.Query(ctx, input.Query, opts...)

    // 4. 转换事件并发布
    for {
        select {
        case toolErr := <-a.toolErrCh:
            // 工具执行失败，立即终止
            agentExecutor.Fail(toolErr)
            return toolErr

        default:
            event, ok := iter.Next()
            if !ok {
                // 迭代结束
                agentExecutor.Complete(&model.ExecutionOutput{
                    Message: &model.Message{
                        Content: model.MessageContent{Content: "completed"},
                    },
                    Usage: &model.TokenUsage{},
                })
                return nil
            }

            if event == nil {
                continue
            }

            // 转换 Eino 事件为领域事件
            a.convertAndPublishEvent(event, agentExecutor)
        }
    }
}

// convertAndPublishEvent 转换并发布事件
func (a *EinoReActAgentAdapter) convertAndPublishEvent(
    event *adk.AgentEvent,
    executor executor.AgentExecutor,
) {
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

// GetID 实现 Agent 接口
func (a *EinoReActAgentAdapter) GetID() string {
    return a.config.AgentID
}

func (a *EinoReActAgentAdapter) GetName() string {
    return a.config.AgentName
}

func (a *EinoReActAgentAdapter) GetDescription() string {
    return a.config.AgentDescription
}

func (a *EinoReActAgentAdapter) GetAppType() appModel.AppType {
    return appModel.AppType(a.config.Type)
}

func (a *EinoReActAgentAdapter) GetTools() []tool.Tool {
    return a.config.ToolsConfig.Tools
}

func (a *EinoReActAgentAdapter) Validate(executor executor.AgentExecutor) error {
    // 参数验证逻辑
    return nil
}
```

### 2.4 调用流程

```
Application Layer (应用服务)
    ↓
AgentBuilder.BuildAgent(app, input, isStreaming)
    ↓ (领域层处理所有业务逻辑)
    ├─ 加载模型配置
    ├─ 处理提示词（变量替换）
    ├─ 加载工具（知识库、插件、MCP、内置、App-as-Tool）
    └─ 根据 AppType 创建 Agent
        ↓
    返回 Agent (EinoReActAgentAdapter/EinoChatAgentAdapter/...)
        ↓
Application Layer 创建 AgentExecutor
        ↓
Application Layer 调用 Agent.Execute(executor)
        ↓
Agent 内部调用 Eino ADK（基础设施层）
        ↓
通过 executor.PublishChunk() 等方法发布事件
        ↓
Application Layer 订阅 executor.Subscribe() 获取事件流
```

---

## 3. 关键优势

### 3.1 清晰的职责分离

**领域层**：
- ✅ 所有业务逻辑集中在 `AgentBuilder`
- ✅ 提示词处理、工具加载、模型配置都在领域层
- ✅ `switch appType` 在领域服务中，符合 DDD 原则

**基础设施层**：
- ✅ 纯技术实现：封装 Eino 组件
- ✅ 事件转换：Eino 事件 → 领域事件
- ✅ 不做任何业务判断

### 3.2 易于测试

```go
// 测试 AgentBuilder（业务逻辑）
func TestAgentBuilder_BuildAgent(t *testing.T) {
    // Mock 依赖
    mockLLMService := &MockLLMService{}
    mockToolOperator := &MockToolOperator{}

    builder := NewAgentBuilder(mockLLMService, mockToolOperator, nil)

    // 测试业务逻辑
    agent, err := builder.BuildAgent(ctx, app, input, true)
    assert.NoError(t, err)
    assert.NotNil(t, agent)
}

// 测试 Eino Adapter（技术实现）
func TestEinoReActAgentAdapter_Execute(t *testing.T) {
    // Mock AgentExecutor
    mockExecutor := &MockAgentExecutor{}

    adapter := NewEinoReActAgentAdapter(ctx, config, nil)

    // 测试技术实现
    err := adapter.Execute(ctx, mockExecutor)
    assert.NoError(t, err)
}
```

### 3.3 易于扩展

**新增应用类型**：
- 只需在 `AgentBuilder.createAgentByType()` 中添加 case
- 基础设施层不需要修改

**新增工具类型**：
- 只需在 `AgentBuilder.loadTools()` 中添加加载逻辑
- 基础设施层不需要修改

### 3.4 符合 DDD 原则

- ✅ 依赖方向正确：基础设施层依赖领域层接口
- ✅ 业务逻辑在领域层：所有配置处理、工具选择在领域服务
- ✅ 技术实现在基础设施层：Eino 组件封装在 adapter

---

## 4. 迁移计划

### 4.1 第一阶段：创建新接口和服务

1. 创建 `domain/agent/agent.go` — Agent 接口
2. 创建 `domain/agent/executor/executor.go` — AgentExecutor 接口
3. 创建 `domain/agent/service/agent_builder.go` — AgentBuilder 接���和实现

### 4.2 第二阶段：实现 Eino Adapter

1. 创建 `infrastructure/ai/adapter/eino_react_adapter.go`
2. 创建 `infrastructure/ai/adapter/eino_chat_adapter.go`
3. 创建 `infrastructure/ai/adapter/eino_text_completion_adapter.go`

### 4.3 第三阶段：重构应用层

1. 修改应用服务，使用 `AgentBuilder` 构建 Agent
2. 创建 `AgentExecutor` 实例
3. 调用 `Agent.Execute(executor)`
4. 订阅 `executor.Subscribe()` 获取事件流

### 4.4 第四阶段：删除旧代码

1. 删除 `infrastructure/ai/engine/factory.go`
2. 删除 `infrastructure/ai/engine/*_executor.go`
3. 更新 DI 配置

---

## 5. 风险与注意事项

### 5.1 兼容性

- 需要确保新架构与现有 API 兼容
- 事件流格式需要保持一致

### 5.2 性能

- `AgentBuilder` 需要加载多种工具，可能影响性能
- 考虑添加缓存机制

### 5.3 错误处理

- 工具加载失败时的降级策略
- Eino 组件异常时的错误传播

---

## 6. 总结

本设计方案通过引入 `Agent` 接口和 `AgentBuilder` 领域服务，将业务逻辑从基础设施层迁移到领域层，实现了清晰的职责分离：

- **领域层**：负责所有业务逻辑（配���处理、工具加载、Agent 构建）
- **基础设施层**：纯技术实现（Eino 组件封装、事件转换）

这种架构符合 DDD 原则，易于测试和扩展，为后续功能迭代奠定了坚实的基础。
