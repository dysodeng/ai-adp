package executor

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AgentExecutor Agent 执行器 - 封装单次 Agent 执行的完整生命周期
type AgentExecutor interface {
	// ========== 上下文信息 ==========
	Ctx() context.Context
	GetTaskID() uuid.UUID
	GetAppID() string
	GetAppType() valueobject.AppType
	GetConversationID() uuid.UUID
	GetMessageID() uuid.UUID
	GetInput() model.ExecutionInput

	// ========== 生命周期管理 ==========
	Start()
	Complete(output *model.ExecutionOutput)
	Fail(err error)
	Cancel()
	Err() error

	// ========== 事件发布 ==========
	PublishChunk(content string)
	PublishThinking(content string)
	PublishToolCall(toolCall *model.ToolCall)
	PublishToolStart(toolCall *model.ToolCall)
	PublishToolResult(toolResult *model.ToolResult)
	PublishToolError(toolCallID, toolName, errMsg string)
	PublishMessage(message *model.Message)
	PublishTokenUsage(usage *model.TokenUsage)

	// ========== 事件订阅 ==========
	Subscribe() <-chan *model.Event
	AddSubscriber() <-chan *model.Event

	// ========== 状态查询 ==========
	GetStatus() model.ExecutionStatus
	IsRunning() bool
	IsCompleted() bool
	Duration() time.Duration
	GetOutput() *model.ExecutionOutput
	HasEventStore() bool
}
