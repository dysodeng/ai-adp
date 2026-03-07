package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	"github.com/dysodeng/ai-adp/internal/application/chat/orchestrator"
	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	appmodel "github.com/dysodeng/ai-adp/internal/domain/app/model"
	"github.com/dysodeng/ai-adp/internal/domain/app/repository"
)

// ChatAppService Chat 应用服务接口
type ChatAppService interface {
	// Chat 执行 Agent 对话，返回 AgentExecutor 用于事件订阅
	Chat(ctx context.Context, appAuthIden string, cmd chatdto.ChatCommand) (executor.AgentExecutor, error)
}

type chatAppService struct {
	orchestrator  orchestrator.ExecutorOrchestrator
	appRepository repository.AppRepository
}

// NewChatAppService 创建 Chat 应用服务
func NewChatAppService(
	orch orchestrator.ExecutorOrchestrator,
	appRepository repository.AppRepository,
) ChatAppService {
	return &chatAppService{
		orchestrator:  orch,
		appRepository: appRepository,
	}
}

func (svc *chatAppService) Chat(
	ctx context.Context,
	appAuthIden string,
	cmd chatdto.ChatCommand,
) (executor.AgentExecutor, error) {
	// 1. 解析 conversation ID
	var conversationID uuid.UUID
	if cmd.ConversationID == "" {
		conversationID, _ = uuid.NewV7()
	} else {
		var err error
		conversationID, err = uuid.Parse(cmd.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("invalid conversation ID: %w", err)
		}
	}

	// 2. 验证 ResponseMode
	if !cmd.ResponseMode.IsValid() {
		return nil, fmt.Errorf("invalid response mode: %s", cmd.ResponseMode)
	}

	// 3. 加载 App 及已发布版本（优先解析为 UUID，否则当作 API Key）
	appUUID, parseErr := uuid.Parse(appAuthIden)

	var app *appmodel.App
	var version *appmodel.AppVersion
	var err error
	if parseErr == nil {
		// 通过 App ID 查找应用及已发布版本
		app, version, err = svc.appRepository.FindAppWithPublishedVersion(ctx, appUUID)
	} else {
		// 通过 API Key 查找应用及已发布版本
		app, version, err = svc.appRepository.FindAppByApiKey(ctx, appAuthIden)
	}
	if err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	// 4. 构建 ExecutionInput
	isStream := cmd.ResponseMode == chatdto.ResponseModeStreaming
	input := model.ExecutionInput{
		Query:     cmd.Query,
		Variables: cmd.Input,
	}

	// 5. 创建 AgentExecutor
	messageID := uuid.New()
	taskID := uuid.New()
	agentExecutor := executor.NewAgentExecutor(
		ctx,
		taskID,
		app.ID().String(),
		app.Type(),
		conversationID,
		messageID,
		input,
	)

	// 6. 通过编排器执行
	if err := svc.orchestrator.Execute(ctx, app, version, agentExecutor, isStream); err != nil {
		return nil, err
	}

	return agentExecutor, nil
}
