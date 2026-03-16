package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

const (
	streamKeyPrefix           = "sse:events:"
	defaultTTL                = 60 * time.Second
	maxReplayEventCount int64 = 10000
)

type RedisEventStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

func NewRedisEventStore(client redis.UniversalClient, keyPrefix string) executor.EventStore {
	return &RedisEventStore{client: client, keyPrefix: keyPrefix}
}

func (s *RedisEventStore) streamKey(taskID string) string {
	return s.keyPrefix + streamKeyPrefix + taskID
}

func (s *RedisEventStore) Append(ctx context.Context, taskID string, event *model.Event) (string, error) {
	key := s.streamKey(taskID)
	data, err := sonic.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal event failed: %w", err)
	}
	streamID, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		Values: map[string]interface{}{
			"type": string(event.Type),
			"data": string(data),
		},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd failed: %w", err)
	}
	s.client.Expire(ctx, key, defaultTTL)
	return streamID, nil
}

func (s *RedisEventStore) ReadAfter(ctx context.Context, taskID string, lastID string, count int64) ([]*model.Event, error) {
	key := s.streamKey(taskID)
	messages, err := s.client.XRangeN(ctx, key, "("+lastID, "+", count).Result()
	if err != nil {
		return nil, fmt.Errorf("xrange failed: %w", err)
	}
	events := make([]*model.Event, 0, len(messages))
	for _, msg := range messages {
		dataStr, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}
		var event model.Event
		if err := sonic.UnmarshalString(dataStr, &event); err != nil {
			continue
		}
		event.StreamID = msg.ID
		events = append(events, &event)
	}
	return events, nil
}

func (s *RedisEventStore) SetTTL(ctx context.Context, taskID string, ttl time.Duration) error {
	return s.client.Expire(ctx, s.streamKey(taskID), ttl).Err()
}

func (s *RedisEventStore) Exists(ctx context.Context, taskID string) (bool, error) {
	n, err := s.client.Exists(ctx, s.streamKey(taskID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

var _ executor.EventStore = (*RedisEventStore)(nil)
