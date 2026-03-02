package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"github.com/dysodeng/ai-adp/internal/application/tenant/dto"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	mocksvc "github.com/dysodeng/ai-adp/internal/application/tenant/service/mock"
)

func TestTenantHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocksvc.NewMockTenantService(ctrl)
	mockSvc.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(&dto.TenantResult{ID: "uuid-001", Name: "Acme", Email: "admin@acme.com", Status: "active"}, nil).
		Times(1)

	h := handler.NewTenantHandler(mockSvc)
	r := gin.New()
	r.POST("/tenants", h.Create)

	body, _ := json.Marshal(map[string]string{"name": "Acme", "email": "admin@acme.com"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "uuid-001")
}

func TestTenantHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocksvc.NewMockTenantService(ctrl)
	mockSvc.EXPECT().
		GetByID(gomock.Any(), "uuid-001").
		Return(&dto.TenantResult{ID: "uuid-001", Name: "Acme", Email: "admin@acme.com", Status: "active"}, nil).
		Times(1)

	h := handler.NewTenantHandler(mockSvc)
	r := gin.New()
	r.GET("/tenants/:id", h.GetByID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/tenants/uuid-001", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Acme")
}

func TestTenantHandler_Create_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocksvc.NewMockTenantService(ctrl)
	mockSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewTenantHandler(mockSvc)
	r := gin.New()
	r.POST("/tenants", h.Create)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/tenants", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
