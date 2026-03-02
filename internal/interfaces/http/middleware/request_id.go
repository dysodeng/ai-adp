package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDKey)
		if id == "" {
			uid, _ := uuid.NewV7()
			id = uid.String()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDKey, id)
		c.Next()
	}
}
