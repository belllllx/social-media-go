package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type SocialUserDTO struct {
	ProviderType models.ProviderType
	Email        string
	Name         string
	AvatarURL    string
}

type FacebookUser struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

type GithubEmail struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

type GithubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type GoogleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type SocialAuthErrorTokenClaims struct {
	SocialAuthVerified bool `json:"socialAuthVerified"`
	jwt.RegisteredClaims
}

type UserAccessTokenClaims struct {
	ID           uuid.UUID `json:"id"`
	AuthVerified bool      `json:"authVerified"`
	jwt.RegisteredClaims
}

type UserRefreshTokenClaims struct {
	ID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

type ResetPasswordTokenClaims struct {
	Email       string `json:"email"`
	OTPVerified bool   `json:"otpVerified"`
	jwt.RegisteredClaims
}

type SendEmailTokenClaims struct {
	Email             string `json:"email"`
	SendEmailVerified bool   `json:"sendEmailVerified"`
	jwt.RegisteredClaims
}

type LoginDTO struct {
	Username string
	Password string
}

type RegisterPayload struct {
	Fullname     string `json:"fullname"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
}

type SendEmailRegisterDTO struct {
	Fullname string
	Username string
	Email    string
	Password string
}

type AuthService interface {
	SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) (token string, err error)
	SendEmailForgotPassword(email string) (token string, err error)
	ResendEmail(email, sendEmailType string) error
	VerifyOTPRegister(email, otp string) error
	VerifyOTPForgotPassword(email, otp string) (token string, err error)
	ValidateUserLogin(loginDTO *LoginDTO) (userID *uuid.UUID, err error)
	CreateTokens(userID uuid.UUID) (tokens *Tokens, err error)
	SocialLogin(providerType models.ProviderType) (url string, err error)
	SocialLoginCallback(socialUserDTO *SocialUserDTO) (tokens *Tokens, url string, err error)
}

type authService struct {
	db             *gorm.DB
	redisClient    *redis.Client
	otpRepository  otp.OTPRepository
	userRepository user.UserRepository
	emailService   email.EmailService
	otpService     otp.OTPService
	googleConfig   *oauth2.Config
	githubConfig   *oauth2.Config
	facebookConfig *oauth2.Config
}

func NewAuthService(
	db *gorm.DB,
	redisClient *redis.Client,
	otpRepository otp.OTPRepository,
	userRepository user.UserRepository,
	emailService email.EmailService,
	otpService otp.OTPService,
	googleConfig *oauth2.Config,
	githubConfig *oauth2.Config,
	facebookConfig *oauth2.Config,
) AuthService {
	return &authService{
		db:             db,
		redisClient:    redisClient,
		otpRepository:  otpRepository,
		userRepository: userRepository,
		emailService:   emailService,
		otpService:     otpService,
		googleConfig:   googleConfig,
		githubConfig:   githubConfig,
		facebookConfig: facebookConfig,
	}
}

func (s *authService) SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) (string, error) {
	userExist, err := s.userRepository.FindByUsername(s.db, sendEmailRegisterDTO.Username)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to find user")
	}
	if userExist != nil {
		logs.Warn(errors.New("Username is already exist"))
		return "", errs.NewBadRequestErrorWithMessage("Username is already exist")
	}

	userExist, err = s.userRepository.FindByEmail(s.db, sendEmailRegisterDTO.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to find user")
	}
	if userExist != nil {
		logs.Warn(errors.New("Email is already exist"))
		return "", errs.NewBadRequestErrorWithMessage("Email is already exist")
	}

	err = s.emailService.SendEmail(s.db, sendEmailRegisterDTO.Email, "register")
	if err != nil {
		return "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailTokenClaims{
			Email:             sendEmailRegisterDTO.Email,
			SendEmailVerified: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.register_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to sign register token")
	}

	passwordHash, err := helpers.HashSecret(sendEmailRegisterDTO.Password)
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to hash password")
	}

	registerPayload := &RegisterPayload{
		Fullname:     sendEmailRegisterDTO.Fullname,
		Username:     sendEmailRegisterDTO.Username,
		Email:        sendEmailRegisterDTO.Email,
		PasswordHash: passwordHash,
	}
	key := fmt.Sprintf("email:register-pending:%s", registerPayload.Email)
	data, err := json.Marshal(registerPayload)
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to marshal json")
	}
	err = helpers.RedisSet(s.redisClient, context.Background(), key, data, time.Minute*15)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to set redis")
	}

	return token, nil
}

func (s *authService) SendEmailForgotPassword(email string) (string, error) {
	userExist, err := s.userRepository.FindByEmail(s.db, email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to find user by email")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Email %s is not found", email))
	}

	// กรณี social account ห้าม
	if userExist.ProviderType == models.ProviderTypeGoogle ||
		userExist.ProviderType == models.ProviderTypeGithub ||
		userExist.ProviderType == models.ProviderTypeFacebook {
		return "", errs.NewBadRequestErrorWithMessage("Cannot reset password for social media account")
	}

	err = s.emailService.SendEmail(s.db, email, "reset password")
	if err != nil {
		return "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailTokenClaims{
			Email:             email,
			SendEmailVerified: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.forgot_password_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to sign forgot password token")
	}

	return token, nil
}

func (s *authService) ResendEmail(email, sendEmailType string) error {
	return s.emailService.SendEmail(s.db, email, sendEmailType)
}

func (s *authService) VerifyOTPRegister(email, otp string) error {
	err := s.otpService.Verify(s.db, email, otp)
	if err != nil {
		return err
	}

	ctx := context.Background()
	key := fmt.Sprintf("email:register-pending:%s", email)
	value, err := helpers.RedisGet(s.redisClient, ctx, key)
	if err == redis.Nil {
		logs.Warn(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to get does not exist key redis")
	} else if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to get value redis")
	}

	registerPayload := &RegisterPayload{}
	err = json.Unmarshal([]byte(value), registerPayload)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to unmarshal json")
	}
	user := &models.User{
		Fullname:     registerPayload.Fullname,
		Username:     &registerPayload.Username,
		Email:        registerPayload.Email,
		PasswordHash: &registerPayload.PasswordHash,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.userRepository.Create(tx, user)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create user")
		}

		err = s.otpRepository.Delete(tx, email)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete otp")
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return err
	}

	err = helpers.RedisDelete(s.redisClient, ctx, key)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to delete key redis")
	}

	return nil
}

func (s *authService) VerifyOTPForgotPassword(email, otp string) (string, error) {
	err := s.otpService.Verify(s.db, email, otp)
	if err != nil {
		return "", err
	}

	token, err := helpers.NewJWT(
		&ResetPasswordTokenClaims{
			Email:       email,
			OTPVerified: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.reset_password_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to sign reset password token")
	}

	err = s.otpRepository.Delete(s.db, email)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to delete otp")
	}

	return token, nil
}

func (s *authService) ValidateUserLogin(loginDTO *LoginDTO) (*uuid.UUID, error) {
	userExist, err := s.userRepository.FindByUsername(s.db, loginDTO.Username)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerError()
	}

	if helpers.IsErrRecordNotFound(err) {
		return nil, errs.NewUnauthorizedError()
	}

	err = helpers.CompareSecret(*userExist.PasswordHash, loginDTO.Password)
	if err != nil {
		return nil, errs.NewUnauthorizedError()
	}

	return &userExist.ID, nil
}

func (s *authService) CreateTokens(userID uuid.UUID) (*Tokens, error) {
	accessToken, err := helpers.NewJWT(
		&UserAccessTokenClaims{
			ID:           userID,
			AuthVerified: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.access_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to sign access token")
	}

	refreshToken, err := helpers.NewJWT(
		&UserRefreshTokenClaims{
			ID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.refresh_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to sign refresh token")
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return tokens, nil
}

func (s *authService) SocialLogin(providerType models.ProviderType) (string, error) {
	state, err := helpers.GenerateRandomState()
	if err != nil {
		logs.Error(err)
		return "", errs.NewUnexpectedErrorWithMessage("Failed to generate oauth2 state")
	}

	key := fmt.Sprintf("auth:oauth2-state:%s", state)
	err = helpers.RedisSet(s.redisClient, context.Background(), key, []byte(state), time.Minute*5)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to set redis")
	}

	url := ""
	switch providerType {
	case models.ProviderTypeGoogle:
		url = s.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	case models.ProviderTypeGithub:
		url = s.githubConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	case models.ProviderTypeFacebook:
		url = s.facebookConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	}
	return url, nil
}

func (s *authService) SocialLoginCallback(socialUserDTO *SocialUserDTO) (*Tokens, string, error) {
	userExist, err := s.userRepository.FindByEmail(s.db, socialUserDTO.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, "", errs.NewInternalServerErrorWithMessage("Failed to find user")
	}

	authSuccessURL := fmt.Sprintf("%s%s", viper.GetString("app.client_url"), viper.GetString("app.client_redirect_auth_success_path"))
	authErrorURL := fmt.Sprintf("%s%s", viper.GetString("app.client_url"), viper.GetString("app.client_redirect_auth_error_path"))

	// ยังไม่มี account -> create
	if helpers.IsErrRecordNotFound(err) {
		createUser := &models.User{
			Fullname:     socialUserDTO.Name,
			Email:        socialUserDTO.Email,
			ProviderType: socialUserDTO.ProviderType,
			ProfileUrl:   &socialUserDTO.AvatarURL,
		}
		err = s.userRepository.Create(s.db, createUser)
		if err != nil {
			logs.Error(err)
			return nil, "", errs.NewInternalServerErrorWithMessage("Failed to create social account")
		}

		tokens, err := s.CreateTokens(createUser.ID)
		if err != nil {
			return nil, "", err
		}

		return tokens, authSuccessURL, nil
	}

	// กรณี provider ไม่ตรงกับ social login ที่ใช้
	if userExist != nil && userExist.ProviderType != socialUserDTO.ProviderType {
		socialAuthErrToken, err := helpers.NewJWT(
			&SocialAuthErrorTokenClaims{
				SocialAuthVerified: true,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			},
			viper.GetString("app.social_login_error_token_secret"),
		)
		if err != nil {
			logs.Error(err)
			return nil, "", errs.NewUnexpectedErrorWithMessage("Failed to sign social login error token")
		}

		msg := "email already registered with a different provider"
		return nil, fmt.Sprintf("%s?message=%s&error_token=%s", authErrorURL, url.PathEscape(msg), socialAuthErrToken), nil
	}

	tokens, err := s.CreateTokens(userExist.ID)
	if err != nil {
		return nil, "", err
	}

	return tokens, authSuccessURL, nil
}
