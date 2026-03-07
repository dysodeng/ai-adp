package entity

import "github.com/dysodeng/ai-adp/internal/domain/app/valueobject"

// AppEntity AI 应用数据库实体
type AppEntity struct {
	Base
	TenantID    string              `gorm:"type:uuid;not null;index:idx_app_tenant"`
	Name        string              `gorm:"size:200;not null"`
	Description string              `gorm:"size:1000"`
	AppType     valueobject.AppType `gorm:"size:30;not null;index:idx_app_type"`
	Icon        string              `gorm:"size:500"`
}

func (AppEntity) TableName() string { return "apps" }
