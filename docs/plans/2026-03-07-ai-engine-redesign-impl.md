# AI 引擎重设计实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重新设计 AI 引擎层，删除旧的底层组件封装，建立以 App 为核心的统一 Executor 架构，一期实现 Agent/Chat/TextGeneration 三种应用类型。

**Architecture:** 新建 `domain/app/` bounded context 定义 AI 应用聚合根（含版本管理），新建统一 `AppExecutor` 端口接口，在 `infrastructure/ai/engine/` 中按 App 类型实现 3 种 Executor（ChatExecutor/TextGenExecutor/AgentExecutor），通过 ExecutorFactory 动态创建。删除旧的 `LLMExecutor`/`AgentExecutor` 端口和对应适配器。

**Tech Stack:** Go 1.25, Eino v0.7.37 (ADK), Eino-ext (openai/ark/ollama/claude), Google Wire, GORM (PostgreSQL), UUID v7

---

## Task 1：删除旧 AI 端口和基础设施代码

**Files:**
- Delete: `internal/domain/shared/port/llm_executor.go`
- Delete: `internal/domain/shared/port/agent_executor.go`
- Delete: `internal/domain/shared/port/agent_executor_test.go`
- Delete: `internal/infrastructure/ai/llm/adapter.go`
- Delete: `internal/infrastructure/ai/llm/adapter_test.go`
- Delete: `internal/infrastructure/ai/llm/factory.go`
- Delete: `internal/infrastructure/ai/agent/agent.go`
- Delete: `internal/infrastructure/ai/agent/agent_test.go`
- Delete: `internal/infrastructure/ai/provider.go`

**Step 1: 删除旧文件**

```bash
rm internal/domain/shared/port/llm_executor.go
rm internal/domain/shared/port/agent_executor.go
rm internal/domain/shared/port/agent_executor_test.go
rm internal/infrastructure/ai/llm/adapter.go
rm internal/infrastructure/ai/llm/adapter_test.go
rm internal/infrastructure/ai/llm/factory.go
rm internal/infrastructure/ai/agent/agent.go
rm internal/infrastructure/ai/agent/agent_test.go
rm internal/infrastructure/ai/provider.go
```

**Step 2: 更新 DI — 移除 AIComponents**

修改 `internal/di/infrastructure.go`，从 `InfrastructureSet` 中移除 `ai.NewComponents`：

```go
var InfrastructureSet = wire.NewSet(
	provideDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
)
```

同时移除 import 中的 `ai "github.com/dysodeng/ai-adp/internal/infrastructure/ai"`.

**Step 3: 更新 DI — 简化 App 结构体**

修改 `internal/di/app.go`，移除 `AIComponents` 字段：

```go
package di

import (
	"context"

	"go.uber.org/zap"

	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// App 应用生命周期容器
type App struct {
	HTTPServer     server.Server
	tracerShutdown telemetry.ShutdownFunc
}

func NewApp(httpServer *server.HTTPServer, _ *zap.Logger, tracerShutdown telemetry.ShutdownFunc) *App {
	return &App{
		HTTPServer:     httpServer,
		tracerShutdown: tracerShutdown,
	}
}

func (a *App) Stop(ctx context.Context) error {
	_ = logger.ZapLogger().Sync()
	return a.tracerShutdown(ctx)
}
```

**Step 4: 重新生成 wire_gen.go**

```bash
wire ./internal/di/
```

**Step 5: 验证编译**

```bash
go build ./...
```

预期：编译通过（embedding 包不受影响）

**Step 6: 提交**

```bash
git add -A
git commit -m "refactor: remove old AI ports and infrastructure (LLMExecutor, AgentExecutor, provider)"
```

---

## Task 2：创建 App 领域层 — 值对象

**Files:**
- Create: `internal/domain/app/valueobject/app_type.go`
- Create: `internal/domain/app/valueobject/version_status.go`
- Create: `internal/domain/app/valueobject/app_config.go`

**Step 1: 创建 AppType 值对象**

创建 `internal/domain/app/valueobject/app_type.go`：

```go
package valueobject

// AppType AI 应用类型
type AppType string

const (
	AppTypeAgent          AppType = "agent"           // Agent (ReAct)：LLM + 工具调用推理循环
	AppTypeChat           AppType = "chat"            // 纯多轮对话，无工具/知识库
	AppTypeTextGeneration AppType = "text_generation" // 单次文本生成，无对话上下文
	AppTypeChatFlow       AppType = "chat_flow"       // 条件分支对话流（二期）
	AppTypeWorkflow       AppType = "workflow"        // 确定性 DAG 工作流（二期）
)

func (t AppType) IsValid() bool {
	switch t {
	case AppTypeAgent, AppTypeChat, AppTypeTextGeneration, AppTypeChatFlow, AppTypeWorkflow:
		return true
	}
	return false
}

func (t AppType) String() string { return string(t) }
```

**Step 2: 创建 VersionStatus 值对象**

创建 `internal/domain/app/valueobject/version_status.go`：

```go
package valueobject

// VersionStatus 应用版本状态
type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "draft"     // 草稿（可编辑）
	VersionStatusPublished VersionStatus = "published" // 已发布（可执行）
	VersionStatusArchived  VersionStatus = "archived"  // 已归档（历史版本）
)

func (s VersionStatus) IsValid() bool {
	switch s {
	case VersionStatusDraft, VersionStatusPublished, VersionStatusArchived:
		return true
	}
	return false
}

func (s VersionStatus) String() string { return string(s) }
```

**Step 3: 创建 AppConfig 值对象**

创建 `internal/domain/app/valueobject/app_config.go`：

```go
package valueobject

import (
	"encoding/json"

	"github.com/google/uuid"
)

// AppConfig AI 应用配置（JSON 序列化存储）
type AppConfig struct {
	ModelID            uuid.UUID        `json:"model_id"`                       // 引用 ModelConfig
	SystemPrompt       string           `json:"system_prompt"`                  // 系统提示词，支持 {{变量名}} 占位符
	Temperature        *float32         `json:"temperature,omitempty"`          // 覆盖模型默认温度
	MaxTokens          int              `json:"max_tokens,omitempty"`           // 覆盖模型默认 token 上限
	Tools              []ToolConfig     `json:"tools,omitempty"`                // Agent 专用：工具配置列表
	OpeningStatement   string           `json:"opening_statement,omitempty"`    // 开场白
	SuggestedQuestions []string         `json:"suggested_questions,omitempty"`  // 推荐问题
}

// ToolConfig 工具配置
type ToolConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
}

// ToJSON 序列化为 JSON 字节
func (c *AppConfig) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// AppConfigFromJSON 从 JSON 字节反序列化
func AppConfigFromJSON(data []byte) (*AppConfig, error) {
	var c AppConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
```

