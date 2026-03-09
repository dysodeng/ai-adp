# 基础设施 pkg 包缓存与 Redis 迁移实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将缓存和 Redis 从旧的 `internal/infrastructure/cache` 实现替换为新的 `internal/infrastructure/pkg/cache` 和 `internal/infrastructure/pkg/redis` 实现，清理旧代码和旧配置。

**Architecture:** 新的 `pkg/redis` 包支持 standalone/cluster/sentinel 三种部署模式，提供 main/cache/mq 三个独立 Redis 实例。新的 `pkg/cache` 包提供统一的 `Cache` 接口，支持 memory 和 redis 两种驱动。`RedisCancelBroadcaster` 需要从旧的 `*redis.Client` 迁移到新的 `redis.Client` 接口。

**Tech Stack:** Go, Wire (DI), go-redis/v9

---

### Task 1: 更新 RedisCancelBroadcaster 使用新的 redis.Client 接口

**Files:**
- Modify: `internal/infrastructure/agent/cancel/redis_cancel_broadcaster.go`

**Step 1: 修改 RedisCancelBroadcaster 的 client 字段和构造函数**

将 `*redis.Client`（go-redis 直接类型）替换为 `redis.Client`（新 pkg 接口，它嵌入了 `redis.UniversalClient`）。

```go
package cancel

import (
	"context"

	pkgredis "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

const cancelChannel = "agent:task:cancel"

// RedisCancelBroadcaster 基于 Redis Pub/Sub 的取消信号广播器
type RedisCancelBroadcaster struct {
	client pkgredis.Client
}

// NewRedisCancelBroadcaster 创建 Redis 取消广播器
func NewRedisCancelBroadcaster(client pkgredis.Client) *RedisCancelBroadcaster {
	return &RedisCancelBroadcaster{client: client}
}

func (b *RedisCancelBroadcaster) Broadcast(ctx context.Context, taskID string) error {
	if err := b.client.Publish(ctx, cancelChannel, taskID).Err(); err != nil {
		logger.Warn(ctx, "[CancelBroadcaster] broadcast failed", logger.ErrorField(err))
		return err
	}
	return nil
}

func (b *RedisCancelBroadcaster) Subscribe(ctx context.Context, registry executor.TaskRegistry) error {
	pubsub := b.client.Subscribe(ctx, cancelChannel)

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				taskID := msg.Payload
				registry.Cancel(taskID)
			}
		}
	}()

	return nil
}

var _ executor.CancelBroadcaster = (*RedisCancelBroadcaster)(nil)
```

**Step 2: 验证编译通过**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/infrastructure/agent/cancel/...`
Expected: 编译成功，无错误

**Step 3: 提交**

```bash
git add internal/infrastructure/agent/cancel/redis_cancel_broadcaster.go
git commit -m "refactor: migrate RedisCancelBroadcaster to pkg/redis.Client interface"
```

---

### Task 2: 删除旧的 cache/redis.go（NewRedisClient）

**Files:**
- Delete: `internal/infrastructure/cache/redis.go`

**Step 1: 删除旧的 NewRedisClient 文件**

```bash
rm internal/infrastructure/cache/redis.go
```

**Step 2: 检查 cache 目录是否还有其他文件**

```bash
ls internal/infrastructure/cache/
```

如果目录为空，删除整个目录：
```bash
rmdir internal/infrastructure/cache/
```

**Step 3: 提交**

```bash
git add -A internal/infrastructure/cache/
git commit -m "refactor: remove legacy cache/redis.go (replaced by pkg/redis)"
```

---

### Task 3: 清理旧的 RedisConfig 结构体和相关默认值

**Files:**
- Modify: `internal/infrastructure/config/config.go`

**Step 1: 删除旧的 RedisConfig 结构体**

从 `config.go` 中移除：

```go
// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`      // 连接池大小，默认 10
	MinIdleConns int    `mapstructure:"min_idle_conns"` // 最小空闲连接数，默认 5
}
```

**Step 2: 清理 setDefaults 中旧的 redis 默认值**

移除 `setDefaults` 函数中的：
```go
v.SetDefault("redis.pool_size", 10)
v.SetDefault("redis.min_idle_conns", 5)
```

**Step 3: 清理 bindEnvKeys 中旧的 redis 环境变量绑定**

移除 `bindEnvKeys` 函数中的：
```go
"redis.addr":           "REDIS_ADDR",
"redis.password":       "REDIS_PASSWORD",
"redis.db":             "REDIS_DB",
```

**Step 4: 验证编译通过**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/infrastructure/config/...`
Expected: 编译成功

