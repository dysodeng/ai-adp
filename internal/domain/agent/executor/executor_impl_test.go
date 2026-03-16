package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

type mockEventStore struct {
	appendCalled int
	lastTaskID   string
	lastEvent    *model.Event
	returnID     string
	returnErr    error
	deleteCalled int
}

func (m *mockEventStore) Append(_ context.Context, taskID string, event *model.Event) (string, error) {
	m.appendCalled++
	m.lastTaskID = taskID
	m.lastEvent = event
	return m.returnID, m.returnErr
}
func (m *mockEventStore) ReadAfter(_ context.Context, _ string, _ string, _ int64) ([]*model.Event, error) {
	return nil, nil
}
func (m *mockEventStore) SetTTL(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (m *mockEventStore) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockEventStore) Delete(_ context.Context, _ string) error {
	m.deleteCalled++
	return nil
}

func newTestExecutor(opts ...Option) AgentExecutor {
	return NewAgentExecutor(
		context.Background(), uuid.New(), "test-app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{Query: "test"}, opts...,
	)
}

func TestNewAgentExecutorWithoutOptions(t *testing.T) {
	exec := newTestExecutor()
	assert.NotNil(t, exec)
	assert.Equal(t, model.ExecutionStatusPending, exec.GetStatus())
}

func TestWithEventStore(t *testing.T) {
	store := &mockEventStore{returnID: "1710000000000-0"}
	exec := newTestExecutor(WithEventStore(store))
	exec.Start()
	assert.Equal(t, 1, store.appendCalled)
	assert.Equal(t, model.EventTypeStart, store.lastEvent.Type)
}

func TestEventStoreDeleteOnComplete(t *testing.T) {
	store := &mockEventStore{returnID: "100-0"}
	exec := newTestExecutor(WithEventStore(store))
	exec.Start()
	exec.Complete(&model.ExecutionOutput{})
	assert.Equal(t, 1, store.deleteCalled)
}

func TestEventStoreDeleteOnFail(t *testing.T) {
	store := &mockEventStore{returnID: "100-0"}
	exec := newTestExecutor(WithEventStore(store))
	exec.Start()
	exec.Fail(fmt.Errorf("test error"))
	assert.Equal(t, 1, store.deleteCalled)
}

func TestEventStoreDeleteOnCancel(t *testing.T) {
	store := &mockEventStore{returnID: "100-0"}
	exec := newTestExecutor(WithEventStore(store))
	exec.Start()
	exec.Cancel()
	assert.Equal(t, 1, store.deleteCalled)
}

func TestWithoutEventStoreNoError(t *testing.T) {
	exec := newTestExecutor()
	exec.Start()
	exec.PublishChunk("hello")
	exec.Complete(&model.ExecutionOutput{})
}
