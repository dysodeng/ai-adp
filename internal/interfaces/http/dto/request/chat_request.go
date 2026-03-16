package request

// ChatRequest Chat 对话请求
type ChatRequest struct {
	ConversationID  string         `json:"conversation_id"`
	Query           string         `json:"query" binding:"required"`
	Input           map[string]any `json:"input"`
	ResponseMode    string         `json:"response_mode"`
	EnableSSEResume bool           `json:"enable_sse_resume"`
}

// ReconnectRequest SSE 重连请求
type ReconnectRequest struct {
	TaskID      string `json:"task_id" binding:"required"`
	LastEventID string `json:"last_event_id" binding:"required"`
}
