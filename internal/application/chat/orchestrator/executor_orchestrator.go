package orchestrator

import (
	"context"
	"fmt"

	domainagent "github.com/dysodeng/ai-adp/internal/domain/agent/agent"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/service"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/ai/adapter"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// ExecutorOrchestrator Agent 执行编排器
type ExecutorOrchestrator interface {
	// Execute 执行 Agent（非阻塞），通过 AgentExecutor 事件流获取结果
	Execute(ctx context.Context, app *appmodel.App, agentExecutor executor.AgentExecutor, isStreaming bool) error
}

type executorOrchestrator struct {
	agentBuilder service.AgentBuilder
	agentFactory *adapter.AgentFactory
}

// NewExecutorOrchestrator 创建执行编排器
func NewExecutorOrchestrator(
	agentBuilder service.AgentBuilder,
	agentFactory *adapter.AgentFactory,
) ExecutorOrchestrator {
	return &executorOrchestrator{
		agentBuilder: agentBuilder,
		agentFactory: agentFactory,
	}
}

func (o *executorOrchestrator) Execute(
	ctx context.Context,
	app *appmodel.App,
	agentExecutor executor.AgentExecutor,
	isStreaming bool,
) error {
	// 1. 构建 Agent 配置
	config, err := o.agentBuilder.BuildAgentConfig(ctx, app, agentExecutor.GetInput().Variables, isStreaming)
	if err != nil {
		return fmt.Errorf("build agent config failed: %w", err)
	}

	// 2. 通过工厂创建 Agent（modelID 存储在 config.AgentID 中，即 appID，后续可从 AppConfig 中获取）
	ag, err := o.agentFactory.CreateAgent(ctx, app.Type(), config, config.AgentID)
	if err != nil {
		return fmt.Errorf("create agent failed: %w", err)
	}

	// 3. 启动执行
	agentExecutor.Start()

	// 4. 异步执行 Agent
	go o.executeAgent(ctx, ag, agentExecutor)

	// 5. 如果是阻塞模式，等待完成
	if !isStreaming {
		eventChan := agentExecutor.AddSubscriber()
		for range eventChan {
			// 消费事件直到 channel 关闭
		}
		if agentExecutor.Err() != nil {
			return agentExecutor.Err()
		}
	}

	return nil
}

func (o *executorOrchestrator) executeAgent(ctx context.Context, ag domainagent.Agent, agentExecutor executor.AgentExecutor) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("agent execution panic: %v", r)
			logger.Error(ctx, "agent execution panic", logger.ErrorField(err))
			agentExecutor.Fail(err)
		}
	}()

	if err := ag.Execute(ctx, agentExecutor); err != nil {
		logger.Error(ctx, "agent execution failed", logger.ErrorField(err))
		return
	}
}
