package stream

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

func TestMemoryExecutorRegistry_RegisterAndGet(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)
	registry.Register("task-1", exec)
	got, ok := registry.Get("task-1")
	assert.True(t, ok)
	assert.Equal(t, exec, got)
}

func TestMemoryExecutorRegistry_GetNotFound(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	got, ok := registry.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMemoryExecutorRegistry_Unregister(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)
	registry.Register("task-1", exec)
	registry.Unregister("task-1")
	_, ok := registry.Get("task-1")
	assert.False(t, ok)
}

func TestMemoryExecutorRegistry_DelayedUnregister(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)
	registry.Register("task-1", exec)
	registry.DelayedUnregister("task-1", 100*time.Millisecond)
	_, ok := registry.Get("task-1")
	assert.True(t, ok)
	time.Sleep(200 * time.Millisecond)
	_, ok = registry.Get("task-1")
	assert.False(t, ok)
}
