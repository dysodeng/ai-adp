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
