//go:build wireinject

package di

import (
	"context"

	"github.com/google/wire"
)

func InitApp(ctx context.Context) (*App, error) {
	wire.Build(
		InfrastructureSet,
		ModulesSet,
		ServerSet,
		NewApp,
	)
	return nil, nil
}
