package tool

import "context"

// Tool 领域层工具接口（不依赖 Eino）
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// InputSchema 输入参数 Schema（JSON Schema）
	InputSchema() map[string]interface{}

	// Invoke 执行工具
	Invoke(ctx context.Context, input map[string]interface{}) (string, error)
}
