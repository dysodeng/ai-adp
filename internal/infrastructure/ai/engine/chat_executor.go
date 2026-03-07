package engine

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// ChatExecutor 多轮对话执行器（纯对话，无工具/知识库），基于 ADK ChatModelAgent
type ChatExecutor struct {
	agent        adk.Agent
	systemPrompt string
}

func NewChatExecutor(ctx context.Context, chatModel einomodel.ToolCallingChatModel, systemPrompt string) (*ChatExecutor, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "chat",
		Description: "Multi-turn conversation agent",
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("chat_executor: %w", err)
	}
	return &ChatExecutor{agent: agent, systemPrompt: systemPrompt}, nil
}

func (e *ChatExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := buildInputMessages(input)
	if rendered := RenderPrompt(e.systemPrompt, input.Variables); rendered != "" {
		messages = prependSystemMessage(rendered, messages)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent})
	iter := runner.Run(ctx, messages)
	return collectResult(iter)
}

func (e *ChatExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := buildInputMessages(input)
	if rendered := RenderPrompt(e.systemPrompt, input.Variables); rendered != "" {
		messages = prependSystemMessage(rendered, messages)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent, EnableStreaming: true})
	iter := runner.Run(ctx, messages)
	return streamEvents(iter), nil
}
