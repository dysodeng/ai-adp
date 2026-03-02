package di

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"github.com/dysodeng/ai-adp/internal/infrastructure/cache"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// InfrastructureSet wires all infrastructure components
var InfrastructureSet = wire.NewSet(
	persistence.NewDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
)

// provideLogger 初始化全局 logger 并返回底层 *zap.Logger
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	if err := logger.InitLogger(cfg.Logger); err != nil {
		return nil, err
	}
	return logger.ZapLogger(), nil
}

// provideTracerShutdown initialises OpenTelemetry tracing (sets the global provider)
// and returns the shutdown function to be called on application exit.
func provideTracerShutdown(cfg *config.Config) (telemetry.ShutdownFunc, error) {
	_, shutdown, err := telemetry.NewTracerProvider(cfg.Tracing)
	return shutdown, err
}
