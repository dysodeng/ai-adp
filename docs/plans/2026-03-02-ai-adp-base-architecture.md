# AI-ADP Base Architecture Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 搭建 AI 应用开发平台的 DDD 基础骨架，包含目录结构、共享类型、基础设施连接、DI 框架、HTTP 服务器，以及 `tenant` 完整 bounded context 作为其他 context 的模板。

**Architecture:** 严格四层 DDD（domain / application / infrastructure / interfaces），7 个 bounded context（tenant / model / prompt / conversation / agent / knowledge / usage），Eino 封装在 infrastructure/ai/ 层，domain 层通过端口接口解耦。参考 dysodeng/app 风格，使用 Google Wire 编译期依赖注入。

**Tech Stack:** Go 1.25, Gin, GORM + PostgreSQL (UUID v7), Redis, Google Wire, Viper, Zap, OpenTelemetry, Eino (cloudwego/eino), testify, go.uber.org/mock

---

## Task 1: 更新 go.mod，添加核心依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加所有核心依赖**

```bash
cd /path/to/ai-adp

go get github.com/gin-gonic/gin@latest
go get gorm.io/gorm@latest
go get gorm.io/driver/postgres@latest
go get github.com/redis/go-redis/v9@latest
go get github.com/google/wire@latest
go get github.com/spf13/viper@latest
go get github.com/spf13/cobra@latest
go get go.uber.org/zap@latest
go get github.com/google/uuid@latest
go get github.com/stretchr/testify@latest
go get go.uber.org/mock@latest
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/trace@latest
```

**Step 2: 验证依赖可解析**

```bash
go mod tidy
```

Expected: 无报错，`go.sum` 生成

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add core dependencies"
```

---

## Task 2: 创建目录骨架

**Files:**
- Create: 以下所有目录（含 `.gitkeep`）

**Step 1: 创建完整目录结构**

```bash
mkdir -p cmd/app
mkdir -p configs
mkdir -p var/logs
mkdir -p scripts
mkdir -p api

# domain 层
mkdir -p internal/domain/shared/port
mkdir -p internal/domain/shared/errors
mkdir -p internal/domain/shared/valueobject
mkdir -p internal/domain/tenant/{model,valueobject,repository,service,errors}
mkdir -p internal/domain/model/{model,valueobject,repository,service,errors}
mkdir -p internal/domain/prompt/{model,valueobject,repository,service,errors}
mkdir -p internal/domain/conversation/{model,valueobject,repository,service,errors}
mkdir -p internal/domain/agent/{model,valueobject,repository,errors}
mkdir -p internal/domain/knowledge/{model,valueobject,repository,service,errors}
mkdir -p internal/domain/usage/{model,valueobject,repository,errors}

# application 层
mkdir -p internal/application/tenant/{dto,service,event,decorator}
mkdir -p internal/application/model/{dto,service,event}
mkdir -p internal/application/prompt/{dto,service,event}
mkdir -p internal/application/conversation/{dto,service,event}
mkdir -p internal/application/agent/{dto,service}
mkdir -p internal/application/knowledge/{dto,service,event}
mkdir -p internal/application/usage/{dto,service}

# infrastructure 层
mkdir -p internal/infrastructure/ai/{provider/domestic,chain,embedding}
mkdir -p internal/infrastructure/persistence/{entity,repository/tenant,repository/model,transactions}
mkdir -p internal/infrastructure/persistence/vector
mkdir -p internal/infrastructure/cache
mkdir -p internal/infrastructure/search
mkdir -p internal/infrastructure/config
mkdir -p internal/infrastructure/event
mkdir -p internal/infrastructure/server

# interfaces 层
mkdir -p internal/interfaces/http/{middleware,handler}

# DI 层
mkdir -p internal/di/modules

# 为空目录添加 .gitkeep
find internal -type d -empty -exec touch {}/.gitkeep \;
touch var/logs/.gitkeep
```

**Step 2: Commit**

```bash
git add .
git commit -m "chore: scaffold directory structure"
```

---

## Task 3: 共享值对象与领域错误基类

**Files:**
- Create: `internal/domain/shared/errors/domain_error.go`
- Create: `internal/domain/shared/errors/domain_error_test.go`
- Create: `internal/domain/shared/valueobject/pagination.go`
- Create: `internal/domain/shared/valueobject/pagination_test.go`

**Step 1: 编写 DomainError 测试**

```go
// internal/domain/shared/errors/domain_error_test.go
package errors_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"
)

func TestDomainError_Error(t *testing.T) {
    err := sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
    assert.Equal(t, "tenant not found", err.Error())
    assert.Equal(t, "TENANT_NOT_FOUND", err.Code())
}

func TestDomainError_Is(t *testing.T) {
    err := sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
    same := sharederrors.New("TENANT_NOT_FOUND", "different message")
    different := sharederrors.New("OTHER_ERROR", "other error")

    assert.True(t, sharederrors.Is(err, "TENANT_NOT_FOUND"))
    assert.True(t, sharederrors.Is(same, "TENANT_NOT_FOUND"))
    assert.False(t, sharederrors.Is(different, "TENANT_NOT_FOUND"))
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/domain/shared/errors/...
```

Expected: FAIL — `package not found`

**Step 3: 实现 DomainError**

```go
// internal/domain/shared/errors/domain_error.go
package errors

// DomainError 领域错误基类
type DomainError struct {
    code    string
    message string
}

func New(code, message string) *DomainError {
    return &DomainError{code: code, message: message}
}

func (e *DomainError) Error() string { return e.message }
func (e *DomainError) Code() string  { return e.code }

// Is 按错误码判断
func Is(err error, code string) bool {
    de, ok := err.(*DomainError)
    if !ok {
        return false
    }
    return de.code == code
}
```

**Step 4: 编写 Pagination 测试**

```go
// internal/domain/shared/valueobject/pagination_test.go
package valueobject_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
)

