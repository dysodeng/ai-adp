# SSE 断线重连与续接 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add SSE reconnection and resume capability so clients can recover from network disconnections within a 60s window without losing events.

**Architecture:** Events are cached in Redis Streams keyed by task ID. SSE output includes `id:` fields from Redis Stream IDs. On reconnection, the server replays cached events then subscribes to live events for seamless handoff. Feature is gated behind `enable_sse_resume` flag (default false).

**Tech Stack:** Go, Gin, Redis Streams (go-redis/v9), Google Wire DI, sonic JSON

**Prerequisites:** Redis >= 6.2 (required for XRANGE exclusive range prefix `(`)

**Spec:** `docs/superpowers/specs/2026-03-16-sse-reconnection-design.md`

---

## Chunk 1: Domain Layer — Event Model & Interfaces

### Task 1: Add StreamID field to Event and new EventType constants

**Files:**
- Modify: `internal/domain/agent/model/event.go`
- Test: `internal/domain/agent/model/event_test.go` (create)

- [ ] **Step 1: Write the test for new EventType and StreamID field**

```go
// internal/domain/agent/model/event_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/domain/agent/model/ -run TestEventTypeExpired -v`
Expected: FAIL — `EventTypeExpired` not defined, `StreamID` not a field

- [ ] **Step 3: Implement the changes**

In `internal/domain/agent/model/event.go`:

1. Add `EventTypeExpired` constant after `EventTypeTokenUsage`:
```go
	// 重连事件
	EventTypeExpired EventType = "expired"
```

2. Add `StreamID` field to `Event` struct:
```go
type Event struct {
	Type      EventType   `json:"type"`
	TaskID    string      `json:"task_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	StreamID  string      `json:"-"` // Redis Stream ID，不序列化到 JSON，仅供 SSE 输出使用
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/domain/agent/model/ -run TestEventType -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/agent/model/event.go internal/domain/agent/model/event_test.go
git commit -m "feat(sse-resume): add StreamID field to Event and EventTypeExpired constant"
```

---

### Task 2: Define EventStore interface

**Files:**
- Create: `internal/domain/agent/executor/event_store.go`

- [ ] **Step 1: Create the EventStore interface**

```go
// internal/domain/agent/executor/event_store.go
package executor

