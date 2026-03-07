package repository

import (
	"context"

	"github.com/google/uuid"

	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppRepository AI 应用仓储接口
type AppRepository interface {
	SaveApp(ctx context.Context, app *appmodel.App) error
	FindAppByID(ctx context.Context, id uuid.UUID) (*appmodel.App, error)
	FindAppsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*appmodel.App, error)
	DeleteApp(ctx context.Context, id uuid.UUID) error

	SaveVersion(ctx context.Context, version *appmodel.AppVersion) error
	FindVersionByID(ctx context.Context, id uuid.UUID) (*appmodel.AppVersion, error)
	FindPublishedVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error)
	FindDraftVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error)
	FindVersionsByApp(ctx context.Context, appID uuid.UUID) ([]*appmodel.AppVersion, error)
	FindVersionsByStatus(ctx context.Context, appID uuid.UUID, status valueobject.VersionStatus) ([]*appmodel.AppVersion, error)
}
