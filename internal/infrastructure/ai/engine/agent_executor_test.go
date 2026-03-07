package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

func TestAgentExecutor_Execute(t *testing.T) {
	exec, err := engine.NewAgentExecutor(context.Background(), &stubModel{reply: "Agent回复"}, "你是一个Agent", nil)
	require.NoError(t, err)

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "帮我查询天气",
	})

	require.NoError(t, err)
	assert.Equal(t, "Agent回复", result.Content)
}

func TestAgentExecutor_NilModel(t *testing.T) {
	_, err := engine.NewAgentExecutor(context.Background(), nil, "system", nil)
	assert.Error(t, err)
}

func TestAgentExecutor_Run(t *testing.T) {
	exec, err := engine.NewAgentExecutor(context.Background(), &stubModel{reply: "流式Agent"}, "你是Agent", nil)
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
