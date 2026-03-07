package response

import (
	"context"

	otelTrace "go.opentelemetry.io/otel/trace"
)

// ParseContextTraceId 从 context 中提取 OpenTelemetry TraceID
func ParseContextTraceId(ctx context.Context) string {
	spanCtx := otelTrace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}
