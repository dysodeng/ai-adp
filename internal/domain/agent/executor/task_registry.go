package executor

import "context"

// TaskRegistry 任务注册表 - 管理 taskID 与 cancelFunc 的映射
type TaskRegistry interface {
	Register(taskID string, cancelFunc context.CancelFunc)
	Unregister(taskID string)
	Cancel(taskID string) bool
}
