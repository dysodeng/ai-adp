package di

import (
	"github.com/google/wire"

	"github.com/dysodeng/ai-adp/internal/infrastructure/cache"
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
)
