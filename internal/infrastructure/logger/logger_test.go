package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

func TestInitLogger(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:      "debug",
		Format:     "json",
		OutputPath: "stdout",
	}
	err := logger.InitLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger.ZapLogger())
}

func TestLoggerInfo_NoTrace(t *testing.T) {
	cfg := config.LoggerConfig{Level: "debug", Format: "json", OutputPath: "stdout"}
	require.NoError(t, logger.InitLogger(cfg))

	assert.NotPanics(t, func() {
		logger.Info(context.Background(), "test message", logger.AddField("key", "value"))
	})
}

func TestLoggerError_WithErrorField(t *testing.T) {
	cfg := config.LoggerConfig{Level: "debug", Format: "json", OutputPath: "stdout"}
	require.NoError(t, logger.InitLogger(cfg))

	assert.NotPanics(t, func() {
		logger.Error(context.Background(), "test error", logger.ErrorField(assert.AnError))
	})
}
