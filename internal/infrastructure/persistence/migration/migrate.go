package migration

import (
	"gorm.io/gorm"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

// AutoMigrate runs GORM auto-migration for all registered entities.
// Add new entities here as bounded contexts are implemented.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.TenantEntity{},
		&entity.WorkspaceEntity{},
	)
}
