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
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type registerPayload struct {
	Fullname     string
	Username     string
	Email        string
	PasswordHash string
}

type AuthService interface {
	SendEmailRegister(sendEmailRegisterRequest *SendEmailRegisterRequest) (result, token string, err error)
	VerifyOTPRegister(registerEmail, otp string) (result string, err error)
}

type authService struct {
	rdb            *redis.Client
	otpRepository  otp.OTPRepository
	userRepository user.UserRepository
	emailService   email.EmailService
	otpService     otp.OTPService
}

func NewAuthService(
	rdb *redis.Client,
	otpRepository otp.OTPRepository,
	userRepository user.UserRepository,
	emailService email.EmailService,
	otpService otp.OTPService,
) AuthService {
	return &authService{
		rdb:            rdb,
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
		jwt.MapClaims{
			"email":             sendEmailRegisterRequest.Email,
			"sendEmailVerified": true,
		},
		[]byte(viper.GetString("app.register_secret")),
	)
	if err != nil {
		logs.Error(err)
		return "Failed to sign jwt", "", errs.NewUnexpectedError()
	}

	passwordHash, err := helpers.HashSecret(sendEmailRegisterRequest.Password)
	if err != nil {
		logs.Error(err)
		return "Failed to hash secret", "", errs.NewUnexpectedError()
	}

	registerPayload := &registerPayload{
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
	err = helpers.RDBSet(s.rdb, key, data, time.Minute*15)
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
	value, err := helpers.RDBGet(s.rdb, key)
	if err == redis.Nil {
		logs.Warn(err)
		return "Failed to register", errs.NewUnexpectedError()
	} else if err != nil {
		logs.Error(err)
		return "Failed to register", errs.NewInternalServerError()
	}

	registerPayload := &registerPayload{}
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
