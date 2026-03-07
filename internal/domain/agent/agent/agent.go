package agent

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/tool"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// Agent 领域核心接口
type Agent interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetAppType() valueobject.AppType

	// Execute 执行 Agent，将结果和事件填充到 agentExecutor 中
	Execute(ctx context.Context, agentExecutor executor.AgentExecutor) error

	GetTools() []tool.Tool
	Validate(agentExecutor executor.AgentExecutor) error
}
