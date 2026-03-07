package handler

import (
	"net/http"

	"github.com/dysodeng/ai-adp/internal/application/tenant/dto"
	"github.com/dysodeng/ai-adp/internal/application/tenant/service"
	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	svc service.TenantService
}

func NewTenantHandler(svc service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) Create(c *gin.Context) {
	var cmd dto.CreateTenantCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *TenantHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TenantHandler) List(c *gin.Context) {
	var q dto.ListTenantsQuery
	_ = c.ShouldBindQuery(&q)
	result, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
