package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	einomodel "github.com/cloudwego/eino/components/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	agentinfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/agent"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/embedding"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	modelrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/model"
)

// Components AI 基础设施组件，可能为 nil（未配置对应 Provider 时）
type Components struct {
	LLMExecutor   port.LLMExecutor   // nil 表示未配置默认 LLM 模型
	Embedder      port.Embedder      // nil 表示未配置默认 Embedding 模型
	AgentExecutor port.AgentExecutor // nil 表示 LLM 不支持工具调用或未配置
}

// NewComponents 从 DB 加载默认模型配置并初始化 AI 组件
// 若某类型无默认模型，对应组件为 nil（不返回错误）
func NewComponents(db *gorm.DB) (*Components, error) {
	repo := modelrepo.NewModelConfigRepository(db)
	ctx := context.Background()
	components := &Components{}

	if err := initLLMComponents(ctx, repo, components); err != nil {
		return nil, err
	}
	if err := initEmbeddingComponent(ctx, repo, components); err != nil {
		return nil, err
	}

	return components, nil
}

func initLLMComponents(ctx context.Context, repo domainrepo.ModelConfigRepository, c *Components) error {
	m, err := repo.FindDefault(ctx, valueobject.ModelCapabilityLLM)
	if err != nil {
		return fmt.Errorf("ai: failed to query default LLM model: %w", err)
	}
	if m == nil {
		logger.Info(ctx, "ai: no default LLM model configured, LLMExecutor and AgentExecutor will be unavailable")
		return nil
	}

	chatModel, err := llm.NewChatModel(ctx, m)
	if err != nil {
		return fmt.Errorf("ai: failed to create LLM provider %q: %w", m.Provider(), err)
	}
	c.LLMExecutor = llm.NewAdapter(chatModel)

	// AgentExecutor requires ToolCallingChatModel; not all providers support it
	toolModel, ok := chatModel.(einomodel.ToolCallingChatModel)
	if !ok {
		logger.Info(ctx, "ai: LLM provider does not support tool calling, AgentExecutor unavailable",
			logger.AddField("provider", m.Provider()))
		return nil
	}
	agentExecutor, err := agentinfra.NewAgentExecutor(toolModel)
	if err != nil {
		logger.Info(ctx, "ai: failed to create AgentExecutor, AgentExecutor unavailable",
			logger.AddField("provider", m.Provider()),
			logger.AddField("reason", err.Error()))
		return nil
	}
	c.AgentExecutor = agentExecutor

	return nil
}

func initEmbeddingComponent(ctx context.Context, repo domainrepo.ModelConfigRepository, c *Components) error {
	m, err := repo.FindDefault(ctx, valueobject.ModelCapabilityEmbedding)
	if err != nil {
		return fmt.Errorf("ai: failed to query default Embedding model: %w", err)
	}
	if m == nil {
		logger.Info(ctx, "ai: no default Embedding model configured, Embedder will be unavailable")
		return nil
	}

	embedder, err := embedding.NewEmbedder(ctx, m)
	if err != nil {
		return fmt.Errorf("ai: failed to create Embedding provider %q: %w", m.Provider(), err)
	}
	c.Embedder = embedding.NewAdapter(embedder)

	return nil
}
