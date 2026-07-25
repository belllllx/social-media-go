package email

import (
	"github.com/belllllx/social-media-go/pkg/helpers"
)

type EmailRepository interface {
	Send(email, otp, verifyEmailType string) error
}

type emailRepositoryImpl struct {
}

func NewEmailRepositoryImpl() EmailRepository {
	return &emailRepositoryImpl{}
}

func (r *emailRepositoryImpl) Send(
	email,
	otp,
	verifyEmailType string,
) error {
	err := helpers.SendEmail(email, otp, verifyEmailType)
	if err != nil {
		return err
	}

	return nil
}
