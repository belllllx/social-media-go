package bootstrap

import (
	"fmt"

	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	Router      *gin.Engine
	Cron        *cron.Cron
}

func NewApp() *App {
	configs.InitTimeZone()
	configs.InitConfig()
	configs.InitValidator()

	db := configs.InitDB()
	redisClient := configs.InitRedis()
	router := gin.New()
	cron := cron.New()

	return &App{
		DB:          db,
		RedisClient: redisClient,
		Router:      router,
		Cron:        cron,
	}
}

func (a *App) Run() {
	port := fmt.Sprintf(":%s", viper.GetString("app.port"))
	a.Router.Run(port)
}

func (a *App) Close() {
	a.RedisClient.Close()
	logs.Sync()
}
