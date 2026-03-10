package tenant

import (
	"context"
	"errors"

	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
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
var _ domainrepo.TenantRepository = (*tenantRepository)(nil)

type tenantRepository struct {
	txManger transactions.TransactionManager
}

func NewTenantRepository(txManger transactions.TransactionManager) domainrepo.TenantRepository {
	return &tenantRepository{txManger: txManger}
}

func (r *tenantRepository) Save(ctx context.Context, tenant *domainmodel.Tenant) error {
	e := &entity.TenantEntity{}
	if id := tenant.ID(); id != uuid.Nil {
		e.ID = id
	}
	e.Name = tenant.Name()
	e.Email = tenant.Email()
	e.Status = string(tenant.Status())
	return r.txManger.GetTx(ctx).Save(e).Error
}

func (r *tenantRepository) FindByID(ctx context.Context, id string) (*domainmodel.Tenant, error) {
	var e entity.TenantEntity
	if err := r.txManger.GetTx(ctx).First(&e, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
		}
		return nil, err
	}
	return domainmodel.Reconstitute(e.ID, e.Name, e.Email, tenantvo.TenantStatus(e.Status)), nil
}

func (r *tenantRepository) FindAll(ctx context.Context, pagination valueobject.Pagination) ([]*domainmodel.Tenant, int64, error) {
	var entities []entity.TenantEntity
	var total int64

	db := r.txManger.GetTx(ctx).Model(&entity.TenantEntity{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(pagination.Offset()).Limit(pagination.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	tenants := make([]*domainmodel.Tenant, 0, len(entities))
	for _, e := range entities {
		tenants = append(tenants, domainmodel.Reconstitute(e.ID, e.Name, e.Email, tenantvo.TenantStatus(e.Status)))
	}
	return tenants, total, nil
}

func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	return r.txManger.GetTx(ctx).Delete(&entity.TenantEntity{}, "id = ?", id).Error
}
