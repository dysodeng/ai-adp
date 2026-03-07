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
