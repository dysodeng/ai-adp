package protocol

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/stream"
)

func TestReconnection_FullFlow(t *testing.T) {
	// Setup Redis
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	eventStore := stream.NewRedisEventStore(client, "test:")

	// Create executor with event store
	taskID := uuid.New()
	exec := executor.NewAgentExecutor(
		context.Background(), taskID, "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{Query: "test"},
		executor.WithEventStore(eventStore),
	)

	// Start and publish some events
	exec.Start()
	exec.PublishChunk("hello ")
	exec.PublishChunk("world")

	// Simulate client disconnect — read events from stream to get last ID
	events, err := eventStore.ReadAfter(context.Background(), taskID.String(), "0-0", 10000)
	require.NoError(t, err)
	require.Len(t, events, 3) // start + 2 chunks

	// Client "disconnects" after receiving 3rd event (index 2, the last one before disconnect)
	lastReceivedID := events[2].StreamID

	// More events arrive while client is disconnected (task still running)
	exec.PublishChunk("!")

	// Client reconnects before task completes — read events after lastReceivedID
	cachedEvents, err := eventStore.ReadAfter(context.Background(), taskID.String(), lastReceivedID, 10000)
	require.NoError(t, err)
	assert.Len(t, cachedEvents, 1) // "!" chunk

	// Task completes (this deletes the Redis stream)
	exec.Complete(&model.ExecutionOutput{
		Message: &model.Message{Role: "assistant", Content: model.MessageContent{Content: "hello world!"}},
	})

	// Replay via SSE
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, true)
	require.NoError(t, err)

	err = adapter.HandleReconnection(context.Background(), cachedEvents, nil) // nil = task already finished
	require.NoError(t, err)

	body := w.Body.String()
	assert.True(t, strings.Contains(body, "event: chunk"))
}

func TestReconnection_ExpiredStream(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	eventStore := stream.NewRedisEventStore(client, "test:")

	// Write an event then expire the key
	taskID := "expired-task"
	_, _ = eventStore.Append(context.Background(), taskID, &model.Event{Type: model.EventTypeStart, TaskID: taskID, Timestamp: time.Now()})

	// Fast-forward time to expire the key
	mr.FastForward(61 * time.Second)

	exists, err := eventStore.Exists(context.Background(), taskID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestReconnection_BackwardCompatibility(t *testing.T) {
	// Without enableResume, SSE should work exactly as before
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, false)
	require.NoError(t, err)

	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{Query: "test"},
	)

	// Run in goroutine since HandleExecution blocks
	go func() {
		exec.Start()
		exec.PublishChunk("hello")
		exec.Complete(&model.ExecutionOutput{})
	}()

	err = adapter.HandleExecution(context.Background(), exec)
	require.NoError(t, err)

	body := w.Body.String()
	assert.NotContains(t, body, "id: ")    // No stream IDs
	assert.NotContains(t, body, "retry: ") // No retry directive
	assert.Contains(t, body, "event: start")
}