**Step 4: 验证编译**

```bash
go build ./internal/domain/app/...
```

**Step 5: 提交**

```bash
git add internal/domain/app/valueobject/
git commit -m "feat(domain/app): add AppType, VersionStatus, AppConfig value objects"
```

---

## Task 3：创建 App 领域层 — 聚合根、仓储接口、错误

**Files:**
- Create: `internal/domain/app/model/app.go`
- Create: `internal/domain/app/model/app_version.go`
- Create: `internal/domain/app/model/app_test.go`
- Create: `internal/domain/app/repository/app_repo.go`
- Create: `internal/domain/app/errors/errors.go`

**Step 1: 先写测试**

创建 `internal/domain/app/model/app_test.go`：

```go
package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

func TestNewApp_Valid(t *testing.T) {
	tenantID := uuid.New()
	app, err := appmodel.NewApp(tenantID, "My Chat App", "A test chat app", valueobject.AppTypeChat, "")
	require.NoError(t, err)
	assert.Equal(t, tenantID, app.TenantID())
	assert.Equal(t, "My Chat App", app.Name())
	assert.Equal(t, valueobject.AppTypeChat, app.Type())
}

func TestNewApp_EmptyName(t *testing.T) {
	_, err := appmodel.NewApp(uuid.New(), "", "desc", valueobject.AppTypeChat, "")
	assert.Error(t, err)
}

func TestNewApp_InvalidType(t *testing.T) {
	_, err := appmodel.NewApp(uuid.New(), "App", "desc", valueobject.AppType("unknown"), "")
	assert.Error(t, err)
}

func TestNewAppVersion_Valid(t *testing.T) {
	cfg := &valueobject.AppConfig{
		SystemPrompt: "you are helpful",
	}
	v, err := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, v.Version())
	assert.Equal(t, valueobject.VersionStatusDraft, v.Status())
}

func TestAppVersion_Publish(t *testing.T) {
	cfg := &valueobject.AppConfig{SystemPrompt: "test"}
	v, _ := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	v.Publish()
	assert.Equal(t, valueobject.VersionStatusPublished, v.Status())
	assert.NotNil(t, v.PublishedAt())
}

func TestAppVersion_Archive(t *testing.T) {
	cfg := &valueobject.AppConfig{SystemPrompt: "test"}
	v, _ := appmodel.NewAppVersion(uuid.New(), 1, cfg)
	v.Publish()
	v.Archive()
	assert.Equal(t, valueobject.VersionStatusArchived, v.Status())
}

func TestApp_Reconstitute(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	app := appmodel.Reconstitute(id, tenantID, "App", "desc", valueobject.AppTypeAgent, "icon.png")
	assert.Equal(t, id, app.ID())
	assert.Equal(t, tenantID, app.TenantID())
	assert.Equal(t, valueobject.AppTypeAgent, app.Type())
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/domain/app/... -v
```

预期：编译失败，包不存在

**Step 3: 创建 AppVersion 实体**

创建 `internal/domain/app/model/app_version.go`：

```go
package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppVersion AI 应用版本实体
type AppVersion struct {
	id          uuid.UUID
	appID       uuid.UUID
	version     int                       // 版本号，自增
	status      valueobject.VersionStatus // Draft | Published | Archived
	config      *valueobject.AppConfig    // 应用配置（JSON 序列化）
	publishedAt *time.Time
}

// NewAppVersion 创建新版本（初始状态为 Draft）
func NewAppVersion(appID uuid.UUID, version int, config *valueobject.AppConfig) (*AppVersion, error) {
	if version < 1 {
		return nil, fmt.Errorf("app_version: version must be >= 1")
	}
	if config == nil {
		return nil, fmt.Errorf("app_version: config cannot be nil")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("app_version: failed to generate ID: %w", err)
	}
	return &AppVersion{
		id:      id,
		appID:   appID,
		version: version,
		status:  valueobject.VersionStatusDraft,
		config:  config,
	}, nil
}

// ReconstituteVersion 从持久化数据重建版本
func ReconstituteVersion(
	id, appID uuid.UUID,
	version int,
	status valueobject.VersionStatus,
	config *valueobject.AppConfig,
	publishedAt *time.Time,
) *AppVersion {
	return &AppVersion{
		id:          id,
		appID:       appID,
		version:     version,
		status:      status,
		config:      config,
		publishedAt: publishedAt,
	}
}

// Getters
func (v *AppVersion) ID() uuid.UUID                   { return v.id }
func (v *AppVersion) AppID() uuid.UUID                { return v.appID }
func (v *AppVersion) Version() int                    { return v.version }
func (v *AppVersion) Status() valueobject.VersionStatus { return v.status }
func (v *AppVersion) Config() *valueobject.AppConfig  { return v.config }
func (v *AppVersion) PublishedAt() *time.Time         { return v.publishedAt }

// Publish 发布版本
func (v *AppVersion) Publish() {
	v.status = valueobject.VersionStatusPublished
	now := time.Now()
	v.publishedAt = &now
}

// Archive 归档版本
func (v *AppVersion) Archive() {
	v.status = valueobject.VersionStatusArchived
}

// UpdateConfig 更新配置（仅 Draft 状态可编辑）
func (v *AppVersion) UpdateConfig(config *valueobject.AppConfig) error {
	if v.status != valueobject.VersionStatusDraft {
		return fmt.Errorf("app_version: can only update config in draft status")
	}
	v.config = config
	return nil
}
```

**Step 4: 创建 App 聚合根**

创建 `internal/domain/app/model/app.go`：

