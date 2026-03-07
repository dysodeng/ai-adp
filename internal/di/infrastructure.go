package di

import (
	"fmt"

	"github.com/google/wire"
	"go.uber.org/zap"
	"gorm.io/gorm"

	agentservice "github.com/dysodeng/ai-adp/internal/domain/agent/service"
	modeldomainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
	"github.com/dysodeng/ai-adp/internal/infrastructure/cache"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/migration"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// InfrastructureSet wires all infrastructure components
var InfrastructureSet = wire.NewSet(
	provideDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
	engine.NewExecutorFactory, // AI 引擎工厂（旧）
	// 新架构组件
	port.NewMockToolService,
	agentservice.NewAgentBuilder,
	provideAgentFactory,
)

// provideDB 初始化 DB 连接并自动执行迁移
func provideDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := persistence.NewDB(cfg)
	if err != nil {
		return nil, err
	}
	if err := migration.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}
	return db, nil
}

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

// provideAgentFactory 提供 AgentFactory
func provideAgentFactory(modelConfigRepo modeldomainrepo.ModelConfigRepository) *adapter.AgentFactory {
	return adapter.NewAgentFactory(modelConfigRepo)
}
