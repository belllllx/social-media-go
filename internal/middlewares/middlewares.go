package middlewares

import (
	"errors"
	"time"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
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
			for _, ginErr := range c.Errors {
				var validateErrs validator.ValidationErrors

				if errors.As(ginErr.Err, &validateErrs) {
					errorField := map[string]string{}
					for _, e := range validateErrs {
						errorField[e.Field()] = helpers.GetErrorMessages(e)
					}
					response.BadRequest(c, errorField)
					c.Abort()
					return
				}
			}

			logs.Error(c.Errors.Last().Err)
			response.InternalServerError(c, c.Errors.Last().Err)
			c.Abort()
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
		registerToken, err := c.Cookie("register_token")
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token, err := helpers.VerifyJWT(registerToken, &auth.SendEmailTokenJWTPayload{}, viper.GetString("app.register_token_secret"))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*auth.SendEmailTokenJWTPayload)
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

		userID, err := authService.ValidateUserLogin(loginRequest)
		if err != nil {
			helpers.HandleError(c, err, err.Error())
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func RequireAuth(userService user.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token, err := helpers.VerifyJWT(accessToken, &auth.UserAccessTokenJWTPayload{}, viper.GetString("app.access_token_secret"))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*auth.UserAccessTokenJWTPayload)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		secureUser, err := userService.SecureFindWithID(claims.ID)
		if err != nil {
			helpers.HandleError(c, err, err.Error())
			c.Abort()
			return
		}

		c.Set("user", secureUser)
		c.Next()
	}
}

func AuthRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token, err := helpers.VerifyJWT(refreshToken, &auth.UserRefreshTokenJWTPayload{}, viper.GetString("app.refresh_token_secret"))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*auth.UserRefreshTokenJWTPayload)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("userID", claims.ID)
		c.Next()
	}
}

func AuthForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		forgotPasswordToken, err := c.Cookie("forgot_password_token")
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token, err := helpers.VerifyJWT(forgotPasswordToken, &auth.SendEmailTokenJWTPayload{}, viper.GetString("app.forgot_password_token_secret"))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*auth.SendEmailTokenJWTPayload)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("email", claims.Email)
		c.Next()
	}
}
