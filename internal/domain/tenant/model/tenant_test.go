package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/dysodeng/ai-adp/internal/domain/tenant/model"
	"github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
)

func TestNewTenant_Valid(t *testing.T) {
	tenant, err := model.NewTenant("Acme Corp", "admin@acme.com")
	assert.NoError(t, err)
	assert.Equal(t, "Acme Corp", tenant.Name())
	assert.Equal(t, "admin@acme.com", tenant.Email())
	assert.Equal(t, valueobject.StatusActive, tenant.Status())
	assert.NotEmpty(t, tenant.ID())
	assert.Len(t, tenant.ID(), 36) // UUID v7
}

func TestNewTenant_EmptyName(t *testing.T) {
	_, err := model.NewTenant("", "admin@acme.com")
	assert.Error(t, err)
}

func TestNewTenant_EmptyEmail(t *testing.T) {
	_, err := model.NewTenant("Acme Corp", "")
	assert.Error(t, err)
}

func TestTenant_Deactivate(t *testing.T) {
	tenant, _ := model.NewTenant("Acme Corp", "admin@acme.com")
	tenant.Deactivate()
	assert.Equal(t, valueobject.StatusInactive, tenant.Status())
}

func TestTenant_Suspend(t *testing.T) {
	tenant, _ := model.NewTenant("Acme Corp", "admin@acme.com")
	tenant.Suspend()
	assert.Equal(t, valueobject.StatusSuspended, tenant.Status())
}

func TestTenant_Activate(t *testing.T) {
	tenant, _ := model.NewTenant("Acme Corp", "admin@acme.com")
	tenant.Deactivate()
	tenant.Activate()
	assert.Equal(t, valueobject.StatusActive, tenant.Status())
}

func TestTenant_Reconstitute(t *testing.T) {
	tenant := model.Reconstitute("test-id", "Acme Corp", "admin@acme.com", valueobject.StatusActive)
	assert.Equal(t, "test-id", tenant.ID())
	assert.Equal(t, "Acme Corp", tenant.Name())
}
