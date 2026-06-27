package main

import (
	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/bootstrap"
	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/middlewares"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/belllllx/social-media-go/docs"

	"github.com/gin-contrib/cors"
)

//	@title			Social Media API
//	@version		1.0
//	@description	This is api for social media application
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:5000
//	@BasePath	/api

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	app := bootstrap.NewApp()
	defer app.Close()

	userRepositoryDB := user.NewUserRepositoryDB(app.DB)
	emailRepositoryImpl := email.NewEmailRepositoryImpl()
	otpRepositoryDB := otp.NewOTPRepositoryDB(app.DB)

	emailService := email.NewEmailService(emailRepositoryImpl, otpRepositoryDB)
	otpService := otp.NewOTPService(otpRepositoryDB)
	authService := auth.NewAuthService(
		app.RedisClient,
		otpRepositoryDB,
		userRepositoryDB,
		emailService,
		otpService,
	)
	userService := user.NewUserService(userRepositoryDB)

	authHandler := auth.NewAuthHandler(authService, emailService)

	app.Cron.AddFunc("*/30 * * * *", func() {
		err := otpService.DeleteWithExpired()
		if err == nil {
			logs.Info("Delete otp expired by cron successfully")
		}
	})
	app.Cron.Start()

	app.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{viper.GetString("app.client_url")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowWebSockets:  true,
		AllowCredentials: true,
	}))
	app.Router.Use(gin.Recovery())
	app.Router.Use(middlewares.ZapLogger())
	app.Router.Use(middlewares.GlobalErrorsHandler())

	app.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	app.Router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})
	api := app.Router.Group("/api")

	{
		auth := api.Group("/auth")

		auth.POST("/login", middlewares.AuthLogin(authService), authHandler.Login)
		auth.GET("/profile", middlewares.RequireAuth(userService), authHandler.Profile)

		register := auth.Group("/register")
		register.POST("/send-email", authHandler.SendEmailRegister)

		register.Use(middlewares.AuthRegister())
		register.POST("/resend-email", authHandler.ResendEmailRegister)
		register.POST("/verify-otp", authHandler.VerifyOTPRegister)
	}

	app.Run()
}
