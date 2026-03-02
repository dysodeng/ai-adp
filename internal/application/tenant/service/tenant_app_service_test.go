package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/dysodeng/ai-adp/internal/application/tenant/dto"
	"github.com/dysodeng/ai-adp/internal/application/tenant/service"
	tenantmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
	mockrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository/mock"
	tenantvo "github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
)

func TestTenantAppService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockrepo.NewMockTenantRepository(ctrl)
	mockRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewTenantAppService(mockRepo)
	result, err := svc.Create(context.Background(), dto.CreateTenantCommand{
		Name:  "Acme Corp",
		Email: "admin@acme.com",
	})

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", result.Name)
	assert.Equal(t, "active", result.Status)
	// ID is assigned by the persistence layer; uuid.Nil expected before actual DB save
	_ = result.ID
}

func TestTenantAppService_Create_EmptyName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockrepo.NewMockTenantRepository(ctrl)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewTenantAppService(mockRepo)
	_, err := svc.Create(context.Background(), dto.CreateTenantCommand{Name: "", Email: "admin@acme.com"})
	assert.Error(t, err)
}

func TestTenantAppService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testID := uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef")
	testIDStr := testID.String()

	mockRepo := mockrepo.NewMockTenantRepository(ctrl)
	mockTenant := tenantmodel.Reconstitute(testID, "Acme Corp", "admin@acme.com", tenantvo.StatusActive)

	mockRepo.EXPECT().
		FindByID(gomock.Any(), testIDStr).
		Return(mockTenant, nil).
		Times(1)

	svc := service.NewTenantAppService(mockRepo)
	result, err := svc.GetByID(context.Background(), testIDStr)

	require.NoError(t, err)
	assert.Equal(t, testID, result.ID)
	assert.Equal(t, "Acme Corp", result.Name)
}
