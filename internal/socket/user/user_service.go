package user

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type activeUser struct {
	SocketID   string    `json:"-"`
	ID         uuid.UUID `json:"id"`
	Fullname   string    `json:"fullname"`
	Email      string    `json:"email"`
	ProfileURL *string   `json:"profileUrl"`
	Active     bool      `json:"active"`
}

var activeUsers []activeUser

type UserFinder interface {
	FindByID(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) (*models.User, error)
}

type UserSocketService interface {
	Connected(
		ctx context.Context,
		socketID,
		userID string,
	) ([]activeUser, error)
	Disconnected(ctx context.Context, socketID string) ([]activeUser, error)
}

type userSocketService struct {
	db             *gorm.DB
	presignClient  *s3.PresignClient
	userRepository UserFinder
}

func NewUserSocketService(
	db *gorm.DB,
	presignClient *s3.PresignClient,
	userRepository UserFinder,
) UserSocketService {
	return &userSocketService{
		db:             db,
		presignClient:  presignClient,
		userRepository: userRepository,
	}
}

func (s *userSocketService) Connected(
	ctx context.Context,
	socketID,
	userID string,
) ([]activeUser, error) {
	err := helpers.ValidateUUID(userID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	userIDParse, err := helpers.ParseUUID(userID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	found := false
	for _, activeUser := range activeUsers {
		if activeUser.ID == *userIDParse {
			found = true
		}
	}

	if !found {
		userByID, err := s.userRepository.FindByID(
			ctx,
			s.db,
			*userIDParse,
		)
		if err != nil {
			if helpers.IsErrContextCanceled(err) {
				logs.Warn(err)
				return nil, err
			}

			if !helpers.IsErrRecordNotFound(err) {
				logs.Error(err)
				return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id")
			}

			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", *userIDParse))
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			userByID,
		)
		if err != nil {
			if helpers.IsErrContextCanceled(err) {
				logs.Warn(err)
				return nil, err
			}

			logs.Error(err)
			return nil, err
		}

		activeUsers = append(activeUsers, activeUser{
			SocketID:   socketID,
			ID:         userByID.ID,
			Fullname:   userByID.Fullname,
			Email:      userByID.Email,
			ProfileURL: userByID.ProfileURL,
			Active:     true,
		})
	}

	return activeUsers, nil
}

func (s *userSocketService) Disconnected(ctx context.Context, socketID string) ([]activeUser, error) {
	err := ctx.Err()
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	return nil, nil
}
