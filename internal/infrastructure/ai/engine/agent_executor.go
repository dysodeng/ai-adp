package engine

import (
	"context"
	"errors"
	"fmt"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// AgentExecutor ReAct Agent 执行器（LLM + Tools 推理循环）
type AgentExecutor struct {
	chatModel    einomodel.ToolCallingChatModel
	systemPrompt string
	tools        []*schema.ToolInfo
}

func NewAgentExecutor(chatModel einomodel.ToolCallingChatModel, systemPrompt string, tools []*schema.ToolInfo) (*AgentExecutor, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("agent_executor: chatModel is required")
	}
	return &AgentExecutor{
		chatModel:    chatModel,
		systemPrompt: systemPrompt,
		tools:        tools,
	}, nil
}

func (e *AgentExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: e.chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("agent_executor: failed to create agent: %w", err)
	}

	msg, err := agent.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &port.AppResult{Content: msg.Content}, nil
}

func (e *AgentExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: e.chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("agent_executor: failed to create agent: %w", err)
	}

	reader, err := agent.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}

	ch := make(chan port.AppEvent, 32)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.AppEvent{Type: port.AppEventDone}
				return
			}
			if err != nil {
				ch <- port.AppEvent{Type: port.AppEventError, Error: err}
				return
			}
			if msg == nil {
				continue
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					ch <- port.AppEvent{
						Type:    port.AppEventToolCall,
						Content: tc.Function.Name,
					}
				}
			} else if msg.Content != "" {
				ch <- port.AppEvent{
					Type:    port.AppEventMessage,
					Content: msg.Content,
				}
			}
		}
	}()
	return ch, nil
}

func (e *AgentExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
	prompt := RenderPrompt(e.systemPrompt, input.Variables)
	messages := make([]*schema.Message, 0, len(input.History)+2)
	if prompt != "" {
		messages = append(messages, schema.SystemMessage(prompt))
	}
	for _, m := range input.History {
		switch m.Role {
		case "system":
			messages = append(messages, schema.SystemMessage(m.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(m.Content, nil))
		default:
			messages = append(messages, schema.UserMessage(m.Content))
		}
	}
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}
