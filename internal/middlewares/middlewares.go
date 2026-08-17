package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	server "github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

func GlobalErrorsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			var validateErrs validator.ValidationErrors

			if errors.As(ginErr.Err, &validateErrs) {
				errFields := map[string]string{}
				for _, e := range validateErrs {
					errFields[e.Field()] = helpers.GetErrorMessages(e)
				}
				response.AbortWithBadRequestErrorFields(c, errFields)
				return
			}
		}

		if len(c.Errors) > 0 {
			logs.Error(c.Errors.Last().Err)
			response.AbortWithInternalServerError(c, c.Errors.Last().Err)
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
			response.AbortWithUnauthorized(c)
			return
		}

		token, err := helpers.VerifyJWT(registerToken, &auth.SendEmailTokenClaims{}, viper.GetString("app.register_token_secret"))
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(*auth.SendEmailTokenClaims)
		if !ok {
			response.AbortWithUnauthorized(c)
			return
		}

		email := claims.Email
		c.Set("email", email)
		c.Next()
	}
}

func AuthLogin(authService auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		loginRequest := &auth.LoginRequest{}
		err := c.ShouldBind(loginRequest)
		if err != nil {
			response.AbortWithError(c, err)
			return
		}

		loginDTO := &auth.LoginDTO{
			Username: loginRequest.Username,
			Password: loginRequest.Password,
		}
		userID, err := authService.ValidateUserLogin(ctx, loginDTO)
		if err != nil {
			helpers.HandleError(c, err)
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func RequireAuth(userService user.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		accessToken, err := c.Cookie("access_token")
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		token, err := helpers.VerifyJWT(accessToken, &auth.UserAccessTokenClaims{}, viper.GetString("app.access_token_secret"))
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(*auth.UserAccessTokenClaims)
		if !ok {
			response.AbortWithUnauthorized(c)
			return
		}

		secureUserWithFollowingRelation, err := userService.FindByIDWithFollowingRelation(ctx, claims.UserID)
		if err != nil {
			helpers.HandleError(c, err)
			return
		}

		c.Set("user", secureUserWithFollowingRelation)
		c.Next()
	}
}

func SocketRequireAuth(socket *server.Socket, next func(*server.ExtendedError)) {
	headers := socket.Handshake().Headers
	req := &http.Request{
		Header: headers.Header(),
	}
	accessToken, err := req.Cookie("access_token")
	if err != nil {
		next(&server.ExtendedError{
			Message: "Unauthorized",
			Data: map[string]int{
				"status": http.StatusUnauthorized,
			},
		})
		return
	}

	token, err := helpers.VerifyJWT(accessToken.Value, &auth.UserAccessTokenClaims{}, viper.GetString("app.access_token_secret"))
	if err != nil {
		next(&server.ExtendedError{
			Message: "Unauthorized",
			Data: map[string]int{
				"status": http.StatusUnauthorized,
			},
		})
		return
	}

	claims, ok := token.Claims.(*auth.UserAccessTokenClaims)
	if !ok {
		next(&server.ExtendedError{
			Message: "Unauthorized",
			Data: map[string]int{
				"status": http.StatusUnauthorized,
			},
		})
		return
	}

	socket.SetData(claims.UserID)
	next(nil)
}

func AuthRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		token, err := helpers.VerifyJWT(refreshToken, &auth.UserRefreshTokenClaims{}, viper.GetString("app.refresh_token_secret"))
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(*auth.UserRefreshTokenClaims)
		if !ok {
			response.AbortWithUnauthorized(c)
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}

func AuthForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		forgotPasswordToken, err := c.Cookie("forgot_password_token")
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		token, err := helpers.VerifyJWT(forgotPasswordToken, &auth.SendEmailTokenClaims{}, viper.GetString("app.forgot_password_token_secret"))
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(*auth.SendEmailTokenClaims)
		if !ok {
			response.AbortWithUnauthorized(c)
			return
		}

		c.Set("email", claims.Email)
		c.Next()
	}
}

