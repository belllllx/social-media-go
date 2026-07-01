package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type SocialUser struct {
	ProviderType user.ProviderType
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

type RegisterPayload struct {
	Fullname     string
	Username     string
	Email        string
	PasswordHash string
}

type AuthService interface {
	SendEmailRegister(sendEmailRegisterRequest *SendEmailRegisterRequest) (result, token string, err error)
	SendEmailForgotPassword(sendEmailForgotPasswordRequest *SendEmailForgotPasswordRequest) (result, token string, err error)
	VerifyOTPRegister(email, otp string) (result string, err error)
	VerifyOTPForgotPassword(email, otp string) (result, token string, err error)
	ValidateUserLogin(loginRequest *LoginRequest) (userID *uuid.UUID, err error)
	Login(userID uuid.UUID) (result string, tokens *Tokens, err error)
	Refresh(userID uuid.UUID) (result string, tokens *Tokens, err error)
	SocialLogin(providerType user.ProviderType) (result, url string, err error)
	SocialLoginCallback(socialUser *SocialUser) (result string, tokens *Tokens, url string, err error)
}

type authService struct {
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
	redisClient *redis.Client,
	otpRepository otp.OTPRepository,
	userRepository user.UserRepository,
	emailService email.EmailService,
	otpService otp.OTPService,
) AuthService {
	googleConfig := configs.InitOAuth2GoogleConfig()
	githubConfig := configs.InitOAuth2GithubConfig()
	facebookConfig := configs.InitOAuth2FacebookConfig()

	return &authService{
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

func (s *authService) SendEmailRegister(sendEmailRegisterRequest *SendEmailRegisterRequest) (string, string, error) {
	userExist, err := s.userRepository.FindByUsername(sendEmailRegisterRequest.Username)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to find user", "", errs.NewInternalServerError()
	}
	if userExist != nil {
		logs.Warn(errors.New("Username is already exist"))
		return "Username is already exist", "", errs.NewBadRequestError()
	}

	userExist, err = s.userRepository.FindByEmail(sendEmailRegisterRequest.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to find user", "", errs.NewInternalServerError()
	}
	if userExist != nil {
		logs.Warn(errors.New("Email is already exist"))
		return "Email is already exist", "", errs.NewBadRequestError()
	}

	result, err := s.emailService.SendEmail(sendEmailRegisterRequest.Email, "register")
	if err != nil {
		return result, "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailTokenClaims{
			Email:             sendEmailRegisterRequest.Email,
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
		return "Failed to sign register token", "", errs.NewUnexpectedError()
	}

	passwordHash, err := helpers.HashSecret(sendEmailRegisterRequest.Password)
	if err != nil {
		logs.Error(err)
		return "Failed to hash password", "", errs.NewUnexpectedError()
	}

	registerPayload := &RegisterPayload{
		Fullname:     sendEmailRegisterRequest.Fullname,
		Username:     sendEmailRegisterRequest.Username,
		Email:        sendEmailRegisterRequest.Email,
		PasswordHash: passwordHash,
	}
	key := fmt.Sprintf("email:register-pending:%s", registerPayload.Email)
	data, err := json.Marshal(registerPayload)
	if err != nil {
		logs.Error(err)
		return "Failed to marshal json", "", errs.NewUnexpectedError()
	}
	err = helpers.RedisSet(s.redisClient, key, data, time.Minute*15)
	if err != nil {
		logs.Error(err)
		return "Failed to set redis", "", errs.NewInternalServerError()
	}

	return result, token, nil
}

func (s *authService) SendEmailForgotPassword(sendEmailForgotPasswordRequest *SendEmailForgotPasswordRequest) (string, string, error) {
	userExist, err := s.userRepository.FindByEmail(sendEmailForgotPasswordRequest.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to find user by email", "", errs.NewInternalServerError()
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return fmt.Sprintf("Email %s is not found", sendEmailForgotPasswordRequest.Email), "", errs.NewNotFoundError()
	}

	// กรณี social account ห้าม
	if userExist.ProviderType == user.ProviderTypeGoogle ||
		userExist.ProviderType == user.ProviderTypeGithub ||
		userExist.ProviderType == user.ProviderTypeFacebook {
		return "Cannot reset password for social media account", "", errs.NewBadRequestError()
	}

	result, err := s.emailService.SendEmail(sendEmailForgotPasswordRequest.Email, "reset password")
	if err != nil {
		return result, "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailTokenClaims{
			Email:             sendEmailForgotPasswordRequest.Email,
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
		return "Failed to sign forgot password token", "", errs.NewUnexpectedError()
	}

	return result, token, nil
}

func (s *authService) VerifyOTPRegister(email, otp string) (string, error) {
	result, err := s.otpService.Verify(email, otp)
	if err != nil {
		return result, err
	}

	key := fmt.Sprintf("email:register-pending:%s", email)
	value, err := helpers.RedisGet(s.redisClient, key)
	if err == redis.Nil {
		logs.Warn(err)
		return "Failed to get does not exist key redis", errs.NewUnexpectedError()
	} else if err != nil {
		logs.Error(err)
		return "Failed to get value redis", errs.NewInternalServerError()
	}

	registerPayload := &RegisterPayload{}
	err = json.Unmarshal([]byte(value), registerPayload)
	if err != nil {
		logs.Error(err)
		return "Failed to unmarshal json", errs.NewUnexpectedError()
	}
	user := &user.User{
		Fullname:     registerPayload.Fullname,
		Username:     &registerPayload.Username,
		Email:        registerPayload.Email,
		PasswordHash: &registerPayload.PasswordHash,
	}
	err = s.userRepository.Create(user)
	if err != nil {
		logs.Error(err)
		return "Failed to create user", errs.NewInternalServerError()
	}

	err = s.otpRepository.Delete(email)
	if err != nil {
		logs.Error(err)
		return "Failed to delete otp", errs.NewInternalServerError()
	}

	err = helpers.RedisDel(s.redisClient, key)
	if err != nil {
		logs.Error(err)
		return "Failed to delete key redis", errs.NewInternalServerError()
	}

	return "Register user successfully", nil
}

func (s *authService) VerifyOTPForgotPassword(email, otp string) (string, string, error) {
	result, err := s.otpService.Verify(email, otp)
	if err != nil {
		return result, "", err
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
		return "Failed to sign reset password token", "", errs.NewUnexpectedError()
	}

	err = s.otpRepository.Delete(email)
	if err != nil {
		logs.Error(err)
		return "Failed to delete otp", "", errs.NewInternalServerError()
	}

	return "Verify otp successfully", token, nil
}

func (s *authService) ValidateUserLogin(loginRequest *LoginRequest) (*uuid.UUID, error) {
	userExist, err := s.userRepository.FindByUsername(loginRequest.Username)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerError()
	}

	if helpers.IsErrRecordNotFound(err) {
		return nil, errs.NewUnauthorizedError()
	}

	err = helpers.CompareSecret(*userExist.PasswordHash, loginRequest.Password)
	if err != nil {
		return nil, errs.NewUnauthorizedError()
	}

	return &userExist.ID, nil
}

func (s *authService) Login(userID uuid.UUID) (string, *Tokens, error) {
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
		return "Failed to sign access token", nil, errs.NewUnexpectedError()
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
		return "Failed to sign refresh token", nil, errs.NewUnexpectedError()
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return "Login successfully", tokens, nil
}

func (s *authService) Refresh(userID uuid.UUID) (string, *Tokens, error) {
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
		return "Failed to sign access token", nil, errs.NewUnexpectedError()
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
		return "Failed to sign refresh token", nil, errs.NewUnexpectedError()
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return "Refresh token successfully", tokens, nil
}

func (s *authService) SocialLogin(providerType user.ProviderType) (string, string, error) {
	state, err := helpers.GenerateRandomState()
	if err != nil {
		logs.Error(err)
		return "Failed to generate oauth2 state", "", errs.NewUnexpectedError()
	}

	key := fmt.Sprintf("auth:oauth2-state:%s", state)
	err = helpers.RedisSet(s.redisClient, key, []byte(state), time.Minute*5)
	if err != nil {
		logs.Error(err)
		return "Failed to set redis", "", errs.NewInternalServerError()
	}

	url := ""
	switch providerType {
	case user.ProviderTypeGoogle:
		url = s.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	case user.ProviderTypeGithub:
		url = s.githubConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	case user.ProviderTypeFacebook:
		url = s.facebookConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	}
	return "Generate url for social login successfully", url, nil
}

func (s *authService) SocialLoginCallback(socialUser *SocialUser) (string, *Tokens, string, error) {
	userExist, err := s.userRepository.FindByEmail(socialUser.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to find user", nil, "", errs.NewInternalServerError()
	}

	authSuccessURL := fmt.Sprintf("%s%s", viper.GetString("app.client_url"), viper.GetString("app.client_redirect_auth_success_path"))
	authErrorURL := fmt.Sprintf("%s%s", viper.GetString("app.client_url"), viper.GetString("app.client_redirect_auth_error_path"))

	// ยังไม่มี account -> create
	if helpers.IsErrRecordNotFound(err) {
		createUser := &user.User{
			Fullname:     socialUser.Name,
			Email:        socialUser.Email,
			ProviderType: socialUser.ProviderType,
			ProfileUrl:   &socialUser.AvatarURL,
		}
		err = s.userRepository.Create(createUser)
		if err != nil {
			logs.Error(err)
			return "Failed to create social account", nil, "", errs.NewInternalServerError()
		}

		accessToken, err := helpers.NewJWT(
			&UserAccessTokenClaims{
				ID:           createUser.ID,
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
			return "Failed to sign access token", nil, "", errs.NewUnexpectedError()
		}

		refreshToken, err := helpers.NewJWT(
			&UserRefreshTokenClaims{
				ID: createUser.ID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			},
			viper.GetString("app.refresh_token_secret"),
		)
		if err != nil {
			logs.Error(err)
			return "Failed to sign refresh token", nil, "", errs.NewUnexpectedError()
		}

		tokens := &Tokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
		return "Login successfully", tokens, authSuccessURL, nil
	}

	// กรณี provider ไม่ตรงกับ social login ที่ใช้
	if userExist != nil && userExist.ProviderType != socialUser.ProviderType {
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
			return "Failed to sign social login error token", nil, "", errs.NewUnexpectedError()
		}

		msg := "email already registered with a different provider"
		return "Failed to login with social", nil, fmt.Sprintf("%s?message=%s&error_token=%s", authErrorURL, url.PathEscape(msg), socialAuthErrToken), nil
	}

	accessToken, err := helpers.NewJWT(
		&UserAccessTokenClaims{
			ID:           userExist.ID,
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
		return "Failed to sign access token", nil, "", errs.NewUnexpectedError()
	}

	refreshToken, err := helpers.NewJWT(
		&UserRefreshTokenClaims{
			ID: userExist.ID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		},
		viper.GetString("app.refresh_token_secret"),
	)
	if err != nil {
		logs.Error(err)
		return "Failed to sign refresh token", nil, "", errs.NewUnexpectedError()
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return "Login successfully", tokens, authSuccessURL, nil
}