func TestPagination_Offset(t *testing.T) {
    p := valueobject.NewPagination(2, 20)
    assert.Equal(t, 20, p.Offset())
    assert.Equal(t, 20, p.Limit())
}

func TestPagination_DefaultLimit(t *testing.T) {
    p := valueobject.NewPagination(1, 0)
    assert.Equal(t, 20, p.Limit()) // 默认 limit 20
}
```

**Step 5: 实现 Pagination**

```go
// internal/domain/shared/valueobject/pagination.go
package valueobject

const defaultLimit = 20

type Pagination struct {
    page  int
    limit int
}

func NewPagination(page, limit int) Pagination {
    if page < 1 {
        page = 1
    }
    if limit <= 0 {
        limit = defaultLimit
    }
    return Pagination{page: page, limit: limit}
}

func (p Pagination) Offset() int { return (p.page - 1) * p.limit }
func (p Pagination) Limit() int  { return p.limit }
func (p Pagination) Page() int   { return p.page }
```

**Step 6: 运行测试确认通过**

```bash
go test ./internal/domain/shared/...
```

Expected: PASS

**Step 7: Commit**

```bash
git add internal/domain/shared/
git commit -m "feat(domain): add shared domain errors and pagination value object"
```

---

## Task 4: LLM 执行端口接口（domain 层）

**Files:**
- Create: `internal/domain/shared/port/llm_executor.go`
- Create: `internal/domain/shared/port/embedder.go`

**Step 1: 创建 LLM 端口接口**

```go
// internal/domain/shared/port/llm_executor.go
package port

import "context"

// Message LLM 消息
type Message struct {
    Role    string // "system" | "user" | "assistant"
    Content string
}

// LLMResponse LLM 响应
type LLMResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
    Model        string
}

// StreamChunk 流式输出片段
type StreamChunk struct {
    Content string
    Done    bool
    Error   error
}

// LLMExecutor LLM 执行端口（由 infrastructure/ai 实现）
type LLMExecutor interface {
    Execute(ctx context.Context, messages []Message) (*LLMResponse, error)
    Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
}
```

**Step 2: 创建 Embedder 端口接口**

```go
// internal/domain/shared/port/embedder.go
package port

import "context"

// Embedder 向量化端口（由 infrastructure/ai/embedding 实现）
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

**Step 3: 验证编译**

```bash
go build ./internal/domain/shared/...
```

Expected: 无报错

**Step 4: Commit**

```bash
git add internal/domain/shared/port/
git commit -m "feat(domain): add LLM executor and embedder port interfaces"
```

---

## Task 5: 配置层（Viper）

**Files:**
- Create: `configs/app.yaml`
- Create: `internal/infrastructure/config/config.go`
- Create: `internal/infrastructure/config/config_test.go`

**Step 1: 编写配置测试**

```go
// internal/infrastructure/config/config_test.go
package config_test

import (
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func TestLoad_FromFile(t *testing.T) {
    // 写临时配置文件
    content := `
app:
  name: ai-adp-test
  env: test
  port: 8080
database:
  host: localhost
  port: 5432
  name: ai_adp_test
  user: postgres
  password: secret
redis:
  addr: localhost:6379
  db: 0
`
    f, err := os.CreateTemp("", "config-*.yaml")
    require.NoError(t, err)
    defer os.Remove(f.Name())
    _, _ = f.WriteString(content)
    f.Close()

    cfg, err := config.Load(f.Name())
    require.NoError(t, err)
    assert.Equal(t, "ai-adp-test", cfg.App.Name)
    assert.Equal(t, 8080, cfg.App.Port)
    assert.Equal(t, "localhost", cfg.Database.Host)
    assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/config/...
```

Expected: FAIL

**Step 3: 实现 Config 结构与加载**

```go
// internal/infrastructure/config/config.go
package config

import "github.com/spf13/viper"

type Config struct {
    App      AppConfig      `mapstructure:"app"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
}

