package dto

// ResponseMode 响应模式
type ResponseMode string

const (
	ResponseModeStreaming ResponseMode = "streaming"
	ResponseModeBlocking  ResponseMode = "blocking"
)

// IsValid 验证响应模式是否合法
func (m ResponseMode) IsValid() bool {
	return m == ResponseModeStreaming || m == ResponseModeBlocking
}

// ChatCommand Chat 对话命令
type ChatCommand struct {
	ConversationID string         `json:"conversation_id"`
	Query          string         `json:"query"`
	Input          map[string]any `json:"input"`
	ResponseMode   ResponseMode   `json:"response_mode"`
}
