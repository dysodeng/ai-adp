package migration

import (
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
	"gorm.io/gorm"
)

// AutoMigrate runs GORM auto-migration for all registered entities.
// Add new entities here as bounded contexts are implemented.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.TenantEntity{},
		&entity.WorkspaceEntity{},
		&entity.ModelConfigEntity{}, // 新增
	)
}
