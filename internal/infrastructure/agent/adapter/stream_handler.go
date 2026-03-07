package adapter

import (
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// extractUsage 从 schema.Message 中提取 token usage 信息
func extractUsage(msg *schema.Message) *model.TokenUsage {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return nil
	}
	u := msg.ResponseMeta.Usage
	return &model.TokenUsage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
}

// handleStreamingResult 处理 ADK 流式结果，将事件发布到 AgentExecutor
func handleStreamingResult(
	iter *adk.AsyncIterator[*adk.AgentEvent],
	agentExecutor executor.AgentExecutor,
) error {
	var fullContent string
	var usage *model.TokenUsage
	for {
		event, ok := iter.Next()
		if !ok {
			// 发送 token_usage 事件
			if usage != nil {
				agentExecutor.PublishTokenUsage(usage)
			}
			agentExecutor.Complete(&model.ExecutionOutput{
				Message: &model.Message{
					Role:    "assistant",
					Content: model.MessageContent{Content: fullContent},
				},
				Usage: usage,
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
				if msg != nil {
					if msg.Content != "" {
						fullContent += msg.Content
						agentExecutor.PublishChunk(msg.Content)
					}
					if u := extractUsage(msg); u != nil {
						usage = u
					}
				}
			}
		} else if mv.Message != nil {
			if mv.Message.Content != "" {
				fullContent += mv.Message.Content
				agentExecutor.PublishChunk(mv.Message.Content)
			}
			if u := extractUsage(mv.Message); u != nil {
				usage = u
			}
		}
	}
}

// handleStreamingResultWithTools 处理 ADK 流式结果（含工具调用），将事件发布到 AgentExecutor
func handleStreamingResultWithTools(
	iter *adk.AsyncIterator[*adk.AgentEvent],
	agentExecutor executor.AgentExecutor,
) error {
	var fullContent string
	var usage *model.TokenUsage
	for {
		event, ok := iter.Next()
		if !ok {
			// 发送 token_usage 事件
			if usage != nil {
				agentExecutor.PublishTokenUsage(usage)
			}
			agentExecutor.Complete(&model.ExecutionOutput{
				Message: &model.Message{
					Role:    "assistant",
					Content: model.MessageContent{Content: fullContent},
				},
				Usage: usage,
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
					if u := extractUsage(msg); u != nil {
						usage = u
					}
				}
			} else if mv.Message != nil {
				agentExecutor.PublishToolResult(&model.ToolResult{
					Output: mv.Message.Content,
				})
				if u := extractUsage(mv.Message); u != nil {
					usage = u
				}
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
					if u := extractUsage(msg); u != nil {
						usage = u
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
			if u := extractUsage(mv.Message); u != nil {
				usage = u
			}
		}
	}
}
