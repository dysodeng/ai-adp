package repository

import (
	"context"

	"github.com/google/uuid"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
)

// ModelConfigRepository AI 模型配置仓储接口
type ModelConfigRepository interface {
	// Save 新建或更新模型配置
	Save(ctx context.Context, m *modelconfig.ModelConfig) error
	// FindByID 按 ID 查询
	FindByID(ctx context.Context, id uuid.UUID) (*modelconfig.ModelConfig, error)
	// FindDefault 查询指定能力类型的默认模型，无结果返回 nil, nil
	FindDefault(ctx context.Context, capability valueobject.ModelCapability) (*modelconfig.ModelConfig, error)
	// FindAllByCapability 查询指定能力类型的所有启用模型
	FindAllByCapability(ctx context.Context, capability valueobject.ModelCapability) ([]*modelconfig.ModelConfig, error)
	// FindAll 查询所有模型
	FindAll(ctx context.Context) ([]*modelconfig.ModelConfig, error)
	// Delete 按 ID 删除
	Delete(ctx context.Context, id uuid.UUID) error
}
