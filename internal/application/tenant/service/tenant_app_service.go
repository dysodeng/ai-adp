package service

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/application/tenant/dto"
	"github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
	domainmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
)

// TenantService defines the tenant use case interface (used for mocking in tests)
type TenantService interface {
	Create(ctx context.Context, cmd dto.CreateTenantCommand) (*dto.TenantResult, error)
	GetByID(ctx context.Context, id string) (*dto.TenantResult, error)
	List(ctx context.Context, q dto.ListTenantsQuery) (*dto.TenantListResult, error)
	Delete(ctx context.Context, id string) error
}

// TenantAppService handles tenant use cases
type TenantAppService struct {
	tenantRepo domainrepo.TenantRepository
}

func NewTenantAppService(tenantRepo domainrepo.TenantRepository) *TenantAppService {
	return &TenantAppService{tenantRepo: tenantRepo}
}

func (s *TenantAppService) Create(ctx context.Context, cmd dto.CreateTenantCommand) (*dto.TenantResult, error) {
	tenant, err := domainmodel.NewTenant(cmd.Name, cmd.Email)
	if err != nil {
		return nil, err
	}
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return nil, err
	}
	return toTenantResult(tenant), nil
}

func (s *TenantAppService) GetByID(ctx context.Context, id string) (*dto.TenantResult, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTenantResult(tenant), nil
}

func (s *TenantAppService) List(ctx context.Context, q dto.ListTenantsQuery) (*dto.TenantListResult, error) {
	pagination := valueobject.NewPagination(q.Page, q.Limit)
	tenants, total, err := s.tenantRepo.FindAll(ctx, pagination)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.TenantResult, 0, len(tenants))
	for _, t := range tenants {
		items = append(items, toTenantResult(t))
	}
	return &dto.TenantListResult{Items: items, Total: total}, nil
}

func (s *TenantAppService) Delete(ctx context.Context, id string) error {
	return s.tenantRepo.Delete(ctx, id)
}

func toTenantResult(t *domainmodel.Tenant) *dto.TenantResult {
	return &dto.TenantResult{
		ID:     t.ID(),
		Name:   t.Name(),
		Email:  t.Email(),
		Status: string(t.Status()),
	}
}