```go
package model

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// App AI 应用聚合根
type App struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	description string
	appType     valueobject.AppType
	icon        string
}

// NewApp 创建新的 AI 应用
func NewApp(tenantID uuid.UUID, name, description string, appType valueobject.AppType, icon string) (*App, error) {
	if name == "" {
		return nil, fmt.Errorf("app: name cannot be empty")
	}
	if !appType.IsValid() {
		return nil, fmt.Errorf("app: invalid type %q", appType)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("app: failed to generate ID: %w", err)
	}
	return &App{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		appType:     appType,
		icon:        icon,
	}, nil
}

// Reconstitute 从持久化数据重建
func Reconstitute(id, tenantID uuid.UUID, name, description string, appType valueobject.AppType, icon string) *App {
	return &App{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		appType:     appType,
		icon:        icon,
	}
}

// Getters
func (a *App) ID() uuid.UUID             { return a.id }
func (a *App) TenantID() uuid.UUID       { return a.tenantID }
func (a *App) Name() string              { return a.name }
func (a *App) Description() string       { return a.description }
func (a *App) Type() valueobject.AppType { return a.appType }
func (a *App) Icon() string              { return a.icon }

// Setters
func (a *App) SetName(name string)               { a.name = name }
func (a *App) SetDescription(description string) { a.description = description }
func (a *App) SetIcon(icon string)               { a.icon = icon }
```

**Step 5: 创建仓储接口**

创建 `internal/domain/app/repository/app_repo.go`：

```go
package repository

import (
	"context"

	"github.com/google/uuid"

	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppRepository AI 应用仓储接口
type AppRepository interface {
	// SaveApp 保存应用
	SaveApp(ctx context.Context, app *appmodel.App) error
	// FindAppByID 按 ID 查询应用
	FindAppByID(ctx context.Context, id uuid.UUID) (*appmodel.App, error)
	// FindAppsByTenant 查询租户下所有应用
	FindAppsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*appmodel.App, error)
	// DeleteApp 删除应用
	DeleteApp(ctx context.Context, id uuid.UUID) error

	// SaveVersion 保存版本
	SaveVersion(ctx context.Context, version *appmodel.AppVersion) error
	// FindVersionByID 按 ID 查询版本
	FindVersionByID(ctx context.Context, id uuid.UUID) (*appmodel.AppVersion, error)
	// FindPublishedVersion 查询应用的已发布版本
	FindPublishedVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error)
	// FindDraftVersion 查询应用的草稿版本
	FindDraftVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error)
	// FindVersionsByApp 查询应用的所有版本
	FindVersionsByApp(ctx context.Context, appID uuid.UUID) ([]*appmodel.AppVersion, error)
	// FindVersionsByStatus 按状态查询版本
	FindVersionsByStatus(ctx context.Context, appID uuid.UUID, status valueobject.VersionStatus) ([]*appmodel.AppVersion, error)
}
```

**Step 6: 创建领域错误**

创建 `internal/domain/app/errors/errors.go`：

```go
package errors

import "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
	ErrAppNotFound           = errors.New("APP_NOT_FOUND", "app not found")
	ErrVersionNotFound       = errors.New("VERSION_NOT_FOUND", "app version not found")
	ErrNoPublishedVersion    = errors.New("NO_PUBLISHED_VERSION", "no published version for this app")
	ErrDraftAlreadyExists    = errors.New("DRAFT_ALREADY_EXISTS", "a draft version already exists for this app")
	ErrCannotEditNonDraft    = errors.New("CANNOT_EDIT_NON_DRAFT", "can only edit draft versions")
)
```

**Step 7: 运行测试**

```bash
go test ./internal/domain/app/... -v
```

预期：7 个测试 PASS

**Step 8: 提交**

```bash
git add internal/domain/app/
git commit -m "feat(domain/app): add App aggregate, AppVersion entity, repository interface, domain errors"
```

---

## Task 4：创建 AppExecutor 领域端口

**Files:**
- Create: `internal/domain/shared/port/app_executor.go`
- Modify: `internal/domain/shared/port/embedder.go` — 保留不变（仅确认）

注意：旧的 `llm_executor.go` 和 `agent_executor.go` 已在 Task 1 删除。`Message` 类型需在新文件中重新定义（或新建 `message.go`）。

**Step 1: 创建消息类型文件**

创建 `internal/domain/shared/port/message.go`：

```go
package port

// Message 通用消息类型
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}
```

**Step 2: 创建 AppExecutor 端口**

创建 `internal/domain/shared/port/app_executor.go`：

```go
package port

import "context"

// AppEventType 应用执行事件类型
type AppEventType string

const (
	AppEventMessage    AppEventType = "message"     // LLM 输出文本片段（流式）
	AppEventToolCall   AppEventType = "tool_call"   // 工具调用开始
	AppEventToolResult AppEventType = "tool_result"  // 工具调用结果
	AppEventDone       AppEventType = "done"        // 执行完成
	AppEventError      AppEventType = "error"       // 执行出错
)

// AppEvent 应用执行事件
type AppEvent struct {
	Type    AppEventType
	Content string // message/tool_call/tool_result 时的内容
	Error   error  // error 类型时非 nil
}

// AppExecutorInput 应用执行输入
type AppExecutorInput struct {
	Query     string            // 用户输入
	Variables map[string]string // 提示词变量注入（{{变量名}} → 值）
	History   []Message         // 对话历史（Chat/Agent 用）
	Stream    bool              // 是否流式输出
}

// AppResult 非流式执行结果
type AppResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
}

// AppExecutor 统一的 AI 应用执行端口
type AppExecutor interface {
	// Run 流式执行，返回事件 channel；调用方需消费至 Done 或 Error
	Run(ctx context.Context, input *AppExecutorInput) (<-chan AppEvent, error)
	// Execute 非流式执行，返回完整结果
	Execute(ctx context.Context, input *AppExecutorInput) (*AppResult, error)
}
```

**Step 3: 验证编译**

```bash
go build ./internal/domain/shared/port/...
```

**Step 4: 提交**

```bash
git add internal/domain/shared/port/message.go internal/domain/shared/port/app_executor.go
git commit -m "feat(port): add AppExecutor domain port with streaming AppEvent and variable injection"
```

---

## Task 5：创建 AI 引擎层 — 模型工厂和提示词渲染

**Files:**
- Create: `internal/infrastructure/ai/engine/model_factory.go`
- Create: `internal/infrastructure/ai/engine/prompt.go`
- Create: `internal/infrastructure/ai/engine/prompt_test.go`

**Step 1: 先写提示词渲染测试**

创建 `internal/infrastructure/ai/engine/prompt_test.go`：

