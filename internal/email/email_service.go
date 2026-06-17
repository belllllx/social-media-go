package email

import (
	"log"
	"time"

	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type SendEmailRegisterDTO struct {
	Fullname string
	Username string
	Email    string
	Password string
}

type EmailService interface {
	SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) error
}

type emailService struct {
	emailRepository EmailRepository
	otpRepository   otp.OTPRepository
}

func NewEmailService(
	emailRepository EmailRepository,
	otpRepository otp.OTPRepository,
) EmailService {
	return &emailService{
		emailRepository: emailRepository,
		otpRepository:   otpRepository,
	}
}

func (s *emailService) SendEmailRegister(sendEmailRegisterDTO *SendEmailRegisterDTO) error {
	isOTPExist, err := s.otpRepository.FindByEmail(sendEmailRegisterDTO.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		log.Println(err)
		return errs.NewInternalServerError()
	}

	if isOTPExist != nil {
		if err := s.otpRepository.Delete(isOTPExist.Email); err != nil {
			log.Println(err)
			return errs.NewInternalServerError()
		}
	}

	otpString, err := helpers.GenerateOTP()
	if err != nil {
		log.Println(err)
		return errs.NewUnexpectedError()
	}

	otpHash, err := helpers.HashSecret(otpString)
	if err != nil {
		log.Println(err)
		return errs.NewUnexpectedError()
	}

	createOTP := otp.OTP{
		Email:     sendEmailRegisterDTO.Email,
		OTPHash:   otpHash,
		ExpiredAt: time.Now().Add(time.Minute * 10),
	}
	err = s.otpRepository.Create(&createOTP)
	if err != nil {
		log.Println(err)
		return errs.NewInternalServerError()
	}

	err = s.emailRepository.Send(sendEmailRegisterDTO.Email, otpString)
	if err != nil {
		log.Println(err)
		if err := s.otpRepository.Delete(sendEmailRegisterDTO.Email); err != nil {
			log.Println(err)
		}

		return errs.NewInternalServerError()
	}

	token, err := helpers.NewJWT(
		jwt.MapClaims{
			"email":             sendEmailRegisterDTO.Email,
			"sendEmailVerified": true,
		},
		[]byte(viper.GetString("app.register_secret")),
	)
	if err != nil {
		log.Println(err)
		return errs.NewUnexpectedError()
	}

	_ = token

	return nil
}
