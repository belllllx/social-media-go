package helpers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func SendEmail(
	email,
	otp,
	verifyEmailType string,
) error {
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

func VerifyJWT(
	token string,
	claims jwt.Claims,
	secret string,
) (*jwt.Token, error) {
	return jwt.ParseWithClaims(
		token,
		claims,
		func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
}

func RedisSet(
	ctx context.Context,
	redisClient *redis.Client,
	key string,
	value []byte,
	expiration time.Duration,
) error {
	return redisClient.Set(ctx, key, string(value), expiration).Err()
}

func RedisGet(
	ctx context.Context,
	redisClient *redis.Client,
	key string,
) (string, error) {
	return redisClient.Get(ctx, key).Result()
}

func RedisDelete(
	ctx context.Context,
	redisClient *redis.Client,
	key string,
) error {
	return redisClient.Del(ctx, key).Err()
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
	case "presignedurl":
		return "This field invalid presigned url"
	default:
		return "Invalid value"
	}
}

func HandleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *errs.AppError:
		c.AbortWithStatusJSON(e.Status, gin.H{
			"status":  e.Status,
			"success": false,
			"message": e.Message,
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

func GenerateFilename(filename string) string {
	fileExt := filepath.Ext(filename)
	newFilename := uuid.NewString() + fileExt
	return newFilename
}

func IsExternalURL(url string) bool {
	if strings.Contains(url, "https") {
		return true
	}
	return false
}

func PutObject(
	ctx context.Context,
	s3Client *s3.Client,
	key string,
	body io.Reader,
	contentType string,
) (*s3.PutObjectOutput, error) {
	return s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(viper.GetString("app.aws_bucket_name")),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
}

func PresignGetObject(
	ctx context.Context,
	presignClient *s3.PresignClient,
	key string,
) (*v4.PresignedHTTPRequest, error) {
	return presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24
	})
}

func DeleteObject(
	ctx context.Context,
	s3Client *s3.Client,
	key string,
) (*s3.DeleteObjectOutput, error) {
	return s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
		Key:    aws.String(key),
	})
}

func DeleteObjects(
	ctx context.Context,
	s3Client *s3.Client,
	keys []string,
) (*s3.DeleteObjectsOutput, error) {
	objectsIdentifier := []types.ObjectIdentifier{}
	for _, key := range keys {
		objectsIdentifier = append(objectsIdentifier, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	return s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
		Delete: &types.Delete{
			Objects: objectsIdentifier,
		},
	})
}

func SplitFilename(filePath string) (fileDIR, filename string) {
	fileSplit := strings.Split(filePath, "/")
	return fileSplit[0], fileSplit[len(fileSplit)-1]
}

func SplitPresignedURL(presignedURL string) (fileDIR, filename string, err error) {
	u, err := url.Parse(presignedURL)
	if err != nil {
		return "", "", err
	}

	fDIR := path.Dir(u.Path)
	fName := path.Base(u.Path)
	return strings.TrimPrefix(fDIR, "/"), fName, nil
}

func OmitUserPasswordHash(db *gorm.DB) *gorm.DB {
	return db.Select(
		"id",
		"fullname",
		"username",
		"email",
		"date_of_birth",
		"profile_url",
		"profile_background_url",
		"info",
		"role",
		"provider_type",
		"created_at",
		"updated_at",
	)
}

func ValidatePresignedURL(fl validator.FieldLevel) bool {
	u, err := url.Parse(fl.Field().String())
	if err != nil {
		return false
	}

	if u.Host != viper.GetString("app.bucket_host") {
		return false
	}

	if !strings.HasPrefix(u.Path, "/post-image/") &&
		!strings.HasPrefix(u.Path, "/post-video/") &&
		!strings.HasPrefix(u.Path, "/comment-image/") &&
		!strings.HasPrefix(u.Path, "/reply-image/") {
		return false
	}

	q := u.Query()

	required := []string{
		"X-Amz-Algorithm",
		"X-Amz-Credential",
		"X-Amz-Date",
		"X-Amz-Expires",
		"X-Amz-Signature",
	}

	for _, k := range required {
		if q.Get(k) == "" {
			return false
		}
	}

	return true
}

func ValidateUUID(s string) error {
	err := uuid.Validate(s)
	if err != nil {
		return errs.NewBadRequestErrorWithMessage(fmt.Sprintf("Invalid uuid for %s", s))
	}

	return nil
}

func ParseUUID(s string) (*uuid.UUID, error) {
	uuid, err := uuid.Parse(s)
	if err != nil {
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to parse uuid")
	}

	return &uuid, nil
}

func GetUserImage(
	ctx context.Context,
	presignClient *s3.PresignClient,
	user *models.User,
) error {
	// ไม่ใช่ avater ของ social login
	// อัพเดต profile url
	if user.ProfileURL != nil && !IsExternalURL(*user.ProfileURL) {
		req, err := PresignGetObject(
			ctx,
			presignClient,
			*user.ProfileURL,
		)
		if err != nil {
			return errs.NewInternalServerErrorWithMessage("Failed to presign get user avatar url object")
		}

		user.ProfileURL = &req.URL
	}

	return nil
}

func GetUserBackgroundImage(
	ctx context.Context,
	presignClient *s3.PresignClient,
	user *models.User,
) error {
	if user.ProfileBackgroundURL != nil {
		req, err := PresignGetObject(
			ctx,
			presignClient,
			*user.ProfileBackgroundURL,
		)
		if err != nil {
			return errs.NewInternalServerErrorWithMessage("Failed to presign get user background url object")
		}

		user.ProfileBackgroundURL = &req.URL
	}

	return nil
}
