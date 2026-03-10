package router

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/interfaces/http"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// RegisterRouter 注册路由
func RegisterRouter(router *gin.Engine, registry *http.HandlerRegistry) {
	// 健康检查
	router.GET("/health", handler.HealthCheck)

	v1 := router.Group("/v1")
	registerV1Routes(v1, registry)
}

// registerV1Routes 注册 v1 版本的所有路由
func registerV1Routes(v1 *gin.RouterGroup, registry *http.HandlerRegistry) {
	// Tenant 路由
	tenants := v1.Group("/tenants")
	{
		tenants.POST("", registry.TenantHandler.Create)
		tenants.GET("", registry.TenantHandler.List)
		tenants.GET("/:id", registry.TenantHandler.GetByID)
		tenants.DELETE("/:id", registry.TenantHandler.Delete)
	}

	// Chat 路由
	chats := v1.Group("/chat", middleware.AppApiKey)
	{
		chats.POST("/send-messages", registry.ChatHandler.Chat)
		chats.POST("/tasks/:task_id/cancel", registry.ChatCancelHandler.Cancel)
	}
}
