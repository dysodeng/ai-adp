package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// compile-time check
var _ Server = (*HTTPServer)(nil)

// HTTPServer HTTP 服务器，实现 Server 接口
type HTTPServer struct {
	cfg           *config.Config
	mu            sync.Mutex
	server        *http.Server
	tenantHandler *handler.TenantHandler
}

func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler) *HTTPServer {
	return &HTTPServer{cfg: cfg, tenantHandler: tenantHandler}
}

func (s *HTTPServer) IsEnabled() bool { return true }
func (s *HTTPServer) Name() string    { return "HTTP" }
func (s *HTTPServer) Addr() string    { return fmt.Sprintf(":%d", s.cfg.Server.HTTP.Port) }

// Start 初始化 Gin engine、注册中间件和路由，同步绑定端口后启动 goroutine
func (s *HTTPServer) Start() error {
	if s.cfg.App.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.Recovery(),
		middleware.Tracing(s.cfg.App.Name),
		middleware.Logger(),
		middleware.RequestID(),
	)

	r.GET("/health", handler.HealthCheck)
	v1 := r.Group("/api/v1")
	s.tenantHandler.RegisterRoutes(v1)

	// 同步绑定端口，立即暴露 bind 错误
	ln, err := net.Listen("tcp", s.Addr())
	if err != nil {
		return fmt.Errorf("HTTP server failed to listen on %s: %w", s.Addr(), err)
	}

	srv := &http.Server{
		Handler:           r,
		ReadTimeout:       time.Duration(s.cfg.Server.HTTP.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(s.cfg.Server.HTTP.WriteTimeout) * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.mu.Lock()
	s.server = srv
	s.mu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error(context.Background(), "HTTP server error", logger.ErrorField(err))
		}
	}()

	return nil
}

// Stop 优雅关闭 HTTP 服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}
