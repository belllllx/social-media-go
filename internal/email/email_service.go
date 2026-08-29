package email

import (
	"context"
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/otp"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"gorm.io/gorm"
)

type EmailService interface {
	SendEmail(
		ctx context.Context,
		db *gorm.DB,
		email,
		sendEmailType string,
	) error
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

func (s *emailService) SendEmail(
	ctx context.Context,
	db *gorm.DB,
	email,
	sendEmailType string,
) error {
	isOTPExist, err := s.otpRepository.FindByEmail(
		ctx,
		db,
		email,
	)
	if err != nil {
		if helpers.IsErrContextCanceled(err) {
			logs.Warn(err)
			return err
		}

		if !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to send email")
		}
	}

	// ลบ otp ของเก่าถ้าเจอ
	if isOTPExist != nil {
		err = s.otpRepository.Delete(
			ctx,
			db,
			isOTPExist.Email,
		)
		if err != nil {
			if helpers.IsErrContextCanceled(err) {
				logs.Warn(err)
				return err
			}

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

	createOTP := models.OTP{
		Email:     email,
		OTPHash:   otpHash,
		ExpiredAt: time.Now().Add(time.Minute * 10),
	}
	err = s.otpRepository.Create(
		ctx,
		db,
		&createOTP,
	)
	if err != nil {
		if helpers.IsErrContextCanceled(err) {
			logs.Warn(err)
			return err
		}

		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create otp")
	}

	err = s.emailRepository.Send(
		ctx,
		email,
		otpString,
		sendEmailType,
	)
	if err != nil {
		if helpers.IsErrContextCanceled(err) {
			logs.Warn(err)
			return err
		}

		logs.Error(err)

		err = s.otpRepository.Delete(
			ctx,
			db,
			email,
		)
		if err != nil {
			if helpers.IsErrContextCanceled(err) {
				logs.Warn(err)
				return err
			}

			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage(fmt.Sprintf("Cannot send email to %s and delete otp", email))
		}

		return errs.NewInternalServerErrorWithMessage(fmt.Sprintf("Cannot send email to %s", email))
	}

	return nil
}
