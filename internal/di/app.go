package di

import (
	"context"

	"go.uber.org/zap"

	agentservice "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	appdomainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// App 持有所有服务实例和清理函数，由 cmd/app 驱动生命周期
type App struct {
	HTTPServer     server.Server
	tracerShutdown telemetry.ShutdownFunc
}

// NewApp 构造 App。_ *zap.Logger 确保 Wire 在构建 App 前初始化全局 logger（顺序依赖）。
func NewApp(
	httpServer *server.HTTPServer,
	_ *engine.ExecutorFactory,
	_ appdomainrepo.AppRepository,
	_ *zap.Logger,
	tracerShutdown telemetry.ShutdownFunc,
	// 新架构组件
	_ port.ToolService,
	_ agentservice.AgentBuilder,
	_ *adapter.AgentFactory,
) *App {
	return &App{
		HTTPServer:     httpServer,
		tracerShutdown: tracerShutdown,
	}
}

// Stop 释放应用资源，在所有 Server 停止后调用
func (a *App) Stop(ctx context.Context) error {
	// 刷新日志缓冲，防止文件输出丢失最后几行
	_ = logger.ZapLogger().Sync()
	return a.tracerShutdown(ctx)
}
