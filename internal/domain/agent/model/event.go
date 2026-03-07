package model

import "time"

// EventType 事件类型
type EventType string

const (
	EventTypeStart      EventType = "start"
	EventTypeChunk      EventType = "chunk"
	EventTypeThinking   EventType = "thinking"
	EventTypeToolCall   EventType = "tool_call"
	EventTypeToolStart  EventType = "tool_start"
	EventTypeToolResult EventType = "tool_result"
	EventTypeToolError  EventType = "tool_error"
	EventTypeMessage    EventType = "message"
	EventTypeTokenUsage EventType = "token_usage"
	EventTypeComplete   EventType = "complete"
	EventTypeError      EventType = "error"
)

// Event 领域事件
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// ExecutionInput 执行输入
type ExecutionInput struct {
	Query     string
	Variables map[string]interface{}
	History   []Message
}

// ExecutionOutput 执行输出
type ExecutionOutput struct {
	Message *Message
	Usage   *TokenUsage
}

// Message 消息
type Message struct {
	Role    string
	Content MessageContent
}

// MessageContent 消息内容
type MessageContent struct {
	Content string
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string
	ToolName string
	Input    map[string]interface{}
}

// ToolResult 工具结果
type ToolResult struct {
	ToolCallID string
	ToolName   string
	Output     string
}

// TokenUsage Token 使用量
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)
