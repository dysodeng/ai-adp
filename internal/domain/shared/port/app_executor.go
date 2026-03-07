package port

import "context"

// AppEventType 应用执行事件类型
type AppEventType string

const (
	AppEventMessage    AppEventType = "message"
	AppEventToolCall   AppEventType = "tool_call"
	AppEventToolResult AppEventType = "tool_result"
	AppEventDone       AppEventType = "done"
	AppEventError      AppEventType = "error"
)

// AppEvent 应用执行事件
type AppEvent struct {
	Type    AppEventType
	Content string
	Error   error
}

// AppExecutorInput 应用执行输入
type AppExecutorInput struct {
	Query     string
	Variables map[string]string
	History   []Message
	Stream    bool
}

// AppResult 非流式执行结果
type AppResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
}

// AppExecutor 统一的 AI 应用执行端口
type AppExecutor interface {
	Run(ctx context.Context, input *AppExecutorInput) (<-chan AppEvent, error)
	Execute(ctx context.Context, input *AppExecutorInput) (*AppResult, error)
}
