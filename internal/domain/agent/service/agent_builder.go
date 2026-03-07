package service

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// AgentBuilder Agent 构建器 - 领域服务
// 负责根据 App 配置构建完整的 Agent
type AgentBuilder interface {
	// BuildAgent 根据 App 和输入构建 Agent
	// 处理所有业务逻辑：提示词、工具、模型配置
	BuildAgent(
		ctx context.Context,
		app *model.App,
		input map[string]any,
		isStreaming bool,
	) (agent.Agent, error)
}
