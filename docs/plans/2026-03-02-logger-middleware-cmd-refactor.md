# Logger、Middleware & cmd/app 重构计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 对齐 dysodeng/app 项目风格：将 Logger 改为支持 context trace 注入的包级函数模式；HTTP 中间件补充 OTel tracing + 访问日志；cmd/app 改为单文件无 cobra 直接启动模式；基础设施 Server 抽象为接口。

**Architecture:** Logger 从 DI 注入的 `*zap.Logger` 改为包级全局函数 `logger.Info(ctx, msg, ...Field)`，自动从 ctx 提取 OTel trace/span ID 注入日志字段。HTTP 中间件链增加 otelgin (创建 request span) + 访问日志中间件（使用 logger 包，带 trace）。cmd/app 改为单文件生命周期模式：initialize → serve → waitForInterruptSignal，参考 dysodeng/app/cmd/app/app.go。

**Tech Stack:** Go 1.25, Gin, Zap, OpenTelemetry (otelgin contrib), Google Wire, GORM (PostgreSQL uuid v7)

---

## Task 1：恢复 Base entity 的 PostgreSQL 默认值 + 清理 SQLite 基础设施测试

**背景：** 之前错误移除了 `Base.ID` 的 `default:uuid_generate_v7()`，该 DB 级默认值对 PostgreSQL 生产环境有意义（直接 INSERT 无需应用层）。基础设施层 repository 的集成测试不应依赖 SQLite，应使用真实 PostgreSQL；单元测试用 mock。删除 SQLite-based repository 测试文件。

**Files:**
- Modify: `internal/infrastructure/persistence/entity/base.go`
- Delete: `internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go`

**Step 1: 恢复 Base entity 的 GORM tag**

编辑 `internal/infrastructure/persistence/entity/base.go`，将 `ID` 字段改回：

```go
ID uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
```

**Step 2: 删除 SQLite-based repository 测试**

```bash
rm internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go
```

Repository 层的集成测试应在真实 PostgreSQL 环境下运行（CI/CD 集成测试），不在 SQLite 单元测试中。应用层已有 mock-based 测试覆盖业务逻辑。

**Step 3: 验证测试通过**

```bash
go test ./internal/infrastructure/persistence/... -v
go test ./... 2>&1 | grep -E "^(ok|FAIL|---)"
```

预期：所有测试 PASS，无 FAIL。

**Step 4: 提交**

```bash
git add internal/infrastructure/persistence/entity/base.go
git rm internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go
git commit -m "fix(entity): restore PostgreSQL uuid_generate_v7() default; remove SQLite-based repo tests"
```

---

## Task 2：添加 OTel Gin 中间件依赖

**Files:**
- Modify: `go.mod`, `go.sum`

**Step 1: 添加 otelgin 依赖**

```bash
go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin
go mod tidy
```

**Step 2: 验证 go.mod 包含新依赖**

```bash
grep otelgin go.mod
```

预期输出类似：
```
go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.x.x
```

**Step 3: 验证构建无误**

```bash
go build ./...
```

**Step 4: 提交**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add otelgin for HTTP OpenTelemetry tracing"
```

---

## Task 3：重构 Logger 包 — context trace 注入 + 包级函数

**背景：** 当前 logger 返回 `*zap.Logger` 通过 DI 注入，无 context 支持，无 trace 信息。改为参考项目风格：全局单例 + 包级函数 `Debug/Info/Warn/Error/Fatal/Panic(ctx, msg, ...Field)`，自动从 ctx 中提取 OTel span 信息附加到日志。

**Files:**
- Modify (rewrite): `internal/infrastructure/logger/logger.go`
- Create: `internal/infrastructure/logger/zap.go`
- Modify: `internal/infrastructure/logger/logger_test.go`

**Step 1: 编写 logger_test.go 的新测试（先写失败测试）**

替换 `internal/infrastructure/logger/logger_test.go` 内容为：

```go
package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

func TestInitLogger(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:      "debug",
		Format:     "json",
		OutputPath: "stdout",
	}
	err := logger.InitLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger.ZapLogger())
}

