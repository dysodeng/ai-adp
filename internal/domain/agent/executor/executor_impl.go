package executor

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// Option 配置 AgentExecutor 的选项函数
type Option func(*agentExecutorImpl)

// WithEventStore 设置事件存储
func WithEventStore(store EventStore) Option {
	return func(e *agentExecutorImpl) {
		e.eventStore = store
	}
}

// agentExecutorImpl AgentExecutor 的默认实现
type agentExecutorImpl struct {
	ctx            context.Context
	taskID         uuid.UUID
	appID          string
	appType        valueobject.AppType
	conversationID uuid.UUID
	messageID      uuid.UUID
	input          model.ExecutionInput

	status    model.ExecutionStatus
	output    *model.ExecutionOutput
	err       error
	startTime time.Time
	endTime   time.Time

	subscribers []chan *model.Event
	mu          sync.RWMutex

	eventStore EventStore
}

// NewAgentExecutor 创建新的 AgentExecutor
func NewAgentExecutor(
	ctx context.Context,
	taskID uuid.UUID,
	appID string,
	appType valueobject.AppType,
	conversationID uuid.UUID,
	messageID uuid.UUID,
	input model.ExecutionInput,
	opts ...Option,
) AgentExecutor {
	e := &agentExecutorImpl{
		ctx:            ctx,
		taskID:         taskID,
		appID:          appID,
		appType:        appType,
		conversationID: conversationID,
		messageID:      messageID,
		input:          input,
		status:         model.ExecutionStatusPending,
		subscribers:    make([]chan *model.Event, 0),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *agentExecutorImpl) Ctx() context.Context            { return e.ctx }
func (e *agentExecutorImpl) GetTaskID() uuid.UUID            { return e.taskID }
func (e *agentExecutorImpl) GetAppID() string                { return e.appID }
func (e *agentExecutorImpl) GetAppType() valueobject.AppType { return e.appType }
func (e *agentExecutorImpl) GetConversationID() uuid.UUID    { return e.conversationID }
func (e *agentExecutorImpl) GetMessageID() uuid.UUID         { return e.messageID }
func (e *agentExecutorImpl) GetInput() model.ExecutionInput  { return e.input }

func (e *agentExecutorImpl) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = model.ExecutionStatusRunning
	e.startTime = time.Now()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeStart,
		Timestamp: time.Now(),
	})
}

// isTerminated 是否已进入终态（调用方必须已持有锁）
func (e *agentExecutorImpl) isTerminated() bool {
	return e.status == model.ExecutionStatusCompleted ||
		e.status == model.ExecutionStatusFailed ||
		e.status == model.ExecutionStatusCancelled
}

func (e *agentExecutorImpl) Complete(output *model.ExecutionOutput) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isTerminated() {
		return
	}
	e.status = model.ExecutionStatusCompleted
	e.output = output
	e.endTime = time.Now()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeComplete,
		Timestamp: time.Now(),
		Data:      output,
	})
	e.closeAllSubscribers()
	if e.eventStore != nil {
		_ = e.eventStore.SetTTL(e.ctx, e.taskID.String(), 30*time.Second)
	}
}

func (e *agentExecutorImpl) Fail(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isTerminated() {
		return
	}
	e.status = model.ExecutionStatusFailed
	e.err = err
	e.endTime = time.Now()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeError,
		Timestamp: time.Now(),
		Data:      err,
	})
	e.closeAllSubscribers()
	if e.eventStore != nil {
		_ = e.eventStore.SetTTL(e.ctx, e.taskID.String(), 30*time.Second)
	}
}

func (e *agentExecutorImpl) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isTerminated() {
		return
	}
	e.status = model.ExecutionStatusCancelled
	e.endTime = time.Now()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeCancelled,
		Timestamp: time.Now(),
		Data: map[string]string{
			"task_id": e.taskID.String(),
			"reason":  "user_cancelled",
		},
	})
	e.closeAllSubscribers()
	if e.eventStore != nil {
		_ = e.eventStore.SetTTL(e.ctx, e.taskID.String(), 30*time.Second)
	}
}

func (e *agentExecutorImpl) Err() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.err
}

func (e *agentExecutorImpl) PublishChunk(content string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeChunk,
		Timestamp: time.Now(),
		Data:      content,
	})
}

