package stream

import (
	"sync"
	"time"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

type MemoryExecutorRegistry struct {
	executors sync.Map
}

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

func (r *MemoryExecutorRegistry) DelayedUnregister(taskID string, delay time.Duration) {
	time.AfterFunc(delay, func() {
		r.executors.Delete(taskID)
	})
}

var _ executor.ExecutorRegistry = (*MemoryExecutorRegistry)(nil)
