package cache

import (
	"github.com/redis/go-redis/v9"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
