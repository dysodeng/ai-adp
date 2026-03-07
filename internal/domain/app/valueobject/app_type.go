package valueobject

// AppType AI 应用类型
type AppType string

const (
	AppTypeAgent          AppType = "agent"
	AppTypeChat           AppType = "chat"
	AppTypeTextGeneration AppType = "text_generation"
	AppTypeChatFlow       AppType = "chat_flow"
	AppTypeWorkflow       AppType = "workflow"
)

func (t AppType) IsValid() bool {
	switch t {
	case AppTypeAgent, AppTypeChat, AppTypeTextGeneration, AppTypeChatFlow, AppTypeWorkflow:
		return true
	}
	return false
}

func (t AppType) String() string { return string(t) }
