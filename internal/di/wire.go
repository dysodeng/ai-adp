//go:build wireinject

package di

import (
	"github.com/google/wire"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// InitApp is the Wire injector. Wire generates wire_gen.go from this.
func InitApp(configPath string) (*App, error) {
	wire.Build(
		config.Load,
		InfrastructureSet,
		NewApp,
	)
	return nil, nil
}
