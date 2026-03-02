package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
)

// Logger 记录 HTTP 访问日志，自动从 Gin context 中获取 OTel trace context。
// 注意：必须在 Tracing() 中间件之后注册，以确保 c.Request.Context() 中已注入 span。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// 使用 c.Request.Context() 以获取 otelgin 注入的 trace span
		ctx := c.Request.Context()

		fields := []logger.Field{
			logger.AddField("status", statusCode),
			logger.AddField("method", method),
			logger.AddField("path", path),
			logger.AddField("ip", clientIP),
			logger.AddField("latency_ms", latency.Milliseconds()),
		}

		if len(c.Errors) > 0 {
			logger.Error(ctx, c.Errors.String(), fields...)
		} else {
			logger.Info(ctx, "http_access", fields...)
		}
	}
}
