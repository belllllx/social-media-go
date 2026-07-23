package email

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"gorm.io/gorm"
)

type EmailService interface {
	SendEmail(db *gorm.DB, email, sendEmailType string) error
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

func (s *emailService) SendEmail(db *gorm.DB, email, sendEmailType string) error {
	isOTPExist, err := s.otpRepository.FindByEmail(db, email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to send email")
	}

	// ลบ otp ของเก่าถ้าเจอ
	if isOTPExist != nil {
		if err := s.otpRepository.Delete(db, isOTPExist.Email); err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete otp")
		}
	}

	otpString, err := helpers.NewOTP()
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to generate otp")
	}
	otpHash, err := helpers.HashSecret(otpString)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to hash otp")
	}

	createOTP := otp.OTP{
		Email:     email,
		OTPHash:   otpHash,
		ExpiredAt: time.Now().Add(time.Minute * 10),
	}
	err = s.otpRepository.Create(db, &createOTP)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create otp")
	}

	err = s.emailRepository.Send(email, otpString, sendEmailType)
	if err != nil {
		logs.Error(err)
		if err := s.otpRepository.Delete(db, email); err != nil {
			logs.Error(err)
		}

		return errs.NewInternalServerErrorWithMessage(fmt.Sprintf("Cannot send email to %s", email))
	}

	return nil
}
