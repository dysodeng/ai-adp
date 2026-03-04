package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentinfra "github.com/dysodeng/ai-adp/internal/infrastructure/ai/agent"
)

func TestNewAgentExecutor_NilModel(t *testing.T) {
	_, err := agentinfra.NewAgentExecutor(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chatModel")
}
