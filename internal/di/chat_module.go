package di

import (
	"github.com/google/wire"

	chatorch "github.com/dysodeng/ai-adp/internal/application/chat/orchestrator"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
)

// ChatModuleSet wires the chat bounded context
var ChatModuleSet = wire.NewSet(
	chatorch.NewExecutorOrchestrator,
	chatservice.NewChatAppService,
	handler.NewChatHandler,
	handler.NewCancelHandler,
)
