package errors

import sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
	ErrTenantNotFound    = sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
	ErrWorkspaceNotFound = sharederrors.New("WORKSPACE_NOT_FOUND", "workspace not found")
	ErrTenantInactive    = sharederrors.New("TENANT_INACTIVE", "tenant is inactive")
	ErrSlugAlreadyExists = sharederrors.New("WORKSPACE_SLUG_EXISTS", "workspace slug already exists")
)