type AppConfig struct {
    Name string `mapstructure:"name"`
	Environment  string `mapstructure:"environment"`
    Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Name     string `mapstructure:"name"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

func Load(path string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(path)
    v.AutomaticEnv()
    if err := v.ReadInConfig(); err != nil {
        return nil, err
    }
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

**Step 4: 创建默认配置文件**

```yaml
# configs/app.yaml
app:
  name: ai-adp
  env: development
  port: 8080

database:
  host: localhost
  port: 5432
  name: ai_adp
  user: postgres
  password: ""
  ssl_mode: disable

redis:
  addr: localhost:6379
  password: ""
  db: 0
```

**Step 5: 运行测试确认通过**

```bash
go test ./internal/infrastructure/config/...
```

Expected: PASS

**Step 6: Commit**

```bash
git add configs/ internal/infrastructure/config/
git commit -m "feat(infra): add config layer with Viper"
```

---

## Task 6: 基础设施 — PostgreSQL 连接 + 基础 GORM 实体

**Files:**
- Create: `internal/infrastructure/persistence/entity/base.go`
- Create: `internal/infrastructure/persistence/entity/base_test.go`
- Create: `internal/infrastructure/persistence/db.go`

**Step 1: 编写 Base Entity 测试**

```go
// internal/infrastructure/persistence/entity/base_test.go
package entity_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

func TestBase_IDGeneratedOnCreate(t *testing.T) {
    b := &entity.Base{}
    assert.Empty(t, b.ID)

    err := b.GenerateID()
    assert.NoError(t, err)
    assert.NotEmpty(t, b.ID)
    assert.Len(t, b.ID, 36) // UUID v7 标准格式
}

func TestBase_IDNotOverwrittenIfSet(t *testing.T) {
    b := &entity.Base{ID: "existing-id"}
    err := b.GenerateID()
    assert.NoError(t, err)
    assert.Equal(t, "existing-id", b.ID)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/persistence/entity/...
```

Expected: FAIL

**Step 3: 实现 Base Entity**

```go
// internal/infrastructure/persistence/entity/base.go
package entity

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Base 所有 GORM 实体的基类，主键使用 UUID v7
type Base struct {
    ID        string         `gorm:"type:varchar(36);primaryKey"`
    CreatedAt time.Time      `gorm:"autoCreateTime"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GenerateID 生成 UUID v7，供 BeforeCreate hook 和测试使用
func (b *Base) GenerateID() error {
    if b.ID != "" {
        return nil
    }
    id, err := uuid.NewV7()
    if err != nil {
        return err
    }
    b.ID = id.String()
    return nil
}

// BeforeCreate GORM hook：自动生成 UUID v7 主键
func (b *Base) BeforeCreate(tx *gorm.DB) error {
    return b.GenerateID()
}
```

**Step 4: 实现 DB 连接**

```go
// internal/infrastructure/persistence/db.go
package persistence

import (
    "fmt"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func NewDB(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.Name,
        cfg.Database.SSLMode,
    )
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Warn),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    return db, nil
}
```

**Step 5: 运行测试**

```bash
go test ./internal/infrastructure/persistence/entity/...
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/infrastructure/persistence/
git commit -m "feat(infra): add PostgreSQL connection and base entity with UUID v7"
```

---

## Task 7: 基础设施 — Redis 连接 + 事务管理器

**Files:**
- Create: `internal/infrastructure/cache/redis.go`
- Create: `internal/infrastructure/persistence/transactions/manager.go`

**Step 1: 实现 Redis 客户端**

```go
// internal/infrastructure/cache/redis.go
package cache

import (
    "fmt"

    "github.com/redis/go-redis/v9"
    "github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     cfg.Redis.Addr,
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })
    // 连接检测推迟到应用启动时执行
    if client == nil {
        return nil, fmt.Errorf("failed to create redis client")
    }
    return client, nil
}
```

**Step 2: 实现事务管理器**

```go
// internal/infrastructure/persistence/transactions/manager.go
package transactions

import (
    "context"

    "gorm.io/gorm"
)

// Manager 数据库事务管理器
type Manager struct {
    db *gorm.DB
}

func NewManager(db *gorm.DB) *Manager {
    return &Manager{db: db}
}

// WithTransaction 在事务中执行 fn，自动提交或回滚
func (m *Manager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        txCtx := context.WithValue(ctx, txKey{}, tx)
        return fn(txCtx)
    })
}

// DB 从 ctx 取事务 DB，否则返回普通 DB
func (m *Manager) DB(ctx context.Context) *gorm.DB {
    if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
        return tx
    }
    return m.db.WithContext(ctx)
}

type txKey struct{}
```

**Step 3: 验证编译**

```bash
go build ./internal/infrastructure/...
```

Expected: 无报错

**Step 4: Commit**

```bash
git add internal/infrastructure/cache/ internal/infrastructure/persistence/transactions/
git commit -m "feat(infra): add Redis client and transaction manager"
```

---

## Task 8: HTTP Server + 基础中间件

**Files:**
- Create: `internal/infrastructure/server/http.go`
- Create: `internal/interfaces/http/middleware/request_id.go`
- Create: `internal/interfaces/http/middleware/recovery.go`
- Create: `internal/interfaces/http/handler/health.go`
- Create: `internal/interfaces/http/handler/health_test.go`

**Step 1: 编写健康检查 handler 测试**

```go
// internal/interfaces/http/handler/health_test.go
package handler_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
)

func TestHealthCheck(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.GET("/health", handler.HealthCheck)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest(http.MethodGet, "/health", nil)
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), `"status":"ok"`)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/interfaces/http/handler/...
```

Expected: FAIL

**Step 3: 实现 HealthCheck handler**

```go
// internal/interfaces/http/handler/health.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
```

**Step 4: 实现中间件**

```go
// internal/interfaces/http/middleware/request_id.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

const RequestIDKey = "X-Request-ID"

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.GetHeader(RequestIDKey)
        if id == "" {
            uid, _ := uuid.NewV7()
            id = uid.String()
        }
        c.Set(RequestIDKey, id)
        c.Header(RequestIDKey, id)
        c.Next()
    }
}
```

```go
// internal/interfaces/http/middleware/recovery.go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
    return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
        c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
            "code":    "INTERNAL_ERROR",
            "message": "internal server error",
        })
    })
}
```

**Step 5: 实现 HTTP Server**

```go
// internal/infrastructure/server/http.go
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

type HTTPServer struct {
    server *http.Server
}

func NewHTTPServer(cfg *config.Config) *HTTPServer {
    r := gin.New()
    r.Use(middleware.Recovery(), middleware.RequestID())

    // 健康检查（不需要认证）
    r.GET("/health", handler.HealthCheck)

    return &HTTPServer{
        server: &http.Server{
            Addr:         fmt.Sprintf(":%d", cfg.App.Port),
            Handler:      r,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 60 * time.Second,
        },
    }
}

func (s *HTTPServer) Start() error {
    return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
    return s.server.Shutdown(ctx)
}

func (s *HTTPServer) Engine() *gin.Engine {
    return s.server.Handler.(*gin.Engine)
}
```

**Step 6: 运行测试**

```bash
go test ./internal/interfaces/http/...
```

Expected: PASS

**Step 7: Commit**

```bash
git add internal/infrastructure/server/ internal/interfaces/http/
git commit -m "feat(server): add HTTP server with health check and base middleware"
```

---

## Task 9: cmd 入口 + 应用主结构

**Files:**
- Create: `cmd/app/root.go`
- Create: `cmd/app/serve.go`
- Create: `main.go`

**Step 1: 实现 cobra 命令**

```go
// cmd/app/root.go
package app

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "ai-adp",
    Short: "AI Development Platform",
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.AddCommand(serveCmd)
}
```

```go
// cmd/app/serve.go
package app

import (
    "log"

    "github.com/spf13/cobra"
    "github.com/dysodeng/ai-adp/internal/di"
)

var configPath string

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the API server",
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := di.InitApp(configPath)
        if err != nil {
            return err
        }
        return app.Run()
    },
}

func init() {
    serveCmd.Flags().StringVarP(&configPath, "config", "c", "configs/app.yaml", "config file path")
    _ = log.Writer()
}
```

**Step 2: 实现 main.go**

```go
// main.go
package main

import (
    "log"

    "github.com/dysodeng/ai-adp/cmd/app"
)

func main() {
    if err := app.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

**Step 3: 验证编译（DI 尚未实现，预期报错）**

```bash
go build ./... 2>&1 | head -5
```

Expected: 编译错误，`di` 包未找到（正常，Task 10 实现）

**Step 4: Commit**

```bash
git add cmd/ main.go
git commit -m "feat: add cobra CLI entry point"
```

---

## Task 10: Google Wire DI 框架搭建

**Files:**
- Create: `internal/di/app.go`
- Create: `internal/di/infrastructure.go`
- Create: `internal/di/module.go`
- Create: `internal/di/wire.go`

**Step 1: 创建 App 主结构**

```go
// internal/di/app.go
package di

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/dysodeng/ai-adp/internal/infrastructure/server"
)

// App 应用主结构，持有所有顶层依赖
type App struct {
    httpServer *server.HTTPServer
}

func NewApp(httpServer *server.HTTPServer) *App {
    return &App{httpServer: httpServer}
}

// Run 启动并阻塞直到收到退出信号
func (a *App) Run() error {
    errCh := make(chan error, 1)
    go func() {
        fmt.Println("HTTP server started")
        if err := a.httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    select {
    case err := <-errCh:
        return err
    case <-quit:
        ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
        defer cancel()
        return a.httpServer.Shutdown(ctx)
    }
}
```

**Step 2: 创建基础设施 Wire Set**

```go
// internal/di/infrastructure.go
package di

import (
    "github.com/google/wire"
    "github.com/dysodeng/ai-adp/internal/infrastructure/cache"
    "github.com/dysodeng/ai-adp/internal/infrastructure/config"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
    "github.com/dysodeng/ai-adp/internal/infrastructure/server"
)

var InfrastructureSet = wire.NewSet(
    config.Load,
    persistence.NewDB,
    cache.NewRedisClient,
    transactions.NewManager,
    server.NewHTTPServer,
)
```

**Step 3: 创建模块 Wire Set（初期为空占位）**

```go
// internal/di/module.go
package di

import "github.com/google/wire"

// ModulesSet 各 bounded context 的 Wire Set 聚合
// 随着 context 实现逐步添加
var ModulesSet = wire.NewSet()
```

**Step 4: 创建 Wire 入口**

```go
// internal/di/wire.go
//go:build wireinject

package di

import (
    "github.com/google/wire"
)

// InitApp 由 Wire 生成，返回完整装配好的 App
func InitApp(configPath string) (*App, error) {
    wire.Build(
        InfrastructureSet,
        ModulesSet,
        NewApp,
    )
    return nil, nil
}
```

**Step 5: 运行 Wire 生成代码**

```bash
cd internal/di && wire
```

Expected: 生成 `wire_gen.go`

> 注意：`config.Load` 签名是 `Load(path string) (*Config, error)`，Wire 会将 `configPath string` 参数透传。

**Step 6: 验证编译**

```bash
go build ./...
```

Expected: 无报错

**Step 7: Commit**

```bash
git add internal/di/
git commit -m "feat(di): add Google Wire DI framework with app and infrastructure sets"
```

---

## Task 11: Tenant Bounded Context — Domain 层

**Files:**
- Create: `internal/domain/tenant/model/tenant.go`
- Create: `internal/domain/tenant/model/tenant_test.go`
- Create: `internal/domain/tenant/model/workspace.go`
- Create: `internal/domain/tenant/valueobject/status.go`
- Create: `internal/domain/tenant/repository/tenant_repo.go`
- Create: `internal/domain/tenant/errors/errors.go`

**Step 1: 编写 Tenant 聚合根测试**

```go
// internal/domain/tenant/model/tenant_test.go
package model_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
    "github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
)

func TestNewTenant_Valid(t *testing.T) {
    tenant, err := model.NewTenant("Acme Corp", "admin@acme.com")
    assert.NoError(t, err)
    assert.Equal(t, "Acme Corp", tenant.Name())
    assert.Equal(t, valueobject.StatusActive, tenant.Status())
    assert.NotEmpty(t, tenant.ID())
}

func TestNewTenant_EmptyName(t *testing.T) {
    _, err := model.NewTenant("", "admin@acme.com")
    assert.Error(t, err)
}

func TestTenant_Deactivate(t *testing.T) {
    tenant, _ := model.NewTenant("Acme Corp", "admin@acme.com")
    tenant.Deactivate()
    assert.Equal(t, valueobject.StatusInactive, tenant.Status())
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/domain/tenant/...
```

Expected: FAIL

**Step 3: 实现值对象**

```go
// internal/domain/tenant/valueobject/status.go
package valueobject

type TenantStatus string

const (
    StatusActive   TenantStatus = "active"
    StatusInactive TenantStatus = "inactive"
    StatusSuspended TenantStatus = "suspended"
)
```

**Step 4: 实现 Tenant 聚合根**

```go
// internal/domain/tenant/model/tenant.go
package model

import (
    "fmt"

    "github.com/google/uuid"
    "github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
)

// Tenant 租户聚合根
type Tenant struct {
    id     string
    name   string
    email  string
    status valueobject.TenantStatus
}

func NewTenant(name, email string) (*Tenant, error) {
    if name == "" {
        return nil, fmt.Errorf("tenant name cannot be empty")
    }
    id, err := uuid.NewV7()
    if err != nil {
        return nil, err
    }
    return &Tenant{
        id:     id.String(),
        name:   name,
        email:  email,
        status: valueobject.StatusActive,
    }, nil
}

// Reconstitute 从持久化数据重建（不生成新 ID）
func Reconstitute(id, name, email string, status valueobject.TenantStatus) *Tenant {
    return &Tenant{id: id, name: name, email: email, status: status}
}

func (t *Tenant) ID() string                       { return t.id }
func (t *Tenant) Name() string                     { return t.name }
func (t *Tenant) Email() string                    { return t.email }
func (t *Tenant) Status() valueobject.TenantStatus { return t.status }

func (t *Tenant) Deactivate() { t.status = valueobject.StatusInactive }
func (t *Tenant) Activate()   { t.status = valueobject.StatusActive }
func (t *Tenant) Suspend()    { t.status = valueobject.StatusSuspended }
```

**Step 5: 实现 Workspace 实体**

```go
// internal/domain/tenant/model/workspace.go
package model

import (
    "fmt"

    "github.com/google/uuid"
)

// Workspace 工作空间实体（归属于 Tenant）
type Workspace struct {
    id       string
    tenantID string
    name     string
    slug     string
}

func NewWorkspace(tenantID, name, slug string) (*Workspace, error) {
    if name == "" || slug == "" {
        return nil, fmt.Errorf("workspace name and slug cannot be empty")
    }
    id, err := uuid.NewV7()
    if err != nil {
        return nil, err
    }
    return &Workspace{
        id:       id.String(),
        tenantID: tenantID,
        name:     name,
        slug:     slug,
    }, nil
}

func (w *Workspace) ID() string       { return w.id }
func (w *Workspace) TenantID() string { return w.tenantID }
func (w *Workspace) Name() string     { return w.name }
func (w *Workspace) Slug() string     { return w.slug }
```

**Step 6: 定义仓储接口和领域错误**

```go
// internal/domain/tenant/repository/tenant_repo.go
package repository

import (
    "context"

    "github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
    "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
)

type TenantRepository interface {
    Save(ctx context.Context, tenant *model.Tenant) error
    FindByID(ctx context.Context, id string) (*model.Tenant, error)
    FindAll(ctx context.Context, pagination valueobject.Pagination) ([]*model.Tenant, int64, error)
    Delete(ctx context.Context, id string) error
}

type WorkspaceRepository interface {
    Save(ctx context.Context, workspace *model.Workspace) error
    FindByID(ctx context.Context, id string) (*model.Workspace, error)
    FindByTenantID(ctx context.Context, tenantID string) ([]*model.Workspace, error)
}
```

```go
// internal/domain/tenant/errors/errors.go
package errors

import sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
    ErrTenantNotFound   = sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
    ErrWorkspaceNotFound = sharederrors.New("WORKSPACE_NOT_FOUND", "workspace not found")
    ErrTenantInactive   = sharederrors.New("TENANT_INACTIVE", "tenant is inactive")
)
```

**Step 7: 运行测试**

```bash
go test ./internal/domain/tenant/...
```

Expected: PASS

**Step 8: Commit**

```bash
git add internal/domain/tenant/
git commit -m "feat(domain/tenant): add tenant aggregate root, workspace entity, repository interfaces"
```

---

## Task 12: Tenant Bounded Context — Infrastructure 层

**Files:**
- Create: `internal/infrastructure/persistence/entity/tenant.go`
- Create: `internal/infrastructure/persistence/repository/tenant/tenant_repo_impl.go`
- Create: `internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go`

**Step 1: 创建 GORM 实体**

```go
// internal/infrastructure/persistence/entity/tenant.go
package entity

// TenantEntity GORM 映射实体
type TenantEntity struct {
    Base
    Name   string `gorm:"type:varchar(255);not null"`
    Email  string `gorm:"type:varchar(255);not null"`
    Status string `gorm:"type:varchar(50);not null;default:'active'"`
}

func (TenantEntity) TableName() string { return "tenants" }

// WorkspaceEntity GORM 映射实体
type WorkspaceEntity struct {
    Base
    TenantID string `gorm:"type:varchar(36);not null;index"`
    Name     string `gorm:"type:varchar(255);not null"`
    Slug     string `gorm:"type:varchar(100);not null;uniqueIndex"`
}

func (WorkspaceEntity) TableName() string { return "workspaces" }
```

**Step 2: 编写仓储实现测试（使用 Mock DB）**

```go
// internal/infrastructure/persistence/repository/tenant/tenant_repo_impl_test.go
package tenant_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    tenantmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
    tenantrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/tenant"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    err = db.AutoMigrate(&entity.TenantEntity{})
    require.NoError(t, err)
    return db
}

func TestTenantRepo_SaveAndFind(t *testing.T) {
    db := setupTestDB(t)
    repo := tenantrepo.NewTenantRepository(db)

    tenant, err := tenantmodel.NewTenant("Acme Corp", "admin@acme.com")
    require.NoError(t, err)

    err = repo.Save(context.Background(), tenant)
    require.NoError(t, err)

    found, err := repo.FindByID(context.Background(), tenant.ID())
    require.NoError(t, err)
    assert.Equal(t, "Acme Corp", found.Name())
}
```

> 注意：此测试需要 `gorm.io/driver/sqlite`。运行前先 `go get gorm.io/driver/sqlite@latest`

**Step 3: 实现仓储**

```go
// internal/infrastructure/persistence/repository/tenant/tenant_repo_impl.go
package tenant

import (
    "context"
    "errors"

    "gorm.io/gorm"
    "github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
    domainmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
    domainerrors "github.com/dysodeng/ai-adp/internal/domain/tenant/errors"
    tenantvo "github.com/dysodeng/ai-adp/internal/domain/tenant/valueobject"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

type TenantRepositoryImpl struct {
    db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepositoryImpl {
    return &TenantRepositoryImpl{db: db}
}

func (r *TenantRepositoryImpl) Save(ctx context.Context, tenant *domainmodel.Tenant) error {
    e := &entity.TenantEntity{}
    e.ID = tenant.ID()
    e.Name = tenant.Name()
    e.Email = tenant.Email()
    e.Status = string(tenant.Status())
    return r.db.WithContext(ctx).Save(e).Error
}

func (r *TenantRepositoryImpl) FindByID(ctx context.Context, id string) (*domainmodel.Tenant, error) {
    var e entity.TenantEntity
    if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, domainerrors.ErrTenantNotFound
        }
        return nil, err
    }
    return domainmodel.Reconstitute(e.ID, e.Name, e.Email, tenantvo.TenantStatus(e.Status)), nil
}

func (r *TenantRepositoryImpl) FindAll(ctx context.Context, pagination valueobject.Pagination) ([]*domainmodel.Tenant, int64, error) {
    var entities []entity.TenantEntity
    var total int64
    if err := r.db.WithContext(ctx).Model(&entity.TenantEntity{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }
    if err := r.db.WithContext(ctx).Offset(pagination.Offset()).Limit(pagination.Limit()).Find(&entities).Error; err != nil {
        return nil, 0, err
    }
    tenants := make([]*domainmodel.Tenant, 0, len(entities))
    for _, e := range entities {
        tenants = append(tenants, domainmodel.Reconstitute(e.ID, e.Name, e.Email, tenantvo.TenantStatus(e.Status)))
    }
    return tenants, total, nil
}

func (r *TenantRepositoryImpl) Delete(ctx context.Context, id string) error {
    return r.db.WithContext(ctx).Delete(&entity.TenantEntity{}, "id = ?", id).Error
}
```

**Step 4: 添加 sqlite 测试依赖并运行测试**

```bash
go get gorm.io/driver/sqlite@latest
go test ./internal/infrastructure/persistence/repository/tenant/...
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/persistence/entity/tenant.go \
        internal/infrastructure/persistence/repository/tenant/ \
        go.mod go.sum
git commit -m "feat(infra/tenant): add tenant GORM entity and repository implementation"
```

---

## Task 13: Tenant Bounded Context — Application 层

**Files:**
- Create: `internal/application/tenant/dto/tenant_dto.go`
- Create: `internal/application/tenant/service/tenant_app_service.go`
- Create: `internal/application/tenant/service/tenant_app_service_test.go`

**Step 1: 创建 DTO**

```go
// internal/application/tenant/dto/tenant_dto.go
package dto

// CreateTenantCommand 创建租户命令
type CreateTenantCommand struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

// TenantResult 租户查询结果
type TenantResult struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Email  string `json:"email"`
    Status string `json:"status"`
}

// ListTenantsQuery 分页查询参数
type ListTenantsQuery struct {
    Page  int `form:"page"`
    Limit int `form:"limit"`
}
```

**Step 2: 编写应用服务测试（Mock Repository）**

```go
// internal/application/tenant/service/tenant_app_service_test.go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
    "github.com/dysodeng/ai-adp/internal/application/tenant/dto"
    "github.com/dysodeng/ai-adp/internal/application/tenant/service"
    mockrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository/mock"
)

