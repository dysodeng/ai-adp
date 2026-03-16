package executor

import "time"

// ExecutorRegistry 执行器注册表 — 用于按 task ID 查找正在运行的 AgentExecutor
type ExecutorRegistry interface {
	// Register 注册执行器
	Register(taskID string, executor AgentExecutor)
	// Get 根据 taskID 获取执行器
	Get(taskID string) (AgentExecutor, bool)
	// Unregister 注销执行器
	Unregister(taskID string)
	// DelayedUnregister 延迟注销执行器
	DelayedUnregister(taskID string, delay time.Duration)
}