func (e *agentExecutorImpl) PublishThinking(content string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeThinking,
		Timestamp: time.Now(),
		Data:      content,
	})
}

func (e *agentExecutorImpl) PublishToolCall(toolCall *model.ToolCall) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeToolCall,
		Timestamp: time.Now(),
		Data:      toolCall,
	})
}

func (e *agentExecutorImpl) PublishToolStart(toolCall *model.ToolCall) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeToolStart,
		Timestamp: time.Now(),
		Data:      toolCall,
	})
}

func (e *agentExecutorImpl) PublishToolResult(toolResult *model.ToolResult) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeToolResult,
		Timestamp: time.Now(),
		Data:      toolResult,
	})
}

func (e *agentExecutorImpl) PublishToolError(toolCallID, toolName, errMsg string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeToolError,
		Timestamp: time.Now(),
		Data: map[string]string{
			"tool_call_id": toolCallID,
			"tool_name":    toolName,
			"error":        errMsg,
		},
	})
}

func (e *agentExecutorImpl) PublishMessage(message *model.Message) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeMessage,
		Timestamp: time.Now(),
		Data:      message,
	})
}

func (e *agentExecutorImpl) PublishTokenUsage(usage *model.TokenUsage) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.broadcastEvent(&model.Event{
		Type:      model.EventTypeTokenUsage,
		Timestamp: time.Now(),
		Data:      usage,
	})
}

func (e *agentExecutorImpl) Subscribe() <-chan *model.Event {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan *model.Event, 100)

	taskID := e.taskID.String()

	// 如果 executor 已经在运行，重放 start 事件
	if e.status == model.ExecutionStatusRunning {
		ch <- &model.Event{
			Type:      model.EventTypeStart,
			TaskID:    taskID,
			Timestamp: e.startTime,
		}
	}

	// 如果 executor 已经结束，重放终止事件并立即关闭 channel，避免订阅者永久阻塞
	switch e.status {
	case model.ExecutionStatusCompleted:
		ch <- &model.Event{
			Type:      model.EventTypeComplete,
			TaskID:    taskID,
			Timestamp: e.endTime,
			Data:      e.output,
		}
		close(ch)
		return ch
	case model.ExecutionStatusFailed:
		ch <- &model.Event{
			Type:      model.EventTypeError,
			TaskID:    taskID,
			Timestamp: e.endTime,
			Data:      e.err,
		}
		close(ch)
		return ch
	case model.ExecutionStatusCancelled:
		close(ch)
		return ch
	}

	e.subscribers = append(e.subscribers, ch)
	return ch
}

func (e *agentExecutorImpl) AddSubscriber() <-chan *model.Event {
	return e.Subscribe()
}

func (e *agentExecutorImpl) GetStatus() model.ExecutionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

func (e *agentExecutorImpl) IsRunning() bool {
	return e.GetStatus() == model.ExecutionStatusRunning
}

func (e *agentExecutorImpl) IsCompleted() bool {
	return e.GetStatus() == model.ExecutionStatusCompleted
}

func (e *agentExecutorImpl) HasEventStore() bool {
	return e.eventStore != nil
}

func (e *agentExecutorImpl) Duration() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.endTime.IsZero() {
		return time.Since(e.startTime)
	}
	return e.endTime.Sub(e.startTime)
}

func (e *agentExecutorImpl) GetOutput() *model.ExecutionOutput {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.output
}

// broadcastEvent 发布事件到所有订阅者（调用方必须已持有锁）
func (e *agentExecutorImpl) broadcastEvent(event *model.Event) {
	event.TaskID = e.taskID.String()

	// 写入事件存储（如果启用）
	// Note: Redis I/O under mutex is acceptable for AI streaming rates (~1ms vs ~50-100ms token interval)
	if e.eventStore != nil {
		streamID, err := e.eventStore.Append(e.ctx, e.taskID.String(), event)
		if err != nil {
			// Write failure doesn't affect normal push, degrades to non-resumable
		} else {
			event.StreamID = streamID
		}
	}

	for _, ch := range e.subscribers {
		select {
		case ch <- event:
		default:
			// 如果 channel 满了，跳过
		}
	}
}

// closeAllSubscribers 关闭所有订阅者的 channel（调用方必须已持有锁）
func (e *agentExecutorImpl) closeAllSubscribers() {
	for _, ch := range e.subscribers {
		close(ch)
	}
}
