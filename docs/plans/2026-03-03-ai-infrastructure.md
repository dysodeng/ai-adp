# AI 基础设施实现计划（Eino ADK + 数据库模型配置）

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 使用 CloudWeGo Eino + Eino-ext 为项目实现完整 AI 基础设施层。模型配置（API Key、Provider、Model ID 等）存储在数据库，支持后续模型管理功能（增删改查）；通过领域层 `model` 有界上下文（已存在骨架）抽象模型实体；AI 组件（LLMExecutor、Embedder、AgentExecutor）通过 Google Wire 注入应用。

**Architecture:**
- **Domain** `internal/domain/model/` — 已有有界上下文骨架，填充 `ModelConfig` 聚合根（存储 provider/model_id/api_key/base_url 等），`ModelConfigRepository` 接口
- **Infrastructure/Persistence** — GORM 实体 + 仓储实现 + 自动迁移
- **Infrastructure/AI** — Eino 适配器：从 DB 加载 `ModelConfig`，按 Provider 工厂创建 Eino 组件，适配为 `port.LLMExecutor` / `port.Embedder` / `port.AgentExecutor`
- **DI** — Wire 接入：`ModelConfigRepository` + `AI Components` 注入

**Tech Stack:** Go 1.25, `github.com/cloudwego/eino`（含 `adk`）, `github.com/cloudwego/eino-ext`（openai/ark/ollama/claude），Google Wire, GORM (PostgreSQL)

---

## Task 1：添加 Eino 依赖

**Files:**
- Modify: `go.mod`, `go.sum`

**Step 1: 添加 eino 核心和 eino-ext 扩展**

```bash
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext@latest
```

**Step 2: 验证 go.mod 包含新依赖**

```bash
grep -E "cloudwego/eino" go.mod
```

预期输出类似：
```
github.com/cloudwego/eino v0.x.x
github.com/cloudwego/eino-ext v0.x.x
```

**Step 3: 验证构建**

```bash
go build ./...
```

**Step 4: 提交**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add cloudwego/eino and eino-ext for AI infrastructure"
```

---

## Task 2：填充 model 领域层

**背景：** `internal/domain/model/` 有界上下文已有骨架目录（`.gitkeep`），根据架构设计文档（`ModelConfig` 聚合根 + `ModelCapability` 值对象）填充实现。领域层只定义抽象，不感知 Eino。

**Files:**
- Create: `internal/domain/model/valueobject/model_capability.go`
- Create: `internal/domain/model/model/model_config.go`
- Create: `internal/domain/model/model/model_config_test.go`
- Create: `internal/domain/model/repository/model_config_repo.go`
- Create: `internal/domain/model/errors/errors.go`

**Step 1: 创建 valueobject/model_capability.go**

```go
package valueobject

// ModelCapability AI 模型能力类型
type ModelCapability string

const (
	ModelCapabilityLLM       ModelCapability = "llm"       // 大语言模型（对话、推理）
	ModelCapabilityEmbedding ModelCapability = "embedding" // 向量化模型
)

func (c ModelCapability) IsValid() bool {
	return c == ModelCapabilityLLM || c == ModelCapabilityEmbedding
}

func (c ModelCapability) String() string { return string(c) }
```

**Step 2: 先写聚合根测试**

创建 `internal/domain/model/model/model_config_test.go`：

```go
package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
)

func TestNewModelConfig_Valid(t *testing.T) {
	m, err := modelconfig.NewModelConfig(
		"GPT-4o",
		"openai",
		valueobject.ModelCapabilityLLM,
		"gpt-4o",
	)
	require.NoError(t, err)
	assert.Equal(t, "GPT-4o", m.Name())
	assert.Equal(t, "openai", m.Provider())
	assert.Equal(t, valueobject.ModelCapabilityLLM, m.Capability())
	assert.Equal(t, "gpt-4o", m.ModelID())
	assert.True(t, m.Enabled())
	assert.False(t, m.IsDefault())
}

func TestNewModelConfig_EmptyName(t *testing.T) {
	_, err := modelconfig.NewModelConfig("", "openai", valueobject.ModelCapabilityLLM, "gpt-4o")
	assert.Error(t, err)
}

func TestNewModelConfig_EmptyProvider(t *testing.T) {
	_, err := modelconfig.NewModelConfig("GPT-4o", "", valueobject.ModelCapabilityLLM, "gpt-4o")
	assert.Error(t, err)
}

func TestNewModelConfig_InvalidCapability(t *testing.T) {
	_, err := modelconfig.NewModelConfig("X", "openai", valueobject.ModelCapability("unknown"), "gpt-4o")
	assert.Error(t, err)
}

func TestModelConfig_SetDefault(t *testing.T) {
	m, _ := modelconfig.NewModelConfig("GPT-4o", "openai", valueobject.ModelCapabilityLLM, "gpt-4o")
	m.SetDefault(true)
	assert.True(t, m.IsDefault())
}

func TestModelConfig_Reconstitute(t *testing.T) {
	id := uuid.New()
	m := modelconfig.Reconstitute(id, "GPT-4o", "openai", valueobject.ModelCapabilityLLM, "gpt-4o",
		"sk-xxx", "https://api.openai.com", 4096, nil, true, true)
	assert.Equal(t, id, m.ID())
	assert.Equal(t, "sk-xxx", m.APIKey())
	assert.True(t, m.IsDefault())
}
```

**Step 3: 运行测试确认失败**

```bash
go test ./internal/domain/model/... -v
```

**Step 4: 创建 internal/domain/model/model/model_config.go**

```go
package model

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
)

