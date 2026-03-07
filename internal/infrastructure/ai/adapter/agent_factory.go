package adapter

import (
	"context"
	"fmt"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// AgentFactory Agent 工厂（基础设施层）
// 根据配置和 AppType 创建具体的 Agent 实现
type AgentFactory struct {
	modelConfigGetter func(ctx context.Context, modelID string) (*modelconfig.ModelConfig, error)
}

func NewAgentFactory(modelConfigGetter func(ctx context.Context, modelID string) (*modelconfig.ModelConfig, error)) *AgentFactory {
	return &AgentFactory{
		modelConfigGetter: modelConfigGetter,
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
	modelConfig, err := f.modelConfigGetter(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load model config: %w", err)
	}

	// 根据 AppType 创建对应的 Agent
	switch appType {
	case valueobject.AppTypeTextCompletion:
		return NewTextCompletionAgent(ctx, config, modelConfig)

	case valueobject.AppTypeChat:
		// TODO: 实现 ChatAgent
		return nil, fmt.Errorf("ChatAgent not implemented yet")

	case valueobject.AppTypeAgent:
		// TODO: 实现 ReActAgent
		return nil, fmt.Errorf("ReActAgent not implemented yet")

	default:
		return nil, fmt.Errorf("unsupported app type: %s", appType)
	}
}
