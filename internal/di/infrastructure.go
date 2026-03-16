package di

import (
	"github.com/google/wire"

	"github.com/dysodeng/ai-adp/internal/di/provider"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	agentService "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/cancel"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/stream"
	pkgredis "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
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
	// SSE 重连组件
	stream.NewMemoryExecutorRegistry,
	wire.Bind(new(executor.ExecutorRegistry), new(*stream.MemoryExecutorRegistry)),
	provideRedisEventStore,
	wire.Bind(new(executor.EventStore), new(*stream.RedisEventStore)),
	// 网关注册
	provider.ProvideGatewayRegistry,
)

func provideRedisEventStore(client pkgredis.Client) *stream.RedisEventStore {
	prefix := pkgredis.MainKey("")
	return stream.NewRedisEventStore(client, prefix)
}

// ServerSet 服务聚合依赖
var ServerSet = wire.NewSet(
	http.NewHandlerRegistry,
	provider.ProvideHTTPServer,
	provider.ProvideHealthServer,
)