// ModelConfig AI 模型配置聚合根（对应架构设计中的 ModelConfig）
// 存储模型的提供商、认证、端点等运行时配置，由管理员通过 API 管理
type ModelConfig struct {
	id         uuid.UUID
	name       string                    // 显示名称，如 "GPT-4o"
	provider   string                    // "openai" | "ark" | "ollama" | "claude" | "openai_compatible"
	capability valueobject.ModelCapability // LLM / Embedding
	modelID    string                    // 实际模型标识，如 "gpt-4o"
	apiKey     string                    // API 密钥（可选，部分部署通过环境变量提供）
	baseURL    string                    // 自定义端点，兼容 OpenAI 接口的私有部署
	maxTokens  int                       // 最大输出 token 数，0 表示使用 Provider 默认值
	temperature *float32                 // 采样温度，nil 表示使用 Provider 默认值
	isDefault  bool                      // 是否为该能力类型的默认模型
	enabled    bool                      // 是否启用
}

// NewModelConfig 创建新的 ModelConfig 聚合（带验证）
func NewModelConfig(name, provider string, capability valueobject.ModelCapability, modelID string) (*ModelConfig, error) {
	if name == "" {
		return nil, fmt.Errorf("model_config: name cannot be empty")
	}
	if provider == "" {
		return nil, fmt.Errorf("model_config: provider cannot be empty")
	}
	if !capability.IsValid() {
		return nil, fmt.Errorf("model_config: invalid capability %q", capability)
	}
	if modelID == "" {
		return nil, fmt.Errorf("model_config: modelID cannot be empty")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("model_config: failed to generate ID: %w", err)
	}
	return &ModelConfig{
		id:         id,
		name:       name,
		provider:   provider,
		capability: capability,
		modelID:    modelID,
		enabled:    true,
	}, nil
}

// Reconstitute 从持久化数据重建聚合（不生成新 ID）
func Reconstitute(
	id uuid.UUID, name, provider string,
	capability valueobject.ModelCapability, modelID string,
	apiKey, baseURL string,
	maxTokens int, temperature *float32,
	isDefault, enabled bool,
) *ModelConfig {
	return &ModelConfig{
		id:          id,
		name:        name,
		provider:    provider,
		capability:  capability,
		modelID:     modelID,
		apiKey:      apiKey,
		baseURL:     baseURL,
		maxTokens:   maxTokens,
		temperature: temperature,
		isDefault:   isDefault,
		enabled:     enabled,
	}
}

// Getters
func (m *ModelConfig) ID() uuid.UUID                        { return m.id }
func (m *ModelConfig) Name() string                         { return m.name }
func (m *ModelConfig) Provider() string                     { return m.provider }
func (m *ModelConfig) Capability() valueobject.ModelCapability { return m.capability }
func (m *ModelConfig) ModelID() string                      { return m.modelID }
func (m *ModelConfig) APIKey() string                       { return m.apiKey }
func (m *ModelConfig) BaseURL() string                      { return m.baseURL }
func (m *ModelConfig) MaxTokens() int                       { return m.maxTokens }
func (m *ModelConfig) Temperature() *float32                { return m.temperature }
func (m *ModelConfig) IsDefault() bool                      { return m.isDefault }
func (m *ModelConfig) Enabled() bool                        { return m.enabled }

// Setters / 命令方法
func (m *ModelConfig) SetAPIKey(key string)          { m.apiKey = key }
func (m *ModelConfig) SetBaseURL(url string)         { m.baseURL = url }
func (m *ModelConfig) SetMaxTokens(n int)            { m.maxTokens = n }
func (m *ModelConfig) SetTemperature(t *float32)     { m.temperature = t }
func (m *ModelConfig) SetDefault(v bool)             { m.isDefault = v }
func (m *ModelConfig) Enable()                       { m.enabled = true }
func (m *ModelConfig) Disable()                      { m.enabled = false }
```

**Step 5: 创建 internal/domain/model/repository/model_config_repo.go**

```go
package repository

import (
	"context"

	"github.com/google/uuid"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
)

// ModelConfigRepository AI 模型配置仓储接口
type ModelConfigRepository interface {
	// Save 新建或更新模型配置
	Save(ctx context.Context, m *modelconfig.ModelConfig) error
	// FindByID 按 ID 查询
	FindByID(ctx context.Context, id uuid.UUID) (*modelconfig.ModelConfig, error)
	// FindDefault 查询指定能力类型的默认模型，无结果返回 nil, nil
	FindDefault(ctx context.Context, capability valueobject.ModelCapability) (*modelconfig.ModelConfig, error)
	// FindAllByCapability 查询指定能力类型的所有启用模型
	FindAllByCapability(ctx context.Context, capability valueobject.ModelCapability) ([]*modelconfig.ModelConfig, error)
	// FindAll 查询所有模型
	FindAll(ctx context.Context) ([]*modelconfig.ModelConfig, error)
	// Delete 按 ID 删除
	Delete(ctx context.Context, id uuid.UUID) error
}
```

**Step 6: 创建 internal/domain/model/errors/errors.go**

```go
package errors

