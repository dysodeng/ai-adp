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
)

// InfrastructureSet wires all infrastructure components
var InfrastructureSet = wire.NewSet(
	persistence.NewDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
)

// provideLogger extracts LoggerConfig from Config and calls logger.New
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	return logger.New(cfg.Logger)
}
