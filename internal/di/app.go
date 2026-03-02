package di

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
)

// App 应用主结构
type App struct {
	httpServer *server.HTTPServer
	logger     *zap.Logger
}

func NewApp(httpServer *server.HTTPServer, logger *zap.Logger) *App {
	return &App{httpServer: httpServer, logger: logger}
}

// Run 启动服务，阻塞直到收到退出信号
func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server starting")
		if err := a.httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		a.logger.Info("Shutting down", zap.String("signal", sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.logger.Error("Shutdown error", zap.Error(err))
			return err
		}
		a.logger.Info("Server stopped gracefully")
		return nil
	}
}
