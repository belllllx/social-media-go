package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
)

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type UserAccessTokenJWTPayload struct {
	ID           uuid.UUID `json:"id"`
	AuthVerified bool      `json:"authVerified"`
	jwt.RegisteredClaims
}

type UserRefreshTokenJWTPayload struct {
	ID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

type ResetPasswordTokenJWTPayload struct {
	Email       string `json:"email"`
	OTPVerified bool   `json:"otpVerified"`
	jwt.RegisteredClaims
}

type SendEmailTokenJWTPayload struct {
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
}

type authService struct {
	redisClient    *redis.Client
	otpRepository  otp.OTPRepository
	userRepository user.UserRepository
	emailService   email.EmailService
	otpService     otp.OTPService
}

func NewAuthService(
	redisClient *redis.Client,
	otpRepository otp.OTPRepository,
	userRepository user.UserRepository,
	emailService email.EmailService,
	otpService otp.OTPService,
) AuthService {
	return &authService{
		redisClient:    redisClient,
		otpRepository:  otpRepository,
		userRepository: userRepository,
		emailService:   emailService,
		otpService:     otpService,
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
		&SendEmailTokenJWTPayload{
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
		return "Failed to sign register token jwt", "", errs.NewUnexpectedError()
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
	if userExist.ProviderType == user.ProviderTypeGoogle || userExist.ProviderType == user.ProviderTypeGithub {
		return "Cannot reset password for social media account", "", errs.NewBadRequestError()
	}

	result, err := s.emailService.SendEmail(sendEmailForgotPasswordRequest.Email, "reset password")
	if err != nil {
		return result, "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailTokenJWTPayload{
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
		return "Failed to sign forgot password token jwt", "", errs.NewUnexpectedError()
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
		return "Failed to register", errs.NewUnexpectedError()
	} else if err != nil {
		logs.Error(err)
		return "Failed to register", errs.NewInternalServerError()
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

	return "Register user successfully", nil
}

func (s *authService) VerifyOTPForgotPassword(email, otp string) (string, string, error) {
	result, err := s.otpService.Verify(email, otp)
	if err != nil {
		return result, "", err
	}

	token, err := helpers.NewJWT(
		&ResetPasswordTokenJWTPayload{
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
		return "Failed to sign reset password token jwt", "", errs.NewUnexpectedError()
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
		&UserAccessTokenJWTPayload{
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
		return "Failed to sign access token jwt", nil, errs.NewUnexpectedError()
	}

	refreshToken, err := helpers.NewJWT(
		&UserRefreshTokenJWTPayload{
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
		return "Failed to sign refresh token jwt", nil, errs.NewUnexpectedError()
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return "Login successfully", tokens, nil
}

func (s *authService) Refresh(userID uuid.UUID) (string, *Tokens, error) {
	accessToken, err := helpers.NewJWT(
		&UserAccessTokenJWTPayload{
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
		return "Failed to sign access token jwt", nil, errs.NewUnexpectedError()
	}

	refreshToken, err := helpers.NewJWT(
		&UserRefreshTokenJWTPayload{
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
		return "Failed to sign refresh token jwt", nil, errs.NewUnexpectedError()
	}

	tokens := &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return "Refresh token successfully", tokens, nil
}
