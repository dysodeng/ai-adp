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

// TextCompletionAgent 文本生成 Agent 适配器（纯技术实现）
type TextCompletionAgent struct {
	adkAgent adk.Agent
	config   *model.Config
}

func NewTextCompletionAgent(
	ctx context.Context,
	config *model.Config,
	modelConfig *modelconfig.ModelConfig,
) (agent.Agent, error) {
	// 使用现有的 engine.NewChatModel 创建模型
	chatModel, err := engine.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
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
	// TODO: 实现执行逻辑，需要了解 ADK Agent 的正确 API
	return fmt.Errorf("not implemented yet")
}

func (a *TextCompletionAgent) GetID() string          { return a.config.AgentID }
func (a *TextCompletionAgent) GetName() string        { return a.config.AgentName }
func (a *TextCompletionAgent) GetDescription() string { return a.config.AgentDescription }
func (a *TextCompletionAgent) GetAppType() valueobject.AppType {
	return valueobject.AppType(a.config.Type)
}
func (a *TextCompletionAgent) GetTools() []tool.Tool                          { return a.config.ToolsConfig.Tools }
func (a *TextCompletionAgent) Validate(executor executor.AgentExecutor) error { return nil }
