package user

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowerData struct {
	ID          int64     `json:"id"`
	FollowerID  uuid.UUID `json:"followerId"`
	FollowingID uuid.UUID `json:"followingId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SecureUserFollow struct {
	ID                   uuid.UUID           `json:"id"`
	Fullname             string              `json:"fullname"`
	Username             *string             `json:"username"`
	Email                string              `json:"email"`
	DateOfBirth          *time.Time          `json:"dateOfBirth"`
	ProfileUrl           *string             `json:"profileUrl"`
	ProfileBackgroundUrl *string             `json:"profileBackgroundUrl"`
	Info                 *string             `json:"info"`
	Role                 models.Role         `json:"role"`
	ProviderType         models.ProviderType `json:"providerType"`
	Followers            []FollowerData      `json:"followers"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type Following struct {
	ID            int64             `json:"id"`
	FollowerID    uuid.UUID         `json:"followerId"`
	FollowingID   uuid.UUID         `json:"followingId"`
	FollowingUser *SecureUserFollow `json:"following"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type Follower struct {
	ID           int64             `json:"id"`
	FollowerID   uuid.UUID         `json:"followerId"`
	FollowingID  uuid.UUID         `json:"followingId"`
	FollowerUser *SecureUserFollow `json:"follower"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type SecureUser struct {
	ID                   uuid.UUID           `json:"id"`
	Fullname             string              `json:"fullname"`
	Username             *string             `json:"username"`
	Email                string              `json:"email"`
	DateOfBirth          *time.Time          `json:"dateOfBirth"`
	ProfileUrl           *string             `json:"profileUrl"`
	ProfileBackgroundUrl *string             `json:"profileBackgroundUrl"`
	Info                 *string             `json:"info"`
	Role                 models.Role         `json:"role"`
	ProviderType         models.ProviderType `json:"providerType"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type SecureUserWithFollowRelations struct {
	ID                   uuid.UUID           `json:"id"`
	Fullname             string              `json:"fullname"`
	Username             *string             `json:"username"`
	Email                string              `json:"email"`
	DateOfBirth          *time.Time          `json:"dateOfBirth"`
	ProfileUrl           *string             `json:"profileUrl"`
	ProfileBackgroundUrl *string             `json:"profileBackgroundUrl"`
	Info                 *string             `json:"info"`
	Role                 models.Role         `json:"role"`
	ProviderType         models.ProviderType `json:"providerType"`
	Followings           []Following         `json:"followings,omitempty"`
	Followers            []Follower          `json:"followers,omitempty"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type UserService interface {
	FindByIDWithFollowRelations(ctx context.Context, userID uuid.UUID) (*SecureUserWithFollowRelations, error)
	ResetPassword(
		ctx context.Context,
		email,
		password string,
	) error
	GetUserImage(ctx context.Context, user *models.User) error
}

type userService struct {
	db             *gorm.DB
	presignClient  *s3.PresignClient
	userRepository UserRepository
}

func NewUserService(
	db *gorm.DB,
	presignClient *s3.PresignClient,
	userRepository UserRepository,
) UserService {
	return &userService{
		db:             db,
		presignClient:  presignClient,
		userRepository: userRepository,
	}
}

func (s *userService) FindByIDWithFollowRelations(ctx context.Context, userID uuid.UUID) (*SecureUserWithFollowRelations, error) {
	user, err := s.userRepository.FindByIDWithFollowRelations(
		ctx,
		s.db,
		userID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id with follow relations")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", userID))
	}

	err = s.GetUserImage(ctx, user)
	if err != nil {
		return nil, err
	}

	followings := []Following{}
	for _, following := range user.Followings {
		err = s.GetUserImage(ctx, &following.Following)
		if err != nil {
			return nil, err
		}

		followerData := []FollowerData{}
		for _, follower := range following.Following.Followers {
			followerData = append(followerData, FollowerData{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		secureUserFollow := &SecureUserFollow{
			ID:                   following.FollowingID,
			Fullname:             following.Following.Fullname,
			Username:             following.Following.Username,
			Email:                following.Following.Email,
			DateOfBirth:          following.Following.DateOfBirth,
			ProfileUrl:           following.Following.ProfileUrl,
			ProfileBackgroundUrl: following.Following.ProfileBackgroundUrl,
			Info:                 following.Following.Info,
			Role:                 following.Following.Role,
			ProviderType:         following.Following.ProviderType,
			Followers:            followerData,
			CreatedAt:            following.Following.CreatedAt,
			UpdatedAt:            following.Following.UpdatedAt,
		}
		followings = append(followings, Following{
			ID:            following.ID,
			FollowerID:    following.FollowerID,
			FollowingID:   following.FollowingID,
			FollowingUser: secureUserFollow,
			CreatedAt:     following.CreatedAt,
			UpdatedAt:     following.UpdatedAt,
		})
	}

	followers := []Follower{}
	for _, follower := range user.Followers {
		err = s.GetUserImage(ctx, &follower.Follower)
		if err != nil {
			return nil, err
		}

		followerData := []FollowerData{}
		for _, follower := range follower.Follower.Followers {
			followerData = append(followerData, FollowerData{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		secureUserFollow := &SecureUserFollow{
			ID:                   follower.FollowerID,
			Fullname:             follower.Follower.Fullname,
			Username:             follower.Follower.Username,
			Email:                follower.Follower.Email,
			DateOfBirth:          follower.Follower.DateOfBirth,
			ProfileUrl:           follower.Follower.ProfileUrl,
			ProfileBackgroundUrl: follower.Follower.ProfileBackgroundUrl,
			Info:                 follower.Follower.Info,
			Role:                 follower.Follower.Role,
			ProviderType:         follower.Follower.ProviderType,
			Followers:            followerData,
			CreatedAt:            follower.Follower.CreatedAt,
			UpdatedAt:            follower.Follower.UpdatedAt,
		}
		followers = append(followers, Follower{
			ID:           follower.ID,
			FollowerID:   follower.FollowerID,
			FollowingID:  follower.FollowingID,
			FollowerUser: secureUserFollow,
			CreatedAt:    follower.CreatedAt,
			UpdatedAt:    follower.UpdatedAt,
		})
	}

	secureUserWithFollowRelations := &SecureUserWithFollowRelations{
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
		Followings:           followings,
		Followers:            followers,
		CreatedAt:            user.CreatedAt,
		UpdatedAt:            user.UpdatedAt,
	}
	return secureUserWithFollowRelations, nil
}

func (s *userService) ResetPassword(
	ctx context.Context,
	email,
	password string,
) error {
	passwordHash, err := helpers.HashSecret(password)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to hash password")
	}

	err = s.userRepository.UpdatePassword(
		ctx,
		s.db,
		email,
		passwordHash,
	)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to update user password")
	}

	return nil
}

func (s *userService) GetUserImage(ctx context.Context, user *models.User) error {
	// ไม่ใช่ avater ของ social login
	// อัพเดต profile url
	if user.ProfileUrl != nil && !helpers.IsExternalURL(*user.ProfileUrl) {
		req, err := helpers.PresignGetObject(
			ctx,
			s.presignClient,
			*user.ProfileUrl,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to presign get user profile url object")
		}

		user.ProfileUrl = &req.URL
	}

	return nil
}