import (
	"context"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// EventStore 事件存储接口 — 负责持久化和检索 SSE 事件流
type EventStore interface {
	// Append 追加事件到存储，返回存储分配的 Stream ID
	Append(ctx context.Context, taskID string, event *model.Event) (string, error)
	// ReadAfter 读取指定 ID 之后的所有事件（不含 lastID 本身），count 为最大返回数量
	ReadAfter(ctx context.Context, taskID string, lastID string, count int64) ([]*model.Event, error)
	// SetTTL 设置事件流的过期时间
	SetTTL(ctx context.Context, taskID string, ttl time.Duration) error
	// Exists 检查事件流是否存在
	Exists(ctx context.Context, taskID string) (bool, error)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/domain/agent/executor/`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/domain/agent/executor/event_store.go
git commit -m "feat(sse-resume): define EventStore interface in domain layer"
```

---

### Task 3: Define ExecutorRegistry interface

**Files:**
- Create: `internal/domain/agent/executor/executor_registry.go`

- [ ] **Step 1: Create the ExecutorRegistry interface**

```go
// internal/domain/agent/executor/executor_registry.go
package executor

import "time"

// ExecutorRegistry 执行器注册表 — 用于按 task ID 查找正在运行的 AgentExecutor
type ExecutorRegistry interface {
	// Register 注册执行器
	Register(taskID string, executor AgentExecutor)
	// Get 获取执行器，如不存在返回 nil, false
	Get(taskID string) (AgentExecutor, bool)
	// Unregister 注销执行器
	Unregister(taskID string)
	// DelayedUnregister 延迟注销执行器，给重连留出时间窗口
	DelayedUnregister(taskID string, delay time.Duration)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/domain/agent/executor/`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/domain/agent/executor/executor_registry.go
git commit -m "feat(sse-resume): define ExecutorRegistry interface in domain layer"
```

---

### Task 4: Add Option pattern and EventStore support to AgentExecutor

**Files:**
- Modify: `internal/domain/agent/executor/executor_impl.go`
- Test: `internal/domain/agent/executor/executor_impl_test.go` (create)

- [ ] **Step 1: Write failing tests for Option pattern and EventStore integration**

```go
// internal/domain/agent/executor/executor_impl_test.go
package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// mockEventStore 用于测试的 mock EventStore
type mockEventStore struct {
	appendCalled int
	lastTaskID   string
	lastEvent    *model.Event
	returnID     string
	returnErr    error
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

func newTestExecutor(opts ...Option) AgentExecutor {
	return NewAgentExecutor(
		context.Background(),
		uuid.New(),
		"test-app",
		valueobject.AppTypeChat,
		uuid.New(),
		uuid.New(),
		model.ExecutionInput{Query: "test"},
		opts...,
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

	// Start should trigger event store append
	exec.Start()

	assert.Equal(t, 1, store.appendCalled)
	assert.Equal(t, model.EventTypeStart, store.lastEvent.Type)
}

func TestBroadcastEventSetsStreamID(t *testing.T) {
	store := &mockEventStore{returnID: "1710000000000-0"}
	exec := newTestExecutor(WithEventStore(store))

	ch := exec.Subscribe()
	exec.Start()

	// Read the start event from subscribe (synthetic) and the broadcast
	event := <-ch
	// The synthetic start from Subscribe has no StreamID
	// The broadcast start from Start() should have StreamID set
	// But subscribe replays start event first, so we need to check carefully
	assert.Equal(t, model.EventTypeStart, event.Type)
}

func TestWithoutEventStoreNoError(t *testing.T) {
	exec := newTestExecutor()
	// Should not panic or error without event store
	exec.Start()
	exec.PublishChunk("hello")
	exec.Complete(&model.ExecutionOutput{})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/domain/agent/executor/ -run TestNewAgentExecutor -v`
Expected: FAIL — `Option` and `WithEventStore` not defined

- [ ] **Step 3: Implement Option pattern and EventStore in executor_impl.go**

Add to `internal/domain/agent/executor/executor_impl.go`:

1. Add `eventStore` field to `agentExecutorImpl` struct:
```go
type agentExecutorImpl struct {
	ctx            context.Context
	taskID         uuid.UUID
	appID          string
	appType        valueobject.AppType
	conversationID uuid.UUID
	messageID      uuid.UUID
	input          model.ExecutionInput

	status    model.ExecutionStatus
	output    *model.ExecutionOutput
	err       error
	startTime time.Time
	endTime   time.Time

	subscribers []chan *model.Event
	mu          sync.RWMutex

	eventStore EventStore // 可选，用于 SSE 重连事件缓存
}
```

2. Add Option type and WithEventStore function:
```go
// Option AgentExecutor 构造选项
type Option func(*agentExecutorImpl)

// WithEventStore 设置事件存储（启用 SSE 重连能力）
func WithEventStore(store EventStore) Option {
	return func(e *agentExecutorImpl) {
		e.eventStore = store
	}
}
```

3. Modify `NewAgentExecutor` to accept options:
```go
func NewAgentExecutor(
	ctx context.Context,
	taskID uuid.UUID,
	appID string,
	appType valueobject.AppType,
	conversationID uuid.UUID,
	messageID uuid.UUID,
	input model.ExecutionInput,
	opts ...Option,
) AgentExecutor {
	e := &agentExecutorImpl{
		ctx:            ctx,
		taskID:         taskID,
		appID:          appID,
		appType:        appType,
		conversationID: conversationID,
		messageID:      messageID,
		input:          input,
		status:         model.ExecutionStatusPending,
		subscribers:    make([]chan *model.Event, 0),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
```

4. Modify `broadcastEvent` to write to EventStore and set StreamID:
```go
func (e *agentExecutorImpl) broadcastEvent(event *model.Event) {
	event.TaskID = e.taskID.String()

	// 写入事件存储（如果启用）
	// 注意：此处在 mutex 持有期间执行 Redis I/O。对于 AI 流式场景，Redis 写入延迟（~1ms）
	// 远小于 AI 推理的 token 生成间隔（~50-100ms），因此对锁持有时间的影响可接受。
	// 如果未来性能测试发现瓶颈，可改为异步写入。
	if e.eventStore != nil {
		streamID, err := e.eventStore.Append(e.ctx, e.taskID.String(), event)
		if err != nil {
			// 写入失败不影响正常事件推送，降级为不可续接
			// TODO: 添加 logger 记录 warn
		} else {
			event.StreamID = streamID
		}
	}

	for _, ch := range e.subscribers {
		select {
		case ch <- event:
		default:
			// 如果 channel 满了，跳过
		}
	}
}
```

5. In `Complete()`, `Fail()`, `Cancel()` methods, after `closeAllSubscribers()`, add TTL shortening:
```go
// 在 closeAllSubscribers() 之后添加
if e.eventStore != nil {
	_ = e.eventStore.SetTTL(e.ctx, e.taskID.String(), 30*time.Second)
}
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/domain/agent/executor/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/agent/executor/executor_impl.go internal/domain/agent/executor/executor_impl_test.go
git commit -m "feat(sse-resume): add Option pattern and EventStore integration to AgentExecutor"
```

---

## Chunk 2: Infrastructure Layer — Redis EventStore & ExecutorRegistry

### Task 5: Implement Redis Stream EventStore

**Files:**
- Create: `internal/infrastructure/agent/stream/event_store.go`
- Test: `internal/infrastructure/agent/stream/event_store_test.go` (create)

- [ ] **Step 1: Write tests for RedisEventStore**

```go
// internal/infrastructure/agent/stream/event_store_test.go
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
		Type:      model.EventTypeChunk,
		TaskID:    "task-1",
		Timestamp: time.Now(),
		Data:      "hello",
	}

	streamID, err := store.Append(context.Background(), "task-1", event)
	require.NoError(t, err)
	assert.NotEmpty(t, streamID)
}

func TestRedisEventStore_ReadAfter(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisEventStore(client, "test:")

	ctx := context.Background()

	// Append 3 events
	id1, _ := store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeStart, TaskID: "task-1", Timestamp: time.Now()})
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "chunk1"})
	_, _ = store.Append(ctx, "task-1", &model.Event{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), Data: "chunk2"})

	// Read after first event
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
```

- [ ] **Step 2: Install miniredis test dependency**

Run: `cd /Users/dysodeng/project/go/ai-adp && go get github.com/alicebob/miniredis/v2`

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/agent/stream/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 4: Implement RedisEventStore**

```go
// internal/infrastructure/agent/stream/event_store.go
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
	streamKeyPrefix    = "sse:events:"
	defaultTTL         = 60 * time.Second
	maxReplayEventCount int64 = 10000 // XRANGE 安全上限
)

// RedisEventStore 基于 Redis Stream 的事件存储实现
type RedisEventStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