func TestTenantAppService_Create(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mockrepo.NewMockTenantRepository(ctrl)
    mockRepo.EXPECT().
        Save(gomock.Any(), gomock.Any()).
        Return(nil).
        Times(1)

    svc := service.NewTenantAppService(mockRepo)
    result, err := svc.Create(context.Background(), dto.CreateTenantCommand{
        Name:  "Acme Corp",
        Email: "admin@acme.com",
    })

    require.NoError(t, err)
    assert.Equal(t, "Acme Corp", result.Name)
    assert.Equal(t, "active", result.Status)
}
```

**Step 3: 生成 Mock（使用 mockgen）**

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=internal/domain/tenant/repository/tenant_repo.go \
        -destination=internal/domain/tenant/repository/mock/mock_tenant_repo.go \
        -package=mockrepo
```

**Step 4: 运行测试确认失败**

```bash
go test ./internal/application/tenant/...
```

Expected: FAIL — `service` 包未实现

**Step 5: 实现应用服务**

```go
// internal/application/tenant/service/tenant_app_service.go
package service

import (
    "context"

    "github.com/dysodeng/ai-adp/internal/application/tenant/dto"
    domainmodel "github.com/dysodeng/ai-adp/internal/domain/tenant/model"
    domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
    "github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
)

type TenantAppService struct {
    tenantRepo domainrepo.TenantRepository
}

func NewTenantAppService(tenantRepo domainrepo.TenantRepository) *TenantAppService {
    return &TenantAppService{tenantRepo: tenantRepo}
}

func (s *TenantAppService) Create(ctx context.Context, cmd dto.CreateTenantCommand) (*dto.TenantResult, error) {
    tenant, err := domainmodel.NewTenant(cmd.Name, cmd.Email)
    if err != nil {
        return nil, err
    }
    if err := s.tenantRepo.Save(ctx, tenant); err != nil {
        return nil, err
    }
    return toTenantResult(tenant), nil
}

func (s *TenantAppService) GetByID(ctx context.Context, id string) (*dto.TenantResult, error) {
    tenant, err := s.tenantRepo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return toTenantResult(tenant), nil
}

func (s *TenantAppService) List(ctx context.Context, q dto.ListTenantsQuery) ([]*dto.TenantResult, int64, error) {
    pagination := valueobject.NewPagination(q.Page, q.Limit)
    tenants, total, err := s.tenantRepo.FindAll(ctx, pagination)
    if err != nil {
        return nil, 0, err
    }
    results := make([]*dto.TenantResult, 0, len(tenants))
    for _, t := range tenants {
        results = append(results, toTenantResult(t))
    }
    return results, total, nil
}

func toTenantResult(t *domainmodel.Tenant) *dto.TenantResult {
    return &dto.TenantResult{
        ID:     t.ID(),
        Name:   t.Name(),
        Email:  t.Email(),
        Status: string(t.Status()),
    }
}
```

