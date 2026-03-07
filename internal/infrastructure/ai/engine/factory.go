package engine

import (
	"context"
	"fmt"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// ExecutorFactory 根据 App 配置创建对应的 Executor
type ExecutorFactory struct{}

// NewExecutorFactory 创建 ExecutorFactory
func NewExecutorFactory() *ExecutorFactory {
	return &ExecutorFactory{}
}

// Create 根据应用类型和配置创建 AppExecutor
// chatModel 由调用方根据 AppConfig.ModelID 加载 ModelConfig 后创建
func (f *ExecutorFactory) Create(
	ctx context.Context,
	appType valueobject.AppType,
	config *valueobject.AppConfig,
	chatModel einomodel.BaseChatModel,
) (port.AppExecutor, error) {
	switch appType {
	case valueobject.AppTypeTextGeneration:
		return NewTextGenExecutor(chatModel, config.SystemPrompt), nil

	case valueobject.AppTypeChat:
		return NewChatExecutor(chatModel, config.SystemPrompt), nil

	case valueobject.AppTypeAgent:
		toolModel, ok := chatModel.(einomodel.ToolCallingChatModel)
		if !ok {
			return nil, fmt.Errorf("engine: agent requires a ToolCallingChatModel, got %T", chatModel)
		}
		return NewAgentExecutor(toolModel, config.SystemPrompt, nil)

	default:
		return nil, fmt.Errorf("engine: unsupported app type %q", appType)
	}
}