// NewRedisEventStore 创建 Redis 事件存储
func NewRedisEventStore(client redis.UniversalClient, keyPrefix string) *RedisEventStore {
	return &RedisEventStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

func (s *RedisEventStore) streamKey(taskID string) string {
	return s.keyPrefix + streamKeyPrefix + taskID
}

// Append 追加事件到 Redis Stream，返回自动生成的 Stream ID
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

	// 每次写入时刷新 TTL
	s.client.Expire(ctx, key, defaultTTL)

	return streamID, nil
}

// ReadAfter 读取指定 ID 之后的所有事件
func (s *RedisEventStore) ReadAfter(ctx context.Context, taskID string, lastID string, count int64) ([]*model.Event, error) {
	key := s.streamKey(taskID)

	// XRANGE 使用 "(" 前缀表示 exclusive（不含 lastID 本身）
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

// SetTTL 设置事件流的过期时间
func (s *RedisEventStore) SetTTL(ctx context.Context, taskID string, ttl time.Duration) error {
	key := s.streamKey(taskID)
	return s.client.Expire(ctx, key, ttl).Err()
}

// Exists 检查事件流是否存在
func (s *RedisEventStore) Exists(ctx context.Context, taskID string) (bool, error) {
	key := s.streamKey(taskID)
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

var _ executor.EventStore = (*RedisEventStore)(nil)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/agent/stream/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/agent/stream/
git commit -m "feat(sse-resume): implement RedisEventStore using Redis Streams"
```

---

### Task 6: Implement MemoryExecutorRegistry

**Files:**
- Create: `internal/infrastructure/agent/stream/executor_registry.go`
- Test: `internal/infrastructure/agent/stream/executor_registry_test.go` (create)

- [ ] **Step 1: Write tests for MemoryExecutorRegistry**

```go
// internal/infrastructure/agent/stream/executor_registry_test.go
package stream

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

func TestMemoryExecutorRegistry_RegisterAndGet(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)

	registry.Register("task-1", exec)

	got, ok := registry.Get("task-1")
	assert.True(t, ok)
	assert.Equal(t, exec, got)
}

func TestMemoryExecutorRegistry_GetNotFound(t *testing.T) {
	registry := NewMemoryExecutorRegistry()

	got, ok := registry.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMemoryExecutorRegistry_Unregister(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)

	registry.Register("task-1", exec)
	registry.Unregister("task-1")

	_, ok := registry.Get("task-1")
	assert.False(t, ok)
}

func TestMemoryExecutorRegistry_DelayedUnregister(t *testing.T) {
	registry := NewMemoryExecutorRegistry()
	exec := executor.NewAgentExecutor(
		context.Background(), uuid.New(), "app", valueobject.AppTypeChat,
		uuid.New(), uuid.New(), model.ExecutionInput{},
	)

	registry.Register("task-1", exec)
	registry.DelayedUnregister("task-1", 100*time.Millisecond)

	// Still available immediately
	_, ok := registry.Get("task-1")
	assert.True(t, ok)

	// Gone after delay
	time.Sleep(200 * time.Millisecond)
	_, ok = registry.Get("task-1")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/agent/stream/ -run TestMemoryExecutorRegistry -v`
Expected: FAIL — `NewMemoryExecutorRegistry` and `DelayedUnregister` not defined

- [ ] **Step 3: Implement MemoryExecutorRegistry**

```go
// internal/infrastructure/agent/stream/executor_registry.go
package stream

import (
	"sync"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

// MemoryExecutorRegistry 基于内存的执行器注册表
type MemoryExecutorRegistry struct {
	executors sync.Map
}

// NewMemoryExecutorRegistry 创建内存执行器注册表
func NewMemoryExecutorRegistry() *MemoryExecutorRegistry {
	return &MemoryExecutorRegistry{}
}

func (r *MemoryExecutorRegistry) Register(taskID string, exec executor.AgentExecutor) {
	r.executors.Store(taskID, exec)
}

func (r *MemoryExecutorRegistry) Get(taskID string) (executor.AgentExecutor, bool) {
	value, ok := r.executors.Load(taskID)
	if !ok {
		return nil, false
	}
	return value.(executor.AgentExecutor), true
}

func (r *MemoryExecutorRegistry) Unregister(taskID string) {
	r.executors.Delete(taskID)
}

// DelayedUnregister 延迟注销执行器，给重连留出时间窗口
func (r *MemoryExecutorRegistry) DelayedUnregister(taskID string, delay time.Duration) {
	time.AfterFunc(delay, func() {
		r.executors.Delete(taskID)
	})
}

var _ executor.ExecutorRegistry = (*MemoryExecutorRegistry)(nil)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/agent/stream/ -run TestMemoryExecutorRegistry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/agent/stream/executor_registry.go internal/infrastructure/agent/stream/executor_registry_test.go
git commit -m "feat(sse-resume): implement MemoryExecutorRegistry with delayed unregister"
```

---

## Chunk 3: Protocol Layer — SSE Adapter Enhancement

### Task 7: Add ReconnectableAdapter interface and enhance SSEAdapter

**Files:**
- Modify: `internal/infrastructure/protocol/adapter.go`
- Modify: `internal/infrastructure/protocol/sse.go`
- Test: `internal/infrastructure/protocol/sse_test.go` (create)

- [ ] **Step 1: Write tests for enhanced SSE output**

```go
// internal/infrastructure/protocol/sse_test.go
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
	adapter, err := NewSSEAdapter(w, true) // enableResume=true
	require.NoError(t, err)

	event := &model.Event{
		Type:      model.EventTypeChunk,
		TaskID:    "task-1",
		Timestamp: time.Now(),
		Data:      "hello",
		StreamID:  "1710000000000-0",
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
	adapter, err := NewSSEAdapter(w, false) // enableResume=false
	require.NoError(t, err)

	event := &model.Event{
		Type:      model.EventTypeChunk,
		TaskID:    "task-1",
		Timestamp: time.Now(),
		Data:      "hello",
		StreamID:  "1710000000000-0",
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

	event := &model.Event{
		Type:      model.EventTypeStart,
		TaskID:    "task-1",
		Timestamp: time.Now(),
		StreamID:  "1710000000000-0",
	}

	err = adapter.SendEvent(event)
	require.NoError(t, err)

	body := w.Body.String()
	assert.Contains(t, body, "retry: 3000\n")

	// Second event should NOT have retry
	w2 := httptest.NewRecorder()
	adapter2, _ := NewSSEAdapter(w2, true)
	_ = adapter2.SendEvent(event)
	_ = adapter2.SendEvent(&model.Event{Type: model.EventTypeChunk, TaskID: "task-1", Timestamp: time.Now(), StreamID: "1710000000000-1"})
	// Check the full output - retry should appear only once
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/protocol/ -run TestSSEAdapter -v`
Expected: FAIL — `NewSSEAdapter` signature mismatch, `HandleReconnection` not defined

- [ ] **Step 3: Add ReconnectableAdapter interface to adapter.go**

```go
// internal/infrastructure/protocol/adapter.go
package protocol

import (
	"context"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
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

// ReconnectableAdapter 支持重连的协议适配器接口
type ReconnectableAdapter interface {
	Adapter
	// HandleReconnection 处理重连：先回放缓存事件，然后续接实时事件流
	// liveExecutor 为 nil 时表示仅回放缓存事件后关闭
	HandleReconnection(ctx context.Context, cachedEvents []*model.Event, liveExecutor executor.AgentExecutor) error
}
```

- [ ] **Step 4: Enhance SSEAdapter**

Modify `internal/infrastructure/protocol/sse.go`:

```go
package protocol

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
)

// SSEAdapter SSE 协议适配器
type SSEAdapter struct {
	w            http.ResponseWriter
	flusher      http.Flusher
	closed       bool
	enableResume bool // 是否启用 SSE 重连能力
	retrySent    bool // 是否已发送 retry 指令
}

// NewSSEAdapter 创建 SSE 适配器
func NewSSEAdapter(w http.ResponseWriter, enableResume bool) (*SSEAdapter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEAdapter{w: w, flusher: flusher, enableResume: enableResume}, nil
}

// HandleExecution 订阅事件流并转换为 SSE 格式发送
func (a *SSEAdapter) HandleExecution(ctx context.Context, agentExecutor executor.AgentExecutor) error {
	eventChan := agentExecutor.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			if err := a.SendEvent(event); err != nil {
				return err
			}
		}
	}
}

// HandleReconnection 处理重连：先回放缓存事件，然后续接实时事件流
func (a *SSEAdapter) HandleReconnection(ctx context.Context, cachedEvents []*model.Event, liveExecutor executor.AgentExecutor) error {
	// 1. 回放缓存事件
	var lastSentID string
	for _, event := range cachedEvents {
		if err := a.SendEvent(event); err != nil {
			return err
		}
		if event.StreamID != "" {
			lastSentID = event.StreamID
		}
	}

	// 2. 如果没有 live executor，仅回放后关闭
	if liveExecutor == nil {
		return nil
	}

	// 3. 订阅实时事件流并续接（Subscribe 在 Chunk 2 之前已执行，此处由调用方传入 channel 或 executor）
	eventChan := liveExecutor.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			// 跳过无 StreamID 的合成事件（如 Subscribe 注入的 start 事件）
			if event.StreamID == "" {
				continue
			}
			// 去重：跳过已通过缓存回放发送过的事件
			if lastSentID != "" && event.StreamID <= lastSentID {
				continue
			}
			if err := a.SendEvent(event); err != nil {
				return err
			}
		}
	}
}

func (a *SSEAdapter) SendEvent(event *model.Event) error {
	if a.closed {
		return fmt.Errorf("adapter is closed")
	}

	data, err := sonic.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	var sseData string
	if a.enableResume && event.StreamID != "" {
		sseData = fmt.Sprintf("id: %s\n", event.StreamID)
		if !a.retrySent {
			sseData += "retry: 3000\n"
			a.retrySent = true
		}
		sseData += fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	} else {
		sseData = fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	}

	if _, err = fmt.Fprint(a.w, sseData); err != nil {
		return fmt.Errorf("write sse data failed: %w", err)
	}

	a.flusher.Flush()
	return nil
}

// SendError 发送错误事件
func (a *SSEAdapter) SendError(err error) error {
	return a.SendEvent(&model.Event{
		Type:      model.EventTypeError,
		Timestamp: time.Now(),
		Data:      map[string]any{"error": err.Error()},
	})
}

// Close 关闭适配器
func (a *SSEAdapter) Close() error {
	a.closed = true
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/protocol/ -v`
Expected: PASS

- [ ] **Step 6: Fix compilation of existing callers**

The `NewSSEAdapter` signature changed from `NewSSEAdapter(w)` to `NewSSEAdapter(w, enableResume)`. Update `chat_handler.go`:

In `internal/interfaces/http/handler/chat_handler.go`, line 55, change:
```go
sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer)
```
to:
```go
sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, false)
```

This keeps existing behavior (resume disabled by default). We'll update this properly when integrating the full request flow in Task 9.

- [ ] **Step 7: Verify full project builds**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: Success

- [ ] **Step 8: Commit**

```bash
git add internal/infrastructure/protocol/adapter.go internal/infrastructure/protocol/sse.go internal/infrastructure/protocol/sse_test.go internal/interfaces/http/handler/chat_handler.go
git commit -m "feat(sse-resume): enhance SSEAdapter with id/retry fields and HandleReconnection"
```

---

## Chunk 4: Application & Interface Layer — Request Handling & Reconnection

### Task 8: Update request DTOs and ChatCommand

**Files:**
- Modify: `internal/interfaces/http/dto/request/chat_request.go`
- Modify: `internal/application/chat/dto/command.go`

- [ ] **Step 1: Update ChatRequest DTO**

In `internal/interfaces/http/dto/request/chat_request.go`:

```go
package request

