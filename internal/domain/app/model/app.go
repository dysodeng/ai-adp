package model

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// App AI 应用聚合根
type App struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	description string
	appType     valueobject.AppType
	icon        string

	// 工具配置（由工具领域处理）
	knowledgeList   []uuid.UUID // 知识库 ID 列表
	toolList        []uuid.UUID // 插件工具 ID 列表
	mcpServerList   []uuid.UUID // MCP 服务器 ID 列表
	builtinToolList []uuid.UUID // 内置工具 ID 列表
	appToolList     []uuid.UUID // App-as-Tool ID 列表
}

func NewApp(tenantID uuid.UUID, name, description string, appType valueobject.AppType, icon string) (*App, error) {
	if name == "" {
		return nil, fmt.Errorf("app: name cannot be empty")
	}
	if !appType.IsValid() {
		return nil, fmt.Errorf("app: invalid type %q", appType)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("app: failed to generate ID: %w", err)
	}
	return &App{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		appType:     appType,
		icon:        icon,
	}, nil
}

func Reconstitute(id, tenantID uuid.UUID, name, description string, appType valueobject.AppType, icon string) *App {
	return &App{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		appType:     appType,
		icon:        icon,
	}
}

func (a *App) ID() uuid.UUID             { return a.id }
func (a *App) TenantID() uuid.UUID       { return a.tenantID }
func (a *App) Name() string              { return a.name }
func (a *App) Description() string       { return a.description }
func (a *App) Type() valueobject.AppType { return a.appType }
func (a *App) Icon() string              { return a.icon }

func (a *App) KnowledgeList() []uuid.UUID   { return a.knowledgeList }
func (a *App) ToolList() []uuid.UUID        { return a.toolList }
func (a *App) McpServerList() []uuid.UUID   { return a.mcpServerList }
func (a *App) BuiltinToolList() []uuid.UUID { return a.builtinToolList }
func (a *App) AppToolList() []uuid.UUID     { return a.appToolList }

// IsToolAgent 判断是否支持工具调用
func (a *App) IsToolAgent() bool {
	return a.appType == valueobject.AppTypeAgent
}

func (a *App) SetName(name string)               { a.name = name }
func (a *App) SetDescription(description string) { a.description = description }
func (a *App) SetIcon(icon string)               { a.icon = icon }