func TestLoggerInfo_NoTrace(t *testing.T) {
	cfg := config.LoggerConfig{Level: "debug", Format: "json", OutputPath: "stdout"}
	require.NoError(t, logger.InitLogger(cfg))

	// Should not panic with empty context (no span)
	assert.NotPanics(t, func() {
		logger.Info(context.Background(), "test message", logger.AddField("key", "value"))
	})
}

func TestLoggerError_WithErrorField(t *testing.T) {
	cfg := config.LoggerConfig{Level: "debug", Format: "json", OutputPath: "stdout"}
	require.NoError(t, logger.InitLogger(cfg))

	assert.NotPanics(t, func() {
		logger.Error(context.Background(), "test error", logger.ErrorField(assert.AnError))
	})
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/logger/... -v
```

预期：FAIL（`InitLogger`, `ZapLogger`, `AddField`, `ErrorField` 未定义）

**Step 3: 重写 internal/infrastructure/logger/logger.go**

```go
package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// Field 日志字段
type Field struct {
	Key     string
	Value   interface{}
	isError bool
}

// ErrorField 创建 error 类型日志字段
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err, isError: true}
}

// AddField 创建通用 key-value 日志字段
func AddField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// _logger 全局单例
var _logger *logger

type logger struct {
	zap *zap.Logger
}

// InitLogger 使用配置初始化全局 logger，必须在应用启动时调用一次
func InitLogger(cfg config.LoggerConfig) error {
	zl, err := newZapLogger(cfg)
	if err != nil {
		return err
	}
	_logger = &logger{zap: zl}
	return nil
}

// ZapLogger 返回底层 *zap.Logger（供需要直接使用 zap 的场景）
func ZapLogger() *zap.Logger {
	if _logger == nil {
		return zap.NewNop()
	}
	return _logger.zap
}

// Debug 记录 debug 级别日志
func Debug(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.DebugLevel, msg, fields...)
}

// Info 记录 info 级别日志
func Info(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.InfoLevel, msg, fields...)
}

// Warn 记录 warn 级别日志
func Warn(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.WarnLevel, msg, fields...)
}

// Error 记录 error 级别日志
func Error(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.ErrorLevel, msg, fields...)
}

// Fatal 记录 fatal 级别日志后调用 os.Exit(1)
func Fatal(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.FatalLevel, msg, fields...)
}

// Panic 记录 panic 级别日志后 panic
func Panic(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.PanicLevel, msg, fields...)
}

func (l *logger) log(ctx context.Context, level zapcore.Level, msg string, fields ...Field) {
	if l == nil || l.zap == nil {
		return
	}
	if ce := l.zap.Check(level, msg); ce != nil {
		zfields := l.extractTrace(ctx)
		for _, f := range fields {
			if f.isError {
				if err, ok := f.Value.(error); ok {
					zfields = append(zfields, zap.Error(err))
				}
			} else {
				zfields = append(zfields, zap.Any(f.Key, f.Value))
			}
		}
		ce.Write(zfields...)
	}
}

// extractTrace 从 context 中提取 OTel span 信息，注入 trace_id / span_id
func (l *logger) extractTrace(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return nil
	}
	sc := span.SpanContext()
	fields := make([]zap.Field, 0, 2)
	if sc.HasTraceID() {
		fields = append(fields, zap.String("trace_id", sc.TraceID().String()))
	}
	if sc.HasSpanID() {
		fields = append(fields, zap.String("span_id", sc.SpanID().String()))
	}
	return fields
}
```

**Step 4: 创建 internal/infrastructure/logger/zap.go**

```go
package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// newZapLogger 根据 LoggerConfig 构建 *zap.Logger
func newZapLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	level := parseLevel(cfg.Level)

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "file",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	if cfg.Format == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	writer, err := buildWriter(cfg.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("logger: failed to build writer: %w", err)
	}

	core := zapcore.NewCore(encoder, writer, zap.NewAtomicLevelAt(level))
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

