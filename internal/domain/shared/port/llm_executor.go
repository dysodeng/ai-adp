package port

import "context"

// Message LLM 消息
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// LLMResponse LLM 非流式响应
type LLMResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Model        string
}

// StreamChunk 流式输出片段
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// LLMExecutor LLM 执行端口（由 infrastructure/ai 实现，domain 层不感知 Eino）
type LLMExecutor interface {
	Execute(ctx context.Context, messages []Message) (*LLMResponse, error)
	Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
}
