package port

import "context"

// Embedder 向量化端口（由 infrastructure/ai/embedding 实现）
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
