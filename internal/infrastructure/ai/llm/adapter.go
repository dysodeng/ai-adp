package llm

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// Adapter 将 Eino BaseChatModel 适配为 port.LLMExecutor
type Adapter struct {
	model einomodel.BaseChatModel
}

// NewAdapter 创建 LLM 适配器
func NewAdapter(model einomodel.BaseChatModel) *Adapter {
	return &Adapter{model: model}
}

// Execute 非流式 LLM 调用
func (a *Adapter) Execute(ctx context.Context, messages []port.Message) (*port.LLMResponse, error) {
	msg, err := a.model.Generate(ctx, toSchemaMessages(messages))
	if err != nil {
		return nil, err
	}
	return &port.LLMResponse{Content: msg.Content}, nil
}

// Stream 流式 LLM 调用，返回 StreamChunk channel
func (a *Adapter) Stream(ctx context.Context, messages []port.Message) (<-chan port.StreamChunk, error) {
	reader, err := a.model.Stream(ctx, toSchemaMessages(messages))
	if err != nil {
		return nil, err
	}
	ch := make(chan port.StreamChunk, 16)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.StreamChunk{Done: true}
				return
			}
			if err != nil {
				ch <- port.StreamChunk{Error: err, Done: true}
				return
			}
			if msg != nil {
				ch <- port.StreamChunk{Content: msg.Content}
			}
		}
	}()
	return ch, nil
}

// toSchemaMessages 将 port.Message 列表转换为 Eino schema.Message 列表
func toSchemaMessages(messages []port.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system":
			result = append(result, schema.SystemMessage(m.Content))
		case "assistant":
			result = append(result, schema.AssistantMessage(m.Content, nil))
		default: // "user" 及其他
			result = append(result, schema.UserMessage(m.Content))
		}
	}
	return result
}
