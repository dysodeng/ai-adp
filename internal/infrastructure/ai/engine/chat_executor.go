package engine

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// ChatExecutor 多轮对话执行器（纯对话，无工具/知识库）
type ChatExecutor struct {
	model        einomodel.BaseChatModel
	systemPrompt string
}

func NewChatExecutor(model einomodel.BaseChatModel, systemPrompt string) *ChatExecutor {
	return &ChatExecutor{model: model, systemPrompt: systemPrompt}
}

func (e *ChatExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)
	msg, err := e.model.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &port.AppResult{Content: msg.Content}, nil
}

func (e *ChatExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)
	reader, err := e.model.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	ch := make(chan port.AppEvent, 16)
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
			if msg != nil && msg.Content != "" {
				ch <- port.AppEvent{Type: port.AppEventMessage, Content: msg.Content}
			}
		}
	}()
	return ch, nil
}

func (e *ChatExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
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