**Step 5: 提交**

```bash
git add internal/infrastructure/config/config.go
git commit -m "refactor: remove legacy RedisConfig struct and old redis defaults"
```

---

### Task 4: 更新配置测试

**Files:**
- Modify: `internal/infrastructure/config/config_test.go`

**Step 1: 更新测试中的 YAML 配置和断言**

将旧的 redis 扁平配置替换为新的结构化配置：

`testConfigYAML` 中的 redis 部分替换为：
```yaml
redis:
  main:
    mode: standalone
    host: 127.0.0.1
    port: "6379"
    password: ""
    db: 0
    pool:
      min_idle_conns: 10
      max_retries: 3
      pool_size: 100
  cache:
    mode: standalone
    host: 127.0.0.1
    port: "6379"
    password: ""
    db: 1
    pool:
      min_idle_conns: 10
      max_retries: 3
      pool_size: 100
```

更新断言：
```go
// Redis
assert.Equal(t, "standalone", cfg.Redis.Main.Mode)
assert.Equal(t, "127.0.0.1", cfg.Redis.Main.Host)
assert.Equal(t, 100, cfg.Redis.Main.Pool.PoolSize)
assert.Equal(t, 1, cfg.Redis.Cache.DB)
```

在 `TestLoad_Defaults` 中，将 redis 部分的 minimal config 替换为：
```yaml
redis:
  main:
    host: 127.0.0.1
    port: "6379"
```

更新默认值断言：
```go
assert.Equal(t, 100, cfg.Redis.Main.Pool.PoolSize)
```

**Step 2: 运行测试**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/config/... -v`
Expected: PASS

**Step 3: 提交**

```bash
git add internal/infrastructure/config/config_test.go
git commit -m "test: update config tests for new redis structure"
```

---

### Task 5: 重新生成 Wire 依赖注入代码

**Files:**
- Modify: `internal/di/wire_gen.go`（自动生成）

**Step 1: 运行 wire 生成**

Run: `cd /Users/dysodeng/project/go/ai-adp/internal/di && wire`
Expected: 生成新的 `wire_gen.go`，使用 `provideRedis` 和 `cache.NewCache` 替代旧的 `cache.NewRedisClient`

**Step 2: 验证生成的代码**

检查 `wire_gen.go` 中：
- 不再导入 `"github.com/dysodeng/ai-adp/internal/infrastructure/cache"`
- 使用 `provideRedis(configConfig)` 获取 redis client
- 使用 `cache.NewCache(configConfig)` 创建缓存
- `NewRedisCancelBroadcaster` 接收 `redis.Client` 而非 `*redis.Client`

**Step 3: 完整构建验证**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: 编译成功

**Step 4: 提交**

```bash
git add internal/di/wire_gen.go
git commit -m "chore: regenerate wire_gen.go with new pkg/redis and pkg/cache"
```

---

### Task 6: 在 App.Stop 中关闭 Redis 连接

**Files:**
- Modify: `internal/di/app.go`

**Step 1: 在 Stop 方法中添加 Redis 关闭调用**

在 `app.go` 的 `Stop` 方法中调用 `redis.Close()`：

```go
import (
	// ... existing imports ...
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
)

func (a *App) Stop(ctx context.Context) error {
	redis.Close()
	_ = logger.ZapLogger().Sync()
	return a.tracerShutdown(ctx)
}
```

**Step 2: 验证编译通过**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: 编译成功

**Step 3: 提交**

```bash
git add internal/di/app.go
git commit -m "feat: close redis connections on app shutdown"
```

---

### Task 7: 最终验证

**Step 1: 完整构建**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: 编译成功

**Step 2: 运行所有测试**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./...`
Expected: 所有测试通过

**Step 3: 确认旧代码已完全清除**

```bash
grep -r "cache\.NewRedisClient\|RedisConfig\|cfg\.Redis\.Addr\|cfg\.Redis\.PoolSize" --include="*.go" .
```
Expected: 只在 docs/ 目录下有匹配（历史计划文档），源码中无匹配