**Step 6: 运行测试**

```bash
go test ./internal/application/tenant/...
```

Expected: PASS

**Step 7: Commit**

```bash
git add internal/application/tenant/ internal/domain/tenant/repository/mock/
git commit -m "feat(app/tenant): add tenant application service with DTOs and mock tests"
```

---

## Task 14: Tenant HTTP Handler + Wire 接入

**Files:**
- Create: `internal/interfaces/http/handler/tenant_handler.go`
- Create: `internal/interfaces/http/handler/tenant_handler_test.go`
- Modify: `internal/infrastructure/server/http.go`
- Modify: `internal/di/module.go`
- Modify: `internal/di/wire.go`

**Step 1: 编写 Tenant Handler 测试**

```go
// internal/interfaces/http/handler/tenant_handler_test.go
package handler_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "go.uber.org/mock/gomock"
    "github.com/dysodeng/ai-adp/internal/application/tenant/dto"
    "github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
    mocksvc "github.com/dysodeng/ai-adp/internal/application/tenant/service/mock"
)

func TestTenantHandler_Create(t *testing.T) {
    gin.SetMode(gin.TestMode)
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockSvc := mocksvc.NewMockTenantAppServiceInterface(ctrl)
    mockSvc.EXPECT().
        Create(gomock.Any(), gomock.Any()).
        Return(&dto.TenantResult{ID: "uuid-1", Name: "Acme", Status: "active"}, nil)

    h := handler.NewTenantHandler(mockSvc)
    r := gin.New()
    r.POST("/tenants", h.Create)

    body, _ := json.Marshal(map[string]string{"name": "Acme", "email": "admin@acme.com"})
    w := httptest.NewRecorder()
    req, _ := http.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)
    assert.Contains(t, w.Body.String(), "uuid-1")
}
```

