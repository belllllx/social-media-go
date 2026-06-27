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

type tokens struct {
	accessToken  string
	refreshToken string
}

type UserAccessTokenJWTPayload struct {
	ID           uuid.UUID
	AuthVerified bool
	jwt.RegisteredClaims
}

type UserRefreshTokenJWTPayload struct {
	ID uuid.UUID
	jwt.RegisteredClaims
}

type SendEmailRegisterJWTPayload struct {
	Email             string
	SendEmailVerified bool
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
	VerifyOTPRegister(registerEmail, otp string) (result string, err error)
	ValidateUserLogin(loginRequest *LoginRequest) (secureUser *user.SecureUser, err error)
	Login(userID uuid.UUID) (result string, tokens *tokens, err error)
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

	result, err := s.emailService.SendEmailRegister(sendEmailRegisterRequest.Email)
	if err != nil {
		return result, "", err
	}

	token, err := helpers.NewJWT(
		&SendEmailRegisterJWTPayload{
			Email:             sendEmailRegisterRequest.Email,
			SendEmailVerified: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
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
		return "Failed to hash secret", "", errs.NewUnexpectedError()
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

func (s *authService) VerifyOTPRegister(registerEmail, otp string) (string, error) {
	result, err := s.otpService.Verify(registerEmail, otp)
	if err != nil {
		return result, err
	}

	key := fmt.Sprintf("email:register-pending:%s", registerEmail)
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

	err = s.otpRepository.Delete(registerEmail)
	if err != nil {
		logs.Error(err)
		return "Failed to delete otp", errs.NewInternalServerError()
	}

	return "Register user successfully", nil
}

func (s *authService) ValidateUserLogin(loginRequest *LoginRequest) (*user.SecureUser, error) {
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

	secureUser := &user.SecureUser{
		ID:                   userExist.ID,
		Fullname:             userExist.Fullname,
		Username:             userExist.Username,
		Email:                userExist.Email,
		DateOfBirth:          userExist.DateOfBirth,
		ProfileUrl:           userExist.ProfileUrl,
		ProfileBackgroundUrl: userExist.ProfileBackgroundUrl,
		Info:                 userExist.Info,
		Role:                 userExist.Role,
		ProviderType:         userExist.ProviderType,
		CreatedAt:            userExist.CreatedAt,
		UpdatedAt:            userExist.UpdatedAt,
	}
	return secureUser, nil
}

func (s *authService) Login(userID uuid.UUID) (string, *tokens, error) {
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

	tokens := &tokens{
		accessToken:  accessToken,
		refreshToken: refreshToken,
	}
	return "Login successfully", tokens, nil
}
