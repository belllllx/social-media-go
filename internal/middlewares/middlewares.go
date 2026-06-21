package middlewares

import (
	"errors"
	"strings"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func GlobalErrorsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if err != nil {
				var validateErrs validator.ValidationErrors
				if errors.As(err, &validateErrs) {
					errorField := map[string]string{}
					for _, e := range validateErrs {
						errorField[strings.ToLower(e.Field())] = helpers.GetErrorMessages(e)
					}
					response.BadRequest(c, errorField)
					return
				}

				logs.Error(err)
				response.InternalServerError(c, err)
			}
		}
	}
}

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
