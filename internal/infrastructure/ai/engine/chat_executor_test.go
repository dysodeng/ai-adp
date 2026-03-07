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
