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

func (a *App) SetName(name string)               { a.name = name }
func (a *App) SetDescription(description string) { a.description = description }
func (a *App) SetIcon(icon string)               { a.icon = icon }
