package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

// compile-time check
var _ Server = (*HTTPServer)(nil)

// HTTPServer HTTP 服务器，实现 Server 接口
type HTTPServer struct {
	cfg           *config.Config
	server        *http.Server
	tenantHandler *handler.TenantHandler
}

func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler) *HTTPServer {
	return &HTTPServer{cfg: cfg, tenantHandler: tenantHandler}
}

func (s *HTTPServer) IsEnabled() bool { return true }
func (s *HTTPServer) Name() string    { return "HTTP" }
func (s *HTTPServer) Addr() string    { return fmt.Sprintf(":%d", s.cfg.Server.HTTP.Port) }

// Start 初始化 Gin engine、注册中间件和路由，在 goroutine 中启动
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

	s.server = &http.Server{
		Addr:              s.Addr(),
		Handler:           r,
		ReadTimeout:       time.Duration(s.cfg.Server.HTTP.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(s.cfg.Server.HTTP.WriteTimeout) * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 短暂等待确认启动无立即错误
	select {
	case err := <-errCh:
		return err
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// Stop 优雅关闭 HTTP 服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