```go
package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

func TestRenderPrompt_WithVariables(t *testing.T) {
	result := engine.RenderPrompt("你是一个{{language}}翻译专家", map[string]string{
		"language": "英语",
	})
	assert.Equal(t, "你是一个英语翻译专家", result)
}

func TestRenderPrompt_MultipleVariables(t *testing.T) {
	result := engine.RenderPrompt("将{{source}}翻译为{{target}}", map[string]string{
		"source": "中文",
		"target": "英语",
	})
	assert.Equal(t, "将中文翻译为英语", result)
}

func TestRenderPrompt_NoVariables(t *testing.T) {
	result := engine.RenderPrompt("你是一个有用的助手", nil)
	assert.Equal(t, "你是一个有用的助手", result)
}

func TestRenderPrompt_MissingVariable(t *testing.T) {
	result := engine.RenderPrompt("你是一个{{language}}专家", map[string]string{})
	assert.Equal(t, "你是一个{{language}}专家", result)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

**Step 3: 创建提示词渲染**

创建 `internal/infrastructure/ai/engine/prompt.go`：

```go
package engine

import "strings"

// RenderPrompt 渲染提示词模板，将 {{变量名}} 替换为 variables 中的值
// 未匹配的变量保留原样
func RenderPrompt(template string, variables map[string]string) string {
	if len(variables) == 0 {
		return template
	}
	result := template
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
```

**Step 4: 运行测试确认通过**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

预期：4 个测试 PASS

**Step 5: 创建模型工厂**

创建 `internal/infrastructure/ai/engine/model_factory.go`：

```go
package engine

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// NewChatModel 根据 ModelConfig 创建对应 Provider 的 Eino BaseChatModel
func NewChatModel(ctx context.Context, m *modelconfig.ModelConfig) (einomodel.BaseChatModel, error) {
	switch m.Provider() {
	case "openai", "openai_compatible":
		cfg := &openai.ChatModelConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			maxTokens := m.MaxTokens()
			cfg.MaxTokens = &maxTokens
		}
		return openai.NewChatModel(ctx, cfg)

	case "ark":
		cfg := &ark.ChatModelConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			maxTokens := m.MaxTokens()
			cfg.MaxTokens = &maxTokens
		}
		return ark.NewChatModel(ctx, cfg)

	case "ollama":
		cfg := &ollama.ChatModelConfig{
			Model: m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return ollama.NewChatModel(ctx, cfg)

	case "claude":
		cfg := &claude.Config{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = &[]string{m.BaseURL()}[0]
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		if m.MaxTokens() > 0 {
			cfg.MaxTokens = m.MaxTokens()
		}
		return claude.NewChatModel(ctx, cfg)

	default:
		return nil, fmt.Errorf("engine: unsupported provider %q", m.Provider())
	}
}

// NewChatModelWithOverrides 创建 ChatModel 并应用 AppConfig 中的覆盖参数
func NewChatModelWithOverrides(ctx context.Context, m *modelconfig.ModelConfig, temperature *float32, maxTokens int) (einomodel.BaseChatModel, error) {
	// 创建一个临时的覆盖配置
	overridden := modelconfig.Reconstitute(
		m.ID(), m.Name(), m.Provider(), m.Capability(), m.ModelID(),
		m.APIKey(), m.BaseURL(),
		func() int {
			if maxTokens > 0 {
				return maxTokens
			}
			return m.MaxTokens()
		}(),
		func() *float32 {
			if temperature != nil {
				return temperature
			}
			return m.Temperature()
		}(),
		m.IsDefault(), m.Enabled(),
	)
	return NewChatModel(ctx, overridden)
}
```

**Step 6: 验证编译**

```bash
go build ./internal/infrastructure/ai/engine/...
```

**Step 7: 提交**

```bash
git add internal/infrastructure/ai/engine/
git commit -m "feat(ai/engine): add model factory and prompt variable renderer"
```

---

## Task 6：创建 AI 引擎层 — TextGenExecutor

**Files:**
- Create: `internal/infrastructure/ai/engine/text_gen_executor.go`
- Create: `internal/infrastructure/ai/engine/text_gen_executor_test.go`

**Step 1: 先写测试**

创建 `internal/infrastructure/ai/engine/text_gen_executor_test.go`：

```go
package engine_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

// stubChatModel 实现 model.BaseChatModel
type stubChatModel struct{ reply string }

func (s *stubChatModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(s.reply, nil), nil
}

func (s *stubChatModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		writer.Send(schema.AssistantMessage(s.reply, nil), nil)
	}()
	return reader, nil
}

func TestTextGenExecutor_Execute(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "翻译结果"}, "你是翻译专家")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "翻译这段话",
	})

	require.NoError(t, err)
	assert.Equal(t, "翻译结果", result.Content)
}

func TestTextGenExecutor_Run(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "流式输出"}, "你是助手")

	ch, err := exec.Run(context.Background(), &port.AppExecutorInput{
		Query: "写一首诗",
	})

	require.NoError(t, err)
	var events []port.AppEvent
	for event := range ch {
		events = append(events, event)
	}
	assert.True(t, len(events) > 0)
	// 最后一个事件应为 Done
	assert.Equal(t, port.AppEventDone, events[len(events)-1].Type)
}

func TestTextGenExecutor_WithVariables(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "ok"}, "你是{{language}}专家")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query:     "翻译",
		Variables: map[string]string{"language": "英语"},
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result.Content)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

**Step 3: 创建 TextGenExecutor**

创建 `internal/infrastructure/ai/engine/text_gen_executor.go`：

```go
package engine

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// TextGenExecutor 单次文本生成执行器（无对话上下文）
type TextGenExecutor struct {
	model        einomodel.BaseChatModel
	systemPrompt string
}

// NewTextGenExecutor 创建文本生成执行器
func NewTextGenExecutor(model einomodel.BaseChatModel, systemPrompt string) *TextGenExecutor {
	return &TextGenExecutor{model: model, systemPrompt: systemPrompt}
}

// Execute 非流式执行
func (e *TextGenExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)
	msg, err := e.model.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &port.AppResult{Content: msg.Content}, nil
}

// Run 流式执行
func (e *TextGenExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)
	reader, err := e.model.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	ch := make(chan port.AppEvent, 16)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.AppEvent{Type: port.AppEventDone}
				return
			}
			if err != nil {
				ch <- port.AppEvent{Type: port.AppEventError, Error: err}
				return
			}
			if msg != nil && msg.Content != "" {
				ch <- port.AppEvent{Type: port.AppEventMessage, Content: msg.Content}
			}
		}
	}()
	return ch, nil
}

func (e *TextGenExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
	prompt := RenderPrompt(e.systemPrompt, input.Variables)
	messages := make([]*schema.Message, 0, 2)
	if prompt != "" {
		messages = append(messages, schema.SystemMessage(prompt))
	}
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}
```