// buildWriter 根据 OutputPath 返回对应的 WriteSyncer
// "stdout" → os.Stdout, "stderr" → os.Stderr, 其他 → 文件
func buildWriter(outputPath string) (zapcore.WriteSyncer, error) {
	switch outputPath {
	case "", "stdout":
		return zapcore.AddSync(os.Stdout), nil
	case "stderr":
		return zapcore.AddSync(os.Stderr), nil
	default:
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return zapcore.AddSync(f), nil
	}
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
```

**Step 5: 运行测试确认通过**

```bash
go test ./internal/infrastructure/logger/... -v
```

预期：3 个测试 PASS。

**Step 6: 更新 internal/di/infrastructure.go — provideLogger 改为调用 InitLogger**

将 `provideLogger` 改为调用 `logger.InitLogger` 并返回 `*zap.Logger`（Wire 图中仍保留该类型，供需要的地方使用）：

```go
// provideLogger 初始化全局 logger 并返回底层 *zap.Logger
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	if err := logger.InitLogger(cfg.Logger); err != nil {
		return nil, err
	}
	return logger.ZapLogger(), nil
}
```

注意：`infrastructure.go` 的 import 需要引入 `logger` 包，同时保留 `"go.uber.org/zap"` import。

**Step 7: 验证构建**

```bash
go build ./...
```

**Step 8: 提交**

```bash
git add internal/infrastructure/logger/logger.go \
        internal/infrastructure/logger/zap.go \
        internal/infrastructure/logger/logger_test.go \
        internal/di/infrastructure.go
git commit -m "feat(logger): refactor to package-level context-aware functions with OTel trace injection"
```

---

## Task 4：HTTP 中间件 — OTel Tracing + 访问日志

**背景：** 补充两个中间件：
1. `Tracing()` — 使用 `otelgin` 为每个请求创建 OTel span，注入 trace context 到 Gin context
2. `Logger()` — 访问日志中间件，请求完成后记录 method/path/status/latency/client_ip，通过 ctx 自动带 trace_id

**Files:**
- Create: `internal/interfaces/http/middleware/tracing.go`
- Create: `internal/interfaces/http/middleware/logger.go`
- Modify: `internal/interfaces/http/middleware/request_id.go` — 将 RequestID 注入 ctx

**Step 1: 创建 internal/interfaces/http/middleware/tracing.go**

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Tracing 为每个 HTTP 请求创建 OpenTelemetry span，传播 trace context
func Tracing(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}
```

**Step 2: 创建 internal/interfaces/http/middleware/logger.go**

```go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// Logger 记录 HTTP 访问日志，自动从 Gin context 中获取 OTel trace context
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// 使用 c.Request.Context() 以获取 otelgin 注入的 trace span
		ctx := c.Request.Context()

		fields := []logger.Field{
			logger.AddField("status", statusCode),
			logger.AddField("method", method),
			logger.AddField("path", path),
			logger.AddField("ip", clientIP),
			logger.AddField("latency_ms", latency.Milliseconds()),
		}

		if len(c.Errors) > 0 {
			logger.Error(ctx, c.Errors.String(), fields...)
		} else {
			logger.Info(ctx, "http_access", fields...)
		}
	}
}
```

**Step 3: 更新 request_id.go — 把 RequestID 注入 ctx（供下游 logger 访问）**

当前 `RequestID()` 只把 ID 放入 gin.Context，不影响 `c.Request.Context()`。如需 request_id 出现在日志字段，在 logger 中间件里从 gin key 中取即可。当前实现已够用，无需修改。

**Step 4: 验证构建**

```bash
go build ./internal/interfaces/...
```

**Step 5: 提交**

```bash
git add internal/interfaces/http/middleware/tracing.go \
        internal/interfaces/http/middleware/logger.go
git commit -m "feat(middleware): add OTel HTTP tracing middleware and access logger with trace injection"
```

---

## Task 5：抽象 Server 接口 + 重构 HTTPServer

**背景：** 参考 dysodeng/app 的 `server.Server` 接口，HTTPServer 实现该接口，使 cmd/app 能统一注册和管理多种服务器（未来扩展 gRPC/WS/HealthCheck）。

**Files:**
- Create: `internal/infrastructure/server/server.go`
- Modify (rewrite): `internal/infrastructure/server/http.go`

**Step 1: 创建 internal/infrastructure/server/server.go**

