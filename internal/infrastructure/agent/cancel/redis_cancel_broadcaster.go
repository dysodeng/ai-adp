package cancel

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
	pkgredis "github.com/dysodeng/ai-adp/internal/infrastructure/pkg/redis"
)

const cancelChannel = "agent:task:cancel"

// RedisCancelBroadcaster 基于 Redis Pub/Sub 的取消信号广播器
type RedisCancelBroadcaster struct {
	client pkgredis.Client
}

// NewRedisCancelBroadcaster 创建 Redis 取消广播器
func NewRedisCancelBroadcaster(client pkgredis.Client) executor.CancelBroadcaster {
	return &RedisCancelBroadcaster{client: client}
}

func (b *RedisCancelBroadcaster) Broadcast(ctx context.Context, taskID string) error {
	if err := b.client.Publish(ctx, cancelChannel, taskID).Err(); err != nil {
		logger.Warn(ctx, "[CancelBroadcaster] broadcast failed", logger.ErrorField(err))
		return err
	}
	return nil
}

func (b *RedisCancelBroadcaster) Subscribe(ctx context.Context, registry executor.TaskRegistry) error {
	pubsub := b.client.Subscribe(ctx, cancelChannel)

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				taskID := msg.Payload
				registry.Cancel(taskID)
			}
		}
	}()

	return nil
}

var _ executor.CancelBroadcaster = (*RedisCancelBroadcaster)(nil)
