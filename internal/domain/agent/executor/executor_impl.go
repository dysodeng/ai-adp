package executor

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

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
) AgentExecutor {
	return &agentExecutorImpl{
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
	e.publishEvent(&model.Event{
		Type:      model.EventTypeStart,
		Timestamp: time.Now(),
	})
}

func (e *agentExecutorImpl) Complete(output *model.ExecutionOutput) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = model.ExecutionStatusCompleted
	e.output = output
	e.endTime = time.Now()
	e.publishEvent(&model.Event{
		Type:      model.EventTypeComplete,
		Timestamp: time.Now(),
		Data:      output,
	})
	e.closeAllSubscribers()
}

func (e *agentExecutorImpl) Fail(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = model.ExecutionStatusFailed
	e.err = err
	e.endTime = time.Now()
	e.publishEvent(&model.Event{
		Type:      model.EventTypeError,
		Timestamp: time.Now(),
		Data:      err,
	})
	e.closeAllSubscribers()
}

func (e *agentExecutorImpl) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = model.ExecutionStatusCancelled
	e.endTime = time.Now()
	e.closeAllSubscribers()
}

func (e *agentExecutorImpl) Err() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.err
}

func (e *agentExecutorImpl) PublishChunk(content string) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeChunk,
		Timestamp: time.Now(),
		Data:      content,
	})
}

func (e *agentExecutorImpl) PublishThinking(content string) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeThinking,
		Timestamp: time.Now(),
		Data:      content,
	})
}

func (e *agentExecutorImpl) PublishToolCall(toolCall *model.ToolCall) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeToolCall,
		Timestamp: time.Now(),
		Data:      toolCall,
	})
}

func (e *agentExecutorImpl) PublishToolStart(toolCall *model.ToolCall) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeToolStart,
		Timestamp: time.Now(),
		Data:      toolCall,
	})
}

func (e *agentExecutorImpl) PublishToolResult(toolResult *model.ToolResult) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeToolResult,
		Timestamp: time.Now(),
		Data:      toolResult,
	})
}

func (e *agentExecutorImpl) PublishToolError(toolCallID, toolName, errMsg string) {
	e.publishEvent(&model.Event{
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
	e.publishEvent(&model.Event{
		Type:      model.EventTypeMessage,
		Timestamp: time.Now(),
		Data:      message,
	})
}

func (e *agentExecutorImpl) PublishTokenUsage(usage *model.TokenUsage) {
	e.publishEvent(&model.Event{
		Type:      model.EventTypeTokenUsage,
		Timestamp: time.Now(),
		Data:      usage,
	})
}

func (e *agentExecutorImpl) Subscribe() <-chan *model.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan *model.Event, 100)
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

// publishEvent 发布事件到所有订阅者
func (e *agentExecutorImpl) publishEvent(event *model.Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, ch := range e.subscribers {
		select {
		case ch <- event:
		default:
			// 如果 channel 满了，跳过
		}
	}
}

// closeAllSubscribers 关闭所有订阅者的 channel
func (e *agentExecutorImpl) closeAllSubscribers() {
	for _, ch := range e.subscribers {
		close(ch)
	}
}