**Step 4: 运行测试**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

预期：7 个测试 PASS（4 prompt + 3 text_gen）

**Step 5: 提交**

```bash
git add internal/infrastructure/ai/engine/text_gen_executor.go internal/infrastructure/ai/engine/text_gen_executor_test.go
git commit -m "feat(ai/engine): TextGenExecutor for single-shot text generation"
```

---

## Task 7：创建 AI 引擎层 — ChatExecutor

**Files:**
- Create: `internal/infrastructure/ai/engine/chat_executor.go`
- Create: `internal/infrastructure/ai/engine/chat_executor_test.go`

**Step 1: 先写测试**

创建 `internal/infrastructure/ai/engine/chat_executor_test.go`：

```go
package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

func TestChatExecutor_Execute(t *testing.T) {
	exec := engine.NewChatExecutor(&stubChatModel{reply: "你好！"}, "你是一个友好的助手")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "你好",
		History: []port.Message{
			{Role: "user", Content: "之前的消息"},
			{Role: "assistant", Content: "之前的回复"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "你好！", result.Content)
}

func TestChatExecutor_Run(t *testing.T) {
	exec := engine.NewChatExecutor(&stubChatModel{reply: "流式回复"}, "你是助手")

	ch, err := exec.Run(context.Background(), &port.AppExecutorInput{
		Query: "你好",
	})

	require.NoError(t, err)
	var events []port.AppEvent
	for event := range ch {
		events = append(events, event)
	}
	assert.True(t, len(events) > 0)
	assert.Equal(t, port.AppEventDone, events[len(events)-1].Type)
}

func TestChatExecutor_WithHistory(t *testing.T) {
	exec := engine.NewChatExecutor(&stubChatModel{reply: "ok"}, "system prompt")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "继续",
		History: []port.Message{
			{Role: "user", Content: "第一轮"},
			{Role: "assistant", Content: "第一轮回复"},
			{Role: "user", Content: "第二轮"},
			{Role: "assistant", Content: "第二轮回复"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result.Content)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/engine/... -run TestChatExecutor -v
```

**Step 3: 创建 ChatExecutor**

创建 `internal/infrastructure/ai/engine/chat_executor.go`：

```go
package engine

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// ChatExecutor 多轮对话执行器（纯对话，无工具/知识库）
type ChatExecutor struct {
	model        einomodel.BaseChatModel
	systemPrompt string
}

// NewChatExecutor 创建对话执行器
func NewChatExecutor(model einomodel.BaseChatModel, systemPrompt string) *ChatExecutor {
	return &ChatExecutor{model: model, systemPrompt: systemPrompt}
}

// Execute 非流式执行
func (e *ChatExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)
	msg, err := e.model.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &port.AppResult{Content: msg.Content}, nil
}

// Run 流式执行
func (e *ChatExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)
	reader, err := e.model.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	ch := make(chan port.AppEvent, 16)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.AppEvent{Type: port.AppEventDone}
				return
			}
			if err != nil {
				ch <- port.AppEvent{Type: port.AppEventError, Error: err}
				return
			}
			if msg != nil && msg.Content != "" {
				ch <- port.AppEvent{Type: port.AppEventMessage, Content: msg.Content}
			}
		}
	}()
	return ch, nil
}

func (e *ChatExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
	prompt := RenderPrompt(e.systemPrompt, input.Variables)
	messages := make([]*schema.Message, 0, len(input.History)+2)
	if prompt != "" {
		messages = append(messages, schema.SystemMessage(prompt))
	}
	// 注入对话历史
	for _, m := range input.History {
		switch m.Role {
		case "system":
			messages = append(messages, schema.SystemMessage(m.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(m.Content, nil))
		default:
			messages = append(messages, schema.UserMessage(m.Content))
		}
	}
	// 当前用户输入
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}
```

**Step 4: 运行测试**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

预期：10 个测试 PASS（4 prompt + 3 text_gen + 3 chat）

**Step 5: 提交**

```bash
git add internal/infrastructure/ai/engine/chat_executor.go internal/infrastructure/ai/engine/chat_executor_test.go
git commit -m "feat(ai/engine): ChatExecutor for multi-turn conversation"
```

---

## Task 8：创建 AI 引擎层 — AgentExecutor

**Files:**
- Create: `internal/infrastructure/ai/engine/agent_executor.go`
- Create: `internal/infrastructure/ai/engine/agent_executor_test.go`

**Step 1: 先写测试**

创建 `internal/infrastructure/ai/engine/agent_executor_test.go`：

```go
package engine_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

// stubToolCallingModel 实现 model.ToolCallingChatModel
type stubToolCallingModel struct {
	reply string
}

func (s *stubToolCallingModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(s.reply, nil), nil
}

func (s *stubToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		writer.Send(schema.AssistantMessage(s.reply, nil), nil)
	}()
	return reader, nil
}

func (s *stubToolCallingModel) BindTools(_ []*schema.ToolInfo) error {
	return nil
}

func TestAgentExecutor_Execute(t *testing.T) {
	exec, err := engine.NewAgentExecutor(&stubToolCallingModel{reply: "Agent回复"}, "你是一个Agent", nil)
	require.NoError(t, err)

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "帮我查询天气",
	})

	require.NoError(t, err)
	assert.Equal(t, "Agent回复", result.Content)
}

func TestAgentExecutor_NilModel(t *testing.T) {
	_, err := engine.NewAgentExecutor(nil, "system", nil)
	assert.Error(t, err)
}

func TestAgentExecutor_Run(t *testing.T) {
	exec, err := engine.NewAgentExecutor(&stubToolCallingModel{reply: "流式Agent"}, "你是Agent", nil)
	require.NoError(t, err)

	ch, err := exec.Run(context.Background(), &port.AppExecutorInput{
		Query: "搜索资料",
	})
	require.NoError(t, err)

	var events []port.AppEvent
	for event := range ch {
		events = append(events, event)
	}
	assert.True(t, len(events) > 0)
	assert.Equal(t, port.AppEventDone, events[len(events)-1].Type)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/engine/... -run TestAgentExecutor -v
```

