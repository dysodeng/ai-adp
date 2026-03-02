package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// New creates a zap.Logger from LoggerConfig.
// OutputPath can be "stdout", "stderr", or a file path.
func New(cfg config.LoggerConfig) (*zap.Logger, error) {
	level := parseLevel(cfg.Level)

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if cfg.Format == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	outputPath := cfg.OutputPath
	if outputPath == "" {
		outputPath = "stdout"
	}

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      level == zapcore.DebugLevel,
		Encoding:         encoding(cfg.Format),
		EncoderConfig:    encoderCfg,
		OutputPaths:      []string{outputPath},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	return log, nil
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func encoding(format string) string {
	if format == "console" {
		return "console"
	}
	return "json"
}
