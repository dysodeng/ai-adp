package service

import (
	"context"
	"fmt"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
	appModel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

type agentBuilderImpl struct {
	toolService port.ToolService
}

func NewAgentBuilder(toolService port.ToolService) AgentBuilder {
	return &agentBuilderImpl{
		toolService: toolService,
	}
}

func (b *agentBuilderImpl) BuildAgentConfig(
	ctx context.Context,
	app *appModel.App,
	input map[string]any,
	isStreaming bool,
) (*model.Config, error) {
	// 1. 加载工具
	tools, cleanup, err := b.loadTools(ctx, app)
	if err != nil {
		return nil, fmt.Errorf("failed to load tools: %w", err)
	}

	// 2. 构建 Agent 配置（业务逻辑在领域层）
	config := &model.Config{
		AgentID:          app.ID().String(),
		AgentName:        app.Name(),
		AgentDescription: app.Description(),
		Type:             app.Type().String(),
		IsStreaming:      isStreaming,
		ToolsConfig: &model.ToolsConfig{
			Enabled: len(tools) > 0,
			Tools:   tools,
		},
		CustomConfig: map[string]any{
			"cleanup": cleanup,
			"app_id":  app.ID().String(),
		},
	}

	return config, nil
}

func (b *agentBuilderImpl) loadTools(
	ctx context.Context,
	app *appModel.App,
) ([]tool.Tool, func(), error) {
	toolConfig := &port.ToolLoadConfig{
		TenantID:        app.TenantID(),
		KnowledgeList:   app.KnowledgeList(),
		ToolList:        app.ToolList(),
		McpServerList:   app.McpServerList(),
		BuiltinToolList: app.BuiltinToolList(),
		AppToolList:     app.AppToolList(),
		IsToolAgent:     app.IsToolAgent(),
	}

	return b.toolService.LoadTools(ctx, toolConfig)
}
