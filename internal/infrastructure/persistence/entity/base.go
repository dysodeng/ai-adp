package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base 所有 GORM 实体的基类，主键使用 UUID v7
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// GenerateID 生成 UUID v7，供 BeforeCreate hook 和测试使用
func (b *Base) GenerateID() error {
	if b.ID != uuid.Nil {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

// BeforeCreate GORM hook：自动生成 UUID v7 主键
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	return b.GenerateID()
}
