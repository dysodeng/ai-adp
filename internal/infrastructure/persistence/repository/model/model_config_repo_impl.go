package model

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domainerrors "github.com/dysodeng/ai-adp/internal/domain/model/errors"
	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

// compile-time interface check
var _ domainrepo.ModelConfigRepository = (*ModelConfigRepositoryImpl)(nil)

// ModelConfigRepositoryImpl GORM-based AI 模型配置仓储
type ModelConfigRepositoryImpl struct {
	db *gorm.DB
}

func NewModelConfigRepository(db *gorm.DB) *ModelConfigRepositoryImpl {
	return &ModelConfigRepositoryImpl{db: db}
}

func (r *ModelConfigRepositoryImpl) Save(ctx context.Context, m *modelconfig.ModelConfig) error {
	e := toEntity(m)
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *ModelConfigRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*modelconfig.ModelConfig, error) {
	var e entity.ModelConfigEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainerrors.ErrModelConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomain(&e), nil
}

func (r *ModelConfigRepositoryImpl) FindDefault(ctx context.Context, capability valueobject.ModelCapability) (*modelconfig.ModelConfig, error) {
	var e entity.ModelConfigEntity
	err := r.db.WithContext(ctx).
		Where("capability = ? AND is_default = true AND enabled = true", capability).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // 无默认模型不是错误，由调用方决定如何处理
	}
	if err != nil {
		return nil, err
	}
	return toDomain(&e), nil
}

func (r *ModelConfigRepositoryImpl) FindAllByCapability(ctx context.Context, capability valueobject.ModelCapability) ([]*modelconfig.ModelConfig, error) {
	var entities []entity.ModelConfigEntity
	err := r.db.WithContext(ctx).
		Where("capability = ? AND enabled = true", capability).
		Order("is_default DESC, created_at ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*modelconfig.ModelConfig, len(entities))
	for i := range entities {
		result[i] = toDomain(&entities[i])
	}
	return result, nil
}

func (r *ModelConfigRepositoryImpl) FindAll(ctx context.Context) ([]*modelconfig.ModelConfig, error) {
	var entities []entity.ModelConfigEntity
	err := r.db.WithContext(ctx).Order("capability, is_default DESC, created_at ASC").Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*modelconfig.ModelConfig, len(entities))
	for i := range entities {
		result[i] = toDomain(&entities[i])
	}
	return result, nil
}

func (r *ModelConfigRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.ModelConfigEntity{}, "id = ?", id).Error
}

// toEntity 将领域对象转换为 GORM 实体
func toEntity(m *modelconfig.ModelConfig) entity.ModelConfigEntity {
	e := entity.ModelConfigEntity{
		Name:        m.Name(),
		Provider:    m.Provider(),
		Capability:  m.Capability(),
		ModelID:     m.ModelID(),
		APIKey:      m.APIKey(),
		BaseURL:     m.BaseURL(),
		MaxTokens:   m.MaxTokens(),
		Temperature: m.Temperature(),
		IsDefault:   m.IsDefault(),
		Enabled:     m.Enabled(),
	}
	e.ID = m.ID()
	return e
}

// toDomain 将 GORM 实体重建为领域聚合根
func toDomain(e *entity.ModelConfigEntity) *modelconfig.ModelConfig {
	return modelconfig.Reconstitute(
		e.ID, e.Name, e.Provider, e.Capability, e.ModelID,
		e.APIKey, e.BaseURL, e.MaxTokens, e.Temperature,
		e.IsDefault, e.Enabled,
	)
}
