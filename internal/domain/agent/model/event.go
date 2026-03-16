package model

import "time"

// EventType 事件类型
type EventType string

const (
	// 执行生命周期事件
	EventTypeStart     EventType = "start"
	EventTypeComplete  EventType = "complete"
	EventTypeError     EventType = "error"
	EventTypeCancelled EventType = "cancelled"

	// 输出内容事件
	EventTypeChunk    EventType = "chunk"
	EventTypeThinking EventType = "thinking"

	// 工具事件
	EventTypeToolCall   EventType = "tool.call"
	EventTypeToolStart  EventType = "tool.start"
	EventTypeToolResult EventType = "tool.result"
	EventTypeToolError  EventType = "tool.error"

	// 消息事件
	EventTypeMessage EventType = "message"

	// 人工介入事件
	EventTypeInterrupt EventType = "interrupt"
	EventTypeResume    EventType = "resume"

	// Token统计事件
	EventTypeTokenUsage EventType = "token.usage"

	// 流过期事件
	EventTypeExpired EventType = "expired"
)

// Event 领域事件
type Event struct {
	Type      EventType   `json:"type"`
	TaskID    string      `json:"task_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	StreamID  string      `json:"-"` // SSE id 字段，不序列化到 JSON
}

// ExecutionInput 执行输入
type ExecutionInput struct {
	Query     string
	Variables map[string]interface{}
	History   []Message
}

// ExecutionOutput 执行输出
type ExecutionOutput struct {
	Message *Message    `json:"message"`
	Usage   *TokenUsage `json:"usage,omitempty"`
}

// Message 消息
type Message struct {
	Role    string         `json:"role"`
	Content MessageContent `json:"content"`
}

// MessageContent 消息内容
type MessageContent struct {
	Content string `json:"content"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
}

// ToolResult 工具结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Output     string `json:"output"`
}

// TokenUsage Token 使用量
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
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
