package telemetry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/telemetry"
)

func TestNewTracer_Disabled(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
		SampleRate:  1.0,
	}
	tp, shutdown, err := telemetry.NewTracerProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, tp)
	assert.NotNil(t, shutdown)
	// Clean up
	shutdown()
}

func TestNewTracer_NoopWhenDisabled(t *testing.T) {
	cfg := config.TracingConfig{Enabled: false}
	_, shutdown, err := telemetry.NewTracerProvider(cfg)
	require.NoError(t, err)
	defer shutdown()
	// Should not panic or error
}
