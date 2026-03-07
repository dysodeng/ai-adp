package entity

import "time"

// AppApiKeyEntity 应用 API Key 数据库实体
type AppApiKeyEntity struct {
	Base
	AppID       string `gorm:"type:uuid;not null;index:idx_api_key_app"`
	ApiKey      string `gorm:"size:100;not null;uniqueIndex:idx_api_key_unique"`
	Description string `gorm:"size:500"`
	IsActive    bool   `gorm:"not null;default:true"`
	LastUsedAt  *time.Time
}

func (AppApiKeyEntity) TableName() string { return "app_api_keys" }
