package main

import (
	"context"
	"net/http"
	"time"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/internal/bootstrap"
	"github.com/belllllx/social-media-go/internal/comment"
	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/follow"
	"github.com/belllllx/social-media-go/internal/like"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/middlewares"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/internal/post"
	"github.com/belllllx/social-media-go/internal/socket"
	commentSocket "github.com/belllllx/social-media-go/internal/socket/comment"
	notificationSocket "github.com/belllllx/social-media-go/internal/socket/notification"
	postSocket "github.com/belllllx/social-media-go/internal/socket/post"
	userSocket "github.com/belllllx/social-media-go/internal/socket/user"
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

	googleConfig := configs.InitOAuth2GoogleConfig()
	githubConfig := configs.InitOAuth2GithubConfig()
	facebookConfig := configs.InitOAuth2FacebookConfig()

	userRepositoryDB := user.NewUserRepositoryDB()
	emailRepositoryImpl := email.NewEmailRepositoryImpl()
	otpRepositoryDB := otp.NewOTPRepositoryDB()
	fileRepositoryDB := file.NewFileRepositoryDB()
	postRepositoryDB := post.NewPostRepositoryDB()
	notificationRepositoryDB := notification.NewNotificationRepositoryDB()
	likeRepositoryDB := like.NewLikeRepositoryDB()
	commentRepositoryDB := comment.NewCommentRepositoryDB()
	followRepositoryDB := follow.NewFollowRepositoryDB()

	userSocketService := userSocket.NewUserSocketService(
		app.DB,
		app.PresignClient,
		userRepositoryDB,
	)
	socket := socket.NewSocketServer(middlewares.SocketRequireAuth, userSocketService)
	defer socket.Close(nil)

	notificationSocket := notificationSocket.NewNotificationSocket(socket)
	postSocket := postSocket.NewPostSocket(socket)
	commentSocket := commentSocket.NewCommentSocket(socket)
	userSocket := userSocket.NewUserSocket(socket)

	emailService := email.NewEmailService(emailRepositoryImpl, otpRepositoryDB)
	otpService := otp.NewOTPService(otpRepositoryDB)
	authService := auth.NewAuthService(
		app.DB,
		app.RedisClient,
		otpRepositoryDB,
		userRepositoryDB,
		emailService,
		otpService,
		googleConfig,
		githubConfig,
		facebookConfig,
	)
	fileService := file.NewFileService(
		app.DB,
		app.S3Client,
		app.PresignClient,
		fileRepositoryDB,
	)
	notificationService := notification.NewNotificationService(
		app.DB,
		app.PresignClient,
		notificationRepositoryDB,
	)
	userService := user.NewUserService(
		app.DB,
		app.RedisClient,
		app.S3Client,
		app.PresignClient,
		userRepositoryDB,
		followRepositoryDB,
		notificationRepositoryDB,
		notificationService,
		userSocket,
		notificationSocket,
	)
	postService := post.NewPostService(
		app.DB,
		app.RedisClient,
		app.S3Client,
		app.PresignClient,
		postRepositoryDB,
		userRepositoryDB,
		fileRepositoryDB,
		notificationRepositoryDB,
		likeRepositoryDB,
		userService,
		notificationService,
		fileService,
		notificationSocket,
		postSocket,
	)
	commentService := comment.NewCommentService(
		app.DB,
		app.RedisClient,
		app.S3Client,
		app.PresignClient,
		commentRepositoryDB,
		userRepositoryDB,
		postRepositoryDB,
		fileRepositoryDB,
		notificationRepositoryDB,
		likeRepositoryDB,
		notificationService,
		userService,
		fileService,
		commentSocket,
		notificationSocket,
	)

	authHandler := auth.NewAuthHandler(
		authService,
		emailService,
		userService,
	)
	postHandler := post.NewPostHandler(fileService, postService)
	commentHandler := comment.NewCommentHandler(commentService, fileService)
	notificationHandler := notification.NewNotificationHandler(notificationService)
	userHandler := user.NewUserHandler(userService)

	app.Cron.AddFunc("*/30 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err := otpService.DeleteWithExpired(ctx, app.DB)
		if err != nil {
			return
		}

		logs.Info("Delete otp expired by cron successfully")
	})
	app.Cron.Start()

	app.Router.MaxMultipartMemory = 10 << 20 // 10 MB
	app.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{viper.GetString("app.client_url")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowWebSockets:  true,
		AllowCredentials: true,
	}))
	app.Router.Use(gin.Recovery())
	app.Router.Use(middlewares.ZapLogger())
	app.Router.Use(middlewares.GlobalErrorsHandler())

	{
		socketIO := app.Router.Group("/socket.io")

		socketIO.GET("/*any", gin.WrapH(socket.ServeHandler(nil)))
		socketIO.POST("/*any", gin.WrapH(socket.ServeHandler(nil)))
	}

	{
		swagger := app.Router.Group("/swagger")

		swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		swagger.GET("", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/swagger/index.html")
		})
	}

	api := app.Router.Group("/api")

	{
		auth := api.Group("/auth")

		auth.POST("/login", middlewares.AuthLogin(authService), authHandler.Login)
		auth.GET("/profile", middlewares.RequireAuth(userService), authHandler.Profile)
		auth.POST("/refresh-token", middlewares.AuthRefresh(), authHandler.Refresh)
		auth.POST("/logout", middlewares.RequireAuth(userService), authHandler.Logout)

		google := auth.Group("/google")
		google.GET("/", authHandler.GoogleLogin)
		google.GET("/callback", middlewares.AuthGoogleCallback(app.RedisClient, app.Verifier), authHandler.GoogleCallback)

		facebook := auth.Group("/facebook")
		facebook.GET("/", authHandler.FacebookLogin)
		facebook.GET("/callback", middlewares.AuthFacebookCallback(app.RedisClient), authHandler.FacebookCallback)

		github := auth.Group("github")
		github.GET("/", authHandler.GithubLogin)
		github.GET("/callback", middlewares.AuthGithubCallback(app.RedisClient), authHandler.GithubCallback)

		forgotPassword := auth.Group("/forgot-password")
		forgotPassword.POST("/send-email", authHandler.SendEmailForgotPassword)

		forgotPassword.Use(middlewares.AuthForgotPassword())
		forgotPassword.POST("/verify-otp", authHandler.VerifyOTPForgotPassword)
		forgotPassword.POST("/resend-email", authHandler.ResendEmailForgotPassword)
		forgotPassword.PATCH("/reset-password", authHandler.ResetPassword)

		register := auth.Group("/register")
		register.POST("/send-email", authHandler.SendEmailRegister)

		register.Use(middlewares.AuthRegister())
		register.POST("/resend-email", authHandler.ResendEmailRegister)
		register.POST("/verify-otp", authHandler.VerifyOTPRegister)
	}

	api.Use(middlewares.RequireAuth(userService))

	{
		post := api.Group("/post")

		post.POST("/upload-files", postHandler.UploadFiles)
		post.DELETE("/delete-file", postHandler.DeleteFile)
		post.POST("/create", postHandler.CreatePost)
		post.POST("/share/create/:parentID", postHandler.CreateSharePost)
		post.GET("/finds", postHandler.FindsCursorPagination)
		post.GET("/finds/:userID", postHandler.FindsWithUserIDCursorPagination)
		post.GET("/find/:postID", postHandler.FindWithID)
		post.PATCH("/update/:postID", postHandler.UpdatePost)
		post.DELETE("/delete/:postID", postHandler.DeletePost)
		post.POST("/toggle-like/:postID", postHandler.ToggleLike)
	}

	{
		comment := api.Group("/comment")

		comment.POST("/upload-file", commentHandler.UploadFile)
		comment.DELETE("/delete-file", commentHandler.DeleteFile)
		comment.POST("/create/:postID", commentHandler.CreateComment)
		comment.POST("/reply/create/:postID/:parentID", commentHandler.CreateReplyComment)
		comment.POST("/tag/create/:postID/:parentID/:replyID", commentHandler.CreateTagReply)
		comment.GET("/finds/:postID", commentHandler.FindsWithPostIDCursorPagination)
		comment.PATCH("/update/:commentID", commentHandler.UpdateComment)
		comment.DELETE("/delete/:commentID", commentHandler.DeleteComment)
		comment.POST("/toggle-like/:postID/:commentID", commentHandler.ToggleLike)
	}

	{
		notification := api.Group("/notification")
		notification.GET("/finds", notificationHandler.FindsWithReceiverIDCursorPagination)
		notification.PATCH("/read-all", notificationHandler.UpdateIsRead)
	}

	{
		user := api.Group("/user")
		user.GET("/finds-with-fullname", userHandler.FindsWithFullnameCursorPagination)
		user.GET("/finds", userHandler.FindsCursorPaginationWithFollowerRelation)
		user.GET("/find/:userID", userHandler.FindByIDWithFollowRelations)
		user.POST("/toggle-follow/:followingID", userHandler.ToggleFollow)
		user.PATCH("/avatar/upload-file", userHandler.UploadEditUserAvatar)
		user.PATCH("/background/upload-file", userHandler.UploadEditUserBackground)
		user.PATCH("/avatar/delete-file", userHandler.ClearUserAvatar)
		user.PATCH("/background/delete-file", userHandler.ClearUserBackground)
		user.PUT("/edit-info", userHandler.UpdatesInfo)
	}

	app.Run()
}
