package di

import (
	"github.com/google/wire"

	"github.com/dysodeng/ai-adp/internal/di/provider"
	agentService "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/cancel"
	"github.com/dysodeng/ai-adp/internal/interfaces/http"
)

// InfrastructureSet wires all infrastructure components
var InfrastructureSet = wire.NewSet(
	provider.ProvideConfig,
	provider.ProvideMonitor,
	provider.ProvideLogger,
	provider.ProvideDB,
	provider.ProvideRedis,
	provider.ProvideCache,
	port.NewMockToolService,
	agentService.NewAgentBuilder,
	provider.ProvideAgentFactory,
	// 取消能力组件
	cancel.NewMemoryTaskRegistry,
	cancel.NewRedisCancelBroadcaster,
	// 网关注册
	provider.ProvideGatewayRegistry,
)

// ServerSet 服务聚合依赖
var ServerSet = wire.NewSet(
	http.NewHandlerRegistry,
	provider.ProvideHTTPServer,
	provider.ProvideHealthServer,
)
