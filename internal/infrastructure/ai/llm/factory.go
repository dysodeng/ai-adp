package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// NewChatModel 根据 ModelConfig 配置创建对应 Provider 的 Eino BaseChatModel
func NewChatModel(ctx context.Context, m *modelconfig.ModelConfig) (einomodel.BaseChatModel, error) {
	switch m.Provider() {
	case "openai", "openai_compatible":
		cfg := &openai.ChatModelConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			maxTokens := m.MaxTokens()
			cfg.MaxTokens = &maxTokens
		}
		return openai.NewChatModel(ctx, cfg)

	case "ark":
		cfg := &ark.ChatModelConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			maxTokens := m.MaxTokens()
			cfg.MaxTokens = &maxTokens
		}
		return ark.NewChatModel(ctx, cfg)

	case "ollama":
		cfg := &ollama.ChatModelConfig{
			Model: m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return ollama.NewChatModel(ctx, cfg)

	case "claude":
		cfg := &claude.Config{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = &[]string{m.BaseURL()}[0]
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			cfg.MaxTokens = m.MaxTokens()
		}
		return claude.NewChatModel(ctx, cfg)

	default:
		return nil, fmt.Errorf("llm: unsupported provider %q", m.Provider())
	}
}