import "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
	ErrModelConfigNotFound = errors.NewDomainError("MODEL_CONFIG_NOT_FOUND", "model config not found")
	ErrNoDefaultModel      = errors.NewDomainError("NO_DEFAULT_MODEL", "no default model configured for this capability")
	ErrModelDisabled       = errors.NewDomainError("MODEL_DISABLED", "model config is disabled")
)
```

**Step 7: 运行测试确认通过**

```bash
go test ./internal/domain/model/... -v
```

预期：5 个测试 PASS

**Step 8: 验证编译**

```bash
go build ./...
```

**Step 9: 提交**

```bash
git add internal/domain/model/
git commit -m "feat(domain/model): add ModelConfig aggregate, repository interface, domain errors"
```

---

## Task 3：持久化层 — GORM 实体 + 仓储实现 + 迁移

**Files:**
- Create: `internal/infrastructure/persistence/entity/model_config.go`
- Create: `internal/infrastructure/persistence/repository/model/model_config_repo_impl.go`
- Modify: `internal/infrastructure/persistence/migration/migrate.go`

**Step 1: 查看现有 Base 实体结构**

先读取 `internal/infrastructure/persistence/entity/` 中已有文件，了解 `Base` 结构体字段（ID、CreatedAt、UpdatedAt 等）：

```bash
ls internal/infrastructure/persistence/entity/
```

然后读取其中一个已有实体文件确认 `Base` 结构。

**Step 2: 创建 GORM 实体**

创建 `internal/infrastructure/persistence/entity/model_config.go`：

```go
package entity

import "github.com/dysodeng/ai-adp/internal/domain/model/valueobject"

// ModelConfigEntity AI 模型配置数据库实体
type ModelConfigEntity struct {
	Base
	Name        string                       `gorm:"size:100;not null"`
	Provider    string                       `gorm:"size:50;not null;index:idx_provider_capability"`
	Capability  valueobject.ModelCapability  `gorm:"size:20;not null;index:idx_provider_capability"`
	ModelID     string                       `gorm:"size:200;not null"`
	APIKey      string                       `gorm:"size:500"`
	BaseURL     string                       `gorm:"size:500"`
	MaxTokens   int                          `gorm:"default:0"`
	Temperature *float32
	IsDefault   bool `gorm:"default:false;index:idx_default_capability"`
	Enabled     bool `gorm:"default:true"`
}

func (ModelConfigEntity) TableName() string { return "model_configs" }
```

**Step 3: 创建仓储实现**

创建 `internal/infrastructure/persistence/repository/model/model_config_repo_impl.go`：

```go
package model

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domainerrors "github.com/dysodeng/ai-adp/internal/domain/model/errors"
	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

// compile-time interface check
var _ domainrepo.ModelConfigRepository = (*ModelConfigRepositoryImpl)(nil)

// ModelConfigRepositoryImpl GORM-based AI 模型配置仓储
type ModelConfigRepositoryImpl struct {
	db *gorm.DB
}

func NewModelConfigRepository(db *gorm.DB) *ModelConfigRepositoryImpl {
	return &ModelConfigRepositoryImpl{db: db}
}

func (r *ModelConfigRepositoryImpl) Save(ctx context.Context, m *modelconfig.ModelConfig) error {
	e := toEntity(m)
	return r.db.WithContext(ctx).Save(&e).Error
}

func (r *ModelConfigRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*modelconfig.ModelConfig, error) {
	var e entity.ModelConfigEntity
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainerrors.ErrModelConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomain(&e), nil
}

func (r *ModelConfigRepositoryImpl) FindDefault(ctx context.Context, capability valueobject.ModelCapability) (*modelconfig.ModelConfig, error) {
	var e entity.ModelConfigEntity
	err := r.db.WithContext(ctx).
		Where("capability = ? AND is_default = true AND enabled = true", capability).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // 无默认模型不是错误，由调用方决定如何处理
	}
	if err != nil {
		return nil, err
	}
	return toDomain(&e), nil
}

func (r *ModelConfigRepositoryImpl) FindAllByCapability(ctx context.Context, capability valueobject.ModelCapability) ([]*modelconfig.ModelConfig, error) {
	var entities []entity.ModelConfigEntity
	err := r.db.WithContext(ctx).
		Where("capability = ? AND enabled = true", capability).
		Order("is_default DESC, created_at ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*modelconfig.ModelConfig, len(entities))
	for i := range entities {
		result[i] = toDomain(&entities[i])
	}
	return result, nil
}

func (r *ModelConfigRepositoryImpl) FindAll(ctx context.Context) ([]*modelconfig.ModelConfig, error) {
	var entities []entity.ModelConfigEntity
	err := r.db.WithContext(ctx).Order("capability, is_default DESC, created_at ASC").Find(&entities).Error
	if err != nil {
		return nil, err
	}
	result := make([]*modelconfig.ModelConfig, len(entities))
	for i := range entities {
		result[i] = toDomain(&entities[i])
	}
	return result, nil
}

func (r *ModelConfigRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.ModelConfigEntity{}, "id = ?", id).Error
}

