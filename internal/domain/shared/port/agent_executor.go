package port

import "context"

// AgentEventType Agent 执行事件类型
type AgentEventType string

const (
	AgentEventTypeMessage  AgentEventType = "message"   // LLM 输出文本片段（流式）
	AgentEventTypeToolCall AgentEventType = "tool_call" // 工具调用（工具名存 Content）
	AgentEventTypeDone     AgentEventType = "done"      // 执行完成
	AgentEventTypeError    AgentEventType = "error"     // 执行出错（Error 字段非 nil）
)

// AgentEvent Agent 执行事件
type AgentEvent struct {
	Type    AgentEventType
	Content string // message/tool_call 时的内容
	Error   error  // error 类型时非 nil
}

// AgentExecutor ADK Agent 执行端口（由 infrastructure/ai/agent 实现）
// domain 层不感知 Eino 细节
type AgentExecutor interface {
	// Run 执行 Agent，返回事件 channel；调用方需消费至 Done 或 Error
	Run(ctx context.Context, query string, history []Message) (<-chan AgentEvent, error)
}
