package otp

import (
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
)

type OTPService interface {
	Verify(email, otp string) (result string, err error)
}

type otpService struct {
	otpRepository OTPRepository
}

func NewOTPService(otpRepository OTPRepository) OTPService {
	return &otpService{otpRepository: otpRepository}
}

func (s *otpService) Verify(email, otp string) (string, error) {
	isOTPExist, err := s.otpRepository.FindNotExpired(email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to verify otp", errs.NewInternalServerError()
	}

	// กรณี otp หมดอายุ
	if helpers.IsErrRecordNotFound(err) {
		if err := s.otpRepository.Delete(email); err != nil {
			logs.Error(err)
			return "Failed to delete expired otp", errs.NewInternalServerError()
		}
		return "OTP has expired", errs.NewBadRequestError()
	}

	err = helpers.CompareSecret(isOTPExist.OTPHash, otp)
	// กรณี otp ไม่ถูกต้อง
	if err != nil {
		return "Invalid otp", errs.NewBadRequestError()
	}

	return "Verify otp successfully", nil
}