```go
package server

import "context"

// Server 服务器接口，所有服务类型（HTTP/gRPC/WS/Health）均实现此接口
type Server interface {
	// IsEnabled 是否启用此服务
	IsEnabled() bool
	// Name 服务名称，用于日志
	Name() string
	// Addr 监听地址，如 ":8080"
	Addr() string
	// Start 启动服务（非阻塞，内部启动 goroutine）
	Start() error
	// Stop 优雅停止服务
	Stop(ctx context.Context) error
}
```

**Step 2: 重写 internal/infrastructure/server/http.go**

```go
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// compile-time check
var _ Server = (*HTTPServer)(nil)

// HTTPServer HTTP 服务器，实现 Server 接口
type HTTPServer struct {
	cfg           *config.Config
	engine        *gin.Engine
	server        *http.Server
	tenantHandler *handler.TenantHandler
}

func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler) *HTTPServer {
	return &HTTPServer{cfg: cfg, tenantHandler: tenantHandler}
}

func (s *HTTPServer) IsEnabled() bool { return true }
func (s *HTTPServer) Name() string    { return "HTTP" }
func (s *HTTPServer) Addr() string    { return fmt.Sprintf(":%d", s.cfg.Server.HTTP.Port) }

// Start 初始化 Gin engine、注册中间件和路由，在 goroutine 中启动
func (s *HTTPServer) Start() error {
	if s.cfg.App.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.Recovery(),
		middleware.Tracing(s.cfg.App.Name),
		middleware.Logger(),
		middleware.RequestID(),
	)

	r.GET("/health", handler.HealthCheck)
	v1 := r.Group("/api/v1")
	s.tenantHandler.RegisterRoutes(v1)

	s.engine = r
	s.server = &http.Server{
		Addr:              s.Addr(),
		Handler:           r,
		ReadTimeout:       time.Duration(s.cfg.Server.HTTP.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(s.cfg.Server.HTTP.WriteTimeout) * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 短暂等待确认启动无立即错误
	select {
	case err := <-errCh:
		return err
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// Stop 优雅关闭 HTTP 服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// Engine 返回底层 gin.Engine（供路由注册使用）
func (s *HTTPServer) Engine() *gin.Engine {
	return s.engine
}
```

**Step 3: 验证编译**

```bash
go build ./internal/infrastructure/server/...
```

**Step 4: 提交**

```bash
git add internal/infrastructure/server/server.go \
        internal/infrastructure/server/http.go
git commit -m "feat(server): add Server interface; HTTPServer implements it with tracing+logger middleware"
```

---

## Task 6：重构 di/App — 持有 Server 引用，提供 Stop()

**背景：** 参考项目的 `di.App` 不负责生命周期，只持有各 Server 实例和清理函数。生命周期由 `cmd/app/app.go` 管理。

**Files:**
- Modify (rewrite): `internal/di/app.go`
- Modify: `internal/di/infrastructure.go` — 移除 `*zap.Logger` 不再注入 App
- Modify: `internal/di/wire.go` — 更新签名
- Regenerate: `internal/di/wire_gen.go`

**Step 1: 重写 internal/di/app.go**

```go
package di

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// App 持有所有服务实例和清理函数，由 cmd/app 驱动生命周期
type App struct {
	HTTPServer     server.Server
	tracerShutdown telemetry.ShutdownFunc
}

func NewApp(httpServer *server.HTTPServer, tracerShutdown telemetry.ShutdownFunc) *App {
	return &App{
		HTTPServer:     httpServer,
		tracerShutdown: tracerShutdown,
	}
}

// Stop 释放应用资源（tracer flush 等），在所有 Server 停止后调用
func (a *App) Stop(ctx context.Context) error {
	a.tracerShutdown()
	return nil
}
```

**Step 2: 更新 internal/di/wire.go — 接受 context.Context**

```go
//go:build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func InitApp(ctx context.Context, configPath string) (*App, error) {
	wire.Build(
		config.Load,
		InfrastructureSet,
		ModulesSet,
		NewApp,
	)
	return nil, nil
}
```

注意：`config.Load` 只接受 `string`；`ctx` 暂不注入 providers（后续扩展可绑定）。Wire 允许 `InitApp` 有额外参数，但未被 provider 消费的参数需要通过 `wire.Value` 绑定。

