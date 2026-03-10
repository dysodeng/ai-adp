package cancel

import (
	"context"
	"sync"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
)

// MemoryTaskRegistry 基于内存的任务注册表
type MemoryTaskRegistry struct {
	tasks sync.Map
}

// NewMemoryTaskRegistry 创建内存任务注册表
func NewMemoryTaskRegistry() executor.TaskRegistry {
	return &MemoryTaskRegistry{}
}

func (r *MemoryTaskRegistry) Register(taskID string, cancelFunc context.CancelFunc) {
	r.tasks.Store(taskID, cancelFunc)
}

func (r *MemoryTaskRegistry) Unregister(taskID string) {
	r.tasks.Delete(taskID)
}

func (r *MemoryTaskRegistry) Cancel(taskID string) bool {
	value, ok := r.tasks.LoadAndDelete(taskID)
	if !ok {
		return false
	}
	cancelFunc := value.(context.CancelFunc)
	cancelFunc()
	return true
}

var _ executor.TaskRegistry = (*MemoryTaskRegistry)(nil)
