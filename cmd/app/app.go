package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dysodeng/ai-adp/internal/di"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
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

func (app *application) run() {
	app.initialize()
	app.serve()
	app.waitForInterruptSignal()
}

func (app *application) initialize() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	mainApp, err := di.InitApp(app.ctx)
	if err != nil {
		// logger 此时可能还未初始化，直接输出到 stderr
		_, _ = fmt.Fprintf(os.Stderr, "应用初始化失败: %v\n", err)
		os.Exit(1)
	}
	app.mainApp = mainApp
}

func (app *application) registerServer(servers ...server.Server) {
	for _, s := range servers {
		if s.IsEnabled() {
			app.servers = append(app.servers, s)
		}
	}
}

func (app *application) serve() {
	logger.Info(app.ctx, "starting application servers...")

	app.registerServer(
		app.mainApp.HTTPServer,
		app.mainApp.HealthServer,
		// 未来扩展: a.mainApp.GRPCServer, a.mainApp.WSServer
	)

	// 启动取消信号订阅
	if err := app.mainApp.StartCancelSubscriber(app.ctx); err != nil {
		logger.Error(app.ctx, "failed to start cancel subscriber", logger.ErrorField(err))
	}

	for _, s := range app.servers {
		if err := s.Start(); err != nil {
			logger.Fatal(app.ctx, fmt.Sprintf("%s server failed to start", s.Name()), logger.ErrorField(err))
		}
		logger.Info(app.ctx, fmt.Sprintf("%s server started", s.Name()), logger.AddField("addr", s.Addr()))
	}

	// 注册服务到网关
	if err := app.mainApp.RegisterGateway(app.ctx); err != nil {
		logger.Error(app.ctx, "failed to register service to gateway", logger.ErrorField(err))
	}
}

func (app *application) waitForInterruptSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	// 取消信号订阅，避免 shutdown 期间再次触发
	signal.Stop(quit)

	logger.Info(app.ctx, "shutting down servers...")

	// 5 秒关闭预算由所有 Server.Stop 和 mainApp.Stop 共享
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, s := range app.servers {
		if err := s.Stop(ctx); err != nil {
			logger.Error(ctx, fmt.Sprintf("%s server shutdown error", s.Name()), logger.ErrorField(err))
		} else {
			logger.Info(ctx, fmt.Sprintf("%s server stopped", s.Name()))
		}
	}

	if err := app.mainApp.Stop(ctx); err != nil {
		logger.Error(ctx, "application cleanup error", logger.ErrorField(err))
	}

	logger.Info(ctx, "application stopped gracefully")
}

// Execute 应用程序入口点
func Execute() {
	ctx := context.Background()
	newApplication(ctx).run()
}
