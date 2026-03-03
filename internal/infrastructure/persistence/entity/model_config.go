package entity

import "github.com/dysodeng/ai-adp/internal/domain/model/valueobject"

// ModelConfigEntity AI 模型配置数据库实体
type ModelConfigEntity struct {
	Base
	Name        string                      `gorm:"size:100;not null"`
	Provider    string                      `gorm:"size:50;not null;index:idx_provider_capability"`
	Capability  valueobject.ModelCapability `gorm:"size:20;not null;index:idx_provider_capability"`
	ModelID     string                      `gorm:"size:200;not null"`
	APIKey      string                      `gorm:"size:500"`
	BaseURL     string                      `gorm:"size:500"`
	MaxTokens   int                         `gorm:"default:0"`
	Temperature *float32
	IsDefault   bool `gorm:"default:false;index:idx_default_capability"`
	Enabled     bool `gorm:"default:true"`
}

func (ModelConfigEntity) TableName() string { return "model_configs" }
