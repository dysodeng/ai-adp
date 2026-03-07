package protocol

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// BlockingAdapter 阻塞式响应适配器
// 职责：等待 AgentExecutor 执行完成，将最终结果以 JSON 格式返回
type BlockingAdapter struct {
	w http.ResponseWriter
}

// NewBlockingAdapter 创建阻塞式适配器
func NewBlockingAdapter(w http.ResponseWriter) *BlockingAdapter {
	return &BlockingAdapter{w: w}
}

// HandleExecution 阻塞等待执行结果
func (a *BlockingAdapter) HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	// 订阅事件流并等待结束
	eventChan := agentExecutor.Subscribe()
	for range eventChan {
		// 消费所有事件，等待 channel 关闭
	}

	a.w.Header().Set("Content-Type", "application/json")

	if err := agentExecutor.Err(); err != nil {
		resp := response.Fail(ctx, err.Error(), response.CodeFail)
		return json.NewEncoder(a.w).Encode(resp)
	}

	output := agentExecutor.GetOutput()
	if output == nil {
		resp := response.Fail(ctx, "execution completed but no output produced", response.CodeInternalServerError)
		return json.NewEncoder(a.w).Encode(resp)
	}

	result := map[string]any{
		"conversation_id": agentExecutor.GetConversationID(),
		"message_id":      agentExecutor.GetMessageID(),
		"task_id":         agentExecutor.GetTaskID(),
		"status":          agentExecutor.GetStatus(),
		"duration":        agentExecutor.Duration().Milliseconds(),
		"output": map[string]any{
			"content": output.Message.Content.Content,
		},
	}

	resp := response.Success(ctx, result)
	return json.NewEncoder(a.w).Encode(resp)
}

// Close 关闭适配器（无操作）
func (a *BlockingAdapter) Close() error { return nil }

// SendError 发送错误（无操作，HandleExecution 内部处理）
func (a *BlockingAdapter) SendError(_ error) error { return nil }
