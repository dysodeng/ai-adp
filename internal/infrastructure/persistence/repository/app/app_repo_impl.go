package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/dysodeng/ai-adp/internal/domain/app/errors"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

var _ domainrepo.AppRepository = (*AppRepositoryImpl)(nil)

type AppRepositoryImpl struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) *AppRepositoryImpl {
	return &AppRepositoryImpl{db: db}
}

func (r *AppRepositoryImpl) SaveApp(ctx context.Context, app *appmodel.App) error {
	e := toAppEntity(app)
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *AppRepositoryImpl) FindAppByID(ctx context.Context, id uuid.UUID) (*appmodel.App, error) {
	var e entity.AppEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return toAppDomain(&e), nil
}

func (r *AppRepositoryImpl) FindAppsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*appmodel.App, error) {
	var entities []entity.AppEntity
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*appmodel.App, len(entities))
	for i := range entities {
		result[i] = toAppDomain(&entities[i])
	}
	return result, nil
}

func (r *AppRepositoryImpl) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.AppEntity{}, "id = ?", id).Error
}

func (r *AppRepositoryImpl) SaveVersion(ctx context.Context, version *appmodel.AppVersion) error {
	e, err := toVersionEntity(version)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *AppRepositoryImpl) FindVersionByID(ctx context.Context, id uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindPublishedVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, valueobject.VersionStatusPublished).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindDraftVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, valueobject.VersionStatusDraft).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindVersionsByApp(ctx context.Context, appID uuid.UUID) ([]*appmodel.AppVersion, error) {
	var entities []entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ?", appID).
		Order("version DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return toVersionDomainList(entities)
}

func (r *AppRepositoryImpl) FindVersionsByStatus(ctx context.Context, appID uuid.UUID, status valueobject.VersionStatus) ([]*appmodel.AppVersion, error) {
	var entities []entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, status).
		Order("version DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return toVersionDomainList(entities)
}

func (r *AppRepositoryImpl) SaveApiKey(ctx context.Context, apiKey *appmodel.AppApiKey) error {
	e := toApiKeyEntity(apiKey)
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *AppRepositoryImpl) FindApiKeyByKey(ctx context.Context, key string) (*appmodel.AppApiKey, error) {
	var e entity.AppApiKeyEntity
	err := r.db.WithContext(ctx).
		Where("api_key = ? AND is_active = ?", key, true).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrApiKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return toApiKeyDomain(&e), nil
}

func (r *AppRepositoryImpl) FindApiKeysByApp(ctx context.Context, appID uuid.UUID) ([]*appmodel.AppApiKey, error) {
	var entities []entity.AppApiKeyEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ?", appID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*appmodel.AppApiKey, len(entities))
	for i := range entities {
		result[i] = toApiKeyDomain(&entities[i])
	}
	return result, nil
}

func (r *AppRepositoryImpl) DeleteApiKey(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.AppApiKeyEntity{}, "id = ?", id).Error
}

func (r *AppRepositoryImpl) FindAppWithPublishedVersion(ctx context.Context, appID uuid.UUID) (*appmodel.App, *appmodel.AppVersion, error) {
	var appEntity entity.AppEntity
	err := r.db.WithContext(ctx).First(&appEntity, "id = ?", appID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperrors.ErrAppNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	var versionEntity entity.AppVersionEntity
	err = r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, valueobject.VersionStatusPublished).
		First(&versionEntity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperrors.ErrNoPublishedVersion
	}
	if err != nil {
		return nil, nil, err
	}

	app := toAppDomain(&appEntity)
	version, err := toVersionDomain(&versionEntity)
	if err != nil {
		return nil, nil, err
	}
	return app, version, nil
}

func (r *AppRepositoryImpl) FindAppByApiKey(ctx context.Context, key string) (*appmodel.App, *appmodel.AppVersion, error) {
	var apiKeyEntity entity.AppApiKeyEntity
	err := r.db.WithContext(ctx).
		Where("api_key = ? AND is_active = ?", key, true).
		First(&apiKeyEntity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperrors.ErrApiKeyNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	appID, _ := uuid.Parse(apiKeyEntity.AppID)
	return r.FindAppWithPublishedVersion(ctx, appID)
}

func toApiKeyEntity(k *appmodel.AppApiKey) entity.AppApiKeyEntity {
	e := entity.AppApiKeyEntity{
		AppID:       k.AppID().String(),
		ApiKey:      k.ApiKey(),
		Description: k.Description(),
		IsActive:    k.IsActive(),
		LastUsedAt:  k.LastUsedAt(),
	}
	e.ID = k.ID()
	return e
}

func toApiKeyDomain(e *entity.AppApiKeyEntity) *appmodel.AppApiKey {
	appID, _ := uuid.Parse(e.AppID)
	return appmodel.ReconstituteAppApiKey(e.ID, appID, e.ApiKey, e.Description, e.IsActive, e.LastUsedAt, e.CreatedAt)
}

func toAppEntity(a *appmodel.App) entity.AppEntity {
	e := entity.AppEntity{
		TenantID:    a.TenantID().String(),
		Name:        a.Name(),
		Description: a.Description(),
		AppType:     a.Type(),
		Icon:        a.Icon(),
	}
	e.ID = a.ID()
	return e
}

func toAppDomain(e *entity.AppEntity) *appmodel.App {
	tenantID, _ := uuid.Parse(e.TenantID)
	return appmodel.Reconstitute(e.ID, tenantID, e.Name, e.Description, e.AppType, e.Icon)
}

func toVersionEntity(v *appmodel.AppVersion) (entity.AppVersionEntity, error) {
	configJSON, err := v.Config().ToJSON()
	if err != nil {
		return entity.AppVersionEntity{}, err
	}
	e := entity.AppVersionEntity{
		AppID:       v.AppID().String(),
		Version:     v.Version(),
		Status:      v.Status(),
		Config:      string(configJSON),
		PublishedAt: v.PublishedAt(),
	}
	e.ID = v.ID()
	return e, nil
}

func toVersionDomain(e *entity.AppVersionEntity) (*appmodel.AppVersion, error) {
	appID, _ := uuid.Parse(e.AppID)
	config, err := valueobject.AppConfigFromJSON([]byte(e.Config))
	if err != nil {
		return nil, err
	}
	return appmodel.ReconstituteVersion(e.ID, appID, e.Version, e.Status, config, e.PublishedAt), nil
}

func toVersionDomainList(entities []entity.AppVersionEntity) ([]*appmodel.AppVersion, error) {
	result := make([]*appmodel.AppVersion, len(entities))
	for i := range entities {
		v, err := toVersionDomain(&entities[i])
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}
