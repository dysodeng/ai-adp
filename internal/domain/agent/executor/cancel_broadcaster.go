package executor

import "context"

// CancelBroadcaster 取消信号广播器 - 在分布式环境中广播取消信号
type CancelBroadcaster interface {
	Broadcast(ctx context.Context, taskID string) error
	Subscribe(ctx context.Context, registry TaskRegistry) error
}
