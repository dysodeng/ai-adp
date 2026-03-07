package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// SSEAdapter SSE 协议适配器
// 职责：订阅 AgentExecutor 的事件流，转换为 SSE 格式输出
type SSEAdapter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	closed  bool
}

// NewSSEAdapter 创建 SSE 适配器
func NewSSEAdapter(w http.ResponseWriter) (*SSEAdapter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEAdapter{w: w, flusher: flusher}, nil
}

// HandleExecution 订阅事件流并转换为 SSE 格式发送
func (a *SSEAdapter) HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	eventChan := agentExecutor.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			if err := a.sendEvent(event); err != nil {
				return err
			}
		}
	}
}

func (a *SSEAdapter) sendEvent(event *model.Event) error {
	if a.closed {
		return fmt.Errorf("adapter is closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	sseData := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	if _, err = fmt.Fprint(a.w, sseData); err != nil {
		return fmt.Errorf("write sse data failed: %w", err)
	}

	a.flusher.Flush()
	return nil
}

// SendError 发送错误事件
func (a *SSEAdapter) SendError(err error) error {
	return a.sendEvent(&model.Event{
		Type:      model.EventTypeError,
		Timestamp: time.Now(),
		Data:      map[string]any{"error": err.Error()},
	})
}

// Close 关闭适配器
func (a *SSEAdapter) Close() error {
	a.closed = true
	return nil
}
