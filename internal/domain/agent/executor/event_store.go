package executor

import (
	"context"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// EventStore 事件存储接口 — 负责持久化和检索 SSE 事件流
type EventStore interface {
	// Append 追加事件到指定任务的事件流，返回事件的 stream ID
	Append(ctx context.Context, taskID string, event *model.Event) (string, error)
	// ReadAfter 读取指定任务在 lastID 之后的事件，最多返回 count 条
	ReadAfter(ctx context.Context, taskID string, lastID string, count int64) ([]*model.Event, error)
	// SetTTL 设置指定任务事件流的过期时间
	SetTTL(ctx context.Context, taskID string, ttl time.Duration) error
	// Exists 检查指定任务的事件流是否存在
	Exists(ctx context.Context, taskID string) (bool, error)
	// Delete 删除指定任务的事件流
	Delete(ctx context.Context, taskID string) error
}
