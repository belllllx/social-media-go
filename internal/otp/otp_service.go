package otp

import (
	"context"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"gorm.io/gorm"
)

type OTPService interface {
	Verify(
		ctx context.Context,
		db *gorm.DB,
		email,
		otp string,
	) error
	DeleteWithExpired(ctx context.Context, db *gorm.DB) error
}

type otpService struct {
	otpRepository OTPRepository
}

func NewOTPService(otpRepository OTPRepository) OTPService {
	return &otpService{
		otpRepository: otpRepository,
	}
}

func (s *otpService) Verify(
	ctx context.Context,
	db *gorm.DB,
	email,
	otp string,
) error {
	isOTPExist, err := s.otpRepository.FindNotExpired(
		ctx,
		db,
		email,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to verify otp")
	}

	// กรณี otp หมดอายุ
	if helpers.IsErrRecordNotFound(err) {
		if err := s.otpRepository.Delete(
			ctx,
			db,
			email,
		); err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete expired otp")
		}
		return errs.NewBadRequestErrorWithMessage("OTP has expired")
	}

	err = helpers.CompareSecret(isOTPExist.OTPHash, otp)
	// กรณี otp ไม่ถูกต้อง
	if err != nil {
		return errs.NewBadRequestErrorWithMessage("Invalid otp")
	}

	return nil
}

func (s *otpService) DeleteWithExpired(ctx context.Context, db *gorm.DB) error {
	err := s.otpRepository.DeleteByExpired(ctx, db)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to delete otp has expired")
	}
	return nil
}
