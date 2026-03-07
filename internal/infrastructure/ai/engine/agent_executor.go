package engine

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

// AgentExecutor ReAct Agent 执行器（LLM + Tools 推理循环），基于 ADK ChatModelAgent
type AgentExecutor struct {
	agent        adk.Agent
	systemPrompt string
}

func NewAgentExecutor(
	ctx context.Context,
	chatModel einomodel.ToolCallingChatModel,
	systemPrompt string,
	tools []tool.BaseTool,
) (*AgentExecutor, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("agent_executor: chatModel is required")
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "agent",
		Description: "ReAct agent with tool calling capabilities",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("agent_executor: %w", err)
	}

	return &AgentExecutor{agent: agent, systemPrompt: systemPrompt}, nil
}

func (e *AgentExecutor) Execute(ctx context.Context, input *port.AppExecutorInput) (*port.AppResult, error) {
	messages := buildInputMessages(input)
	if rendered := RenderPrompt(e.systemPrompt, input.Variables); rendered != "" {
		messages = prependSystemMessage(rendered, messages)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent})
	iter := runner.Run(ctx, messages)
	return collectResult(iter)
}

func (e *AgentExecutor) Run(ctx context.Context, input *port.AppExecutorInput) (<-chan port.AppEvent, error) {
	messages := buildInputMessages(input)
	if rendered := RenderPrompt(e.systemPrompt, input.Variables); rendered != "" {
		messages = prependSystemMessage(rendered, messages)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: e.agent, EnableStreaming: true})
	iter := runner.Run(ctx, messages)
	return streamEvents(iter), nil
}