**Step 3: 创建 AgentExecutor**

创建 `internal/infrastructure/ai/engine/agent_executor.go`：

```go
package engine

import (
	"context"
	"errors"
	"fmt"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/schema/tool"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// AgentExecutor ReAct Agent 执行器（LLM + Tools 推理循环）
type AgentExecutor struct {
	chatModel    einomodel.ToolCallingChatModel
	systemPrompt string
	tools        []*tool.Info
}

// NewAgentExecutor 创建 Agent 执行器
func NewAgentExecutor(chatModel einomodel.ToolCallingChatModel, systemPrompt string, tools []*tool.Info) (*AgentExecutor, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("agent_executor: chatModel is required")
	}
	return &AgentExecutor{
		chatModel:    chatModel,
		systemPrompt: systemPrompt,
		tools:        tools,
	}, nil
}

// Execute 非流式执行
func (e *AgentExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := e.buildMessages(input)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: e.chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("agent_executor: failed to create agent: %w", err)
	}

	msg, err := agent.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &port.AppResult{Content: msg.Content}, nil
}

// Run 流式执行
func (e *AgentExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := e.buildMessages(input)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: e.chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("agent_executor: failed to create agent: %w", err)
	}

	reader, err := agent.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}

	ch := make(chan port.AppEvent, 32)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.AppEvent{Type: port.AppEventDone}
				return
			}
			if err != nil {
				ch <- port.AppEvent{Type: port.AppEventError, Error: err}
				return
			}
			if msg == nil {
				continue
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					ch <- port.AppEvent{
						Type:    port.AppEventToolCall,
						Content: tc.Function.Name,
					}
				}
			} else if msg.Content != "" {
				ch <- port.AppEvent{
					Type:    port.AppEventMessage,
					Content: msg.Content,
				}
			}
		}
	}()
	return ch, nil
}

func (e *AgentExecutor) buildMessages(input *port.AppExecutorInput) []*schema.Message {
	prompt := RenderPrompt(e.systemPrompt, input.Variables)
	messages := make([]*schema.Message, 0, len(input.History)+2)
	if prompt != "" {
		messages = append(messages, schema.SystemMessage(prompt))
	}
	for _, m := range input.History {
		switch m.Role {
		case "system":
			messages = append(messages, schema.SystemMessage(m.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(m.Content, nil))
		default:
			messages = append(messages, schema.UserMessage(m.Content))
		}
	}
	messages = append(messages, schema.UserMessage(input.Query))
	return messages
}
```

> **注意：** `tool.Info` 和 `react.AgentConfig` 的字段需与实际 Eino 版本匹配。若编译报错，运行 `go doc github.com/cloudwego/eino/flow/agent/react AgentConfig` 和 `go doc github.com/cloudwego/eino/schema/tool Info` 确认。`schema/tool` 路径可能是 `schema.ToolInfo`，需根据实际 API 调整。

**Step 4: 运行测试**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

预期：13 个测试 PASS

**Step 5: 提交**

```bash
git add internal/infrastructure/ai/engine/agent_executor.go internal/infrastructure/ai/engine/agent_executor_test.go
git commit -m "feat(ai/engine): AgentExecutor with ReAct loop for tool-calling agent"
```

---

## Task 9：创建 AI 引擎层 — ExecutorFactory

**Files:**
- Create: `internal/infrastructure/ai/engine/factory.go`
- Create: `internal/infrastructure/ai/engine/factory_test.go`

**Step 1: 先写测试**

创建 `internal/infrastructure/ai/engine/factory_test.go`：

```go
package engine_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

func TestExecutorFactory_UnsupportedType(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "test",
	}

	_, err := factory.Create(context.Background(), valueobject.AppTypeChatFlow, cfg, &stubChatModel{reply: "x"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestExecutorFactory_Chat(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "你是助手",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeChat, cfg, &stubChatModel{reply: "hello"})
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestExecutorFactory_TextGeneration(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "翻译",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeTextGeneration, cfg, &stubChatModel{reply: "translated"})
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestExecutorFactory_Agent(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "你是Agent",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeAgent, cfg, &stubToolCallingModel{reply: "ok"})
	require.NoError(t, err)
	assert.NotNil(t, exec)
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/engine/... -run TestExecutorFactory -v
```

**Step 3: 创建 ExecutorFactory**

创建 `internal/infrastructure/ai/engine/factory.go`：

```go
package engine

import (
	"context"
	"fmt"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// ExecutorFactory 根据 App 配置创建对应的 Executor
type ExecutorFactory struct{}

// NewExecutorFactory 创建 ExecutorFactory
func NewExecutorFactory() *ExecutorFactory {
	return &ExecutorFactory{}
}

// Create 根据应用类型和配置创建 AppExecutor
// chatModel 由调用方根据 AppConfig.ModelID 加载 ModelConfig 后创建
func (f *ExecutorFactory) Create(
	ctx context.Context,
	appType valueobject.AppType,
	config *valueobject.AppConfig,
	chatModel einomodel.BaseChatModel,
) (port.AppExecutor, error) {
	switch appType {
	case valueobject.AppTypeTextGeneration:
		return NewTextGenExecutor(chatModel, config.SystemPrompt), nil

	case valueobject.AppTypeChat:
		return NewChatExecutor(chatModel, config.SystemPrompt), nil

	case valueobject.AppTypeAgent:
		toolModel, ok := chatModel.(einomodel.ToolCallingChatModel)
		if !ok {
			return nil, fmt.Errorf("engine: agent requires a ToolCallingChatModel, got %T", chatModel)
		}
		return NewAgentExecutor(toolModel, config.SystemPrompt, nil)

	default:
		return nil, fmt.Errorf("engine: unsupported app type %q", appType)
	}
}
```

**Step 4: 运行测试**

```bash
go test ./internal/infrastructure/ai/engine/... -v
```

预期：17 个测试 PASS

**Step 5: 验证完整编译**

```bash
go build ./...
```

**Step 6: 提交**

```bash
git add internal/infrastructure/ai/engine/factory.go internal/infrastructure/ai/engine/factory_test.go
git commit -m "feat(ai/engine): ExecutorFactory dispatching by AppType"
```

---

