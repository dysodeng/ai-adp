# Gateway Service Registration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Register the ai-adp service to the gateway via etcd using `github.com/dysodeng/gateway/sdk`, with configuration driven by existing `.env` GATEWAY_* variables.

**Architecture:** Add a `Gateway` config struct that reads GATEWAY_* env vars, create a provider that initializes the gateway SDK registry, wire it into the DI graph, and hook registration into the app lifecycle (register on start, close on shutdown).

**Tech Stack:** `github.com/dysodeng/gateway/sdk`, `github.com/dysodeng/gateway/sdk/etcd`, Google Wire DI

---

## Chunk 1: Configuration & Gateway Registration

### Task 1: Add Gateway Config Struct

**Files:**
- Create: `internal/infrastructure/config/gateway.go`

- [ ] **Step 1: Create gateway config file**

```go
package config

import "github.com/spf13/viper"

// Gateway 网关注册配置
type Gateway struct {
	Enabled  bool       `mapstructure:"enabled"`
	Type     string     `mapstructure:"type"` // etcd
	Etcd     GatewayEtcd `mapstructure:"etcd"`
}

// GatewayEtcd etcd网关注册配置
type GatewayEtcd struct {
	Endpoints []string `mapstructure:"endpoints"`
	Prefix    string   `mapstructure:"prefix"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

func gatewayBindEnv(v *viper.Viper) {
	_ = v.BindEnv("enabled", "GATEWAY_ENABLED")
	_ = v.BindEnv("type", "GATEWAY_TYPE")
	_ = v.BindEnv("etcd.endpoints", "GATEWAY_ETCD_ENDPOINTS")
	_ = v.BindEnv("etcd.prefix", "GATEWAY_ETCD_PREFIX")
	_ = v.BindEnv("etcd.username", "GATEWAY_ETCD_USERNAME")
	_ = v.BindEnv("etcd.password", "GATEWAY_ETCD_PASSWORD")

	v.SetDefault("enabled", false)
	v.SetDefault("type", "etcd")
	v.SetDefault("etcd.prefix", "/services/")
}
```

- [ ] **Step 2: Add Gateway field to Config struct**

Modify: `internal/infrastructure/config/config.go`

Add `Gateway` field to `Config` struct:
```go
type Config struct {
	App      App            `mapstructure:"app"`
	Server   Server         `mapstructure:"server"`
	Security Security       `mapstructure:"security"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    Redis          `mapstructure:"redis"`
	Cache    Cache          `mapstructure:"cache"`
	Monitor  Monitor        `mapstructure:"monitor"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Gateway  Gateway        `mapstructure:"gateway"`
}
```

Add gateway config loading in `Load()` function, after the monitor config block (line 104, before `setDefaults`):
```go
var gatewayConfig Gateway
if gateway := v.Sub("gateway"); gateway != nil {
	gatewayBindEnv(gateway)
	if err := gateway.Unmarshal(&gatewayConfig); err != nil {
		return nil, err
	}
}
```

Add `cfg.Gateway = gatewayConfig` to the assignment block (after `cfg.Cache = cacheConfig` at line 123):
```go
cfg.Gateway = gatewayConfig
```

- [ ] **Step 3: Add gateway section to configs/app.yaml**

Append to `configs/app.yaml`:
```yaml
# 网关注册配置
gateway:
  enabled: false
  type: etcd
  etcd:
    endpoints:
      - "127.0.0.1:2379"
    prefix: "/services/"
    username: ""
    password: ""
```

- [ ] **Step 4: Verify config compiles**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/infrastructure/config/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/config/gateway.go internal/infrastructure/config/config.go configs/app.yaml
git commit -m "feat(config): add gateway registration config"
```

---

### Task 2: Add Gateway SDK Dependency & Create Provider

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/di/provider/gateway.go`

- [ ] **Step 1: Add gateway SDK dependency**

Run: `cd /Users/dysodeng/project/go/ai-adp && go get github.com/dysodeng/gateway/sdk@latest`
Expected: go.mod updated with `github.com/dysodeng/gateway/sdk`

- [ ] **Step 2: Create gateway provider**

```go
package provider

