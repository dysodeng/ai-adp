package embedding

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"
	openaieembed "github.com/cloudwego/eino-ext/components/embedding/openai"
	einoembed "github.com/cloudwego/eino/components/embedding"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// NewEmbedder 根据 ModelConfig 创建对应 Provider 的 Eino Embedder
func NewEmbedder(ctx context.Context, m *modelconfig.ModelConfig) (einoembed.Embedder, error) {
	switch m.Provider() {
	case "openai", "openai_compatible":
		cfg := &openaieembed.EmbeddingConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return openaieembed.NewEmbedder(ctx, cfg)

	case "ark":
		cfg := &ark.EmbeddingConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return ark.NewEmbedder(ctx, cfg)

	case "ollama":
		cfg := &ollamaembed.EmbeddingConfig{
			Model: m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return ollamaembed.NewEmbedder(ctx, cfg)

	default:
		return nil, fmt.Errorf("embedding: unsupported provider %q", m.Provider())
	}
}