## Task 10：持久化层 — App/AppVersion GORM 实体和仓储实现

**Files:**
- Create: `internal/infrastructure/persistence/entity/app.go`
- Create: `internal/infrastructure/persistence/entity/app_version.go`
- Create: `internal/infrastructure/persistence/repository/app/app_repo_impl.go`
- Modify: `internal/infrastructure/persistence/migration/migrate.go`

**Step 1: 创建 App GORM 实体**

创建 `internal/infrastructure/persistence/entity/app.go`：

```go
package entity

import "github.com/dysodeng/ai-adp/internal/domain/app/valueobject"

// AppEntity AI 应用数据库实体
type AppEntity struct {
	Base
	TenantID    string           `gorm:"type:uuid;not null;index:idx_app_tenant"`
	Name        string           `gorm:"size:200;not null"`
	Description string           `gorm:"size:1000"`
	AppType     valueobject.AppType `gorm:"size:30;not null;index:idx_app_type"`
	Icon        string           `gorm:"size:500"`
}

func (AppEntity) TableName() string { return "apps" }
```

**Step 2: 创建 AppVersion GORM 实体**

创建 `internal/infrastructure/persistence/entity/app_version.go`：

```go
package entity

import (
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppVersionEntity AI 应用版本数据库实体
type AppVersionEntity struct {
	Base
	AppID       string                   `gorm:"type:uuid;not null;index:idx_version_app"`
	Version     int                      `gorm:"not null"`
	Status      valueobject.VersionStatus `gorm:"size:20;not null;index:idx_version_status"`
	Config      string                   `gorm:"type:jsonb;not null"` // JSON 存储
	PublishedAt *time.Time
}

func (AppVersionEntity) TableName() string { return "app_versions" }
```

**Step 3: 创建仓储实现**

创建 `internal/infrastructure/persistence/repository/app/app_repo_impl.go`：

```go
package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/dysodeng/ai-adp/internal/domain/app/errors"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

var _ domainrepo.AppRepository = (*AppRepositoryImpl)(nil)

// AppRepositoryImpl GORM-based AI 应用仓储
type AppRepositoryImpl struct {
	db *gorm.DB
}

// NewAppRepository 创建应用仓储
func NewAppRepository(db *gorm.DB) *AppRepositoryImpl {
	return &AppRepositoryImpl{db: db}
}

// --- App CRUD ---

func (r *AppRepositoryImpl) SaveApp(ctx context.Context, app *appmodel.App) error {
	e := toAppEntity(app)
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *AppRepositoryImpl) FindAppByID(ctx context.Context, id uuid.UUID) (*appmodel.App, error) {
	var e entity.AppEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return toAppDomain(&e), nil
}

func (r *AppRepositoryImpl) FindAppsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*appmodel.App, error) {
	var entities []entity.AppEntity
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*appmodel.App, len(entities))
	for i := range entities {
		result[i] = toAppDomain(&entities[i])
	}
	return result, nil
}

func (r *AppRepositoryImpl) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.AppEntity{}, "id = ?", id).Error
}

// --- AppVersion CRUD ---

func (r *AppRepositoryImpl) SaveVersion(ctx context.Context, version *appmodel.AppVersion) error {
	e, err := toVersionEntity(version)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *AppRepositoryImpl) FindVersionByID(ctx context.Context, id uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindPublishedVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, valueobject.VersionStatusPublished).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindDraftVersion(ctx context.Context, appID uuid.UUID) (*appmodel.AppVersion, error) {
	var e entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, valueobject.VersionStatusDraft).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toVersionDomain(&e)
}

func (r *AppRepositoryImpl) FindVersionsByApp(ctx context.Context, appID uuid.UUID) ([]*appmodel.AppVersion, error) {
	var entities []entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ?", appID).
		Order("version DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return toVersionDomainList(entities)
}

func (r *AppRepositoryImpl) FindVersionsByStatus(ctx context.Context, appID uuid.UUID, status valueobject.VersionStatus) ([]*appmodel.AppVersion, error) {
	var entities []entity.AppVersionEntity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND status = ?", appID, status).
		Order("version DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return toVersionDomainList(entities)
}

// --- Conversion helpers ---

func toAppEntity(a *appmodel.App) entity.AppEntity {
	e := entity.AppEntity{
		TenantID:    a.TenantID().String(),
		Name:        a.Name(),
		Description: a.Description(),
		AppType:     a.Type(),
		Icon:        a.Icon(),
	}
	e.ID = a.ID()
	return e
}

func toAppDomain(e *entity.AppEntity) *appmodel.App {
	tenantID, _ := uuid.Parse(e.TenantID)
	return appmodel.Reconstitute(e.ID, tenantID, e.Name, e.Description, e.AppType, e.Icon)
}

func toVersionEntity(v *appmodel.AppVersion) (entity.AppVersionEntity, error) {
	configJSON, err := v.Config().ToJSON()
	if err != nil {
		return entity.AppVersionEntity{}, err
	}
	e := entity.AppVersionEntity{
		AppID:       v.AppID().String(),
		Version:     v.Version(),
		Status:      v.Status(),
		Config:      string(configJSON),
		PublishedAt: v.PublishedAt(),
	}
	e.ID = v.ID()
	return e, nil
}

func toVersionDomain(e *entity.AppVersionEntity) (*appmodel.AppVersion, error) {
	appID, _ := uuid.Parse(e.AppID)
	config, err := valueobject.AppConfigFromJSON([]byte(e.Config))
	if err != nil {
		return nil, err
	}
	return appmodel.ReconstituteVersion(e.ID, appID, e.Version, e.Status, config, e.PublishedAt), nil
}

func toVersionDomainList(entities []entity.AppVersionEntity) ([]*appmodel.AppVersion, error) {
	result := make([]*appmodel.AppVersion, len(entities))
	for i := range entities {
		v, err := toVersionDomain(&entities[i])
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}
```

**Step 4: 更新迁移文件**

修改 `internal/infrastructure/persistence/migration/migrate.go`，添加 App 和 AppVersion 实体：

在 `AutoMigrate` 调用中追加 `&entity.AppEntity{}` 和 `&entity.AppVersionEntity{}`：

```go
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.TenantEntity{},
		&entity.WorkspaceEntity{},
		&entity.ModelConfigEntity{},
		&entity.AppEntity{},        // 新增
		&entity.AppVersionEntity{}, // 新增
	)
}
```

