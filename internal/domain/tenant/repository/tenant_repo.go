package repository

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
	"github.com/dysodeng/ai-adp/internal/domain/tenant/model"
)

type TenantRepository interface {
	Save(ctx context.Context, tenant *model.Tenant) error
	FindByID(ctx context.Context, id string) (*model.Tenant, error)
	FindAll(ctx context.Context, pagination valueobject.Pagination) ([]*model.Tenant, int64, error)
	Delete(ctx context.Context, id string) error
}

type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *model.Workspace) error
	FindByID(ctx context.Context, id string) (*model.Workspace, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]*model.Workspace, error)
}