// toEntity 将领域对象转换为 GORM 实体
func toEntity(m *modelconfig.ModelConfig) entity.ModelConfigEntity {
	e := entity.ModelConfigEntity{
		Name:        m.Name(),
		Provider:    m.Provider(),
		Capability:  m.Capability(),
		ModelID:     m.ModelID(),
		APIKey:      m.APIKey(),
		BaseURL:     m.BaseURL(),
		MaxTokens:   m.MaxTokens(),
		Temperature: m.Temperature(),
		IsDefault:   m.IsDefault(),
		Enabled:     m.Enabled(),
	}
	e.ID = m.ID()
	return e
}

// toDomain 将 GORM 实体重建为领域聚合根
func toDomain(e *entity.ModelConfigEntity) *modelconfig.ModelConfig {
	return modelconfig.Reconstitute(
		e.ID, e.Name, e.Provider, e.Capability, e.ModelID,
		e.APIKey, e.BaseURL, e.MaxTokens, e.Temperature,
		e.IsDefault, e.Enabled,
	)
}
```

**Step 4: 更新迁移文件**

在 `internal/infrastructure/persistence/migration/migrate.go` 的 `AutoMigrate` 调用中追加 `&entity.ModelConfigEntity{}`：

```go
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.TenantEntity{},
		&entity.WorkspaceEntity{},
		&entity.ModelConfigEntity{}, // 新增
	)
}
```

**Step 5: 验证编译**

```bash
go build ./internal/infrastructure/persistence/...
```

**Step 6: 提交**

```bash
git add internal/infrastructure/persistence/entity/model_config.go \
        internal/infrastructure/persistence/repository/model/ \
        internal/infrastructure/persistence/migration/migrate.go
git commit -m "feat(persistence/model): add ModelConfigEntity, repository impl, auto-migrate"
```

---

## Task 4：添加 port.AgentExecutor 领域端口

**Files:**
- Create: `internal/domain/shared/port/agent_executor.go`
- Create: `internal/domain/shared/port/agent_executor_test.go`

**Step 1: 先写测试**

```go
package port_test

