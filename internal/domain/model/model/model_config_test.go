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