**Step 2: 定义应用服务接口，生成 Mock**

```go
// 在 internal/application/tenant/service/tenant_app_service.go 顶部添加接口
type TenantAppServiceInterface interface {
    Create(ctx context.Context, cmd dto.CreateTenantCommand) (*dto.TenantResult, error)
    GetByID(ctx context.Context, id string) (*dto.TenantResult, error)
    List(ctx context.Context, q dto.ListTenantsQuery) ([]*dto.TenantResult, int64, error)
}
```

```bash
mockgen -source=internal/application/tenant/service/tenant_app_service.go \
        -destination=internal/application/tenant/service/mock/mock_tenant_svc.go \
        -package=mocksvc
```

**Step 3: 实现 Tenant Handler**

```go
// internal/interfaces/http/handler/tenant_handler.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/dysodeng/ai-adp/internal/application/tenant/dto"
    "github.com/dysodeng/ai-adp/internal/application/tenant/service"
)

type TenantHandler struct {
    svc service.TenantAppServiceInterface
}

func NewTenantHandler(svc service.TenantAppServiceInterface) *TenantHandler {
    return &TenantHandler{svc: svc}
}

func (h *TenantHandler) Create(c *gin.Context) {
    var cmd dto.CreateTenantCommand
    if err := c.ShouldBindJSON(&cmd); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
        return
    }
    result, err := h.svc.Create(c.Request.Context(), cmd)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, result)
}

func (h *TenantHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    result, err := h.svc.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": err.Error()})
        return
    }
    c.JSON(http.StatusOK, result)
}

func (h *TenantHandler) List(c *gin.Context) {
    var q dto.ListTenantsQuery
    _ = c.ShouldBindQuery(&q)
    results, total, err := h.svc.List(c.Request.Context(), q)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": results, "total": total})
}

// RegisterRoutes 注册租户路由
func (h *TenantHandler) RegisterRoutes(rg *gin.RouterGroup) {
    tenants := rg.Group("/tenants")
    tenants.POST("", h.Create)
    tenants.GET("", h.List)
    tenants.GET("/:id", h.GetByID)
}
```

