package email

import (
	"context"

	"github.com/belllllx/social-media-go/pkg/helpers"
)

type EmailRepository interface {
	Send(
		ctx context.Context,
		email,
		otp,
		verifyEmailType string,
	) error
}

type emailRepositoryImpl struct {
}

func NewEmailRepositoryImpl() EmailRepository {
	return &emailRepositoryImpl{}
}

func (r *emailRepositoryImpl) Send(
	ctx context.Context,
	email,
	otp,
	verifyEmailType string,
) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	err = helpers.SendEmail(
		email,
		otp,
		verifyEmailType,
	)
	if err != nil {
		return err
	}

	return nil
}
