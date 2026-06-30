package helpers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

func NewOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	otp := fmt.Sprintf("%06d", n.Int64())
	return otp, nil
}

func HashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CompareSecret(hashedSecret, secret string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(secret))
}

func IsErrRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func SendEmail(email, otp, verifyEmailType string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("email.from"))
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Verify Your Email")
	m.SetBody("text/html", fmt.Sprintf(`
		<p>Enter <b>%s</b> in the page to verify your email address and complete to %s process.</p>
    <p>This code <b>expires in 10 minutes</b>.</p>
	`, otp, verifyEmailType))

	d := gomail.NewDialer(
		viper.GetString("email.host"),
		viper.GetInt("email.port"),
		viper.GetString("email.user"),
		viper.GetString("email.password"),
	)
	return d.DialAndSend(m)
}

func NewJWT(claims jwt.Claims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func VerifyJWT(token string, claims jwt.Claims, secret string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(
		token,
		claims,
		func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
}

func RedisSet(redisClient *redis.Client, key string, value []byte, expiration time.Duration) error {
	return redisClient.Set(context.Background(), key, string(value), expiration).Err()
}

func RedisGet(redisClient *redis.Client, key string) (string, error) {
	return redisClient.Get(context.Background(), key).Result()
}

func RedisDel(redisClient *redis.Client, key string) error {
	return redisClient.Del(context.Background(), key).Err()
}

func GetErrorMessages(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("This field minimum length is %s", err.Param())
	case "max":
		return fmt.Sprintf("This field maximum length is %s", err.Param())
	case "len":
		return fmt.Sprintf("This field length is %s", err.Param())
	case "eqfield":
		return fmt.Sprintf("This field does not match the %s field", strings.ToLower(err.Param()))
	default:
		return "Invalid value"
	}
}

func HandleError(c *gin.Context, err error, msg string) {
	switch e := err.(type) {
	case *errs.AppError:
		c.AbortWithStatusJSON(e.Status, gin.H{
			"status":  e.Status,
			"success": false,
			"message": msg,
		})
	case error:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  http.StatusInternalServerError,
			"success": false,
			"message": e.Error(),
		})
	}
}

func GenerateRandomState() (string, error) {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
