package stream

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, client
}

func TestRedisEventStore_Append(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")
	event := &model.Event{
		Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "hello",
	}
	streamID, err := store.Append(context.Background(), "task-1", event)
	require.NoError(t, err)
	assert.NotEmpty(t, streamID)
}

func TestRedisEventStore_ReadAfter(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")
	ctx := context.Background()
	id1, _ := store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeStart, TaskID: "task-1", Timestamp: time.Now()})
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "chunk1"})
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "chunk2"})
	events, err := store.ReadAfter(ctx, "task-1", id1, 10000)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, model.EventTypeChunk, events[0].Type)
	assert.NotEmpty(t, events[0].StreamID)
}

func TestRedisEventStore_ReadAfter_EmptyStream(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")
	events, err := store.ReadAfter(context.Background(), "nonexistent", "0-0", 10000)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestRedisEventStore_SetTTL(t *testing.T) {
	mr, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")
	ctx := context.Background()
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeStart, TaskID: "task-1", Timestamp: time.Now()})
	err := store.SetTTL(ctx, "task-1", 30*time.Second)
	require.NoError(t, err)
	ttl := mr.TTL("test:sse:events:task-1")
	assert.True(t, ttl > 0 && ttl <= 30*time.Second)
}

func TestRedisEventStore_Exists(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")
	ctx := context.Background()
	exists, err := store.Exists(ctx, "task-1")
	require.NoError(t, err)
	assert.False(t, exists)
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeStart, TaskID: "task-1", Timestamp: time.Now()})
	exists, err = store.Exists(ctx, "task-1")
	require.NoError(t, err)
	assert.True(t, exists)
}
