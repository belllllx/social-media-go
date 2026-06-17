package email

import (
	"github.com/belllllx/social-media-go/pkg/helpers"
)

type EmailRepository interface {
	Send(email, otp string) error
}

type emailRepositoryImpl struct {
}

func NewEmailRepositoryImpl() EmailRepository {
	return &emailRepositoryImpl{}
}

func (r *emailRepositoryImpl) Send(email, otp string) error {
	err := helpers.SendEmail(email, otp, "register")
	if err != nil {
		return err
	}

	return nil
}
