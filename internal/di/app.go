package di

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

// App 应用主结构
type App struct {
	httpServer      *server.HTTPServer
	logger          *zap.Logger
	tracerShutdown  telemetry.ShutdownFunc
}

func NewApp(httpServer *server.HTTPServer, logger *zap.Logger, tracerShutdown telemetry.ShutdownFunc) *App {
	return &App{httpServer: httpServer, logger: logger, tracerShutdown: tracerShutdown}
}

// Run 启动服务，阻塞直到收到退出信号
func (a *App) Run() error {
	a.logger.Info("HTTP server starting")
	if err := a.httpServer.Start(); err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.logger.Info("Shutting down", zap.String("signal", sig.String()))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.httpServer.Stop(ctx); err != nil {
		a.logger.Error("Shutdown error", zap.Error(err))
		return err
	}
	a.tracerShutdown()
	a.logger.Info("Server stopped gracefully")
	return nil
}
