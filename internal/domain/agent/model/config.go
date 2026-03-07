package model

import "github.com/dysodeng/ai-adp/internal/domain/agent/tool"

// Config Agent 配置
type Config struct {
	AgentID          string
	AgentName        string
	AgentDescription string
	Type             string
	IsStreaming      bool
	MaxIterations    int

	LLMConfig    *LLMConfig
	Prompt       *PromptConfig
	ToolsConfig  *ToolsConfig
	CustomConfig map[string]interface{}
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Provider    string
	Model       string
	APIKey      string
	BaseURL     string
	Temperature *float64
	MaxTokens   *int
	TopP        *float64
}

// PromptConfig 提示词配置
type PromptConfig struct {
	SystemPrompt string
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Enabled bool
	Tools   []tool.Tool
}