**Step 5: 验证编译**

```bash
go build ./internal/infrastructure/persistence/...
```

**Step 6: 提交**

```bash
git add internal/infrastructure/persistence/entity/app.go \
        internal/infrastructure/persistence/entity/app_version.go \
        internal/infrastructure/persistence/repository/app/ \
        internal/infrastructure/persistence/migration/migrate.go
git commit -m "feat(persistence/app): add App/AppVersion entities, repository impl, auto-migrate"
```

---

## Task 11：更新 DI — 注入 ExecutorFactory 和 App 仓储

**Files:**
- Modify: `internal/di/infrastructure.go` — 添加 ExecutorFactory
- Modify: `internal/di/module.go` — 添加 AppModuleSet
- Regenerate: `internal/di/wire_gen.go`

**Step 1: 更新 infrastructure.go**

在 `InfrastructureSet` 中添加 `engine.NewExecutorFactory`：

```go
import (
	// ... 现有 import ...
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

var InfrastructureSet = wire.NewSet(
	provideDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
	engine.NewExecutorFactory, // AI 引擎工厂
)
```

**Step 2: 更新 module.go**

添加 App 模块仓储绑定：

```go
import (
	// ... 现有 import ...
	appdomainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	apprepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/app"
)

var AppModuleSet = wire.NewSet(
	apprepo.NewAppRepository,
	wire.Bind(new(appdomainrepo.AppRepository), new(*apprepo.AppRepositoryImpl)),
)

var ModulesSet = wire.NewSet(
	TenantModuleSet,
	AppModuleSet,
)
```

**Step 3: 重新生成 wire_gen.go**

```bash
wire ./internal/di/
```

> **注意：** 如果 Wire 报 `*engine.ExecutorFactory` 或 `*apprepo.AppRepositoryImpl` 未被消费的错误，需要将它们添加到 `NewApp` 的参数中或找到使用它们的组件。若暂时无消费者，可以在 `NewApp` 中加入但不赋值字段（用 `_` 忽略）。

**Step 4: 验证构建**

```bash
go build ./...
```

**Step 5: 运行全量测试**

```bash
go test ./...
```

预期：所有测试 PASS

**Step 6: 提交**

```bash
git add internal/di/
git commit -m "feat(di): wire ExecutorFactory and App repository into DI"
```

---

## Task 12：全量验证

**Step 1: 构建 + vet**

```bash
go build ./...
go vet ./...
```

**Step 2: 全量测试**

```bash
go test ./... -v 2>&1 | tail -30
```

预期：所有测试 PASS

**Step 3: 清理空目录**

检查并删除旧 AI 目录中的空目录（如果删除文件后目录为空）：

```bash
# 如果 internal/infrastructure/ai/llm/ 为空则删除
rmdir internal/infrastructure/ai/llm/ 2>/dev/null
rmdir internal/infrastructure/ai/agent/ 2>/dev/null
```

**Step 4: 最终提交（如有遗漏）**

```bash
git status
# 如有变更则提交
git add -A && git commit -m "chore: cleanup after AI engine redesign"
```

---

## 变更汇总

| 文件 | 操作 | 说明 |
|---|---|---|
| `internal/domain/shared/port/llm_executor.go` | 删除 | 被 AppExecutor 替代 |
| `internal/domain/shared/port/agent_executor.go` | 删除 | 被 AppExecutor 替代 |
| `internal/domain/shared/port/message.go` | 新建 | Message 类型独立文件 |
| `internal/domain/shared/port/app_executor.go` | 新建 | 统一执行端口 |
| `internal/domain/app/valueobject/*.go` | 新建 | AppType, VersionStatus, AppConfig |
| `internal/domain/app/model/app.go` | 新建 | App 聚合根 |
| `internal/domain/app/model/app_version.go` | 新建 | AppVersion 实体 |
| `internal/domain/app/repository/app_repo.go` | 新建 | App 仓储接口 |
| `internal/domain/app/errors/errors.go` | 新建 | 领域错误 |
| `internal/infrastructure/ai/llm/*` | 删除 | 迁移到 engine/ |
| `internal/infrastructure/ai/agent/*` | 删除 | 重写为 engine/ |
| `internal/infrastructure/ai/provider.go` | 删除 | 不再需要 |
| `internal/infrastructure/ai/engine/model_factory.go` | 新建 | LLM 模型创建工厂 |
| `internal/infrastructure/ai/engine/prompt.go` | 新建 | 提示词变量渲染 |
| `internal/infrastructure/ai/engine/factory.go` | 新建 | ExecutorFactory |
| `internal/infrastructure/ai/engine/text_gen_executor.go` | 新建 | 文本生成执行器 |
| `internal/infrastructure/ai/engine/chat_executor.go` | 新建 | 对话执行器 |
| `internal/infrastructure/ai/engine/agent_executor.go` | 新建 | Agent 执行器 |
| `internal/infrastructure/persistence/entity/app.go` | 新建 | App GORM 实体 |
| `internal/infrastructure/persistence/entity/app_version.go` | 新建 | AppVersion GORM 实体 |
| `internal/infrastructure/persistence/repository/app/*.go` | 新建 | App 仓储实现 |
| `internal/infrastructure/persistence/migration/migrate.go` | 修改 | 添加 App/AppVersion 迁移 |
| `internal/di/infrastructure.go` | 修改 | 移除旧 AI 组件，添加 ExecutorFactory |
| `internal/di/app.go` | 修改 | 移除 AIComponents 字段 |
| `internal/di/module.go` | 修改 | 添加 AppModuleSet |
| `internal/di/wire_gen.go` | 重新生成 | Wire 再生成 |

## 关键注意事项

1. **Eino API 兼容性**：`react.AgentConfig`、`schema.ToolInfo` 等类型字段名需与 Eino v0.7.37 匹配。编译报错时运行 `go doc` 确认。

2. **Wire 未消费错误**：若 `ExecutorFactory` 或 `AppRepository` 暂无消费者，需在 `NewApp` 中添加参数接收或创建占位消费。

3. **Embedding 保留**：`internal/infrastructure/ai/embedding/` 和 `internal/domain/shared/port/embedder.go` 不变，供后续知识库使用。

4. **二期扩展**：ChatFlow 和 Workflow 类型在 `ExecutorFactory.Create()` 中返回 "unsupported" 错误，二期实现时添加对应 Executor。
