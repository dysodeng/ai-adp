package di

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
)

// App 应用主结构
type App struct {
	httpServer *server.HTTPServer
}

func NewApp(httpServer *server.HTTPServer) *App {
	return &App{httpServer: httpServer}
}

// Run 启动服务，阻塞直到收到退出信号
func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		fmt.Println("HTTP server starting...")
		if err := a.httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		fmt.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.httpServer.Shutdown(ctx)
	}
}
