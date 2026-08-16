package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FindsWithFullnameCursorPaginationDTO struct {
	UserID   uuid.UUID
	Fullname string
	Cursor   string
	Limit    string
}

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

type UserCursorPagination struct {
	Users      []SecureUser `json:"users"`
	NextCursor *uuid.UUID   `json:"nextCursor"`
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
	Followings           []Following         `json:"followings"`
	Followers            []Follower          `json:"followers"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type UserService interface {
	FindByID(ctx context.Context, userID string) (*SecureUserWithFollowRelations, error)
	FindByIDWithFollowRelations(ctx context.Context, userID uuid.UUID) (*SecureUserWithFollowRelations, error)
	FindsWithFullnameCursorPagination(
		ctx context.Context,
		findsWithFullnameCursorPaginationDTO *FindsWithFullnameCursorPaginationDTO,
	) (*UserCursorPagination, error)
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

func (s *userService) FindByID(ctx context.Context, userID string) (*SecureUserWithFollowRelations, error) {
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

	return s.FindByIDWithFollowRelations(ctx, *userIDParse)
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

func (s *userService) FindsWithFullnameCursorPagination(
	ctx context.Context,
	findsWithFullnameCursorPaginationDTO *FindsWithFullnameCursorPaginationDTO,
) (*UserCursorPagination, error) {
	var nextCursor *uuid.UUID
	var cursorID *uuid.UUID

	if findsWithFullnameCursorPaginationDTO.Cursor != "" {
		err := helpers.ValidateUUID(findsWithFullnameCursorPaginationDTO.Cursor)
		if err != nil {
			logs.Warn(err)
			return nil, err
		}

		cursorID, err = helpers.ParseUUID(findsWithFullnameCursorPaginationDTO.Cursor)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
	}

	limitInt, err := strconv.Atoi(findsWithFullnameCursorPaginationDTO.Limit)
	if err != nil {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	users := []models.User{}
	usersCursorPagination := []SecureUser{}
	if findsWithFullnameCursorPaginationDTO.Cursor == "" {
		users, err = s.userRepository.FindsByFullnameCursorPagination(
			ctx,
			s.db,
			findsWithFullnameCursorPaginationDTO.UserID,
			findsWithFullnameCursorPaginationDTO.Fullname,
			nil,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find users by fullname cursor pagination")
		}
	} else {
		cursor, err := s.userRepository.FindByIDCursor(
			ctx,
			s.db,
			findsWithFullnameCursorPaginationDTO.UserID,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find user cursor by user id")
		}

		if helpers.IsErrRecordNotFound(err) {
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by user id %v is not found", *cursorID))
		}

		users, err = s.userRepository.FindsByFullnameCursorPagination(
			ctx,
			s.db,
			findsWithFullnameCursorPaginationDTO.UserID,
			findsWithFullnameCursorPaginationDTO.Fullname,
			cursor,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find users by fullname cursor pagination")
		}
	}

	for _, user := range users {
		err = s.GetUserImage(ctx, &user)
		if err != nil {
			return nil, err
		}

		usersCursorPagination = append(usersCursorPagination, SecureUser{
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
		})
	}

	if len(usersCursorPagination) == limitInt {
		nextCursor = &usersCursorPagination[len(usersCursorPagination)-1].ID
	}
	userCursorPagination := &UserCursorPagination{
		Users:      usersCursorPagination,
		NextCursor: nextCursor,
	}
	return userCursorPagination, nil
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
