package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

func TestNewApp_Valid(t *testing.T) {
	tenantID := uuid.New()
	app, err := appmodel.NewApp(tenantID, "My Chat App", "A test chat app", valueobject.AppTypeChat, "")
	require.NoError(t, err)
	assert.Equal(t, tenantID, app.TenantID())
	assert.Equal(t, "My Chat App", app.Name())
	assert.Equal(t, valueobject.AppTypeChat, app.Type())
}

func TestNewApp_EmptyName(t *testing.T) {
	_, err := appmodel.NewApp(uuid.New(), "", "desc", valueobject.AppTypeChat, "")
	assert.Error(t, err)
}

func TestNewApp_InvalidType(t *testing.T) {
	_, err := appmodel.NewApp(uuid.New(), "App", "desc", valueobject.AppType("unknown"), "")
	assert.Error(t, err)
}

func TestNewAppVersion_Valid(t *testing.T) {
	cfg := &valueobject.AppConfig{SystemPrompt: "you are helpful"}
	v, err := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, v.Version())
	assert.Equal(t, valueobject.VersionStatusDraft, v.Status())
}

func TestAppVersion_Publish(t *testing.T) {
	cfg := &valueobject.AppConfig{SystemPrompt: "test"}
	v, _ := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	v.Publish()
	assert.Equal(t, valueobject.VersionStatusPublished, v.Status())
	assert.NotNil(t, v.PublishedAt())
}

func TestAppVersion_Archive(t *testing.T) {
	cfg := &valueobject.AppConfig{SystemPrompt: "test"}
	v, _ := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	v.Publish()
	v.Archive()
	assert.Equal(t, valueobject.VersionStatusArchived, v.Status())
}

func TestApp_Reconstitute(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	app := appmodel.Reconstitute(id, tenantID, "App", "desc", valueobject.AppTypeAgent, "icon.png")
	assert.Equal(t, id, app.ID())
	assert.Equal(t, tenantID, app.TenantID())
	assert.Equal(t, valueobject.AppTypeAgent, app.Type())
}
