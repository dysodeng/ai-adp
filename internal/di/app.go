package di

import (
	"context"
	"fmt"
	"net"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry"
	"go.uber.org/zap"

	"github.com/dysodeng/ai-adp/internal/di/provider"
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
	"github.com/dysodeng/gateway/sdk"
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
	gatewayRegistry   *provider.GatewayRegistry
	gatewayInstances  []sdk.ServiceInstance // 已注册的网关服务实例，用于退出时注销
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
	// 网关注册
	gatewayRegistry *provider.GatewayRegistry,
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
		gatewayRegistry:   gatewayRegistry,
	}
}

// StartCancelSubscriber 启动取消信号订阅
func (a *App) StartCancelSubscriber(ctx context.Context) error {
	return a.cancelBroadcaster.Subscribe(ctx, a.taskRegistry)
}

// Stop 释放应用资源，在所有 Server 停止后调用
func (a *App) Stop(ctx context.Context) error {
	// 先注销网关服务实例，再关闭注册器
	if a.gatewayRegistry != nil && a.gatewayRegistry.Registry != nil {
		for _, instance := range a.gatewayInstances {
			if err := a.gatewayRegistry.Registry.Deregister(ctx, instance); err != nil {
				logger.Error(ctx, fmt.Sprintf("deregister service [%s] from gateway failed", instance.Name), logger.ErrorField(err))
			}
		}
		_ = a.gatewayRegistry.Registry.Close()
	}
	redis.Close()
	_ = logger.ZapLogger().Sync()
	return nil
}

// RegisterGateway 注册服务到网关
func (a *App) RegisterGateway(ctx context.Context) error {
	if a.gatewayRegistry == nil || a.gatewayRegistry.Registry == nil {
		return nil
	}

	instances := buildServiceInstances(a.cfg)
	for _, instance := range instances {
		registered, err := a.gatewayRegistry.Registry.Register(ctx, instance)
		if err != nil {
			return fmt.Errorf("register service [%s] to gateway failed: %w", instance.Name, err)
		}
		a.gatewayInstances = append(a.gatewayInstances, registered)
	}
	return nil
}

func buildServiceInstances(cfg *config.Config) []sdk.ServiceInstance {
	var instances []sdk.ServiceInstance
	serviceName := cfg.App.Name
	host := outboundIP()

	if cfg.Server.HTTP.Enabled {
		instances = append(instances, sdk.ServiceInstance{
			Name:    serviceName,
			Host:    host,
			Port:    cfg.Server.HTTP.Port,
			Version: cfg.Monitor.ServiceVersion,
			Metadata: map[string]string{
				"protocol": "http",
				"env":      cfg.App.Environment,
			},
		})
	}

	return instances
}

// outboundIP 获取本机出口IP
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
