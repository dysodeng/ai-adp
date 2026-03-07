package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
)

// ToolService 工具服务接口
// 负责将 App 的工具配置转换为领域层的 Tool 实例
type ToolService interface {
	// LoadTools 根据 App 的工具配置加载所有工具
	// 返回工具列表和清理函数（用于 MCP 连接清理）
	LoadTools(ctx context.Context, config *ToolLoadConfig) ([]tool.Tool, func(), error)
}

// ToolLoadConfig 工具加载配置
type ToolLoadConfig struct {
	TenantID        uuid.UUID // 租户 ID
	KnowledgeList   []uuid.UUID
	ToolList        []uuid.UUID
	McpServerList   []uuid.UUID
	BuiltinToolList []uuid.UUID
	AppToolList     []uuid.UUID
	IsToolAgent     bool // 是否支持工具调用
}

// MockToolService 临时 mock 实现，工具领域实现后删除
type MockToolService struct{}

func NewMockToolService() ToolService {
	return &MockToolService{}
}

func (m *MockToolService) LoadTools(ctx context.Context, config *ToolLoadConfig) ([]tool.Tool, func(), error) {
	// 暂时返回空工具列表
	return []tool.Tool{}, func() {}, nil
}
