package adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// ReActAgent ReAct Agent 适配器（支持工具调用）
type ReActAgent struct {
	adkAgent adk.Agent
	config   *model.Config
}

func NewReActAgent(
	ctx context.Context,
	config *model.Config,
	modelConfig *modelconfig.ModelConfig,
) (agent.Agent, error) {
	chatModel, err := NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	// 转换领域 Tool 为 Eino BaseTool
	einoTools := ConvertDomainToolsToEino(config.ToolsConfig.Tools)

	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        config.AgentName,
		Description: config.AgentDescription,
		Instruction: config.Prompt.SystemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: einoTools,
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

func (a *ReActAgent) Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	input := agentExecutor.GetInput()

	messages := buildInputMessages(&input)
	if a.config.Prompt.SystemPrompt != "" {
		messages = prependSystemMessage(a.config.Prompt.SystemPrompt, messages)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a.adkAgent,
		EnableStreaming: true,
	})
	iter := runner.Run(ctx, messages)

	return handleStreamingResultWithTools(iter, agentExecutor)
}

func (a *ReActAgent) GetID() string                                  { return a.config.AgentID }
func (a *ReActAgent) GetName() string                                { return a.config.AgentName }
func (a *ReActAgent) GetDescription() string                         { return a.config.AgentDescription }
func (a *ReActAgent) GetAppType() valueobject.AppType                { return valueobject.AppType(a.config.Type) }
func (a *ReActAgent) GetTools() []tool.Tool                          { return a.config.ToolsConfig.Tools }
func (a *ReActAgent) Validate(executor executor.AgentExecutor) error { return nil }
