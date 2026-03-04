package embedding_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	embeddinginfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/embedding"
)

// stubEmbedder 实现 eino embedding.Embedder 接口（不发网络请求）
type stubEmbedder struct{}

func (s *stubEmbedder) EmbedStrings(_ context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i := range texts {
		result[i] = []float64{0.1, 0.2, 0.3}
	}
	return result, nil
}

func TestEmbeddingAdapter_Embed(t *testing.T) {
	adapter := embeddinginfra.NewAdapter(&stubEmbedder{})
	vectors, err := adapter.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	assert.Len(t, vectors, 2)
	assert.Len(t, vectors[0], 3)
	assert.InDelta(t, float32(0.1), vectors[0][0], 1e-6)
}

func TestEmbeddingAdapter_Float64ToFloat32(t *testing.T) {
	adapter := embeddinginfra.NewAdapter(&stubEmbedder{})
	vectors, err := adapter.Embed(context.Background(), []string{"test"})
	require.NoError(t, err)
	var _ []float32 = vectors[0] // 编译期检查返回类型为 []float32
	assert.NotEmpty(t, vectors[0])
}

func TestEmbeddingAdapter_MultipleTexts(t *testing.T) {
	adapter := embeddinginfra.NewAdapter(&stubEmbedder{})
	texts := []string{"foo", "bar", "baz"}
	vectors, err := adapter.Embed(context.Background(), texts)
	require.NoError(t, err)
	assert.Len(t, vectors, len(texts))
	for _, vec := range vectors {
		assert.Len(t, vec, 3)
		assert.InDelta(t, float32(0.2), vec[1], 1e-6)
		assert.InDelta(t, float32(0.3), vec[2], 1e-6)
	}
}
