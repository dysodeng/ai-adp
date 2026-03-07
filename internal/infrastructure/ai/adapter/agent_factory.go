package adapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	modelrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
)

// AgentFactory Agent 工厂（基础设施层）
// 根据配置和 AppType 创建具体的 Agent 实现
type AgentFactory struct {
	modelConfigRepo modelrepo.ModelConfigRepository
}

func NewAgentFactory(modelConfigRepo modelrepo.ModelConfigRepository) *AgentFactory {
	return &AgentFactory{
		modelConfigRepo: modelConfigRepo,
	}
}

// CreateAgent 根据 AppType 和配置创建 Agent
func (f *AgentFactory) CreateAgent(
	ctx context.Context,
	appType valueobject.AppType,
	config *model.Config,
	modelID string,
) (agent.Agent, error) {
	// 加载模型配置
	modelUUID, err := uuid.Parse(modelID)
	if err != nil {
		return nil, fmt.Errorf("invalid model ID: %w", err)
	}

	modelConfig, err := f.modelConfigRepo.FindByID(ctx, modelUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load model config: %w", err)
	}
	if modelConfig == nil {
		return nil, fmt.Errorf("model config not found: %s", modelID)
	}

	// 根据 AppType 创建对应的 Agent
	switch appType {
	case valueobject.AppTypeTextCompletion:
		return NewTextCompletionAgent(ctx, config, modelConfig)

	case valueobject.AppTypeChat:
		return NewChatAgent(ctx, config, modelConfig)

	case valueobject.AppTypeAgent:
		return NewReActAgent(ctx, config, modelConfig)

	default:
		return nil, fmt.Errorf("unsupported app type: %s", appType)
	}
}
