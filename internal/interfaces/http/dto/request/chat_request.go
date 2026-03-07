package request

// ChatRequest Chat 对话请求
type ChatRequest struct {
	ConversationID string         `json:"conversation_id"`
	Query          string         `json:"query" binding:"required"`
	Input          map[string]any `json:"input"`
	ResponseMode   string         `json:"response_mode"`
}
