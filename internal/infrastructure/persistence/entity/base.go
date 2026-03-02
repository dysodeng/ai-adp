package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base 所有 GORM 实体的基类，主键使用 UUID v7
type Base struct {
	ID        string         `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GenerateID 生成 UUID v7，供 BeforeCreate hook 和测试使用
func (b *Base) GenerateID() error {
	if b.ID != "" {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	b.ID = id.String()
	return nil
}

// BeforeCreate GORM hook：自动生成 UUID v7 主键
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	return b.GenerateID()
}
