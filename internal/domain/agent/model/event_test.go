package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventTypeExpired(t *testing.T) {
	assert.Equal(t, EventType("expired"), EventTypeExpired)
}

func TestEventStreamIDNotSerialized(t *testing.T) {
	event := &Event{
		Type:      EventTypeChunk,
		TaskID:    "test-task",
		Timestamp: time.Now(),
		Data:      "hello",
		StreamID:  "1710000000000-0",
	}
	assert.Equal(t, "1710000000000-0", event.StreamID)
}
