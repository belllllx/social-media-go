package main

import (
	"fmt"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/configs"
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
	configs.InitTimeZone()
	configs.InitConfig()
	db := configs.InitDB()
	rdb := configs.InitRedis()
	defer rdb.Close()
	defer logs.Sync()

	userRepositoryDB := user.NewUserRepositoryDB(db)
	emailRepositoryImpl := email.NewEmailRepositoryImpl()
	otpRepositoryDB := otp.NewOTPRepositoryDB(db)

	emailService := email.NewEmailService(
		rdb,
		emailRepositoryImpl,
		otpRepositoryDB,
	)
	otpService := otp.NewOTPService(otpRepositoryDB)
	authService := auth.NewAuthService(
		rdb,
		otpRepositoryDB,
		userRepositoryDB,
		emailService,
		otpService,
	)

	authHandler := auth.NewAuthHandler(authService, emailService)

	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{viper.GetString("app.client_url")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowWebSockets:  true,
		AllowCredentials: true,
	}))
	r.Use(gin.Recovery())
	r.Use(middlewares.ZapLogger())
	r.Use(middlewares.GlobalErrorsHandler())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})
	api := r.Group("/api")

	{
		auth := api.Group("/auth")

		auth.POST("/login", middlewares.AuthLogin(authService), authHandler.Login)

		register := auth.Group("/register")
		register.POST("/send-email", authHandler.SendEmailRegister)

		register.Use(middlewares.AuthRegister())
		register.POST("/resend-email", authHandler.ResendEmailRegister)
		register.POST("/verify-otp", authHandler.VerifyOTPRegister)
	}

	port := fmt.Sprintf(":%s", viper.GetString("app.port"))
	r.Run(port)
}
