package email

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type SendEmailRegisterDTO struct {
	Fullname string
	Username string
	Email    string
	Password string
}

type EmailService interface {
	SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) (result, token string, err error)
}

type emailService struct {
	rdb             *redis.Client
	emailRepository EmailRepository
	otpRepository   otp.OTPRepository
}

func NewEmailService(
	rdb *redis.Client,
	emailRepository EmailRepository,
	otpRepository otp.OTPRepository,
) EmailService {
	return &emailService{
		rdb:             rdb,
		emailRepository: emailRepository,
		otpRepository:   otpRepository,
	}
}

func (s *emailService) SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) (string, string, error) {
	isOTPExist, err := s.otpRepository.FindByEmail(sendEmailRegisterDTO.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", "", errs.NewInternalServerError()
	}

	if isOTPExist != nil {
		if err := s.otpRepository.Delete(isOTPExist.Email); err != nil {
			logs.Error(err)
			return "", "", errs.NewInternalServerError()
		}
	}

	otpString, err := helpers.NewOTP()
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewUnexpectedError()
	}

	otpHash, err := helpers.HashSecret(otpString)
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewUnexpectedError()
	}

	createOTP := otp.OTP{
		Email:     sendEmailRegisterDTO.Email,
		OTPHash:   otpHash,
		ExpiredAt: time.Now().Add(time.Minute * 10),
	}
	err = s.otpRepository.Create(&createOTP)
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewInternalServerError()
	}

	result := ""
	err = s.emailRepository.Send(sendEmailRegisterDTO.Email, otpString)
	if err != nil {
		logs.Error(err)
		if err := s.otpRepository.Delete(sendEmailRegisterDTO.Email); err != nil {
			logs.Error(err)
		}

		result = fmt.Sprintf("Cannot send email to %s", sendEmailRegisterDTO.Email)
		return result, "", errs.NewInternalServerError()
	}
	result = fmt.Sprintf("Send email to %s successfully", sendEmailRegisterDTO.Email)

	token, err := helpers.NewJWT(
		jwt.MapClaims{
			"email":             sendEmailRegisterDTO.Email,
			"sendEmailVerified": true,
		},
		[]byte(viper.GetString("app.register_secret")),
	)
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewUnexpectedError()
	}

	key := fmt.Sprintf("email:register-pending:%s", sendEmailRegisterDTO.Email)
	data, err := json.Marshal(sendEmailRegisterDTO)
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewUnexpectedError()
	}
	err = helpers.RDBSet(s.rdb, key, data, time.Minute*15)
	if err != nil {
		logs.Error(err)
		return "", "", errs.NewInternalServerError()
	}

	return result, token, nil
}
