package adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

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
		return nil, fmt.Errorf("text_completion_agent: %w", err)
	}

	return &TextCompletionAgent{
		adkAgent: adkAgent,
		config:   config,
	}, nil
}

func (a *TextCompletionAgent) Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error {
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

	return handleStreamingResult(iter, agentExecutor)
}

func (a *TextCompletionAgent) GetID() string          { return a.config.AgentID }
func (a *TextCompletionAgent) GetName() string        { return a.config.AgentName }
func (a *TextCompletionAgent) GetDescription() string { return a.config.AgentDescription }
func (a *TextCompletionAgent) GetAppType() valueobject.AppType {
	return valueobject.AppType(a.config.Type)
}
func (a *TextCompletionAgent) GetTools() []tool.Tool                          { return a.config.ToolsConfig.Tools }
func (a *TextCompletionAgent) Validate(executor executor.AgentExecutor) error { return nil }

// buildInputMessages 将 ExecutionInput 转换为 Eino Messages（不含 system prompt）
func buildInputMessages(input *model.ExecutionInput) []*schema.Message {
	messages := make([]*schema.Message, 0, len(input.History)+1)
	for _, m := range input.History {
		switch m.Role {
		case "system":
			messages = append(messages, schema.SystemMessage(m.Content.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(m.Content.Content, nil))
		default:
			messages = append(messages, schema.UserMessage(m.Content.Content))
		}
	}
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}

// prependSystemMessage 在消息列表前插入一条 system 消息
func prependSystemMessage(systemPrompt string, messages []*schema.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages)+1)
	result = append(result, schema.SystemMessage(systemPrompt))
	result = append(result, messages...)
	return result
}
