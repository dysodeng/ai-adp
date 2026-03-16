package middleware

import (
	"net/http"
	"strings"

	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
	"github.com/gin-gonic/gin"
)

// AppApiKey 应用ApiKey验证
func AppApiKey(ctx *gin.Context) {
	apiKey := ctx.GetHeader("Authorization")
	if apiKey == "" {
		// 回退到查询参数（支持 EventSource GET 请求）
		apiKey = ctx.Query("api_key")
		if apiKey != "" {
			ctx.Set("api_key", apiKey)
			ctx.Next()
			return
		}
		ctx.AbortWithStatusJSON(http.StatusOK, response.Fail(ctx, "Missing Authorization header", response.CodeUnauthorized))
		return
	}
	if !strings.HasPrefix(apiKey, "Bearer ") || len(apiKey) < 8 {
		ctx.AbortWithStatusJSON(http.StatusOK, response.Fail(ctx, "Invalid authorization format", response.CodeUnauthorized))
		return
	}
	ctx.Set("api_key", apiKey[7:])
	ctx.Next()
}
