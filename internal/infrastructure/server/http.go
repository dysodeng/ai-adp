package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/middleware"
)

type HTTPServer struct {
	engine *gin.Engine
	server *http.Server
}

func NewHTTPServer(cfg *config.Config, tenantHandler *handler.TenantHandler) *HTTPServer {
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.RequestID())
	r.GET("/health", handler.HealthCheck)

	v1 := r.Group("/api/v1")
	tenantHandler.RegisterRoutes(v1)

	return &HTTPServer{
		engine: r,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
			Handler:      r,
			ReadTimeout:  time.Duration(cfg.Server.HTTP.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.HTTP.WriteTimeout) * time.Second,
		},
	}
}

func (s *HTTPServer) Start() error {
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Engine returns the underlying gin.Engine for route registration by modules
func (s *HTTPServer) Engine() *gin.Engine {
	return s.engine
}

// ErrServerClosed re-exports http.ErrServerClosed for callers
var ErrServerClosed = http.ErrServerClosed

// IsErrServerClosed checks if an error is http.ErrServerClosed
func IsErrServerClosed(err error) bool {
	return errors.Is(err, http.ErrServerClosed)
}
