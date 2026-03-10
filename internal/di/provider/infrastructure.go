package provider

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry"
	"go.uber.org/zap"

	modeldomainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/migration"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/cache"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/db"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
)

// ProvideConfig 提供配置
func ProvideConfig() (*config.Config, error) {
	return config.Load("configs/app.yaml")
}

// ProvideMonitor 提供可观测性配置
func ProvideMonitor(cfg *config.Config) (*telemetry.Monitor, error) {
	return telemetry.InitMonitor(cfg)
}

// ProvideLogger 提供日志
func ProvideLogger(cfg *config.Config) (*zap.Logger, error) {
	logger.InitLogger(cfg)
	return logger.ZapLogger(), nil
}

// ProvideRedis 提供redis
func ProvideRedis(cfg *config.Config) (redis.Client, error) {
	cli, err := redis.Initialize(cfg)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func ProvideCache(cfg *config.Config) (cache.Cache, error) {
	return cache.NewCache(cfg)
}

// ProvideDB 提供数据库
func ProvideDB(ctx context.Context, cfg *config.Config) (transactions.TransactionManager, error) {
	tx, err := db.Initialize(cfg)
	if err != nil {
		return nil, err
	}

	txManager := transactions.NewGormTransactionManager(tx)

	if cfg.Database.Migration.Enabled {
		// 执行数据库迁移
		if err = migration.Migrate(ctx, txManager); err != nil {
			logger.Fatal(ctx, "数据库迁移失败", logger.ErrorField(err))
		}

		// 填充初始数据
		if err = migration.Seed(ctx, txManager); err != nil {
			logger.Fatal(ctx, "初始数据填充失败", logger.ErrorField(err))
		}
	}
	return txManager, nil
}

// ProvideAgentFactory 提供 AgentFactory
func ProvideAgentFactory(modelConfigRepo modeldomainrepo.ModelConfigRepository) *adapter.AgentFactory {
	return adapter.NewAgentFactory(modelConfigRepo)
}
