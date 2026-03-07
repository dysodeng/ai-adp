package orchestrator

import (
	"context"
	"errors"
	"fmt"

	domainagent "github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/service"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	infraagent "github.com/dysodeng/ai-adp/internal/infrastructure/agent"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// ExecutorOrchestrator Agent 执行编排器
type ExecutorOrchestrator interface {
	// Execute 执行 Agent（非阻塞），通过 AgentExecutor 事件流获取结果
	Execute(ctx context.Context, app *appmodel.App, version *appmodel.AppVersion, agentExecutor executor.AgentExecutor, isStreaming bool) error
}

type executorOrchestrator struct {
	agentBuilder service.AgentBuilder
	agentFactory *infraagent.AgentFactory
	taskRegistry executor.TaskRegistry
}

// NewExecutorOrchestrator 创建执行编排器
func NewExecutorOrchestrator(
	agentBuilder service.AgentBuilder,
	agentFactory *infraagent.AgentFactory,
	taskRegistry executor.TaskRegistry,
) ExecutorOrchestrator {
	return &executorOrchestrator{
		agentBuilder: agentBuilder,
		agentFactory: agentFactory,
		taskRegistry: taskRegistry,
	}
}

func (o *executorOrchestrator) Execute(
	ctx context.Context,
	app *appmodel.App,
	version *appmodel.AppVersion,
	agentExecutor executor.AgentExecutor,
	isStreaming bool,
) error {
	logger.Info(ctx, "[orchestrator] Execute start",
		logger.AddField("appID", app.ID().String()),
		logger.AddField("modelID", version.Config().ModelID.String()),
		logger.AddField("isStreaming", isStreaming),
	)

	// 1. 构建 Agent 配置
	config, err := o.agentBuilder.BuildAgentConfig(ctx, app, version, agentExecutor.GetInput().Variables, isStreaming)
	if err != nil {
		return fmt.Errorf("build agent config failed: %w", err)
	}
	logger.Info(ctx, "[orchestrator] BuildAgentConfig done")

	// 2. 通过工厂创建 Agent，使用 version 中的 ModelID
	modelID := version.Config().ModelID.String()
	ag, err := o.agentFactory.CreateAgent(ctx, app.Type(), config, modelID)
	if err != nil {
		return fmt.Errorf("create agent failed: %w", err)
	}
	logger.Info(ctx, "[orchestrator] CreateAgent done")

	// 3. 创建可取消的 context 并注册到 TaskRegistry
	taskID := agentExecutor.GetTaskID().String()
	execCtx, cancel := context.WithCancel(ctx)
	o.taskRegistry.Register(taskID, cancel)

	// 4. 启动执行
	agentExecutor.Start()
	logger.Info(ctx, "[orchestrator] agentExecutor.Start() done, launching goroutine")

	// 5. 异步执行 Agent
	go func() {
		defer o.taskRegistry.Unregister(taskID)
		o.executeAgent(execCtx, ag, agentExecutor)
	}()

	// 6. 如果是阻塞模式，等待完成
	if !isStreaming {
		eventChan := agentExecutor.AddSubscriber()
		for range eventChan {
			// 消费事件直到 channel 关闭
		}
		if agentExecutor.Err() != nil {
			return agentExecutor.Err()
		}
	}

	logger.Info(ctx, "[orchestrator] Execute returning")
	return nil
}

func (o *executorOrchestrator) executeAgent(ctx context.Context, ag domainagent.Agent, agentExecutor executor.AgentExecutor) {
	logger.Info(ctx, "[orchestrator] executeAgent goroutine started")
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("agent execution panic: %v", r)
			logger.Error(ctx, "[orchestrator] agent execution panic", logger.ErrorField(err))
			agentExecutor.Fail(err)
		}
	}()

	logger.Info(ctx, "[orchestrator] calling ag.Execute ...")
	if err := ag.Execute(ctx, agentExecutor); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info(ctx, "[orchestrator] agent execution cancelled")
			agentExecutor.Cancel()
			return
		}
		logger.Error(ctx, "[orchestrator] ag.Execute failed", logger.ErrorField(err))
		agentExecutor.Fail(err)
		return
	}
	logger.Info(ctx, "[orchestrator] ag.Execute completed successfully")
}
