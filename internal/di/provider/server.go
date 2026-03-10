package provider

import (
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server/health"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server/http"
	ifaceHttp "github.com/dysodeng/ai-adp/internal/interfaces/http"
)

// ProvideHTTPServer 提供HTTP服务器
func ProvideHTTPServer(cfg *config.Config, handlerRegistry *ifaceHttp.HandlerRegistry) *http.Server {
	return http.NewServer(cfg, handlerRegistry)
}

// ProvideHealthServer 提供容器环境健康检查服务
func ProvideHealthServer(cfg *config.Config) *health.Server {
	return health.NewServer(cfg)
}
