package engine_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/engine"
)

// stubChatModel 实现 model.BaseChatModel
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

func TestTextGenExecutor_Execute(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "翻译结果"}, "你是翻译专家")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "翻译这段话",
	})

	require.NoError(t, err)
	assert.Equal(t, "翻译结果", result.Content)
}

func TestTextGenExecutor_Run(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "流式输出"}, "你是助手")

	ch, err := exec.Run(context.Background(), &port.AppExecutorInput{
		Query: "写一首诗",
	})

	require.NoError(t, err)
	var events []port.AppEvent
	for event := range ch {
		events = append(events, event)
	}
	assert.True(t, len(events) > 0)
	assert.Equal(t, port.AppEventDone, events[len(events)-1].Type)
}

func TestTextGenExecutor_WithVariables(t *testing.T) {
	exec := engine.NewTextGenExecutor(&stubChatModel{reply: "ok"}, "你是{{language}}专家")

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query:     "翻译",
		Variables: map[string]string{"language": "英语"},
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result.Content)
}
