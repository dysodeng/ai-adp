package cache

import (
	"github.com/redis/go-redis/v9"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func NewRedisClient(cfg *config.Config) *redis.Client {
	poolSize := cfg.Redis.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}
	minIdle := cfg.Redis.MinIdleConns
	if minIdle <= 0 {
		minIdle = 5
	}
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
	})
}