import (
	"fmt"
	"strings"

	"github.com/dysodeng/gateway/sdk"
	"github.com/dysodeng/gateway/sdk/etcd"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// GatewayRegistry 网关注册器包装，用于 DI 注入
type GatewayRegistry struct {
	Registry sdk.Registry
	Config   *config.Config
}

// ProvideGatewayRegistry 提供网关注册器
func ProvideGatewayRegistry(cfg *config.Config) (*GatewayRegistry, error) {
	if !cfg.Gateway.Enabled {
		return &GatewayRegistry{Config: cfg}, nil
	}

	switch cfg.Gateway.Type {
	case "etcd":
		return provideEtcdRegistry(cfg)
	default:
		return nil, fmt.Errorf("unsupported gateway type: %s", cfg.Gateway.Type)
	}
}

func provideEtcdRegistry(cfg *config.Config) (*GatewayRegistry, error) {
	endpoints := cfg.Gateway.Etcd.Endpoints
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("gateway etcd endpoints is empty")
	}

	// 处理逗号分隔的 endpoints（兼容环境变量单字符串格式）
	var parsedEndpoints []string
	for _, ep := range endpoints {
		for _, e := range strings.Split(ep, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				parsedEndpoints = append(parsedEndpoints, e)
			}
		}
	}

	opts := []etcd.Option{
		etcd.WithPrefix(cfg.Gateway.Etcd.Prefix),
	}
	if cfg.Gateway.Etcd.Username != "" {
		opts = append(opts, etcd.WithAuth(cfg.Gateway.Etcd.Username, cfg.Gateway.Etcd.Password))
	}

	registry, err := etcd.NewRegistry(parsedEndpoints, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gateway registry failed: %w", err)
	}

	return &GatewayRegistry{
		Registry: registry,
		Config:   cfg,
	}, nil
}
```

- [ ] **Step 2: Verify provider compiles**

Run: `cd /Users/dysodeng/project/go/ai-adp && go mod tidy && go build ./internal/di/provider/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum internal/di/provider/gateway.go
git commit -m "feat(gateway): add gateway SDK dependency and registry provider"
```

---

### Task 3: Wire Gateway into DI and App Lifecycle

**Files:**
- Modify: `internal/di/infrastructure.go`
- Modify: `internal/di/app.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Add gateway provider to InfrastructureSet**

Modify `internal/di/infrastructure.go`, add `provider.ProvideGatewayRegistry` to `InfrastructureSet`:

```go
var InfrastructureSet = wire.NewSet(
	provider.ProvideConfig,
	provider.ProvideMonitor,
	provider.ProvideLogger,
	provider.ProvideDB,
	provider.ProvideRedis,
	provider.ProvideCache,
	port.NewMockToolService,
	agentService.NewAgentBuilder,
	provider.ProvideAgentFactory,
	// 取消能力组件
	cancel.NewMemoryTaskRegistry,
	cancel.NewRedisCancelBroadcaster,
	// 网关注册
	provider.ProvideGatewayRegistry,
)
```

- [ ] **Step 2: Add gateway to App struct and NewApp**

Modify `internal/di/app.go`:

Add imports to `internal/di/app.go`:
- `"fmt"`
- `"github.com/dysodeng/gateway/sdk"`
- `"github.com/dysodeng/ai-adp/internal/di/provider"`

Add field to `App` struct:
```go
gatewayRegistry *provider.GatewayRegistry
```

Add parameter to `NewApp` function and assign it:
```go
func NewApp(
	cfg *config.Config,
	monitor *telemetry.Monitor,
	logger *zap.Logger,
	txManager transactions.TransactionManager,
	redisClient redis.Client,
	cache cache.Cache,
	handlerRegistry *ifaceHTTP.HandlerRegistry,
	httpServer *http.Server,
	healthServer *health.Server,
	_ port.ToolService,
	_ agentService.AgentBuilder,
	_ *adapter.AgentFactory,
	// 取消能力组件
	cancelBroadcaster executor.CancelBroadcaster,
	taskRegistry executor.TaskRegistry,
	// 网关注册
	gatewayRegistry *provider.GatewayRegistry,
) *App {
	return &App{
		// ... existing fields ...
		gatewayRegistry: gatewayRegistry,
	}
}
```

