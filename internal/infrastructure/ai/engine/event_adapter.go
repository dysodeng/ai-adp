package engine

import (
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// collectResult 从 ADK AsyncIterator 中收集完整的非流式结果
func collectResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (*port.AppResult, error) {
	var content string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.IsStreaming {
			msg, err := schema.ConcatMessageStream(mv.MessageStream)
			if err != nil {
				return nil, err
			}
			if msg != nil {
				content += msg.Content
			}
		} else if mv.Message != nil {
			content += mv.Message.Content
		}
	}
	return &port.AppResult{Content: content}, nil
}

// streamEvents 将 ADK AsyncIterator 转换为 port.AppEvent channel
func streamEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) <-chan port.AppEvent {
	ch := make(chan port.AppEvent, 32)
	go func() {
		defer close(ch)
		for {
			event, ok := iter.Next()
			if !ok {
				ch <- port.AppEvent{Type: port.AppEventDone}
				return
			}
			if event.Err != nil {
				ch <- port.AppEvent{Type: port.AppEventError, Error: event.Err}
				return
			}
			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			mv := event.Output.MessageOutput
			if mv.Role == schema.Tool {
				// 工具调用结果
				if mv.IsStreaming {
					msg, err := schema.ConcatMessageStream(mv.MessageStream)
					if err != nil {
						ch <- port.AppEvent{Type: port.AppEventError, Error: err}
						return
					}
					if msg != nil {
						ch <- port.AppEvent{Type: port.AppEventToolResult, Content: msg.Content}
					}
				} else if mv.Message != nil {
					ch <- port.AppEvent{Type: port.AppEventToolResult, Content: mv.Message.Content}
				}
				continue
			}
			// Assistant 消息
			if mv.IsStreaming {
				// 流式消息逐 chunk 发送
				for {
					msg, err := mv.MessageStream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						ch <- port.AppEvent{Type: port.AppEventError, Error: err}
						return
					}
					if msg != nil {
						if len(msg.ToolCalls) > 0 {
							for _, tc := range msg.ToolCalls {
								ch <- port.AppEvent{Type: port.AppEventToolCall, Content: tc.Function.Name}
							}
						}
						if msg.Content != "" {
							ch <- port.AppEvent{Type: port.AppEventMessage, Content: msg.Content}
						}
					}
				}
			} else if mv.Message != nil {
				if len(mv.Message.ToolCalls) > 0 {
					for _, tc := range mv.Message.ToolCalls {
						ch <- port.AppEvent{Type: port.AppEventToolCall, Content: tc.Function.Name}
					}
				}
				if mv.Message.Content != "" {
					ch <- port.AppEvent{Type: port.AppEventMessage, Content: mv.Message.Content}
				}
			}
		}
	}()
	return ch
}

// prependSystemMessage 在消息列表前插入一条 system 消息
func prependSystemMessage(systemPrompt string, messages []*schema.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages)+1)
	result = append(result, schema.SystemMessage(systemPrompt))
	result = append(result, messages...)
	return result
}

// buildInputMessages 将 port.AppExecutorInput 转换为 Eino Messages（不含 system prompt）
func buildInputMessages(input *port.AppExecutorInput) []*schema.Message {
	messages := make([]*schema.Message, 0, len(input.History)+1)
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