import (
	"testing"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

func TestAgentEventTypes_Defined(t *testing.T) {
	types := []port.AgentEventType{
		port.AgentEventTypeMessage,
		port.AgentEventTypeToolCall,
		port.AgentEventTypeDone,
		port.AgentEventTypeError,
	}
	for _, tt := range types {
		if tt == "" {
			t.Fatalf("AgentEventType constant should not be empty")
		}
	}
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/domain/shared/port/... -run TestAgentEventTypes_Defined -v
```

**Step 3: 创建 agent_executor.go**

```go
package port

import "context"

// AgentEventType Agent 执行事件类型
type AgentEventType string

const (
	AgentEventTypeMessage  AgentEventType = "message"   // LLM 输出文本片段（流式）
	AgentEventTypeToolCall AgentEventType = "tool_call" // 工具调用（工具名存 Content）
	AgentEventTypeDone     AgentEventType = "done"      // 执行完成
	AgentEventTypeError    AgentEventType = "error"     // 执行出错（Error 字段非 nil）
)

// AgentEvent Agent 执行事件
type AgentEvent struct {
	Type    AgentEventType
	Content string // message/tool_call 时的内容
	Error   error  // error 类型时非 nil
}

// AgentExecutor ADK Agent 执行端口（由 infrastructure/ai/agent 实现）
// domain 层不感知 Eino 细节
type AgentExecutor interface {
	// Run 执行 Agent，返回事件 channel；调用方需消费至 Done 或 Error
	Run(ctx context.Context, query string, history []Message) (<-chan AgentEvent, error)
}
```

**Step 4: 运行测试确认通过**

```bash
go test ./internal/domain/shared/port/... -v
```

**Step 5: 提交**

```bash
git add internal/domain/shared/port/agent_executor.go \
        internal/domain/shared/port/agent_executor_test.go
git commit -m "feat(port): add AgentExecutor domain port with streaming AgentEvent"
```

---

## Task 5：实现 infrastructure/ai — LLM 适配器

**Files:**
- Create: `internal/infrastructure/ai/llm/factory.go`
- Create: `internal/infrastructure/ai/llm/adapter.go`
- Create: `internal/infrastructure/ai/llm/adapter_test.go`

**背景：** 工厂函数接受 `*modelconfig.ModelConfig` 领域对象创建 Eino ChatModel；适配器实现 `port.LLMExecutor`。

**Step 1: 先写测试（stub-based，不依赖真实 API）**

创建 `internal/infrastructure/ai/llm/adapter_test.go`：

```go
package llm_test

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
)

// stubChatModel 实现 model.BaseChatModel（不发网络请求）
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

func TestLLMAdapter_Execute(t *testing.T) {
	adapter := llm.NewAdapter(&stubChatModel{reply: "hello"})

	result, err := adapter.Execute(context.Background(), []port.Message{
		{Role: "user", Content: "hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "hello", result.Content)
}

func TestLLMAdapter_Stream(t *testing.T) {
	adapter := llm.NewAdapter(&stubChatModel{reply: "chunk"})

	ch, err := adapter.Stream(context.Background(), []port.Message{
		{Role: "user", Content: "hi"},
	})
	require.NoError(t, err)

	var got []port.StreamChunk
	for chunk := range ch {
		got = append(got, chunk)
	}

	assert.True(t, len(got) > 0)
	// 最后一个 chunk 应为 Done
	assert.True(t, got[len(got)-1].Done)
}

func TestToSchemaMessages(t *testing.T) {
	// 验证消息角色正确映射
	adapter := llm.NewAdapter(&stubChatModel{reply: "ok"})
	_, err := adapter.Execute(context.Background(), []port.Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})
	assert.NoError(t, err)
}

// 确保编译时引用了 io 包（避免 linter 警告）
var _ = io.EOF
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/llm/... -v
```

**Step 3: 创建 factory.go**

```go
package llm

import (
	"context"
	"fmt"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// NewChatModel 根据 ModelConfig 配置创建对应 Provider 的 Eino BaseChatModel
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
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		return ark.NewChatModel(ctx, cfg)

	case "ollama":
		cfg := &ollama.ChatModelConfig{
			BaseURL: m.BaseURL(),
			Model:   m.ModelID(),
		}
		return ollama.NewChatModel(ctx, cfg)

	case "claude":
		maxTokens := m.MaxTokens()
		cfg := &claude.Config{
			APIKey:    m.APIKey(),
			Model:     m.ModelID(),
			MaxTokens: maxTokens,
		}
		if m.Temperature() != nil {
			cfg.Temperature = m.Temperature()
		}
		return claude.NewChatModel(ctx, cfg)

	default:
		return nil, fmt.Errorf("llm: unsupported provider %q", m.Provider())
	}
}
```

**Step 4: 创建 adapter.go**

```go
package llm

import (
	"context"
	"errors"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// Adapter 将 Eino BaseChatModel 适配为 port.LLMExecutor
type Adapter struct {
	model einomodel.BaseChatModel
}

// NewAdapter 创建 LLM 适配器
func NewAdapter(model einomodel.BaseChatModel) *Adapter {
	return &Adapter{model: model}
}

// Execute 非流式 LLM 调用
func (a *Adapter) Execute(ctx context.Context, messages []port.Message) (*port.LLMResponse, error) {
	msg, err := a.model.Generate(ctx, toSchemaMessages(messages))
	if err != nil {
		return nil, err
	}
	return &port.LLMResponse{Content: msg.Content}, nil
}

// Stream 流式 LLM 调用，返回 StreamChunk channel
func (a *Adapter) Stream(ctx context.Context, messages []port.Message) (<-chan port.StreamChunk, error) {
	reader, err := a.model.Stream(ctx, toSchemaMessages(messages))
	if err != nil {
		return nil, err
	}
	ch := make(chan port.StreamChunk, 16)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, err := reader.Recv()
			if errors.Is(err, io.EOF) {
				ch <- port.StreamChunk{Done: true}
				return
			}
			if err != nil {
				ch <- port.StreamChunk{Error: err, Done: true}
				return
			}
			if msg != nil {
				ch <- port.StreamChunk{Content: msg.Content}
			}
		}
	}()
	return ch, nil
}

// toSchemaMessages 将 port.Message 列表转换为 Eino schema.Message 列表
func toSchemaMessages(messages []port.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system":
			result = append(result, schema.SystemMessage(m.Content))
		case "assistant":
			result = append(result, schema.AssistantMessage(m.Content, nil))
		default: // "user" 及其他
			result = append(result, schema.UserMessage(m.Content))
		}
	}
	return result
}
```

**Step 5: 运行测试**

```bash
go test ./internal/infrastructure/ai/llm/... -v
```

预期：3 个测试 PASS

**Step 6: 验证编译**

```bash
go build ./...
```

**Step 7: 提交**

```bash
git add internal/infrastructure/ai/llm/
git commit -m "feat(ai/llm): LLMExecutor adapter with DB-driven ModelConfig (openai/ark/ollama/claude)"
```

---

## Task 6：实现 infrastructure/ai — Embedding 适配器

**Files:**
- Create: `internal/infrastructure/ai/embedding/factory.go`
- Create: `internal/infrastructure/ai/embedding/adapter.go`
- Create: `internal/infrastructure/ai/embedding/adapter_test.go`

**Step 1: 先写测试**

```go
package embedding_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	embeddinginfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/embedding"
)

// stubEmbedder 满足 embedding.Embedder 接口
type stubEmbedder struct{}

func (s *stubEmbedder) EmbedStrings(_ context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i := range texts {
		result[i] = []float64{0.1, 0.2, 0.3}
	}
	return result, nil
}

func TestEmbeddingAdapter_Embed(t *testing.T) {
	adapter := embeddinginfra.NewAdapter(&stubEmbedder{})

	vectors, err := adapter.Embed(context.Background(), []string{"hello", "world"})

	require.NoError(t, err)
	assert.Len(t, vectors, 2)
	assert.Len(t, vectors[0], 3)
	assert.InDelta(t, float32(0.1), vectors[0][0], 1e-6)
}