如果 Wire 报错 `ctx context.Context` 未使用，改为：

```go
func InitApp(_ context.Context, configPath string) (*App, error) {
	wire.Build(
		config.Load,
		InfrastructureSet,
		ModulesSet,
		NewApp,
	)
	return nil, nil
}
```

或者直接保持原有签名 `InitApp(configPath string) (*App, error)` 不变，在 `cmd/app/app.go` 内部处理 ctx。

**Step 3: 重新生成 wire_gen.go**

```bash
wire ./internal/di/
```

预期输出：
```
wire: github.com/dysodeng/ai-adp/internal/di: wrote .../internal/di/wire_gen.go
```

**Step 4: 验证构建**

```bash
go build ./...
```

**Step 5: 提交**

```bash
git add internal/di/app.go internal/di/wire.go internal/di/wire_gen.go
git commit -m "refactor(di): App holds Server interface; lifecycle moved to cmd; Stop() for cleanup"
```

---

## Task 7：重构 cmd/app — 单文件无 cobra，参考 dysodeng/app

**背景：** 当前有 root.go + serve.go + migrate.go 三个 cobra 子命令文件。参考项目是单文件 `app.go`，`Execute()` 直接启动所有服务。DB 迁移在 `di.InitApp` 内自动执行（调用 `migration.AutoMigrate`），不再作为独立子命令。

**Files:**
- Delete: `cmd/app/root.go`, `cmd/app/serve.go`, `cmd/app/migrate.go`
- Create: `cmd/app/app.go`
- Modify: `main.go`
- Modify: `internal/di/infrastructure.go` — DB 初始化后自动迁移
- Modify: `go.mod` — 移除 cobra 和 viper 的 cobra 依赖（如果不再使用 cobra）

**Step 1: 删除旧 cmd/app 文件**

```bash
rm cmd/app/root.go cmd/app/serve.go cmd/app/migrate.go
```

**Step 2: 创建 cmd/app/app.go**

```go
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dysodeng/ai-adp/internal/di"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

const defaultConfigPath = "configs/app.yaml"

type application struct {
	ctx     context.Context
	mainApp *di.App
	servers []server.Server
}

func newApplication(ctx context.Context) *application {
	return &application{ctx: ctx}
}

func (a *application) run() {
	a.initialize()
	a.serve()
	a.waitForInterruptSignal()
}

func (a *application) initialize() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	mainApp, err := di.InitApp(configPath)
	if err != nil {
		// logger 此时可能还未初始化，直接 stderr
		fmt.Fprintf(os.Stderr, "应用初始化失败: %v\n", err)
		os.Exit(1)
	}
	a.mainApp = mainApp
}

func (a *application) registerServer(servers ...server.Server) {
	for _, s := range servers {
		if s.IsEnabled() {
			a.servers = append(a.servers, s)
		}
	}
}

func (a *application) serve() {
	logger.Info(a.ctx, "starting application servers...")

	a.registerServer(
		a.mainApp.HTTPServer,
		// 未来扩展: a.mainApp.GRPCServer, a.mainApp.WSServer, a.mainApp.HealthServer
	)

	for _, s := range a.servers {
		if err := s.Start(); err != nil {
			logger.Fatal(a.ctx, fmt.Sprintf("%s server failed to start", s.Name()), logger.ErrorField(err))
		}
		logger.Info(a.ctx, fmt.Sprintf("%s server started", s.Name()), logger.AddField("addr", s.Addr()))
	}
}

func (a *application) waitForInterruptSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(a.ctx, "shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, s := range a.servers {
		if err := s.Stop(ctx); err != nil {
			logger.Error(ctx, fmt.Sprintf("%s server shutdown error", s.Name()), logger.ErrorField(err))
		} else {
			logger.Info(ctx, fmt.Sprintf("%s server stopped", s.Name()))
		}
	}

	if err := a.mainApp.Stop(ctx); err != nil {
		logger.Error(ctx, "application cleanup error", logger.ErrorField(err))
	}

	logger.Info(ctx, "application stopped gracefully")
}

// Execute 应用程序入口点
func Execute() {
	ctx := context.Background()
	newApplication(ctx).run()
}
```