**Step 4: 将租户 Handler 注册到 HTTP Server**

修改 `internal/infrastructure/server/http.go` 的 `NewHTTPServer`，接受 `handlers` 参数：

```go
// NewHTTPServer 增加 handlers 注册回调
func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler) *HTTPServer {
    r := gin.New()
    r.Use(middleware.Recovery(), middleware.RequestID())
    r.GET("/health", handler.HealthCheck)

    v1 := r.Group("/api/v1")
    tenantHandler.RegisterRoutes(v1)

    return &HTTPServer{
        server: &http.Server{
            Addr:         fmt.Sprintf(":%d", cfg.App.Port),
            Handler:      r,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 60 * time.Second,
        },
    }
}
```

**Step 5: 更新 DI Module Set**

```go
// internal/di/module.go
package di

import (
    "github.com/google/wire"
    tenantrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/tenant"
    tenantappsvc "github.com/dysodeng/ai-adp/internal/application/tenant/service"
    tenanthandler "github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
    domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
)

var TenantModuleSet = wire.NewSet(
    tenantrepo.NewTenantRepository,
    wire.Bind(new(domainrepo.TenantRepository), new(*tenantrepo.TenantRepositoryImpl)),
    tenantappsvc.NewTenantAppService,
    wire.Bind(new(tenantappsvc.TenantAppServiceInterface), new(*tenantappsvc.TenantAppService)),
    tenanthandler.NewTenantHandler,
)

var ModulesSet = wire.NewSet(
    TenantModuleSet,
)
```