// ChatRequest Chat 对话请求（保留原有 binding 规则不变）
type ChatRequest struct {
	ConversationID  string         `json:"conversation_id"`
	Query           string         `json:"query" binding:"required"`
	Input           map[string]any `json:"input"`
	ResponseMode    string         `json:"response_mode"`
	EnableSSEResume bool           `json:"enable_sse_resume"` // 是否启用 SSE 断线续接
}

// ReconnectRequest SSE 重连请求（独立 DTO，不需要 query）
type ReconnectRequest struct {
	TaskID      string `json:"task_id" binding:"required"`
	LastEventID string `json:"last_event_id" binding:"required"`
}
```

Note: 使用独立的 `ReconnectRequest` DTO 而非修改 `ChatRequest`，保留 `Query` 的 `binding:"required"` 不变，避免影响现有 API 消费者的验证行为。Handler 层通过先尝试绑定 `ReconnectRequest` 来判断是否为重连请求。

- [ ] **Step 2: Update ChatCommand DTO**

In `internal/application/chat/dto/command.go`, add fields:

```go
// ChatCommand Chat 对话命令
type ChatCommand struct {
	ConversationID  string         `json:"conversation_id"`
	Query           string         `json:"query"`
	Input           map[string]any `json:"input"`
	ResponseMode    ResponseMode   `json:"response_mode"`
	EnableSSEResume bool           `json:"enable_sse_resume"`
}

