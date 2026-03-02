package model

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
)

// Tenant 租户聚合根
type Tenant struct {
	id     string
	name   string
	email  string
	status valueobject.TenantStatus
}

// NewTenant creates a new Tenant aggregate with validation
func NewTenant(name, email string) (*Tenant, error) {
	if name == "" {
		return nil, fmt.Errorf("tenant name cannot be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("tenant email cannot be empty")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate tenant ID: %w", err)
	}
	return &Tenant{
		id:     id.String(),
		name:   name,
		email:  email,
		status: valueobject.StatusActive,
	}, nil
}

// Reconstitute rebuilds a Tenant from persistence (no new ID generated)
func Reconstitute(id, name, email string, status valueobject.TenantStatus) *Tenant {
	return &Tenant{id: id, name: name, email: email, status: status}
}

func (t *Tenant) ID() string                       { return t.id }
func (t *Tenant) Name() string                     { return t.name }
func (t *Tenant) Email() string                    { return t.email }
func (t *Tenant) Status() valueobject.TenantStatus { return t.status }

func (t *Tenant) Deactivate() { t.status = valueobject.StatusInactive }
func (t *Tenant) Activate()   { t.status = valueobject.StatusActive }
func (t *Tenant) Suspend()    { t.status = valueobject.StatusSuspended }
