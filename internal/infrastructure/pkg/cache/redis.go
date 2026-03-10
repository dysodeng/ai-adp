package cache

import (
	"context"
	"fmt"
	"time"

	serializerIface "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/serializer"
	"github.com/redis/go-redis/v9"
)

// Redis redis缓存
type Redis struct {
	keyPrefix  string
	client     redis.UniversalClient
	serializer serializerIface.Serializer
}

// NewRedisWithClient 使用redis连接创建缓存实例
func NewRedisWithClient(redisClient redis.UniversalClient, keyPrefix string, serializer serializerIface.Serializer) Cache {
	return &Redis{
		client:     redisClient,
		keyPrefix:  keyPrefix,
		serializer: serializer,
	}
}

func (r *Redis) key(key string) string {
	if r.keyPrefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", r.keyPrefix, key)
}

func (r *Redis) IsExist(key string) bool {
	if v, err := r.client.Exists(context.Background(), r.key(key)).Result(); err == nil && v > 0 {
		return true
	}
	return false
}

func (r *Redis) Get(key string, dest any) error {
	data, err := r.client.Get(context.Background(), r.key(key)).Result()
	if err != nil {
		return err
	}
	return r.serializer.Unmarshal(data, dest)
}

func (r *Redis) Put(key string, value any, expiration time.Duration) error {
	data, err := r.serializer.Marshal(value)
	if err != nil {
		return err
	}
	_, err = r.client.Set(context.Background(), r.key(key), data, expiration).Result()
	return err
}

func (r *Redis) GetString(key string) (string, error) {
	return r.client.Get(context.Background(), r.key(key)).Result()
}

func (r *Redis) PutString(key string, value string, expiration time.Duration) error {
	_, err := r.client.Set(context.Background(), r.key(key), value, expiration).Result()
	return err
}

func (r *Redis) Delete(key string) error {
	_, err := r.client.Del(context.Background(), r.key(key)).Result()
	return err
}

// BatchDelete 批量删除
func (r *Redis) BatchDelete(prefix string) error {
	var cursor uint64
	var keys []string
	var err error

	ctx := context.Background()

	for {
		keys, cursor, err = r.client.Scan(ctx, cursor, prefix+"*", 10).Result()
		if err != nil {
			return err
		}

		for _, key := range keys {
			err = r.client.Del(ctx, key).Err()
			if err != nil {
				fmt.Printf("failed to delete key: %s\n", key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return err
}
