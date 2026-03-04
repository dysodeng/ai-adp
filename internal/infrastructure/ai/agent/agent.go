package agent

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// compile-time interface check
var _ port.AgentExecutor = (*AgentExecutorImpl)(nil)

// AgentExecutorImpl 使用 Eino ReAct Agent 实现 port.AgentExecutor
type AgentExecutorImpl struct {
	chatModel einomodel.ToolCallingChatModel
}

// NewAgentExecutor 创建 AgentExecutor；chatModel 不能为 nil
func NewAgentExecutor(chatModel einomodel.ToolCallingChatModel) (*AgentExecutorImpl, error) {
	if chatModel == nil {
		return nil, errors.New("agent: chatModel is required")
	}
	return &AgentExecutorImpl{chatModel: chatModel}, nil
}

// Run 执行 ReAct Agent，以流式方式将输出写入 AgentEvent channel。
// 调用方需消费 channel 直到收到 AgentEventTypeDone 或 AgentEventTypeError。
func (a *AgentExecutorImpl) Run(ctx context.Context, query string, history []port.Message) (<-chan port.AgentEvent, error) {
	// 构建输入消息列表：历史记录 + 当前查询
	input := toSchemaMessages(history)
	input = append(input, schema.UserMessage(query))

	// 构造 ReAct Agent（无工具；若需要工具可通过 WithToolList 选项注入）
	agentInstance, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: a.chatModel,
	})
	if err != nil {
		return nil, err
	}

	// 流式调用 Agent
	reader, err := agentInstance.Stream(ctx, input)
	if err != nil {
		return nil, err
	}

	ch := make(chan port.AgentEvent, 32)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, recvErr := reader.Recv()
			if errors.Is(recvErr, io.EOF) {
				ch <- port.AgentEvent{Type: port.AgentEventTypeDone}
				return
			}
			if recvErr != nil {
				ch <- port.AgentEvent{Type: port.AgentEventTypeError, Error: recvErr}
				return
			}
			if msg == nil {
				continue
			}
			// 判断事件类型：有 ToolCalls 则为工具调用，否则为普通消息片段
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					ch <- port.AgentEvent{
						Type:    port.AgentEventTypeToolCall,
						Content: tc.Function.Name,
					}
				}
			} else if msg.Content != "" {
				ch <- port.AgentEvent{
					Type:    port.AgentEventTypeMessage,
					Content: msg.Content,
				}
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
