package cache

import (
	"strings"
	"sync"
	"time"

	serializerIface "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/serializer"
)

// item 缓存项
type item struct {
	value    string
	created  time.Time
	lifetime time.Duration
}

// isExpire 缓存项是否过期
func (item *item) isExpire() bool {
	if item.lifetime == 0 {
		return false
	}
	return time.Since(item.created) > item.lifetime
}

const defaultGcInterval = time.Minute // 1分钟gc一次

// MemoryCache 内存缓存
type MemoryCache struct {
	duration   time.Duration
	items      sync.Map
	gcTicker   *time.Ticker
	serializer serializerIface.Serializer
}

func NewMemoryCache(serializer serializerIface.Serializer) Cache {
	m := &MemoryCache{
		duration:   defaultGcInterval,
		gcTicker:   time.NewTicker(defaultGcInterval),
		serializer: serializer,
	}
	go m.gc()
	return m
}

// IsExist 缓存项是否存在
func (c *MemoryCache) IsExist(key string) bool {
	_, exists := c.items.Load(key)
	return exists && !c.isItemExpired(key)
}

// Get 获取缓存项并反序列化到dest
func (c *MemoryCache) Get(key string, dest any) error {
	val, ok := c.items.Load(key)
	if !ok {
		return ErrKeyNotExist
	}
	cacheItem := val.(*item)
	if cacheItem.isExpire() {
		return ErrKeyExpired
	}
	return c.serializer.Unmarshal(cacheItem.value, dest)
}

// Put 序列化value并设置缓存项
func (c *MemoryCache) Put(key string, value any, expiration time.Duration) error {
	data, err := c.serializer.Marshal(value)
	if err != nil {
		return err
	}
	c.items.Store(key, &item{
		value:    data,
		lifetime: expiration,
		created:  time.Now(),
	})
	return nil
}

// GetString 获取原始字符串值
func (c *MemoryCache) GetString(key string) (string, error) {
	val, ok := c.items.Load(key)
	if !ok {
		return "", ErrKeyNotExist
	}
	cacheItem := val.(*item)
	if cacheItem.isExpire() {
		return "", ErrKeyExpired
	}
	return cacheItem.value, nil
}

// PutString 直接存储字符串值
func (c *MemoryCache) PutString(key string, value string, expiration time.Duration) error {
	c.items.Store(key, &item{
		value:    value,
		lifetime: expiration,
		created:  time.Now(),
	})
	return nil
}

// Delete 删除缓存项
func (c *MemoryCache) Delete(key string) error {
	c.items.Delete(key)
	return nil
}

// BatchDelete 批量删除
func (c *MemoryCache) BatchDelete(prefix string) error {
	c.items.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasPrefix(key, prefix) {
			c.items.Delete(key)
		}
		return true
	})
	return nil
}

// gc 定时清除已过期缓存
func (c *MemoryCache) gc() {
	for range c.gcTicker.C {
		c.clearExpired()
	}
}

func (c *MemoryCache) clearExpired() {
	c.items.Range(func(k, v interface{}) bool {
		cacheItem := v.(*item)
		if cacheItem.isExpire() {
			c.items.Delete(k)
		}
		return true
	})
}

func (c *MemoryCache) isItemExpired(key string) bool {
	val, ok := c.items.Load(key)
	if !ok {
		return true
	}
	cacheItem := val.(*item)
	return cacheItem.isExpire()
}
