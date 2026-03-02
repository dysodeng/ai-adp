//go:build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func InitApp(configPath string) (*App, error) {
	wire.Build(
		config.Load,
		InfrastructureSet,
		ModulesSet,
		NewApp,
	)
	return nil, nil
}
