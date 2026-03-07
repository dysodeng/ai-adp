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

// stubToolCallingModel 实现 model.ToolCallingChatModel
type stubToolCallingModel struct {
	reply string
}

func (s *stubToolCallingModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(s.reply, nil), nil
}

func (s *stubToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		writer.Send(schema.AssistantMessage(s.reply, nil), nil)
	}()
	return reader, nil
}

func (s *stubToolCallingModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return s, nil
}

func TestAgentExecutor_Execute(t *testing.T) {
	exec, err := engine.NewAgentExecutor(&stubToolCallingModel{reply: "Agent回复"}, "你是一个Agent", nil)
	require.NoError(t, err)

	result, err := exec.Execute(context.Background(), &port.AppExecutorInput{
		Query: "帮我查询天气",
	})

	require.NoError(t, err)
	assert.Equal(t, "Agent回复", result.Content)
}

func TestAgentExecutor_NilModel(t *testing.T) {
	_, err := engine.NewAgentExecutor(nil, "system", nil)
	assert.Error(t, err)
}

func TestAgentExecutor_Run(t *testing.T) {
	exec, err := engine.NewAgentExecutor(&stubToolCallingModel{reply: "流式Agent"}, "你是Agent", nil)
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