func AuthGoogleCallback(redisClient *redis.Client, verifier *oidc.IDTokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		state := c.Query("state")
		code := c.Query("code")

		key := fmt.Sprintf("auth:oauth2-state:%s", state)
		oauth2State, err := helpers.RedisGet(
			ctx,
			redisClient,
			key,
		)
		if err == redis.Nil {
			response.AbortWithUnauthorized(c)
			return
		} else if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		if state != oauth2State {
			response.AbortWithUnauthorized(c)
			return
		}

		googleConfig := configs.InitOAuth2GoogleConfig()
		token, err := googleConfig.Exchange(ctx, code)
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			response.AbortWithUnauthorized(c)
			return
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		googleClaims := &auth.GoogleClaims{}
		err = idToken.Claims(googleClaims)
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		err = helpers.RedisDelete(
			ctx,
			redisClient,
			key,
		)
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		socialUserDTO := &auth.SocialUserDTO{
			ProviderType: models.ProviderTypeGoogle,
			Email:        googleClaims.Email,
			Name:         googleClaims.Name,
			AvatarURL:    googleClaims.Picture,
		}

		c.Set("socialUser", socialUserDTO)
		c.Next()
	}
}

func AuthGithubCallback(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		state := c.Query("state")
		code := c.Query("code")

		key := fmt.Sprintf("auth:oauth2-state:%s", state)
		oauth2State, err := helpers.RedisGet(
			ctx,
			redisClient,
			key,
		)
		if err == redis.Nil {
			response.AbortWithUnauthorized(c)
			return
		} else if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		if state != oauth2State {
			response.AbortWithUnauthorized(c)
			return
		}

		githubConfig := configs.InitOAuth2GithubConfig()
		token, err := githubConfig.Exchange(ctx, code)
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		client := githubConfig.Client(ctx, token)
		resp, err := client.Get(viper.GetString("app.github_user_resources_api"))
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			response.AbortWithUnauthorized(c)
			return
		}

		if resp.StatusCode != http.StatusOK {
			response.AbortWithInternalServerError(c, errors.New("Failed to get github user data"))
			return
		}

		githubUser := &auth.GithubUser{}
		err = json.NewDecoder(resp.Body).Decode(githubUser)
		if err != nil {
			response.AbortWithBadRequest(c)
			return
		}

		resp, err = client.Get(viper.GetString("app.github_user_email_api"))
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			response.AbortWithUnauthorized(c)
			return
		}

		if resp.StatusCode != http.StatusOK {
			response.AbortWithInternalServerError(c, errors.New("Failed to get github user email"))
			return
		}

		githubEmails := []auth.GithubEmail{}
		err = json.NewDecoder(resp.Body).Decode(&githubEmails)
		if err != nil {
			response.AbortWithBadRequest(c)
			return
		}

		for _, e := range githubEmails {
			if e.Primary && e.Verified {
				githubUser.Email = e.Email
			}
		}

		err = helpers.RedisDelete(
			ctx,
			redisClient,
			key,
		)
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		socialUserDTO := &auth.SocialUserDTO{
			ProviderType: models.ProviderTypeGithub,
			Email:        githubUser.Email,
			Name:         githubUser.Name,
			AvatarURL:    githubUser.AvatarURL,
		}

		c.Set("socialUser", socialUserDTO)
		c.Next()
	}
}

func AuthFacebookCallback(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		state := c.Query("state")
		code := c.Query("code")

		key := fmt.Sprintf("auth:oauth2-state:%s", state)
		oauth2State, err := helpers.RedisGet(
			ctx,
			redisClient,
			key,
		)
		if err == redis.Nil {
			response.AbortWithUnauthorized(c)
			return
		} else if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		if state != oauth2State {
			response.AbortWithUnauthorized(c)
			return
		}

		facebookConfig := configs.InitOAuth2FacebookConfig()
		token, err := facebookConfig.Exchange(ctx, code)
		if err != nil {
			response.AbortWithUnauthorized(c)
			return
		}

		client := facebookConfig.Client(ctx, token)
		resp, err := client.Get(viper.GetString("app.facebook_user_resources_api"))
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			response.AbortWithUnauthorized(c)
			return
		}

		if resp.StatusCode != http.StatusOK {
			response.AbortWithInternalServerError(c, errors.New("Failed to get facebook user data"))
			return
		}

		facebookUser := &auth.FacebookUser{}
		err = json.NewDecoder(resp.Body).Decode(facebookUser)
		if err != nil {
			response.AbortWithBadRequest(c)
			return
		}

		err = helpers.RedisDelete(
			ctx,
			redisClient,
			key,
		)
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		socialUserDTO := &auth.SocialUserDTO{
			ProviderType: models.ProviderTypeFacebook,
			Email:        facebookUser.Email,
			Name:         facebookUser.Name,
			AvatarURL:    facebookUser.Picture.Data.URL,
		}

		c.Set("socialUser", socialUserDTO)
		c.Next()
	}
}
