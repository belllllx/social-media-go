package bootstrap

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/socket"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	server "github.com/zishang520/socket.io/servers/socket/v3"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	RedisClient   *redis.Client
	Router        *gin.Engine
	Cron          *cron.Cron
	Verifier      *oidc.IDTokenVerifier
	S3Client      *s3.Client
	PresignClient *s3.PresignClient
	Socket        *server.Server
}

func NewApp() *App {
	configs.InitTimeZone()
	configs.InitConfig()
	configs.InitValidator()

	db := configs.InitDB()
	configs.Migrate(db)

	redisClient := configs.InitRedis()
	verifier := configs.InitOIDCVerifier()
	s3Client := configs.InitS3Client()
	presignClient := s3.NewPresignClient(s3Client)

	router := gin.New()
	cron := cron.New()
	socket := socket.NewSocketServer()

	return &App{
		DB:            db,
		RedisClient:   redisClient,
		Router:        router,
		Cron:          cron,
		Verifier:      verifier,
		S3Client:      s3Client,
		PresignClient: presignClient,
		Socket:        socket,
	}
}

func (a *App) Run() {
	port := fmt.Sprintf(":%s", viper.GetString("app.port"))
	a.Router.Run(port)
}

func (a *App) Close() {
	a.RedisClient.Close()
	logs.Sync()
	a.Socket.Close(func(error) {})
}
