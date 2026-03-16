package protocol

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// SSEAdapter SSE 协议适配器
// 职责：订阅 AgentExecutor 的事件流，转换为 SSE 格式输出
type SSEAdapter struct {
	w            http.ResponseWriter
	flusher      http.Flusher
	closed       bool
	enableResume bool
	retrySent    bool
}

// NewSSEAdapter 创建 SSE 适配器
func NewSSEAdapter(w http.ResponseWriter, enableResume bool) (*SSEAdapter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEAdapter{w: w, flusher: flusher, enableResume: enableResume}, nil
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
			if err := a.SendEvent(event); err != nil {
				return err
			}
		}
	}
}

// HandleReconnection replays cached events then continues with live executor
func (a *SSEAdapter) HandleReconnection(ctx context.Context, cachedEvents []*model.Event, liveExecutor executor.AgentExecutor) error {
	var lastSentID string
	for _, event := range cachedEvents {
		if err := a.SendEvent(event); err != nil {
			return err
		}
		if event.StreamID != "" {
			lastSentID = event.StreamID
		}
	}

	if liveExecutor == nil {
		return nil
	}

	eventChan := liveExecutor.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			// Skip synthetic events (no StreamID) from Subscribe()
			if event.StreamID == "" {
				continue
			}
			// Dedup: skip events already sent via cache replay
			if lastSentID != "" && event.StreamID <= lastSentID {
				continue
			}
			if err := a.SendEvent(event); err != nil {
				return err
			}
		}
	}
}

// SendEvent sends a single event in SSE format (exported for use by handler)
func (a *SSEAdapter) SendEvent(event *model.Event) error {
	if a.closed {
		return fmt.Errorf("adapter is closed")
	}

	data, err := sonic.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	var sseData string
	if a.enableResume && event.StreamID != "" {
		sseData = fmt.Sprintf("id: %s\n", event.StreamID)
		if !a.retrySent {
			sseData += "retry: 3000\n"
			a.retrySent = true
		}
		sseData += fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	} else {
		sseData = fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	}

	if _, err = fmt.Fprint(a.w, sseData); err != nil {
		return fmt.Errorf("write sse data failed: %w", err)
	}

	a.flusher.Flush()
	return nil
}

// SendError 发送错误事件
func (a *SSEAdapter) SendError(err error) error {
	return a.SendEvent(&model.Event{
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
