package port_test

import (
	"testing"

	"github.com/dysodeng/ai-adp/internal/domain/shared/port"
)

func TestAgentEventTypes_Defined(t *testing.T) {
	types := []port.AgentEventType{
		port.AgentEventTypeMessage,
		port.AgentEventTypeToolCall,
		port.AgentEventTypeDone,
		port.AgentEventTypeError,
	}
	for _, tt := range types {
		if tt == "" {
			t.Fatalf("AgentEventType constant should not be empty")
		}
	}
}
