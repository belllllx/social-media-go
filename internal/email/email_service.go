package email

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/redis/go-redis/v9"
)

type EmailService interface {
	SendEmailRegister(email string) (result string, err error)
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

func (s *emailService) SendEmailRegister(email string) (string, error) {
	isOTPExist, err := s.otpRepository.FindByEmail(email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to send email register", errs.NewInternalServerError()
	}

	// ลบ otp ของเก่าถ้าเจอ
	if isOTPExist != nil {
		if err := s.otpRepository.Delete(isOTPExist.Email); err != nil {
			logs.Error(err)
			return "Failed to delete otp", errs.NewInternalServerError()
		}
	}

	otpString, err := helpers.NewOTP()
	if err != nil {
		logs.Error(err)
		return "Failed to generate otp", errs.NewUnexpectedError()
	}
	otpHash, err := helpers.HashSecret(otpString)
	if err != nil {
		logs.Error(err)
		return "Failed to hash otp", errs.NewUnexpectedError()
	}

	createOTP := otp.OTP{
		Email:     email,
		OTPHash:   otpHash,
		ExpiredAt: time.Now().Add(time.Minute * 10),
	}
	err = s.otpRepository.Create(&createOTP)
	if err != nil {
		logs.Error(err)
		return "Failed to create otp", errs.NewInternalServerError()
	}

	result := ""
	err = s.emailRepository.Send(email, otpString)
	if err != nil {
		logs.Error(err)
		if err := s.otpRepository.Delete(email); err != nil {
			logs.Error(err)
		}

		result = fmt.Sprintf("Cannot send email to %s", email)
		return result, errs.NewInternalServerError()
	}
	result = fmt.Sprintf("Send email to %s successfully", email)
	return result, nil
}
