package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"
	"github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
	domainmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
	tenantvo "github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

// Ensure interface is implemented
var _ domainrepo.TenantRepository = (*TenantRepositoryImpl)(nil)

type TenantRepositoryImpl struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepositoryImpl {
	return &TenantRepositoryImpl{db: db}
}

func (r *TenantRepositoryImpl) Save(ctx context.Context, tenant *domainmodel.Tenant) error {
	e := &entity.TenantEntity{}
	if id := tenant.ID(); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		e.ID = parsed
	}
	e.Name = tenant.Name()
	e.Email = tenant.Email()
	e.Status = string(tenant.Status())
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *TenantRepositoryImpl) FindByID(ctx context.Context, id string) (*domainmodel.Tenant, error) {
	var e entity.TenantEntity
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
		}
		return nil, err
	}
	return domainmodel.Reconstitute(e.ID.String(), e.Name, e.Email, tenantvo.TenantStatus(e.Status)), nil
}

func (r *TenantRepositoryImpl) FindAll(ctx context.Context, pagination valueobject.Pagination) ([]*domainmodel.Tenant, int64, error) {
	var entities []entity.TenantEntity
	var total int64

	db := r.db.WithContext(ctx).Model(&entity.TenantEntity{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(pagination.Offset()).Limit(pagination.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	tenants := make([]*domainmodel.Tenant, 0, len(entities))
	for _, e := range entities {
		tenants = append(tenants, domainmodel.Reconstitute(e.ID.String(), e.Name, e.Email, tenantvo.TenantStatus(e.Status)))
	}
	return tenants, total, nil
}

func (r *TenantRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.TenantEntity{}, "id = ?", id).Error
}
