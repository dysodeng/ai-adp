package llm_test

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/llm"
)

// stubChatModel 实现 model.BaseChatModel（不发网络请求）
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

func TestLLMAdapter_Execute(t *testing.T) {
	adapter := llm.NewAdapter(&stubChatModel{reply: "hello"})

	result, err := adapter.Execute(context.Background(), []port.Message{
		{Role: "user", Content: "hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "hello", result.Content)
}

func TestLLMAdapter_Stream(t *testing.T) {
	adapter := llm.NewAdapter(&stubChatModel{reply: "chunk"})

	ch, err := adapter.Stream(context.Background(), []port.Message{
		{Role: "user", Content: "hi"},
	})
	require.NoError(t, err)

	var got []port.StreamChunk
	for chunk := range ch {
		got = append(got, chunk)
	}

	assert.True(t, len(got) > 0)
	assert.True(t, got[len(got)-1].Done)
}

func TestToSchemaMessages(t *testing.T) {
	adapter := llm.NewAdapter(&stubChatModel{reply: "ok"})
	_, err := adapter.Execute(context.Background(), []port.Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})
	assert.NoError(t, err)
}

var _ = io.EOF