Add `RegisterGateway` method to `App`:
```go
// RegisterGateway 注册服务到网关
func (a *App) RegisterGateway(ctx context.Context) error {
	if a.gatewayRegistry == nil || a.gatewayRegistry.Registry == nil {
		return nil
	}

	instances := buildServiceInstances(a.cfg)
	for _, instance := range instances {
		if err := a.gatewayRegistry.Registry.Register(ctx, instance); err != nil {
			return fmt.Errorf("register service [%s] to gateway failed: %w", instance.Name, err)
		}
	}
	return nil
}
```

Add `buildServiceInstances` helper (in `app.go`):
```go
func buildServiceInstances(cfg *config.Config) []sdk.ServiceInstance {
	var instances []sdk.ServiceInstance
	serviceName := cfg.App.Name

	if cfg.Server.HTTP.Enabled {
		instances = append(instances, sdk.ServiceInstance{
			Name:    serviceName,
			Host:    cfg.Server.HTTP.Host,
			Port:    cfg.Server.HTTP.Port,
			Version: cfg.Monitor.ServiceVersion,
			Metadata: map[string]string{
				"protocol": "http",
				"env":      cfg.App.Environment,
			},
		})
	}

	if cfg.Server.GRPC.Enabled {
		instances = append(instances, sdk.ServiceInstance{
			Name:    serviceName,
			Host:    cfg.Server.GRPC.Host,
			Port:    cfg.Server.GRPC.Port,
			Version: cfg.Monitor.ServiceVersion,
			Metadata: map[string]string{
				"protocol": "grpc",
				"env":      cfg.App.Environment,
			},
		})
	}

	return instances
}
```

Update `Stop` method — insert at the beginning, before `redis.Close()`:
```go
func (a *App) Stop(ctx context.Context) error {
	if a.gatewayRegistry != nil && a.gatewayRegistry.Registry != nil {
		_ = a.gatewayRegistry.Registry.Close()
	}
	redis.Close()
	_ = logger.ZapLogger().Sync()
	return nil
}
```

- [ ] **Step 3: Call RegisterGateway in app.go serve()**

Modify `cmd/app/app.go`, add gateway registration in `serve()` after server start:

```go
func (app *application) serve() {
	logger.Info(app.ctx, "starting application servers...")

	app.registerServer(
		app.mainApp.HTTPServer,
		app.mainApp.HealthServer,
	)

	// 启动取消信号订阅
	if err := app.mainApp.StartCancelSubscriber(app.ctx); err != nil {
		logger.Error(app.ctx, "failed to start cancel subscriber", logger.ErrorField(err))
	}

	for _, s := range app.servers {
		if err := s.Start(); err != nil {
			logger.Fatal(app.ctx, fmt.Sprintf("%s server failed to start", s.Name()), logger.ErrorField(err))
		}
		logger.Info(app.ctx, fmt.Sprintf("%s server started", s.Name()), logger.AddField("addr", s.Addr()))
	}

	// 注册服务到网关
	if err := app.mainApp.RegisterGateway(app.ctx); err != nil {
		logger.Error(app.ctx, "failed to register service to gateway", logger.ErrorField(err))
	}
}
```

- [ ] **Step 4: Regenerate Wire**

Run: `cd /Users/dysodeng/project/go/ai-adp && wire ./internal/di/`
Expected: `wire_gen.go` regenerated successfully

- [ ] **Step 5: Verify full build**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add internal/di/infrastructure.go internal/di/app.go internal/di/wire_gen.go cmd/app/app.go
git commit -m "feat(gateway): wire gateway registration into app lifecycle"
```