func TestEmbeddingAdapter_Float64ToFloat32(t *testing.T) {
	adapter := embeddinginfra.NewAdapter(&stubEmbedder{})
	vectors, err := adapter.Embed(context.Background(), []string{"test"})
	require.NoError(t, err)
	// 验证类型为 float32
	var _ []float32 = vectors[0]
	assert.NotEmpty(t, vectors[0])
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/embedding/... -v
```

**Step 3: 创建 factory.go**

```go
package embedding

import (
	"context"
	"fmt"

	einoembed "github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"

	modelconfig "github.com/dysodeng/ai-adp/internal/domain/model/model"
)

// NewEmbedder 根据 ModelConfig 配置创建对应 Provider 的 Eino Embedder
func NewEmbedder(ctx context.Context, m *modelconfig.ModelConfig) (einoembed.Embedder, error) {
	switch m.Provider() {
	case "openai", "openai_compatible":
		cfg := &openai.EmbeddingConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		if m.BaseURL() != "" {
			cfg.BaseURL = m.BaseURL()
		}
		return openai.NewEmbedder(ctx, cfg)

	case "ark":
		cfg := &ark.EmbeddingConfig{
			APIKey: m.APIKey(),
			Model:  m.ModelID(),
		}
		return ark.NewEmbedder(ctx, cfg)

	case "ollama":
		cfg := &ollama.EmbeddingConfig{
			BaseURL: m.BaseURL(),
			Model:   m.ModelID(),
		}
		return ollama.NewEmbedder(ctx, cfg)

	default:
		return nil, fmt.Errorf("embedding: unsupported provider %q", m.Provider())
	}
}
```

**Step 4: 创建 adapter.go**

```go
package embedding

import (
	"context"

	einoembed "github.com/cloudwego/eino/components/embedding"
)

// Adapter 将 Eino Embedder 适配为 port.Embedder（float64 → float32）
type Adapter struct {
	embedder einoembed.Embedder
}

// NewAdapter 创建 Embedding 适配器
func NewAdapter(embedder einoembed.Embedder) *Adapter {
	return &Adapter{embedder: embedder}
}

// Embed 执行文本向量化
func (a *Adapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vectors, err := a.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
	}
	return toFloat32(vectors), nil
}

// toFloat32 将 float64 向量转换为 float32（精度损失可忽略）
func toFloat32(in [][]float64) [][]float32 {
	out := make([][]float32, len(in))
	for i, vec := range in {
		out[i] = make([]float32, len(vec))
		for j, v := range vec {
			out[i][j] = float32(v)
		}
	}
	return out
}
```

**Step 5: 运行测试**

```bash
go test ./internal/infrastructure/ai/embedding/... -v
```

预期：2 个测试 PASS

**Step 6: 提交**

```bash
git add internal/infrastructure/ai/embedding/
git commit -m "feat(ai/embedding): Embedder adapter with DB-driven ModelConfig (openai/ark/ollama)"
```

---

## Task 7：实现 infrastructure/ai — ADK Agent 适配器

**Files:**
- Create: `internal/infrastructure/ai/agent/agent.go`
- Create: `internal/infrastructure/ai/agent/agent_test.go`

**Step 1: 先写测试**

```go
package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentinfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/agent"
)

