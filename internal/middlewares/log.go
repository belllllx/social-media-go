package middlewares

import (
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ZapLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		logs.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		)
	}
}
