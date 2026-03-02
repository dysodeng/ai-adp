package tenant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"
	"github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
	tenantmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
	tenantrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/tenant"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.TenantEntity{}))
	return db
}

func TestTenantRepo_SaveAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := tenantrepo.NewTenantRepository(db)

	tenant, err := tenantmodel.NewTenant("Acme Corp", "admin@acme.com")
	require.NoError(t, err)

	err = repo.Save(context.Background(), tenant)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), tenant.ID())
	require.NoError(t, err)
	assert.Equal(t, tenant.ID(), found.ID())
	assert.Equal(t, "Acme Corp", found.Name())
	assert.Equal(t, "admin@acme.com", found.Email())
}

func TestTenantRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := tenantrepo.NewTenantRepository(db)

	_, err := repo.FindByID(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.True(t, sharederrors.Is(err, "TENANT_NOT_FOUND"))
}

func TestTenantRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := tenantrepo.NewTenantRepository(db)

	for _, name := range []string{"Tenant A", "Tenant B", "Tenant C"} {
		ten, _ := tenantmodel.NewTenant(name, name+"@example.com")
		_ = repo.Save(context.Background(), ten)
	}

	pagination := valueobject.NewPagination(1, 10)
	tenants, total, err := repo.FindAll(context.Background(), pagination)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, tenants, 3)
}

func TestTenantRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := tenantrepo.NewTenantRepository(db)

	tenant, _ := tenantmodel.NewTenant("Delete Me", "delete@example.com")
	_ = repo.Save(context.Background(), tenant)

	err := repo.Delete(context.Background(), tenant.ID())
	require.NoError(t, err)

	_, err = repo.FindByID(context.Background(), tenant.ID())
	assert.Error(t, err)
}