func TestNewAgentExecutor_NilModel(t *testing.T) {
	_, err := agentinfra.NewAgentExecutor(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chatModel")
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/infrastructure/ai/agent/... -v
```

**Step 3: 创建 agent.go**

```go
package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// AgentExecutorImpl 使用 Eino ADK 实现 port.AgentExecutor
type AgentExecutorImpl struct {
	runner *adk.Runner
}

// NewAgentExecutor 创建 ADK Agent 执行器
// chatModel 必须实现 model.ToolCallingChatModel（支持工具调用）
func NewAgentExecutor(chatModel einomodel.BaseChatModel) (*AgentExecutorImpl, error) {
	if chatModel == nil {
		return nil, errors.New("agent: chatModel is required")
	}

	toolModel, ok := chatModel.(einomodel.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("agent: provider does not implement ToolCallingChatModel (required for ADK)")
	}

	ctx := context.Background()
	agentImpl, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: toolModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: nil, // 初始无工具；可通过 AgentRunOption 动态注入
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("agent: failed to create ChatModelAgent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agentImpl})
	return &AgentExecutorImpl{runner: runner}, nil
}

// Run 执行 Agent，将 Eino 事件流转换为 port.AgentEvent channel
func (a *AgentExecutorImpl) Run(ctx context.Context, query string, history []port.Message) (<-chan port.AgentEvent, error) {
	var opts []adk.AgentRunOption
	if len(history) > 0 {
		msgs := make([]adk.Message, 0, len(history))
		for _, m := range history {
			msgs = append(msgs, adk.Message{Role: m.Role, Content: m.Content})
		}
		opts = append(opts, adk.WithHistoryModifier(func(_ context.Context, _ []adk.Message) []adk.Message {
			return msgs
		}))
	}

	iter := a.runner.Query(ctx, query, opts...)

	ch := make(chan port.AgentEvent, 16)
	go func() {
		defer close(ch)
		for {
			event, ok := iter.Next()
			if !ok {
				ch <- port.AgentEvent{Type: port.AgentEventTypeDone}
				return
			}
			if event == nil {
				continue
			}
			ch <- convertEvent(event)
		}
	}()
	return ch, nil
}

// convertEvent 将 adk.AgentEvent 转换为 port.AgentEvent
func convertEvent(e *adk.AgentEvent) port.AgentEvent {
	if e.Err != nil {
		return port.AgentEvent{Type: port.AgentEventTypeError, Error: e.Err}
	}
	if e.Output != nil {
		msg := e.Output.Message()
		if msg != nil && msg.Content != "" {
			return port.AgentEvent{Type: port.AgentEventTypeMessage, Content: msg.Content}
		}
	}
	// Action 事件（工具调用）
	if e.Action != nil {
		return port.AgentEvent{Type: port.AgentEventTypeToolCall, Content: e.Action.String()}
	}
	return port.AgentEvent{Type: port.AgentEventTypeMessage}
}
```

> **注意：** `adk.AgentEvent` 的字段（`Err`、`Output`、`Action`）需与实际 Eino ADK 包一致。若编译报字段名错误，参考 `go doc github.com/cloudwego/eino/adk AgentEvent` 调整。

**Step 4: 运行测试**

```bash
go test ./internal/infrastructure/ai/agent/... -v
```

**Step 5: 验证编译**

```bash
go build ./internal/infrastructure/ai/...
```

**Step 6: 提交**

```bash
git add internal/infrastructure/ai/agent/
git commit -m "feat(ai/agent): AgentExecutor using Eino ADK ChatModelAgent + Runner"
```

---

## Task 8：AI Provider 容器 + Wire DI 接入

**背景：** `ai.Components` 从 DB 加载默认模型配置，创建 Eino 组件并包装为 port 接口。若 DB 中暂无对应类型的默认模型，对应组件为 nil（应用正常启动，调用时返回"未配置"错误）。

**Files:**
- Create: `internal/infrastructure/ai/provider.go`
- Modify: `internal/di/infrastructure.go`
- Modify: `internal/di/app.go`
- Regenerate: `internal/di/wire_gen.go`

**Step 1: 查看当前 di/infrastructure.go 和 di/app.go**

先读取这两个文件了解现有结构。

**Step 2: 创建 internal/infrastructure/ai/provider.go**

```go
package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	agentinfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/agent"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/embedding"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
	modelrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// Components AI 基础设施组件，可能为 nil（未配置对应 Provider 时）
type Components struct {
	LLMExecutor   port.LLMExecutor   // nil 表示未配置默认 LLM 模型
	Embedder      port.Embedder      // nil 表示未配置默认 Embedding 模型
	AgentExecutor port.AgentExecutor // nil 表示 LLM 不支持工具调用或未配置
}

// NewComponents 从 DB 加载默认模型配置并初始化 AI 组件
// 若某类型无默认模型，对应组件为 nil（不返回错误）
func NewComponents(db *gorm.DB) (*Components, error) {
	repo := modelrepo.NewModelConfigRepository(db)
	ctx := context.Background()
	components := &Components{}

	// 初始化 LLM 组件
	if err := initLLMComponents(ctx, repo, components); err != nil {
		return nil, err
	}

	// 初始化 Embedding 组件
	if err := initEmbeddingComponent(ctx, repo, components); err != nil {
		return nil, err
	}

	return components, nil
}

func initLLMComponents(ctx context.Context, repo domainrepo.ModelConfigRepository, c *Components) error {
	m, err := repo.FindDefault(ctx, valueobject.ModelCapabilityLLM)
	if err != nil {
		return fmt.Errorf("ai: failed to query default LLM model: %w", err)
	}
	if m == nil {
		logger.Info(ctx, "ai: no default LLM model configured, LLMExecutor and AgentExecutor will be unavailable")
		return nil
	}

	chatModel, err := llm.NewChatModel(ctx, m)
	if err != nil {
		return fmt.Errorf("ai: failed to create LLM provider %q: %w", m.Provider(), err)
	}
	c.LLMExecutor = llm.NewAdapter(chatModel)

	// Agent 需要 ToolCallingChatModel；不支持时只记录日志，不返回错误
	agentExecutor, err := agentinfra.NewAgentExecutor(chatModel)
	if err != nil {
		logger.Info(ctx, "ai: LLM provider does not support tool calling, AgentExecutor unavailable",
			logger.AddField("provider", m.Provider()), logger.AddField("reason", err.Error()))
	} else {
		c.AgentExecutor = agentExecutor
	}
	return nil
}

func initEmbeddingComponent(ctx context.Context, repo domainrepo.ModelConfigRepository, c *Components) error {
	m, err := repo.FindDefault(ctx, valueobject.ModelCapabilityEmbedding)
	if err != nil {
		return fmt.Errorf("ai: failed to query default Embedding model: %w", err)
	}
	if m == nil {
		logger.Info(ctx, "ai: no default Embedding model configured, Embedder will be unavailable")
		return nil
	}

	embedder, err := embedding.NewEmbedder(ctx, m)
	if err != nil {
		return fmt.Errorf("ai: failed to create Embedding provider %q: %w", m.Provider(), err)
	}
	c.Embedder = embedding.NewAdapter(embedder)
	return nil
}
```

**Step 3: 更新 internal/di/infrastructure.go**

在 `InfrastructureSet` 中添加 `ai.NewComponents` 和 `modelrepo.NewModelConfigRepository`：

```go
var InfrastructureSet = wire.NewSet(
	provideDB,
	cache.NewRedisClient,
	transactions.NewManager,
	server.NewHTTPServer,
	provideLogger,
	provideTracerShutdown,
	ai.NewComponents,                       // 新增：AI 组件（从 DB 加载模型配置）
	modelrepo.NewModelConfigRepository,     // 新增：AI 模型配置仓储
)
```

同时在 import 中添加：
```go
import (
	// ... 现有 import ...
	ai "github.com/dysodeng/ai-adp/internal/infrastructure/ai"
	modelrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/model"
)
```

**Step 4: 更新 internal/di/app.go**

在 `App` 结构体中添加 `AIComponents`，供应用层使用：

```go
type App struct {
	HTTPServer     server.Server
	AIComponents   *ai.Components    // AI 基础设施组件（nil 表示未配置）
	tracerShutdown telemetry.ShutdownFunc
}

func NewApp(
	httpServer *server.HTTPServer,
	aiComponents *ai.Components,
	_ *zap.Logger,
	tracerShutdown telemetry.ShutdownFunc,
) *App {
	return &App{
		HTTPServer:     httpServer,
		AIComponents:   aiComponents,
		tracerShutdown: tracerShutdown,
	}
}
```

同时在 import 中添加：
```go
ai "github.com/dysodeng/ai-adp/internal/infrastructure/ai"
```

**Step 5: 重新生成 wire_gen.go**

```bash
wire ./internal/di/
```

**Step 6: 验证构建**

```bash
go build ./...
```

**Step 7: 运行全量测试**

```bash
go test ./... 2>&1 | grep -v "^ld: warning"
```

**Step 8: 提交**

```bash
git add internal/infrastructure/ai/provider.go \
        internal/di/infrastructure.go \
        internal/di/app.go \
        internal/di/wire_gen.go
git commit -m "feat(di): wire AI infrastructure into DI; Components loaded from DB model config"
```

---

## Task 9：全量验证

**Step 1: 构建 + vet**

```bash
go build ./...
go vet ./...
```

**Step 2: 全量测试**

```bash
go test ./... 2>&1 | grep -v "^ld: warning"
```

预期：所有已有测试 + 新增测试全部 PASS，无 FAIL

**Step 3: 静态检查**

```bash
golangci-lint run ./internal/domain/model/... \
                  ./internal/infrastructure/ai/... \
                  ./internal/domain/shared/port/...
```

**Step 4: 最终提交（如有遗漏）**

```bash
git add -A
git commit -m "chore: final cleanup after AI infrastructure implementation"
```

---

## 变更汇总

| 文件 | 操作 | 说明 |
|---|---|---|
| `go.mod` / `go.sum` | 修改 | 添加 cloudwego/eino、eino-ext |
| `internal/domain/model/valueobject/model_capability.go` | 新建 | ModelCapability 值对象（LLM/Embedding） |
| `internal/domain/model/model/model_config.go` | 新建 | ModelConfig 聚合根 |
| `internal/domain/model/repository/model_config_repo.go` | 新建 | 仓储接口 |
| `internal/domain/model/errors/errors.go` | 新建 | 领域错误 |
| `internal/infrastructure/persistence/entity/model_config.go` | 新建 | GORM 实体（表名 model_configs） |
| `internal/infrastructure/persistence/repository/model/` | 新建 | 仓储实现 |
| `internal/infrastructure/persistence/migration/migrate.go` | 修改 | 添加 ModelConfigEntity 迁移 |
| `internal/domain/shared/port/agent_executor.go` | 新建 | AgentExecutor 端口 |
| `internal/infrastructure/ai/llm/factory.go` | 新建 | LLM ChatModel 工厂 |
| `internal/infrastructure/ai/llm/adapter.go` | 新建 | LLMExecutor 适配器 |
| `internal/infrastructure/ai/embedding/factory.go` | 新建 | Embedding 工厂 |
| `internal/infrastructure/ai/embedding/adapter.go` | 新建 | Embedder 适配器 |
| `internal/infrastructure/ai/agent/agent.go` | 新建 | ADK AgentExecutor |
| `internal/infrastructure/ai/provider.go` | 新建 | AI 组件容器（从 DB 加载） |
| `internal/di/infrastructure.go` | 修改 | 添加 AI 组件 + 仓储 provider |
| `internal/di/app.go` | 修改 | App 持有 *ai.Components |
| `internal/di/wire_gen.go` | 重新生成 | Wire 再生成 |

---

## 关键注意事项

1. **adk.AgentEvent 字段名**：`convertEvent` 中 `e.Err`、`e.Output`、`e.Action` 等字段需与实际 Eino ADK 版本匹配。编译报错时运行 `go doc github.com/cloudwego/eino/adk AgentEvent` 确认字段名。

2. **adk.Message 类型**：`WithHistoryModifier` 回调中使用的 `adk.Message` 若与 `schema.Message` 不同，需调整 `Run` 方法中的消息构建逻辑。

3. **AI 组件可选性**：DB 中无默认模型时对应组件为 nil，调用方（应用服务）需在使用前判空并返回业务错误。

4. **Wire 消费 *ai.Components**：若 Wire 报未消费错误，将 `*ai.Components` 加入 `NewApp` 参数（Task 8 Step 4）。

5. **API Key 安全**：当前 API Key 明文存储。生产环境建议集成 AES 加密或 Vault，本期暂不实现。
