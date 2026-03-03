package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dysodeng/ai-adp/internal/di"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/server"
)

const defaultConfigPath = "configs/app.yaml"

type application struct {
	ctx     context.Context
	mainApp *di.App
	servers []server.Server
}

func newApplication(ctx context.Context) *application {
	return &application{ctx: ctx}
}

func (a *application) run() {
	a.initialize()
	a.serve()
	a.waitForInterruptSignal()
}

func (a *application) initialize() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	mainApp, err := di.InitApp(configPath)
	if err != nil {
		// logger 此时可能还未初始化，直接输出到 stderr
		fmt.Fprintf(os.Stderr, "应用初始化失败: %v\n", err)
		os.Exit(1)
	}
	a.mainApp = mainApp
}

func (a *application) registerServer(servers ...server.Server) {
	for _, s := range servers {
		if s.IsEnabled() {
			a.servers = append(a.servers, s)
		}
	}
}

func (a *application) serve() {
	logger.Info(a.ctx, "starting application servers...")

	a.registerServer(
		a.mainApp.HTTPServer,
		// 未来扩展: a.mainApp.GRPCServer, a.mainApp.WSServer
	)

	for _, s := range a.servers {
		if err := s.Start(); err != nil {
			logger.Fatal(a.ctx, fmt.Sprintf("%s server failed to start", s.Name()), logger.ErrorField(err))
		}
		logger.Info(a.ctx, fmt.Sprintf("%s server started", s.Name()), logger.AddField("addr", s.Addr()))
	}
}

func (a *application) waitForInterruptSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(a.ctx, "shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, s := range a.servers {
		if err := s.Stop(ctx); err != nil {
			logger.Error(ctx, fmt.Sprintf("%s server shutdown error", s.Name()), logger.ErrorField(err))
		} else {
			logger.Info(ctx, fmt.Sprintf("%s server stopped", s.Name()))
		}
	}

	if err := a.mainApp.Stop(ctx); err != nil {
		logger.Error(ctx, "application cleanup error", logger.ErrorField(err))
	}

	logger.Info(ctx, "application stopped gracefully")
}

// Execute 应用程序入口点
func Execute() {
	ctx := context.Background()
	newApplication(ctx).run()
}
