package http

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// Router HTTP 路由管理器，集中管理所有路由注册
type Router struct {
	tenantHandler *handler.TenantHandler
	chatHandler   *handler.ChatHandler
}

// NewRouter 创建路由管理器
func NewRouter(tenantHandler *handler.TenantHandler, chatHandler *handler.ChatHandler) *Router {
	return &Router{
		tenantHandler: tenantHandler,
		chatHandler:   chatHandler,
	}
}

// Setup 配置 Gin Engine 的中间件和路由
func (r *Router) Setup(engine *gin.Engine, appName string) {
	engine.Use(
		middleware.Recovery(),
		middleware.Tracing(appName),
		middleware.Logger(),
		middleware.RequestID(),
	)

	engine.GET("/health", handler.HealthCheck)

	v1 := engine.Group("/v1")
	r.registerV1Routes(v1)
}

// registerV1Routes 注册 v1 版本的所有路由
func (r *Router) registerV1Routes(v1 *gin.RouterGroup) {
	// Tenant 路由
	tenants := v1.Group("/tenants")
	{
		tenants.POST("", r.tenantHandler.Create)
		tenants.GET("", r.tenantHandler.List)
		tenants.GET("/:id", r.tenantHandler.GetByID)
		tenants.DELETE("/:id", r.tenantHandler.Delete)
	}

	// Chat 路由
	chats := v1.Group("/chat", middleware.AppApiKey)
	{
		chats.POST("/send-messages", r.chatHandler.Chat)
	}
}
