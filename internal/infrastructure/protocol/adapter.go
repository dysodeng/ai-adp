package protocol

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

// Adapter 协议适配器接口
type Adapter interface {
	// HandleExecution 订阅 AgentExecutor 事件流并转发给客户端
	HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error
	// Close 关闭适配器
	Close() error
	// SendError 发送错误事件
	SendError(err error) error
}
