package http

import (
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
)

// HandlerRegistry 控制器注册表
type HandlerRegistry struct {
	TenantHandler     *handler.TenantHandler
	ChatHandler       *handler.ChatHandler
	ChatCancelHandler *handler.CancelHandler
}

func NewHandlerRegistry(
	tenantHandler *handler.TenantHandler,
	chatHandler *handler.ChatHandler,
	chatCancelHandler *handler.CancelHandler,
) *HandlerRegistry {
	return &HandlerRegistry{
		TenantHandler:     tenantHandler,
		ChatHandler:       chatHandler,
		ChatCancelHandler: chatCancelHandler,
	}
}
