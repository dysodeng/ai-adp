package entity

import (
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppVersionEntity AI 应用版本数据库实体
type AppVersionEntity struct {
	Base
	AppID       string                    `gorm:"type:uuid;not null;index:idx_version_app"`
	Version     int                       `gorm:"not null"`
	Status      valueobject.VersionStatus `gorm:"size:20;not null;index:idx_version_status"`
	Config      string                    `gorm:"type:jsonb;not null"`
	PublishedAt *time.Time
}

func (AppVersionEntity) TableName() string { return "app_versions" }
