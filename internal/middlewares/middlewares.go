package middlewares

import (
	"errors"
	"strings"
	"time"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
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
					c.Abort()
					return
				}

				logs.Error(err)
				response.InternalServerError(c, err)
				c.Abort()
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

func AuthRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("register_token")
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		t, err := helpers.VerifyJWT(token, &auth.SendEmailRegisterJWTPayload{}, viper.GetString("app.register_secret"))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, ok := t.Claims.(*auth.SendEmailRegisterJWTPayload)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		email := claims.Email
		c.Set("email", email)
		c.Next()
	}
}

func AuthLogin(authService auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginRequest := &auth.LoginRequest{}
		err := c.ShouldBind(loginRequest)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		secureUser, err := authService.ValidateUserLogin(loginRequest)
		if err != nil {
			helpers.HandleError(c, err, err.Error())
			c.Abort()
			return
		}

		c.Set("user", secureUser)
		c.Next()
	}
}