**Step 3: 更新 main.go**

```go
package main

import "github.com/dysodeng/ai-adp/cmd/app"

func main() {
	app.Execute()
}
```

**Step 4: 将 AutoMigrate 集成到 DB 初始化流程**

在 `internal/di/infrastructure.go` 中，将 `persistence.NewDB` 包装成带迁移的 provider：

```go
// provideDB 初始化 DB 连接并自动执行迁移
func provideDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := persistence.NewDB(cfg)
	if err != nil {
		return nil, err
	}
	if err := migration.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}
	return db, nil
}
```

`InfrastructureSet` 中用 `provideDB` 代替 `persistence.NewDB`：

```go
var InfrastructureSet = wire.NewSet(
	provideDB,       // was: persistence.NewDB
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
)
```

> 注意：`provideDB` 需要 import `gorm.io/gorm`、`github.com/dysodeng/ai-adp/internal/infrastructure/persistence`、`migration`。

**Step 5: 重新生成 wire_gen.go**

```bash
wire ./internal/di/
```

**Step 6: 验证构建**

```bash
go build ./...
```

**Step 7: 移除 cobra 依赖（如果不再有其他地方使用）**

```bash
grep -r "cobra" --include="*.go" .
```

如果没有其他使用，从 `go.mod` require 中移除并 tidy：

```bash
go mod tidy
```

**Step 8: 运行所有测试**

```bash
go test ./... 2>&1 | grep -v "^ld: warning"
```

预期：全部 PASS，无 FAIL。

**Step 9: 提交**

```bash
git add cmd/app/app.go main.go \
        internal/di/infrastructure.go internal/di/wire.go internal/di/wire_gen.go \
        go.mod go.sum
git commit -m "refactor(cmd): replace cobra subcommands with single app.go; auto-migrate on startup; follow dysodeng/app pattern"
```

---

## Task 8：全量验证

**Step 1: 构建验证**

```bash
go build ./...
go vet ./...
```

预期：无错误。

**Step 2: 全量测试**

```bash
go test ./... 2>&1 | grep -v "^ld: warning"
```

预期：所有测试包 PASS，无 FAIL。

**Step 3: 静态检查（可选）**

```bash
golangci-lint run
```

**Step 4: 最终提交（如有遗漏的修改）**

```bash
git add -A
git commit -m "chore: final cleanup after logger/middleware/cmd refactor"
```

---

## 变更汇总

| 文件 | 操作 | 说明 |
|---|---|---|
| `internal/infrastructure/persistence/entity/base.go` | 修改 | 恢复 `default:uuid_generate_v7()` |
| `internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go` | 修改 | SQLite 用原生 CREATE TABLE |
| `go.mod` / `go.sum` | 修改 | 添加 otelgin |
| `internal/infrastructure/logger/logger.go` | 重写 | 包级函数 + OTel trace 注入 |
| `internal/infrastructure/logger/zap.go` | 新建 | Zap 配置封装 |
| `internal/interfaces/http/middleware/tracing.go` | 新建 | otelgin HTTP tracing |
| `internal/interfaces/http/middleware/logger.go` | 新建 | 访问日志（含 trace） |
| `internal/infrastructure/server/server.go` | 新建 | Server 接口 |
| `internal/infrastructure/server/http.go` | 重写 | 实现 Server 接口 |
| `internal/di/app.go` | 重写 | 持有 Server 引用，Stop() |
| `internal/di/wire.go` | 修改 | 更新注入签名 |
| `internal/di/wire_gen.go` | 重新生成 | wire 再生成 |
| `internal/di/infrastructure.go` | 修改 | provideDB with migrate, provideLogger |
| `cmd/app/root.go` | 删除 | 移除 cobra |
| `cmd/app/serve.go` | 删除 | 移除 cobra |
| `cmd/app/migrate.go` | 删除 | 移除独立 migrate 命令 |
| `cmd/app/app.go` | 新建 | 单文件生命周期管理 |
| `main.go` | 修改 | 简化为 `app.Execute()` |
