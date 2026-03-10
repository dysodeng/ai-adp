package di

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry"
	"go.uber.org/zap"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	agentService "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/cache"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server/health"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server/http"
	ifaceHTTP "github.com/dysodeng/ai-adp/internal/interfaces/http"
)

// App 持有所有服务实例和清理函数，由 cmd/app 驱动生命周期
type App struct {
	cfg               *config.Config
	monitor           *telemetry.Monitor
	logger            *zap.Logger
	txManager         transactions.TransactionManager
	redisClient       redis.Client
	cache             cache.Cache
	HandlerRegistry   *ifaceHTTP.HandlerRegistry
	HTTPServer        *http.Server
	HealthServer      *health.Server
	cancelBroadcaster executor.CancelBroadcaster
	taskRegistry      executor.TaskRegistry
}

// NewApp 构造 App。_ *zap.Logger 确保 Wire 在构建 App 前初始化全局 logger（顺序依赖）。
func NewApp(
	cfg *config.Config,
	monitor *telemetry.Monitor,
	logger *zap.Logger,
	txManager transactions.TransactionManager,
	redisClient redis.Client,
	cache cache.Cache,
	handlerRegistry *ifaceHTTP.HandlerRegistry,
	httpServer *http.Server,
	healthServer *health.Server,
	_ port.ToolService,
	_ agentService.AgentBuilder,
	_ *adapter.AgentFactory,
	// 取消能力组件
	cancelBroadcaster executor.CancelBroadcaster,
	taskRegistry executor.TaskRegistry,
) *App {
	return &App{
		cfg:               cfg,
		monitor:           monitor,
		logger:            logger,
		txManager:         txManager,
		redisClient:       redisClient,
		cache:             cache,
		HandlerRegistry:   handlerRegistry,
		HTTPServer:        httpServer,
		HealthServer:      healthServer,
		cancelBroadcaster: cancelBroadcaster,
		taskRegistry:      taskRegistry,
	}
}

// StartCancelSubscriber 启动取消信号订阅
func (a *App) StartCancelSubscriber(ctx context.Context) error {
	return a.cancelBroadcaster.Subscribe(ctx, a.taskRegistry)
}

// Stop 释放应用资源，在所有 Server 停止后调用
func (a *App) Stop(ctx context.Context) error {
	redis.Close()
	_ = logger.ZapLogger().Sync()
	return nil
}
