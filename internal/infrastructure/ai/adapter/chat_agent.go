package adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

// ChatAgent 对话 Agent 适配器
type ChatAgent struct {
	adkAgent adk.Agent
	config   *model.Config
}

func NewChatAgent(
	ctx context.Context,
	config *model.Config,
	modelConfig *modelconfig.ModelConfig,
) (agent.Agent, error) {
	chatModel, err := engine.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        config.AgentName,
		Description: config.AgentDescription,
		Instruction: config.Prompt.SystemPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("chat_agent: %w", err)
	}

	return &ChatAgent{
		adkAgent: adkAgent,
		config:   config,
	}, nil
}

func (a *ChatAgent) Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	// TODO: 实现执行逻辑
	return fmt.Errorf("not implemented yet")
}

func (a *ChatAgent) GetID() string                                  { return a.config.AgentID }
func (a *ChatAgent) GetName() string                                { return a.config.AgentName }
func (a *ChatAgent) GetDescription() string                         { return a.config.AgentDescription }
func (a *ChatAgent) GetAppType() valueobject.AppType                { return valueobject.AppType(a.config.Type) }
func (a *ChatAgent) GetTools() []tool.Tool                          { return a.config.ToolsConfig.Tools }
func (a *ChatAgent) Validate(executor executor.AgentExecutor) error { return nil }
