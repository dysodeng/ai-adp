package service

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
)

// AgentBuilder Agent 构建器 - 领域服务
// 负责根据 App 配置构建 Agent 配置
type AgentBuilder interface {
	// BuildAgentConfig 根据 App 和输入构建 Agent 配置
	// 处理所有业务逻辑：提示词、工具、模型配置
	// 返回配置，由调用方使用基础设施层工厂创建 Agent
	BuildAgentConfig(
		ctx context.Context,
		app *appModel.App,
		input map[string]any,
		isStreaming bool,
	) (*model.Config, error)
}