**Step 6: 重新生成 Wire**

```bash
cd internal/di && wire
```

**Step 7: 运行所有测试**

```bash
go test ./...
```

Expected: PASS

**Step 8: Commit**

```bash
git add internal/interfaces/http/handler/ \
        internal/infrastructure/server/http.go \
        internal/di/ \
        internal/application/tenant/service/mock/
git commit -m "feat(tenant): complete tenant bounded context with HTTP handler and DI wiring"
```

---

## Task 15: 数据库 Migration 支持

**Files:**
- Create: `internal/infrastructure/persistence/migration/migrate.go`
- Create: `cmd/app/migrate.go`

**Step 1: 实现 AutoMigrate**

```go
// internal/infrastructure/persistence/migration/migrate.go
package migration

import (
    "gorm.io/gorm"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

// AutoMigrate 自动建表（开发/测试环境使用）
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &entity.TenantEntity{},
        &entity.WorkspaceEntity{},
    )
}
```

**Step 2: 添加 migrate 命令**

```go
// cmd/app/migrate.go
package app

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/dysodeng/ai-adp/internal/infrastructure/config"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence"
    "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/migration"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Run database migrations",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load(configPath)
        if err != nil {
            return err
        }
        db, err := persistence.NewDB(cfg)
        if err != nil {
            return err
        }
        if err := migration.AutoMigrate(db); err != nil {
            return err
        }
        fmt.Println("Migration completed successfully")
        return nil
    },
}

func init() {
    migrateCmd.Flags().StringVarP(&configPath, "config", "c", "configs/app.yaml", "config file path")
    rootCmd.AddCommand(migrateCmd)
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

Expected: 无报错

**Step 4: Commit**

```bash
git add internal/infrastructure/persistence/migration/ cmd/app/migrate.go
git commit -m "feat(infra): add database migration command"
```

---

## Task 16: 最终验证

**Step 1: 运行全量测试**

```bash
go test ./... -v -count=1
```

Expected: 全部 PASS

**Step 2: 验证完整编译**

```bash
go build -o bin/ai-adp ./...
```

Expected: 生成 `bin/ai-adp` 二进制

**Step 3: 冒烟测试（需本地 PostgreSQL + Redis）**

```bash
# 启动
./bin/ai-adp serve -c configs/app.yaml &

# 健康检查
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# 创建租户
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Tenant","email":"test@example.com"}'
# Expected: 201 + JSON with id

kill %1
```

**Step 4: 最终 Commit**

```bash
git add bin/ .gitignore  # 确保 bin/ 在 .gitignore 中
git commit -m "feat: complete base architecture skeleton

- DDD 4-layer: domain/application/infrastructure/interfaces
- 7 bounded context directories scaffolded
- Tenant context fully implemented as template
- PostgreSQL (UUID v7) + Redis + Wire DI
- HTTP server (Gin) with health check, request ID, recovery middleware
- TDD throughout with gomock
"
```

---

## 架构验证清单

完成后确认：
- [ ] `domain` 层无任何 `infrastructure` 或 `application` 包的 import
- [ ] 所有 repository 在 domain 层定义接口，infrastructure 层实现
- [ ] 所有测试使用 mock 而非真实数据库
- [ ] Wire 生成代码无手动修改
- [ ] `go vet ./...` 无警告
- [ ] UUID v7 作为所有实体主键
