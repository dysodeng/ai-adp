package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	domainTool "github.com/dysodeng/ai-adp/internal/domain/agent/tool"
)

// ToolAdapter 将领域层 Tool 适配为 Eino BaseTool
type ToolAdapter struct {
	domainTool domainTool.Tool
}

func NewToolAdapter(t domainTool.Tool) einotool.InvokableTool {
	return &ToolAdapter{domainTool: t}
}

func (a *ToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	// TODO: 正确转换 InputSchema 为 schema.ParamsOneOf
	return &schema.ToolInfo{
		Name:        a.domainTool.Name(),
		Desc:        a.domainTool.Description(),
		ParamsOneOf: nil, // TODO: 转换 InputSchema
	}, nil
}

func (a *ToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error) {
	// 将 JSON 字符串解析为 map
	var input map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	return a.domainTool.Invoke(ctx, input)
}

// ConvertDomainToolsToEino 批量转换领域 Tool 为 Eino BaseTool
func ConvertDomainToolsToEino(domainTools []domainTool.Tool) []einotool.BaseTool {
	einoTools := make([]einotool.BaseTool, 0, len(domainTools))
	for _, t := range domainTools {
		einoTools = append(einoTools, NewToolAdapter(t))
	}
	return einoTools
}
