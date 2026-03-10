package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		// 记录请求日志
		statusCode := c.Writer.Status()

		// 状态码颜色
		var statusColor string
		switch {
		case statusCode >= 200 && statusCode < 300:
			statusColor = "\033[97;42m" // 绿色
		case statusCode >= 300 && statusCode < 400:
			statusColor = "\033[90;47m" // 白色
		case statusCode >= 400 && statusCode < 500:
			statusColor = "\033[97;43m" // 黄色
		default:
			statusColor = "\033[97;41m" // 红色
		}

		// 方法颜色
		var methodColor string
		switch method {
		case "GET":
			methodColor = "\033[97;44m" // 蓝色
		case "POST":
			methodColor = "\033[97;42m" // 绿色
		case "PUT":
			methodColor = "\033[97;43m" // 黄色
		case "DELETE":
			methodColor = "\033[97;41m" // 红色
		case "PATCH":
			methodColor = "\033[97;45m" // 紫色
		case "HEAD":
			methodColor = "\033[97;46m" // 青色
		default:
			methodColor = "\033[97;44m" // 蓝色
		}

		// 重置颜色
		resetColor := "\033[0m"

		_, _ = gin.DefaultWriter.Write([]byte(
			"[GIN] " + time.Now().Format("2006/01/02 - 15:04:05") +
				" | " + methodColor + method + resetColor +
				" | " + path +
				" | " + c.ClientIP() +
				" | " + c.Request.UserAgent() +
				" | " + time.Since(start).String() +
				" | " + statusColor + strconv.Itoa(statusCode) + resetColor +
				" | " + c.Errors.String() +
				"\n",
		))
	}
}
