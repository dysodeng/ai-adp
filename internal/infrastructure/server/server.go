package server

import "context"

// Server 服务器接口，所有服务类型（HTTP/gRPC/WS/Health）均实现此接口
type Server interface {
	// IsEnabled 是否启用此服务
	IsEnabled() bool
	// Name 服务名称，用于日志
	Name() string
	// Addr 监听地址，如 ":8080"
	Addr() string
	// Start 启动服务（非阻塞，内部启动 goroutine）
	Start() error
	// Stop 优雅停止服务
	Stop(ctx context.Context) error
}
