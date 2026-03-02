package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// Field 日志字段
type Field struct {
	Key     string
	Value   interface{}
	isError bool
}

// ErrorField 创建 error 类型日志字段
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err, isError: true}
}

// AddField 创建通用 key-value 日志字段
func AddField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

var _logger *logger

type logger struct {
	zap *zap.Logger
}

// InitLogger 使用配置初始化全局 logger，必须在应用启动时调用一次
func InitLogger(cfg config.LoggerConfig) error {
	zl, err := newZapLogger(cfg)
	if err != nil {
		return err
	}
	_logger = &logger{zap: zl}
	return nil
}

// ZapLogger 返回底层 *zap.Logger（供需要直接使用 zap 的场景）
func ZapLogger() *zap.Logger {
	if _logger == nil {
		return zap.NewNop()
	}
	return _logger.zap
}

// Debug 记录 debug 级别日志
func Debug(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.DebugLevel, msg, fields...)
}

// Info 记录 info 级别日志
func Info(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.InfoLevel, msg, fields...)
}

// Warn 记录 warn 级别日志
func Warn(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.WarnLevel, msg, fields...)
}

// Error 记录 error 级别日志
func Error(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.ErrorLevel, msg, fields...)
}

// Fatal 记录 fatal 级别日志后调用 os.Exit(1)
func Fatal(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.FatalLevel, msg, fields...)
}

// Panic 记录 panic 级别日志后 panic
func Panic(ctx context.Context, msg string, fields ...Field) {
	_logger.log(ctx, zapcore.PanicLevel, msg, fields...)
}

func (l *logger) log(ctx context.Context, level zapcore.Level, msg string, fields ...Field) {
	if l == nil || l.zap == nil {
		return
	}
	if ce := l.zap.Check(level, msg); ce != nil {
		zfields := l.extractTrace(ctx)
		for _, f := range fields {
			if f.isError {
				if err, ok := f.Value.(error); ok {
					zfields = append(zfields, zap.Error(err))
				}
			} else {
				zfields = append(zfields, zap.Any(f.Key, f.Value))
			}
		}
		ce.Write(zfields...)
	}
}

// extractTrace 从 context 中提取 OTel span 信息，注入 trace_id / span_id
func (l *logger) extractTrace(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return nil
	}
	sc := span.SpanContext()
	fields := make([]zap.Field, 0, 2)
	if sc.HasTraceID() {
		fields = append(fields, zap.String("trace_id", sc.TraceID().String()))
	}
	if sc.HasSpanID() {
		fields = append(fields, zap.String("span_id", sc.SpanID().String()))
	}
	return fields
}
