package embedding

import (
	"context"

	einoembed "github.com/cloudwego/eino/components/embedding"
)

// Adapter 将 Eino Embedder 适配为 port.Embedder
// Eino Embedder 返回 [][]float64，port.Embedder 要求 [][]float32，此处完成类型转换
type Adapter struct {
	embedder einoembed.Embedder
}

// NewAdapter 创建 Embedding 适配器
func NewAdapter(embedder einoembed.Embedder) *Adapter {
	return &Adapter{embedder: embedder}
}

// Embed 实现 port.Embedder 接口，将文本列表转换为向量列表
func (a *Adapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vectors, err := a.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
	}
	return toFloat32(vectors), nil
}

// toFloat32 将 Eino 返回的 [][]float64 转换为 [][]float32
func toFloat32(in [][]float64) [][]float32 {
	out := make([][]float32, len(in))
	for i, vec := range in {
		out[i] = make([]float32, len(vec))
		for j, v := range vec {
			out[i][j] = float32(v)
		}
	}
	return out
}
