package port

// Message 通用消息类型
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}
