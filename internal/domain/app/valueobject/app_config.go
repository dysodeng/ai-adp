package valueobject

import (
	"encoding/json"

	"github.com/google/uuid"
)

// AppConfig AI 应用配置（JSON 序列化存储）
type AppConfig struct {
	ModelID            uuid.UUID    `json:"model_id"`
	SystemPrompt       string       `json:"system_prompt"`
	Temperature        *float32     `json:"temperature,omitempty"`
	MaxTokens          int          `json:"max_tokens,omitempty"`
	Tools              []ToolConfig `json:"tools,omitempty"`
	OpeningStatement   string       `json:"opening_statement,omitempty"`
	SuggestedQuestions []string     `json:"suggested_questions,omitempty"`
}

// ToolConfig 工具配置
type ToolConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
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
