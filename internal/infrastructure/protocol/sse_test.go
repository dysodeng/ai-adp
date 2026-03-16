package protocol

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

func TestSSEAdapter_SendEventWithStreamID(t *testing.T) {
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, true)
	require.NoError(t, err)
	event := &model.Event{
		Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(),
		Data: "hello", StreamID: "1710000000000-0",
	}
	err = adapter.SendEvent(event)
	require.NoError(t, err)
	body := w.Body.String()
	assert.Contains(t, body, "id: 1710000000000-0\n")
	assert.Contains(t, body, "event: chunk\n")
	assert.Contains(t, body, "data: ")
}

func TestSSEAdapter_SendEventWithoutResume(t *testing.T) {
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, false)
	require.NoError(t, err)
	event := &model.Event{
		Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(),
		Data: "hello", StreamID: "1710000000000-0",
	}
	err = adapter.SendEvent(event)
	require.NoError(t, err)
	body := w.Body.String()
	assert.NotContains(t, body, "id: ")
	assert.Contains(t, body, "event: chunk\n")
}

func TestSSEAdapter_FirstEventIncludesRetry(t *testing.T) {
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, true)
	require.NoError(t, err)
	_ = adapter.SendEvent(&model.Event{
		Type: model.EventTypeStart, TaskID: "task-1", Timestamp: time.Now(), StreamID: "100-0",
	})
	_ = adapter.SendEvent(&model.Event{
		Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), StreamID: "100-1",
	})
	body := w.Body.String()
	assert.Contains(t, body, "retry: 3000\n")
	// Count occurrences - retry should appear exactly once
	count := 0
	for i := 0; i < len(body)-10; i++ {
		if body[i:i+11] == "retry: 3000" {
			count++
		}
	}
	assert.Equal(t, 1, count, "retry should appear exactly once")
}

func TestSSEAdapter_HandleReconnection_CachedOnly(t *testing.T) {
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, true)
	require.NoError(t, err)
	cached := []*model.Event{
		{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "chunk1", StreamID: "100-0"},
		{Type: model.EventTypeComplete, TaskID: "task-1", Timestamp: time.Now(), StreamID: "100-1"},
	}
	err = adapter.HandleReconnection(context.Background(), cached, nil)
	require.NoError(t, err)
	body := w.Body.String()
	assert.Contains(t, body, "id: 100-0")
	assert.Contains(t, body, "id: 100-1")
}
