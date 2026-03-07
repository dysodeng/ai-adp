package engine

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// TextCompletionExecutor 单次文本生成执行器（无对话上下文），基于 ADK ChatModelAgent
type TextCompletionExecutor struct {
	agent        adk.Agent
	systemPrompt string
}

func NewTextCompletionExecutor(ctx context.Context, chatModel einomodel.ToolCallingChatModel, systemPrompt string) (*TextCompletionExecutor, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "text_completion",
		Description: "Single-shot text completion agent",
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("text_gen_executor: %w", err)
	}
	return &TextCompletionExecutor{agent: agent, systemPrompt: systemPrompt}, nil
}

func (e *TextCompletionExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent})
	iter := runner.Run(ctx, messages)
	return collectResult(iter)
}

func (e *TextCompletionExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent, EnableStreaming: true})
	iter := runner.Run(ctx, messages)
	return streamEvents(iter), nil
}

func (e *TextCompletionExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
	prompt := RenderPrompt(e.systemPrompt, input.Variables)
	messages := make([]*schema.Message, 0, 2)
	if prompt != "" {
		messages = append(messages, schema.SystemMessage(prompt))
	}
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}
