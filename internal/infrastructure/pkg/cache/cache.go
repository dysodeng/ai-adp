package cache

import (
	"sync"
	"time"

	serializer2 "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/serializer"
	"github.com/pkg/errors"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
)

var (
	ErrKeyExpired  = errors.New("key expired")
	ErrKeyNotExist = errors.New("key not exist")
)

// Cache 缓存接口
type Cache interface {
	IsExist(key string) bool
	Get(key string, dest any) error
	Put(key string, value any, expiration time.Duration) error
	GetString(key string) (string, error)
	PutString(key string, value string, expiration time.Duration) error
	Delete(key string) error
	BatchDelete(prefix string) error
}

var cache Cache
var cacheInstanceOnce sync.Once

// NewCache 创建缓存实例
func NewCache(cfg *config.Config) (Cache, error) {
	serializer := serializer2.NewSerializer(cfg.Cache.Serializer)
	cacheInstanceOnce.Do(func() {
		switch cfg.Cache.Driver {
		case "memory": // 内存
			cache = NewMemoryCache(serializer)
		case "redis": // redis
			cache = NewRedisWithClient(redis.CacheClient(), "", serializer)
		default:
			panic("缓存驱动错误")
		}
	})
	return cache, nil
}
