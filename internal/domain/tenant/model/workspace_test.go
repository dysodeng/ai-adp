package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/dysodeng/ai-adp/internal/domain/tenant/model"
)

func TestNewWorkspace_Valid(t *testing.T) {
	ws, err := model.NewWorkspace("tenant-id-1", "My Workspace", "my-workspace")
	assert.NoError(t, err)
	assert.Equal(t, "tenant-id-1", ws.TenantID())
	assert.Equal(t, "My Workspace", ws.Name())
	assert.Equal(t, "my-workspace", ws.Slug())
	assert.NotEmpty(t, ws.ID())
}

func TestNewWorkspace_EmptyName(t *testing.T) {
	_, err := model.NewWorkspace("tenant-id-1", "", "slug")
	assert.Error(t, err)
}

func TestNewWorkspace_EmptySlug(t *testing.T) {
	_, err := model.NewWorkspace("tenant-id-1", "name", "")
	assert.Error(t, err)
}
