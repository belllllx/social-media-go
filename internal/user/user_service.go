package user

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
)

type SecureUser struct {
	ID                   uuid.UUID    `json:"id"`
	Fullname             string       `json:"fullname"`
	Username             *string      `json:"username"`
	Email                string       `json:"email"`
	DateOfBirth          *time.Time   `json:"dateOfBirth"`
	ProfileUrl           *string      `json:"profileUrl"`
	ProfileBackgroundUrl *string      `json:"profileBackgroundUrl"`
	Info                 *string      `json:"info"`
	Role                 Role         `json:"role"`
	ProviderType         ProviderType `json:"providerType"`
	CreatedAt            time.Time    `json:"createdAt"`
	UpdatedAt            time.Time    `json:"updatedAt"`
}

type UserService interface {
	SecureFindWithID(ID uuid.UUID) (*SecureUser, error)
	ResetPassword(email, password string) error
	GetPostUserImage(post *Post) error
	GetNotificationUserImage(notification *Notification) error
}

type userService struct {
	presignClient  *s3.PresignClient
	userRepository UserRepository
}

func NewUserService(presignClient *s3.PresignClient, userRepository UserRepository) UserService {
	return &userService{
		presignClient:  presignClient,
		userRepository: userRepository,
	}
}

func (s *userService) SecureFindWithID(ID uuid.UUID) (*SecureUser, error) {
	user, err := s.userRepository.FindByID(ID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerError()
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundError()
	}

	secureUser := &SecureUser{
		ID:                   user.ID,
		Fullname:             user.Fullname,
		Username:             user.Username,
		Email:                user.Email,
		DateOfBirth:          user.DateOfBirth,
		ProfileUrl:           user.ProfileUrl,
		ProfileBackgroundUrl: user.ProfileBackgroundUrl,
		Info:                 user.Info,
		Role:                 user.Role,
		ProviderType:         user.ProviderType,
		CreatedAt:            user.CreatedAt,
		UpdatedAt:            user.UpdatedAt,
	}
	return secureUser, nil
}

func (s *userService) ResetPassword(email, password string) error {
	passwordHash, err := helpers.HashSecret(password)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to hash password")
	}

	err = s.userRepository.UpdatePassword(email, passwordHash)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to update user password")
	}

	return nil
}

func (s *userService) GetPostUserImage(post *Post) error {
	ctx := context.Background()

	// ไม่ใช่ avater ของ social login
	// อัพเดต profile url
	if post.User.ProfileUrl != nil && !helpers.IsExternalURL(*post.User.ProfileUrl) {
		req, err := helpers.PresignGetObject(s.presignClient, ctx, *post.User.ProfileUrl)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to presign get post user profile url object")
		}

		post.User.ProfileUrl = &req.URL
	}

	return nil
}

func (s *userService) GetNotificationUserImage(notification *Notification) error {
	ctx := context.Background()

	// ไม่ใช่ avater ของ social login
	// อัพเดต profile url
	if notification.Sender.ProfileUrl != nil && !helpers.IsExternalURL(*notification.Sender.ProfileUrl) {
		req, err := helpers.PresignGetObject(s.presignClient, ctx, *notification.Sender.ProfileUrl)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to presign get notification sender profile url object")
		}

		notification.Sender.ProfileUrl = &req.URL
	}

	return nil
}
