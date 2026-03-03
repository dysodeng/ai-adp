package dto

import "github.com/google/uuid"

// CreateTenantCommand 创建租户命令
type CreateTenantCommand struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateTenantCommand 更新租户命令
type UpdateTenantCommand struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// TenantResult 租户查询结果
type TenantResult struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Status string    `json:"status"`
}

// ListTenantsQuery 分页查询参数
type ListTenantsQuery struct {
	Page  int `json:"page" form:"page"`
	Limit int `json:"limit" form:"limit"`
}

// TenantListResult 租户列表结果
type TenantListResult struct {
	Items []*TenantResult `json:"items"`
	Total int64           `json:"total"`
}
