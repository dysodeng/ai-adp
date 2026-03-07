package adapter

import (
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// handleStreamingResult 处理 ADK 流式结果，将事件发布到 AgentExecutor
func handleStreamingResult(
	iter *adk.AsyncIterator[*adk.AgentEvent],
	agentExecutor executor.AgentExecutor,
) error {
	var fullContent string
	for {
		event, ok := iter.Next()
		if !ok {
			agentExecutor.Complete(&model.ExecutionOutput{
				Message: &model.Message{
					Role:    "assistant",
					Content: model.MessageContent{Content: fullContent},
				},
			})
			return nil
		}
		if event.Err != nil {
			agentExecutor.Fail(event.Err)
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.IsStreaming {
			for {
				msg, err := mv.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					agentExecutor.Fail(err)
					return err
				}
				if msg != nil && msg.Content != "" {
					fullContent += msg.Content
					agentExecutor.PublishChunk(msg.Content)
				}
			}
		} else if mv.Message != nil && mv.Message.Content != "" {
			fullContent += mv.Message.Content
			agentExecutor.PublishChunk(mv.Message.Content)
		}
	}
}

// handleStreamingResultWithTools 处理 ADK 流式结果（含工具调用），将事件发布到 AgentExecutor
func handleStreamingResultWithTools(
	iter *adk.AsyncIterator[*adk.AgentEvent],
	agentExecutor executor.AgentExecutor,
) error {
	var fullContent string
	for {
		event, ok := iter.Next()
		if !ok {
			agentExecutor.Complete(&model.ExecutionOutput{
				Message: &model.Message{
					Role:    "assistant",
					Content: model.MessageContent{Content: fullContent},
				},
			})
			return nil
		}
		if event.Err != nil {
			agentExecutor.Fail(event.Err)
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// 处理工具调用结果
		if mv.Role == schema.Tool {
			if mv.IsStreaming {
				msg, err := schema.ConcatMessageStream(mv.MessageStream)
				if err != nil {
					agentExecutor.Fail(err)
					return err
				}
				if msg != nil {
					agentExecutor.PublishToolResult(&model.ToolResult{
						Output: msg.Content,
					})
				}
			} else if mv.Message != nil {
				agentExecutor.PublishToolResult(&model.ToolResult{
					Output: mv.Message.Content,
				})
			}
			continue
		}

		// 处理 Assistant 消息
		if mv.IsStreaming {
			for {
				msg, err := mv.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					agentExecutor.Fail(err)
					return err
				}
				if msg != nil {
					if len(msg.ToolCalls) > 0 {
						for _, tc := range msg.ToolCalls {
							agentExecutor.PublishToolCall(&model.ToolCall{
								ID:       tc.ID,
								ToolName: tc.Function.Name,
							})
						}
					}
					if msg.Content != "" {
						fullContent += msg.Content
						agentExecutor.PublishChunk(msg.Content)
					}
				}
			}
		} else if mv.Message != nil {
			if len(mv.Message.ToolCalls) > 0 {
				for _, tc := range mv.Message.ToolCalls {
					agentExecutor.PublishToolCall(&model.ToolCall{
						ID:       tc.ID,
						ToolName: tc.Function.Name,
					})
				}
			}
			if mv.Message.Content != "" {
				fullContent += mv.Message.Content
				agentExecutor.PublishChunk(mv.Message.Content)
			}
		}
	}
}
