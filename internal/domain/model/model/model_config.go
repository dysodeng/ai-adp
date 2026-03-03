package model

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/model/valueobject"
)

// ModelConfig AI 模型配置聚合根（对应架构设计中的 ModelConfig）
// 存储模型的提供商、认证、端点等运行时配置，由管理员通过 API 管理
type ModelConfig struct {
	id          uuid.UUID
	name        string                      // 显示名称，如 "GPT-4o"
	provider    string                      // "openai" | "ark" | "ollama" | "claude" | "openai_compatible"
	capability  valueobject.ModelCapability // LLM / Embedding
	modelID     string                      // 实际模型标识，如 "gpt-4o"
	apiKey      string                      // API 密钥（可选，部分部署通过环境变量提供）
	baseURL     string                      // 自定义端点，兼容 OpenAI 接口的私有部署
	maxTokens   int                         // 最大输出 token 数，0 表示使用 Provider 默认值
	temperature *float32                    // 采样温度，nil 表示使用 Provider 默认值
	isDefault   bool                        // 是否为该能力类型的默认模型
	enabled     bool                        // 是否启用
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
func (m *ModelConfig) ID() uuid.UUID                           { return m.id }
func (m *ModelConfig) Name() string                            { return m.name }
func (m *ModelConfig) Provider() string                        { return m.provider }
func (m *ModelConfig) Capability() valueobject.ModelCapability { return m.capability }
func (m *ModelConfig) ModelID() string                         { return m.modelID }
func (m *ModelConfig) APIKey() string                          { return m.apiKey }
func (m *ModelConfig) BaseURL() string                         { return m.baseURL }
func (m *ModelConfig) MaxTokens() int                          { return m.maxTokens }
func (m *ModelConfig) Temperature() *float32                   { return m.temperature }
func (m *ModelConfig) IsDefault() bool                         { return m.isDefault }
func (m *ModelConfig) Enabled() bool                           { return m.enabled }

// Setters / 命令方法
func (m *ModelConfig) SetAPIKey(key string)      { m.apiKey = key }
func (m *ModelConfig) SetBaseURL(url string)     { m.baseURL = url }
func (m *ModelConfig) SetMaxTokens(n int)        { m.maxTokens = n }
func (m *ModelConfig) SetTemperature(t *float32) { m.temperature = t }
func (m *ModelConfig) SetDefault(v bool)         { m.isDefault = v }
func (m *ModelConfig) Enable()                   { m.enabled = true }
func (m *ModelConfig) Disable()                  { m.enabled = false }
