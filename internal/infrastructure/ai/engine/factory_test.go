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

	_, err := factory.Create(context.Background(), valueobject.AppTypeChatFlow, cfg, &stubModel{reply: "x"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestExecutorFactory_Chat(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "你是助手",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeChat, cfg, &stubModel{reply: "hello"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestExecutorFactory_TextCompletion(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "翻译",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeTextCompletion, cfg, &stubModel{reply: "translated"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestExecutorFactory_Agent(t *testing.T) {
	factory := engine.NewExecutorFactory()
	cfg := &valueobject.AppConfig{
		ModelID:      uuid.New(),
		SystemPrompt: "你是Agent",
	}

	exec, err := factory.Create(context.Background(), valueobject.AppTypeAgent, cfg, &stubModel{reply: "ok"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}
