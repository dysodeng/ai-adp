package entity

// TenantEntity GORM 映射实体
type TenantEntity struct {
	Base
	Name   string `gorm:"type:varchar(255);not null"`
	Email  string `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status string `gorm:"type:varchar(50);not null;default:'active'"`
}

func (TenantEntity) TableName() string { return "tenants" }

// WorkspaceEntity GORM 映射实体
type WorkspaceEntity struct {
	Base
	TenantID string `gorm:"type:varchar(36);not null;index"`
	Name     string `gorm:"type:varchar(255);not null"`
	Slug     string `gorm:"type:varchar(100);not null;uniqueIndex"`
}

func (WorkspaceEntity) TableName() string { return "workspaces" }
