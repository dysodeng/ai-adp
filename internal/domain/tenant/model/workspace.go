package model

import (
	"fmt"

	"github.com/google/uuid"
)

// Workspace 工作空间实体（归属于 Tenant）
type Workspace struct {
	id       string
	tenantID string
	name     string
	slug     string
}

// NewWorkspace creates a new Workspace entity
func NewWorkspace(tenantID, name, slug string) (*Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("workspace name cannot be empty")
	}
	if slug == "" {
		return nil, fmt.Errorf("workspace slug cannot be empty")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate workspace ID: %w", err)
	}
	return &Workspace{
		id:       id.String(),
		tenantID: tenantID,
		name:     name,
		slug:     slug,
	}, nil
}

func (w *Workspace) ID() string       { return w.id }
func (w *Workspace) TenantID() string { return w.tenantID }
func (w *Workspace) Name() string     { return w.name }
func (w *Workspace) Slug() string     { return w.slug }
