package middleware

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/metrics"
)

var (
	// httpRequestCounter HTTP请求计数器
	httpRequestCounter metric.Int64Counter
	// httpRequestDuration HTTP请求耗时直方图
	httpRequestDuration metric.Float64Histogram
	// httpRequestsInflight 当前并发处理的请求数
	httpRequestsInflight metric.Int64UpDownCounter
	httpMetricsOnce      sync.Once
)

func initHTTPMetrics() {
	m := metrics.Meter()
	if m == nil {
		return
	}
	c, _ := m.Int64Counter("http.server.requests_total")
	h, _ := m.Float64Histogram("http.server.duration", metric.WithUnit("s"))
	u, _ := m.Int64UpDownCounter("http.server.inflight")
	httpRequestCounter = c
	httpRequestDuration = h
	httpRequestsInflight = u
}

// Metrics 指标中间件
func Metrics() gin.HandlerFunc {
	httpMetricsOnce.Do(initHTTPMetrics)
	return func(c *gin.Context) {
		if httpRequestCounter == nil || httpRequestDuration == nil || httpRequestsInflight == nil {
			return
		}

		route := c.FullPath()
		method := c.Request.Method

		ctx := context.Background()

		commonAttrs := []attribute.KeyValue{
			attribute.String("http.request.method", method),
			attribute.String("http.route", route),
		}

		httpRequestsInflight.Add(ctx, 1, metric.WithAttributes(commonAttrs...))

		start := time.Now()

		defer func() {
			httpRequestsInflight.Add(ctx, -1, metric.WithAttributes(commonAttrs...))

			duration := time.Since(start).Seconds()
			status := c.Writer.Status()

			finalAttrs := append(commonAttrs, attribute.String("http.response.status_code", strconv.Itoa(status)))

			httpRequestCounter.Add(ctx, 1, metric.WithAttributes(finalAttrs...))
			httpRequestDuration.Record(ctx, duration, metric.WithAttributes(finalAttrs...))
		}()

		c.Next()
	}
}
