package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

func TestNewLogger_InfoLevel(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestNewLogger_DebugLevel(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:      "invalid",
		Format:     "json",
		OutputPath: "stdout",
	}
	// Invalid level should fall back to info, not error
	log, err := logger.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}