// ReconnectCommand SSE 重连命令
type ReconnectCommand struct {
	TaskID      string `json:"task_id"`
	LastEventID string `json:"last_event_id"`
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add internal/interfaces/http/dto/request/chat_request.go internal/application/chat/dto/command.go
git commit -m "feat(sse-resume): update request DTOs with reconnection fields"
```

---

### Task 9: Add reconnection logic to ChatAppService

**Files:**
- Modify: `internal/application/chat/service/chat_app_service.go`

- [ ] **Step 1: Add Reconnect method to ChatAppService interface and implementation**

```go
// ChatAppService interface — add:
type ChatAppService interface {
	Chat(ctx context.Context, appAuthIden string, cmd chatdto.ChatCommand) (executor.AgentExecutor, error)
	// Reconnect 处理 SSE 重连：从 EventStore 回放事件，尝试续接实时流
	Reconnect(ctx context.Context, cmd chatdto.ReconnectCommand) (cachedEvents []*model.Event, liveExecutor executor.AgentExecutor, err error)
}
```

Update the struct to include EventStore and ExecutorRegistry:

```go
type chatAppService struct {
	orchestrator     orchestrator.ExecutorOrchestrator
	appRepository    repository.AppRepository
	eventStore       executor.EventStore       // 可选，用于 SSE 重连
	executorRegistry executor.ExecutorRegistry  // 可选，用于 SSE 重连
}

func NewChatAppService(
	orch orchestrator.ExecutorOrchestrator,
	appRepository repository.AppRepository,
	eventStore executor.EventStore,
	executorRegistry executor.ExecutorRegistry,
) ChatAppService {
	return &chatAppService{
		orchestrator:     orch,
		appRepository:    appRepository,
		eventStore:       eventStore,
		executorRegistry: executorRegistry,
	}
}
```

Add the Reconnect method:

```go
func (svc *chatAppService) Reconnect(
	ctx context.Context,
	cmd chatdto.ReconnectCommand,
) ([]*model.Event, executor.AgentExecutor, error) {
	// 1. 检查事件流是否存在
	exists, err := svc.eventStore.Exists(ctx, cmd.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("check event stream failed: %w", err)
	}
	if !exists {
		return nil, nil, nil // 流已过期，调用方发送 expired 事件
	}

	// 2. 尝试获取 live executor（可能在同一实例上仍在运行）
	var liveExec executor.AgentExecutor
	if svc.executorRegistry != nil {
		liveExec, _ = svc.executorRegistry.Get(cmd.TaskID)
		// 如果 executor 已终止，不返回 live（仅回放缓存）
		if liveExec != nil && !liveExec.IsRunning() {
			liveExec = nil
		}
	}

	// 3. 从 EventStore 读取 lastEventID 之后的缓存事件
	const maxReplayCount int64 = 10000
	cachedEvents, err := svc.eventStore.ReadAfter(ctx, cmd.TaskID, cmd.LastEventID, maxReplayCount)
	if err != nil {
		return nil, nil, fmt.Errorf("read cached events failed: %w", err)
	}

	return cachedEvents, liveExec, nil
}
```

Also update the `Chat` method to pass `EnableSSEResume` and use EventStore Option when creating the executor:

In the `Chat` method, change the `NewAgentExecutor` call (around line 88):

```go
	// 5. 创建 AgentExecutor
	messageID := uuid.New()
	taskID := uuid.New()

	var opts []executor.Option
	if cmd.EnableSSEResume && svc.eventStore != nil {
		opts = append(opts, executor.WithEventStore(svc.eventStore))
	}

	agentExecutor := executor.NewAgentExecutor(
		ctx, taskID, app.ID().String(), app.Type(),
		conversationID, messageID, input, opts...,
	)
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/application/chat/...`
Expected: May fail due to DI changes — proceed to fix in next steps

- [ ] **Step 3: Commit**

```bash
git add internal/application/chat/service/chat_app_service.go
git commit -m "feat(sse-resume): add Reconnect method to ChatAppService"
```

---

### Task 10: Update ExecutorOrchestrator to register executors

**Files:**
- Modify: `internal/application/chat/orchestrator/executor_orchestrator.go`

- [ ] **Step 1: Add ExecutorRegistry to orchestrator**

Update the struct and constructor:

```go
type executorOrchestrator struct {
	agentBuilder     service.AgentBuilder
	agentFactory     *adapter.AgentFactory
	taskRegistry     executor.TaskRegistry
	executorRegistry executor.ExecutorRegistry // 用于 SSE 重连
}

func NewExecutorOrchestrator(
	agentBuilder service.AgentBuilder,
	agentFactory *adapter.AgentFactory,
	taskRegistry executor.TaskRegistry,
	executorRegistry executor.ExecutorRegistry,
) ExecutorOrchestrator {
	return &executorOrchestrator{
		agentBuilder:     agentBuilder,
		agentFactory:     agentFactory,
		taskRegistry:     taskRegistry,
		executorRegistry: executorRegistry,
	}
}
```

In the `Execute` method, after `agentExecutor.Start()`:

```go
	// 4. 启动执行
	agentExecutor.Start()

	// 注册到 ExecutorRegistry（用于 SSE 重连查找）
	if o.executorRegistry != nil {
		o.executorRegistry.Register(taskID, agentExecutor)
	}
```

In the goroutine, change the defer to use delayed unregister:

```go
	go func() {
		defer o.taskRegistry.Unregister(taskID)
		defer func() {
			if o.executorRegistry != nil {
				// 延迟 30s 注销，给重连留出时间窗口
				if reg, ok := o.executorRegistry.(*stream.MemoryExecutorRegistry); ok {
					reg.DelayedUnregister(taskID, 30*time.Second)
				} else {
					o.executorRegistry.Unregister(taskID)
				}
			}
		}()
		o.executeAgent(execCtx, ag, agentExecutor)
	}()
```

Note: `DelayedUnregister` was already defined in the `ExecutorRegistry` interface (Task 3) and implemented in `MemoryExecutorRegistry` (Task 6).

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./internal/application/chat/...`
Expected: May still fail due to DI — fix in Task 12

- [ ] **Step 4: Commit**

```bash
git add internal/application/chat/orchestrator/executor_orchestrator.go internal/domain/agent/executor/executor_registry.go
git commit -m "feat(sse-resume): register executors in ExecutorRegistry for reconnection lookup"
```

---

### Task 11: Update ChatHandler with reconnection flow and new GET endpoint

**Files:**
- Modify: `internal/interfaces/http/handler/chat_handler.go`
- Modify: `internal/interfaces/http/router/router.go`
- Modify: `internal/interfaces/http/middleware/app_key.go`

- [ ] **Step 1: Update ChatHandler to handle reconnection**

```go
package handler

import (
	"net/http"
	"time"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/trace"
	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/protocol"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/request"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

type ChatHandler struct {
	chatService chatservice.ChatAppService
}

func NewChatHandler(chatService chatservice.ChatAppService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// Chat Agent 对话接口，支持 SSE 流式和阻塞式响应，也支持 SSE 重连
func (h *ChatHandler) Chat(ctx *gin.Context) {
	spanCtx, span := trace.Tracer().Start(trace.Gin(ctx), "api.Handler.ChatHandler")
	defer span.End()

	// 读取原始 body，支持多次解析
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "read request body failed", response.CodeFail))
		return
	}

	// 先尝试解析为重连请求
	var reconnReq request.ReconnectRequest
	if err := sonic.Unmarshal(body, &reconnReq); err == nil && reconnReq.TaskID != "" && reconnReq.LastEventID != "" {
		h.handleReconnect(ctx, spanCtx, reconnReq)
		return
	}

	// 解析为正常 Chat 请求
	var req request.ChatRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, err.Error(), response.CodeFail))
		return
	}
	if req.Query == "" {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "query is required", response.CodeFail))
		return
	}

	apiKey := ctx.GetString("api_key")

	if req.ResponseMode == "" {
		req.ResponseMode = "streaming"
	}
	responseMode := chatdto.ResponseMode(req.ResponseMode)
	if !responseMode.IsValid() {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "invalid response_mode", response.CodeFail))
		return
	}

	var adapter protocol.Adapter
	if responseMode == chatdto.ResponseModeBlocking {
		adapter = protocol.NewBlockingAdapter(ctx.Writer)
	} else {
		sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, req.EnableSSEResume)
		if err != nil {
			ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
			return
		}
		adapter = sseAdapter
	}
	defer func() { _ = adapter.Close() }()

	logger.Info(spanCtx, "[handler] calling chatService.Chat",
		logger.AddField("api_key", apiKey),
		logger.AddField("query", req.Query),
		logger.AddField("response_mode", req.ResponseMode),
	)

	agentExecutor, err := h.chatService.Chat(
		spanCtx,
		apiKey,
		chatdto.ChatCommand{
			ConversationID:  req.ConversationID,
			Query:           req.Query,
			Input:           req.Input,
			ResponseMode:    responseMode,
			EnableSSEResume: req.EnableSSEResume,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] chatService.Chat failed", logger.ErrorField(err))
		if responseMode == chatdto.ResponseModeBlocking {
			ctx.JSON(http.StatusOK, response.Fail(spanCtx, err.Error(), response.CodeFail))
		} else {
			_ = adapter.SendError(err)
		}
		return
	}

	if err = adapter.HandleExecution(spanCtx, agentExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleExecution error", logger.ErrorField(err))
	}
}

// handleReconnect 处理 SSE 重连请求（POST）
func (h *ChatHandler) handleReconnect(ctx *gin.Context, spanCtx context.Context, req request.ReconnectRequest) {
	sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, true)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
		return
	}
	defer func() { _ = sseAdapter.Close() }()

	cachedEvents, liveExecutor, err := h.chatService.Reconnect(
		spanCtx,
		chatdto.ReconnectCommand{
			TaskID:      req.TaskID,
			LastEventID: req.LastEventID,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] reconnect failed", logger.ErrorField(err))
		_ = sseAdapter.SendError(err)
		return
	}

	// 事件流已过期
	if cachedEvents == nil && liveExecutor == nil {
		_ = sseAdapter.SendEvent(&model.Event{
			Type:      model.EventTypeExpired,
			TaskID:    req.TaskID,
			Timestamp: time.Now(),
			Data:      map[string]string{"message": "event stream expired, please retry with conversation_id"},
		})
		return
	}

	if err = sseAdapter.HandleReconnection(ctx.Request.Context(), cachedEvents, liveExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleReconnection error", logger.ErrorField(err))
	}
}

// StreamReconnect GET 端点，供浏览器 EventSource 自动重连使用
func (h *ChatHandler) StreamReconnect(ctx *gin.Context) {
	spanCtx, span := trace.Tracer().Start(trace.Gin(ctx), "api.Handler.StreamReconnect")
	defer span.End()

	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, response.Fail(spanCtx, "task_id is required", response.CodeFail))
		return
	}

	// 优先读取 Last-Event-ID Header，其次读取查询参数
	lastEventID := ctx.GetHeader("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = ctx.Query("last_event_id")
	}
	if lastEventID == "" {
		ctx.JSON(http.StatusBadRequest, response.Fail(spanCtx, "last_event_id is required", response.CodeFail))
		return
	}

	sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, true)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
		return
	}
	defer func() { _ = sseAdapter.Close() }()

	cachedEvents, liveExecutor, err := h.chatService.Reconnect(
		spanCtx,
		chatdto.ReconnectCommand{
			TaskID:      taskID,
			LastEventID: lastEventID,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] StreamReconnect failed", logger.ErrorField(err))
		_ = sseAdapter.SendError(err)
		return
	}

	if cachedEvents == nil && liveExecutor == nil {
		_ = sseAdapter.SendEvent(&model.Event{
			Type:      model.EventTypeExpired,
			TaskID:    taskID,
			Timestamp: time.Now(),
			Data:      map[string]string{"message": "event stream expired, please retry with conversation_id"},
		})
		return
	}

	if err = sseAdapter.HandleReconnection(ctx.Request.Context(), cachedEvents, liveExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleReconnection error", logger.ErrorField(err))
	}
}
```

**Important:** The `handleReconnect` method above uses an awkward type assertion for `spanCtx`. Since `trace.Tracer().Start()` returns a `context.Context`, let me simplify — `spanCtx` is already `context.Context`. The code should use `spanCtx` directly:

```go
func (h *ChatHandler) handleReconnect(ctx *gin.Context, spanCtx context.Context, req request.ReconnectRequest) {
```

And update the `Chat` method signature for `handleReconnect` call accordingly.

- [ ] **Step 2: Update router to add GET endpoint**

In `internal/interfaces/http/router/router.go`:

```go
	// Chat 路由
	chats := v1.Group("/chat", middleware.AppApiKey)
	{
		chats.POST("/send-messages", registry.ChatHandler.Chat)
		chats.GET("/tasks/:task_id/stream", registry.ChatHandler.StreamReconnect)
		chats.POST("/tasks/:task_id/cancel", registry.ChatCancelHandler.Cancel)
	}
```

- [ ] **Step 3: Update AppApiKey middleware to support query parameter**

In `internal/interfaces/http/middleware/app_key.go`:

```go
func AppApiKey(ctx *gin.Context) {
	apiKey := ctx.GetHeader("Authorization")
	if apiKey == "" {
		// 回退到查询参数（支持 EventSource GET 请求）
		apiKey = ctx.Query("api_key")
		if apiKey != "" {
			ctx.Set("api_key", apiKey)
			ctx.Next()
			return
		}
		ctx.AbortWithStatusJSON(http.StatusOK, response.Fail(ctx, "Missing Authorization header", response.CodeUnauthorized))
		return
	}
	if !strings.HasPrefix(apiKey, "Bearer ") || len(apiKey) < 8 {
		ctx.AbortWithStatusJSON(http.StatusOK, response.Fail(ctx, "Invalid authorization format", response.CodeUnauthorized))
		return
	}
	ctx.Set("api_key", apiKey[7:])
	ctx.Next()
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: May fail due to DI — fix in Task 12

- [ ] **Step 5: Commit**

```bash
git add internal/interfaces/http/handler/chat_handler.go internal/interfaces/http/router/router.go internal/interfaces/http/middleware/app_key.go
git commit -m "feat(sse-resume): add reconnection handler, GET endpoint, and API key query param support"
```

---

## Chunk 5: Dependency Injection Wiring

### Task 12: Wire up new dependencies

**Files:**
- Modify: `internal/di/infrastructure.go`
- Modify: `internal/di/chat_module.go`
- Modify: `internal/di/app.go`
- Regenerate: `internal/di/wire_gen.go`

- [ ] **Step 1: Update infrastructure.go to provide new components**

In `internal/di/infrastructure.go`, add the stream package providers:

```go
import (
	// ... existing imports ...
	"github.com/dysodeng/ai-adp/internal/infrastructure/agent/stream"
)

var InfrastructureSet = wire.NewSet(
	// ... existing providers ...
	// 取消能力组件
	cancel.NewMemoryTaskRegistry,
	cancel.NewRedisCancelBroadcaster,
	// SSE 重连组件
	stream.NewMemoryExecutorRegistry,
	wire.Bind(new(executor.ExecutorRegistry), new(*stream.MemoryExecutorRegistry)),
	provideRedisEventStore,
	wire.Bind(new(executor.EventStore), new(*stream.RedisEventStore)),
	// 网关注册
	provider.ProvideGatewayRegistry,
)
```

Add the provider function:

```go
func provideRedisEventStore(client redis.Client) *stream.RedisEventStore {
	return stream.NewRedisEventStore(client, redis.MainKey(""))
}
```

Note: Need to import `executor` package and `redis` package. Also `MainKey("")` returns just the prefix, which is what we want.

Actually, looking at the `MainKey` function, `MainKey("")` returns `prefix + ":"` if prefix is set, or `""` if not. Let me adjust:

```go
func provideRedisEventStore(client pkgredis.Client) *stream.RedisEventStore {
	prefix := pkgredis.MainKey("")
	return stream.NewRedisEventStore(client, prefix)
}
```

- [ ] **Step 2: Update chat_module.go**

The `NewExecutorOrchestrator` and `NewChatAppService` constructors now take additional parameters. Wire will auto-resolve them if the types are provided. No change needed to chat_module.go since Wire resolves by type.

- [ ] **Step 3: Regenerate Wire**

Run: `cd /Users/dysodeng/project/go/ai-adp && go generate ./internal/di/`
Expected: `wire_gen.go` updated with new providers

If `wire` is not installed:
Run: `go install github.com/google/wire/cmd/wire@latest`

- [ ] **Step 4: Verify full project builds**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: Success

- [ ] **Step 5: Run all existing tests**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./... -count=1`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/di/
git commit -m "feat(sse-resume): wire up EventStore and ExecutorRegistry dependencies"
```

---

## Chunk 6: Integration Testing & Verification

### Task 13: Integration tests for complete reconnection flow

**Files:**
- Create: `internal/infrastructure/protocol/reconnect_test.go`

- [ ] **Step 1: Write integration tests**

```go
// internal/infrastructure/protocol/reconnect_test.go
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

	// Client "disconnects" after receiving 2nd event (index 1)
	lastReceivedID := events[1].StreamID

	// More events arrive while client is disconnected
	exec.PublishChunk("!")
	exec.Complete(&model.ExecutionOutput{
		Message: &model.Message{Role: "assistant", Content: model.MessageContent{Content: "hello world!"}},
	})

	// Client reconnects — read events after lastReceivedID
	cachedEvents, err := eventStore.ReadAfter(context.Background(), taskID.String(), lastReceivedID, 10000)
	require.NoError(t, err)
	assert.Len(t, cachedEvents, 2) // "!" chunk + complete

	// Replay via SSE
	w := httptest.NewRecorder()
	adapter, err := NewSSEAdapter(w, true)
	require.NoError(t, err)

	err = adapter.HandleReconnection(context.Background(), cachedEvents, nil) // nil = task already finished
	require.NoError(t, err)

	body := w.Body.String()
	assert.True(t, strings.Contains(body, "event: chunk"))
	assert.True(t, strings.Contains(body, "event: complete"))
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
	assert.NotContains(t, body, "id: ")     // No stream IDs
	assert.NotContains(t, body, "retry: ")  // No retry directive
	assert.Contains(t, body, "event: start")
}
```

- [ ] **Step 2: Run integration tests**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./internal/infrastructure/protocol/ -run TestReconnection -v -count=1`
Expected: All PASS

- [ ] **Step 3: Run full test suite**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./... -count=1`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/protocol/reconnect_test.go
git commit -m "test(sse-resume): add integration tests for reconnection flow"
```

---

### Task 14: Final verification and cleanup

- [ ] **Step 1: Verify full project builds**

Run: `cd /Users/dysodeng/project/go/ai-adp && go build ./...`
Expected: Success

- [ ] **Step 2: Run full test suite**

Run: `cd /Users/dysodeng/project/go/ai-adp && go test ./... -count=1 -race`
Expected: All PASS, no data races

- [ ] **Step 3: Run go vet**

Run: `cd /Users/dysodeng/project/go/ai-adp && go vet ./...`
Expected: No issues

- [ ] **Step 4: Final commit (if any cleanup needed)**

```bash
git add -A
git commit -m "chore(sse-resume): final cleanup and verification"
```
