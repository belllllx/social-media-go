package configs

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitTimeZone() {
	ict, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		panic(err)
	}
	time.Local = ict
}

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func InitDB() *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Bangkok",
		viper.GetString("db.host"),
		viper.GetString("db.user"),
		viper.GetString("db.password"),
		viper.GetString("db.name"),
		viper.GetInt("db.port"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}
	return db
}

func InitRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr: viper.GetString("redis.host"),
	})
	return redisClient
}

func InitValidator() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	v.RegisterValidation("presignedurl", helpers.ValidatePresignedURL)
}

func InitOAuth2GoogleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     viper.GetString("app.google_client_id"),
		ClientSecret: viper.GetString("app.google_client_secret"),
		Endpoint:     google.Endpoint,
		RedirectURL:  viper.GetString("app.google_callback_url"),
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
	}
}

func InitOAuth2GithubConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     viper.GetString("app.github_client_id"),
		ClientSecret: viper.GetString("app.github_client_secret"),
		Endpoint:     github.Endpoint,
		RedirectURL:  viper.GetString("app.github_callback_url"),
		Scopes: []string{
			"read:user",
			"user:email",
		},
	}
}

func InitOAuth2FacebookConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     viper.GetString("app.facebook_client_id"),
		ClientSecret: viper.GetString("app.facebook_client_secret"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  viper.GetString("app.facebook_auth_url"),
			TokenURL: viper.GetString("app.facebook_token_url"),
		},
		RedirectURL: viper.GetString("app.facebook_callback_url"),
		Scopes: []string{
			"email",
			"public_profile",
		},
	}
}

func InitOIDCVerifier() *oidc.IDTokenVerifier {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		panic(err)
	}

	return provider.Verifier(&oidc.Config{
		ClientID: viper.GetString("app.google_client_id"),
	})
}

func InitS3Client() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(viper.GetString("app.aws_bucket_region")),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				viper.GetString("app.aws_access_key"),
				viper.GetString("app.aws_secret_access_key"),
				"",
			),
		),
	)
	if err != nil {
		panic(err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(viper.GetString("app.r2_endpoint"))
	})
}
