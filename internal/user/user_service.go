package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/dto"
	"github.com/belllllx/social-media-go/internal/follow"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/socket"
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

type Follow struct {
	ID          int64             `json:"id"`
	FollowerID  uuid.UUID         `json:"followerId"`
	Follower    *SecureUserFollow `json:"follower,omitempty"`
	FollowingID uuid.UUID         `json:"followingId"`
	Following   *SecureUserFollow `json:"following,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitzero"`
	UpdatedAt   time.Time         `json:"updatedAt,omitzero"`
}

type Follower struct {
	ID           int64             `json:"id"`
	FollowerID   uuid.UUID         `json:"followerId"`
	FollowingID  uuid.UUID         `json:"followingId"`
	FollowerUser *SecureUserFollow `json:"follower,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type Following struct {
	ID            int64             `json:"id"`
	FollowerID    uuid.UUID         `json:"followerId"`
	FollowingID   uuid.UUID         `json:"followingId"`
	FollowingUser *SecureUserFollow `json:"following"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
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

type SecureUserWithFollowerRelation struct {
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
	Followers            []Follower          `json:"followers"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type UserCursorPagination struct {
	Users      []dto.SecureUser `json:"users"`
	NextCursor *uuid.UUID       `json:"nextCursor"`
}

type UserWithFollowerRelationCursorPagination struct {
	Users      []SecureUserWithFollowerRelation `json:"users"`
	NextCursor *uuid.UUID                       `json:"nextCursor"`
}

type UserService interface {
	FindByIDWithFollowingRelation(ctx context.Context, userID uuid.UUID) (*dto.SecureUserWithFollowingRelation, error)
	FindByIDWithFollowRelations(ctx context.Context, userID string) (*SecureUserWithFollowRelations, error)
	FindsWithFullnameCursorPagination(
		ctx context.Context,
		findsWithFullnameCursorPaginationDTO *FindsWithFullnameCursorPaginationDTO,
	) (*UserCursorPagination, error)
	FindsCursorPaginationWithFollowerRelation(
		ctx context.Context,
		userID uuid.UUID,
		cursor,
		limit string,
	) (*UserWithFollowerRelationCursorPagination, error)
	ResetPassword(
		ctx context.Context,
		email,
		password string,
	) error
	ToggleFollow(
		ctx context.Context,
		followerID uuid.UUID,
		followingID string,
	) (string, *Follow, error)
}

type userService struct {
	db                     *gorm.DB
	presignClient          *s3.PresignClient
	userRepository         UserRepository
	followRepository       follow.FollowRepository
	notificationRepository notification.NotificationRepository
	notificationService    notification.NotificationService
	userSocket             socket.UserSocket
	notificationSocket     socket.NotificationSocket
}

func NewUserService(
	db *gorm.DB,
	presignClient *s3.PresignClient,
	userRepository UserRepository,
	followRepository follow.FollowRepository,
	notificationRepository notification.NotificationRepository,
	notificationService notification.NotificationService,
	userSocket socket.UserSocket,
	notificationSocket socket.NotificationSocket,
) UserService {
	return &userService{
		db:                     db,
		presignClient:          presignClient,
		userRepository:         userRepository,
		followRepository:       followRepository,
		notificationRepository: notificationRepository,
		notificationService:    notificationService,
		userSocket:             userSocket,
		notificationSocket:     notificationSocket,
	}
}

func (s *userService) FindByIDWithFollowingRelation(ctx context.Context, userID uuid.UUID) (*dto.SecureUserWithFollowingRelation, error) {
	user, err := s.userRepository.FindByIDWithFollowingRelation(
		ctx,
		s.db,
		userID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id with following relation")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", userID))
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		user,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	followings := []dto.Following{}
	for _, following := range user.Followings {
		followings = append(followings, dto.Following{
			ID:          following.ID,
			FollowerID:  following.FollowerID,
			FollowingID: following.FollowingID,
			CreatedAt:   following.CreatedAt,
			UpdatedAt:   following.UpdatedAt,
		})
	}

	secureUserWithFollowingRelation := &dto.SecureUserWithFollowingRelation{
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
		CreatedAt:            user.CreatedAt,
		UpdatedAt:            user.UpdatedAt,
	}
	return secureUserWithFollowingRelation, nil
}

func (s *userService) FindByIDWithFollowRelations(ctx context.Context, userID string) (*SecureUserWithFollowRelations, error) {
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

	user, err := s.userRepository.FindByIDWithFollowRelations(
		ctx,
		s.db,
		*userIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id with follow relations")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", userID))
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		user,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	followings := []Following{}
	for _, following := range user.Followings {
		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&following.Following,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		followersData := []FollowerData{}
		for _, follower := range following.Following.Followers {
			followersData = append(followersData, FollowerData{
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
			Followers:            followersData,
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
		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&follower.Follower,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		followersData := []FollowerData{}
		for _, follower := range follower.Follower.Followers {
			followersData = append(followersData, FollowerData{
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
			Followers:            followersData,
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
		logs.Warn(err)
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	users := []models.User{}
	usersCursorPagination := []dto.SecureUser{}
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
			logs.Warn(err)
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
		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&user,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		usersCursorPagination = append(usersCursorPagination, dto.SecureUser{
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

func (s *userService) FindsCursorPaginationWithFollowerRelation(
	ctx context.Context,
	userID uuid.UUID,
	cursor,
	limit string,
) (*UserWithFollowerRelationCursorPagination, error) {
	var nextCursor *uuid.UUID
	var cursorID *uuid.UUID

	if cursor != "" {
		err := helpers.ValidateUUID(cursor)
		if err != nil {
			logs.Warn(err)
			return nil, err
		}

		cursorID, err = helpers.ParseUUID(cursor)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		logs.Warn(err)
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	users := []models.User{}
	usersWithFollowerRelationCursorPagination := []SecureUserWithFollowerRelation{}
	if cursor == "" {
		users, err = s.userRepository.FindsCursorPaginationWithFollowerRelation(
			ctx,
			s.db,
			userID,
			nil,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find users cursor pagination with follower relation")
		}
	} else {
		cursor, err := s.userRepository.FindByIDCursor(
			ctx,
			s.db,
			*cursorID,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find user cursor by user id")
		}

		if helpers.IsErrRecordNotFound(err) {
			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by user id %v is not found", *cursorID))
		}

		users, err = s.userRepository.FindsCursorPaginationWithFollowerRelation(
			ctx,
			s.db,
			userID,
			cursor,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find users cursor pagination with follower relation")
		}
	}

	for _, user := range users {
		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&user,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		followers := []Follower{}
		for _, follower := range user.Followers {
			followers = append(followers, Follower{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		usersWithFollowerRelationCursorPagination = append(usersWithFollowerRelationCursorPagination, SecureUserWithFollowerRelation{
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
			Followers:            followers,
			CreatedAt:            user.CreatedAt,
			UpdatedAt:            user.UpdatedAt,
		})
	}

	if len(usersWithFollowerRelationCursorPagination) == limitInt {
		nextCursor = &usersWithFollowerRelationCursorPagination[len(usersWithFollowerRelationCursorPagination)-1].ID
	}
	userWithFollowerRelationCursorPagination := &UserWithFollowerRelationCursorPagination{
		Users:      usersWithFollowerRelationCursorPagination,
		NextCursor: nextCursor,
	}
	return userWithFollowerRelationCursorPagination, nil
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

func (s *userService) ToggleFollow(
	ctx context.Context,
	followerID uuid.UUID,
	followingID string,
) (string, *Follow, error) {
	err := helpers.ValidateUUID(followingID)
	if err != nil {
		logs.Warn(err)
		return "", nil, err
	}

	followingIDParse, err := helpers.ParseUUID(followingID)
	if err != nil {
		logs.Error(err)
		return "", nil, err
	}

	if followerID == *followingIDParse {
		return "", nil, errs.NewBadRequestErrorWithMessage("You can only follow other users")
	}

	userByID, err := s.userRepository.FindByID(
		ctx,
		s.db,
		*followingIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", *followingIDParse))
	}

	_, err = s.followRepository.FindIsFollowing(
		ctx,
		s.db,
		followerID,
		userByID.ID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find is following")
	}

	// กรณี follow
	if helpers.IsErrRecordNotFound(err) {
		createFollow := &models.Follow{
			FollowerID:  followerID,
			FollowingID: userByID.ID,
		}
		createNotificationDTO := &models.Notification{
			Type:       models.NotificationTypeFollow,
			Message:    "Following you",
			SenderID:   followerID,
			ReceiverID: userByID.ID,
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			err = s.followRepository.Create(
				ctx,
				tx,
				createFollow,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to create follow")
			}

			err = s.notificationService.CreateNotification(
				ctx,
				tx,
				createNotificationDTO,
			)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			_, ok := err.(*errs.AppError)
			if !ok {
				logs.Error(err)
			}
			return "", nil, err
		}

		createdFollow, err := s.followRepository.FindByIDWithFollowingAndFollowerRelations(
			ctx,
			s.db,
			createFollow.ID,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find follow by id with following and follower relations")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&createdFollow.Follower,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, err
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&createdFollow.Following,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, err
		}

		createdNotification, err := s.notificationRepository.FindByIDWithSenderRelation(
			ctx,
			s.db,
			createNotificationDTO.ID,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find notification by id with sender relation")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&createdNotification.Sender,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, err
		}

		followersOfFollower := []socket.FollowerDataDTO{}
		for _, follower := range createdFollow.Follower.Followers {
			followersOfFollower = append(followersOfFollower, socket.FollowerDataDTO{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		followersOfFollowing := []socket.FollowerDataDTO{}
		for _, follower := range createdFollow.Following.Followers {
			followersOfFollowing = append(followersOfFollowing, socket.FollowerDataDTO{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		secureUserFollower := &socket.SecureUserFollowDTO{
			ID:                   createdFollow.FollowerID,
			Fullname:             createdFollow.Follower.Fullname,
			Username:             createdFollow.Follower.Username,
			Email:                createdFollow.Follower.Email,
			DateOfBirth:          createdFollow.Follower.DateOfBirth,
			ProfileUrl:           createdFollow.Follower.ProfileUrl,
			ProfileBackgroundUrl: createdFollow.Follower.ProfileBackgroundUrl,
			Info:                 createdFollow.Follower.Info,
			Role:                 createdFollow.Follower.Role,
			ProviderType:         createdFollow.Follower.ProviderType,
			Followers:            followersOfFollower,
			CreatedAt:            createdFollow.Follower.CreatedAt,
			UpdatedAt:            createdFollow.Follower.UpdatedAt,
		}

		secureUserFollowing := &socket.SecureUserFollowDTO{
			ID:                   createdFollow.FollowingID,
			Fullname:             createdFollow.Following.Fullname,
			Username:             createdFollow.Following.Username,
			Email:                createdFollow.Following.Email,
			DateOfBirth:          createdFollow.Following.DateOfBirth,
			ProfileUrl:           createdFollow.Following.ProfileUrl,
			ProfileBackgroundUrl: createdFollow.Following.ProfileBackgroundUrl,
			Info:                 createdFollow.Following.Info,
			Role:                 createdFollow.Following.Role,
			ProviderType:         createdFollow.Following.ProviderType,
			Followers:            followersOfFollowing,
			CreatedAt:            createdFollow.Following.CreatedAt,
			UpdatedAt:            createdFollow.Following.UpdatedAt,
		}

		followDTO := &socket.FollowDTO{
			ID:          createdFollow.ID,
			FollowerID:  createdFollow.FollowerID,
			Follower:    secureUserFollower,
			FollowingID: createdFollow.FollowingID,
			Following:   secureUserFollowing,
			CreatedAt:   createdFollow.CreatedAt,
			UpdatedAt:   createdFollow.UpdatedAt,
		}

		go s.userSocket.EmitToggleFollow(followDTO)

		secureUserSender := &dto.SecureUser{
			ID:                   createdNotification.SenderID,
			Fullname:             createdNotification.Sender.Fullname,
			Username:             createdNotification.Sender.Username,
			Email:                createdNotification.Sender.Email,
			DateOfBirth:          createdNotification.Sender.DateOfBirth,
			ProfileUrl:           createdNotification.Sender.ProfileUrl,
			ProfileBackgroundUrl: createdNotification.Sender.ProfileBackgroundUrl,
			Info:                 createdNotification.Sender.Info,
			Role:                 createdNotification.Sender.Role,
			ProviderType:         createdNotification.Sender.ProviderType,
			CreatedAt:            createdNotification.Sender.CreatedAt,
			UpdatedAt:            createdNotification.Sender.UpdatedAt,
		}

		emitNotificationDTO := &socket.EmitNotificationDTO{
			ID:         createdNotification.ID,
			Type:       createdNotification.Type,
			Message:    createdNotification.Message,
			IsRead:     createdNotification.IsRead,
			SenderID:   createdNotification.SenderID,
			Sender:     secureUserSender,
			ReceiverID: createdNotification.ReceiverID,
			CreatedAt:  createdNotification.CreatedAt,
			UpdatedAt:  createdNotification.UpdatedAt,
		}

		go func() {
			err := s.notificationSocket.EmitNotification(emitNotificationDTO)
			if err != nil {
				logs.Error(err)
			}
		}()

		followersOfFollowerResp := []FollowerData{}
		for _, follower := range createdFollow.Follower.Followers {
			followersOfFollowerResp = append(followersOfFollowerResp, FollowerData{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		followersOfFollowingResp := []FollowerData{}
		for _, follower := range createdFollow.Following.Followers {
			followersOfFollowingResp = append(followersOfFollowingResp, FollowerData{
				ID:          follower.ID,
				FollowerID:  follower.FollowerID,
				FollowingID: follower.FollowingID,
				CreatedAt:   follower.CreatedAt,
				UpdatedAt:   follower.UpdatedAt,
			})
		}

		secureUserFollowerResp := &SecureUserFollow{
			ID:                   createdFollow.FollowerID,
			Fullname:             createdFollow.Follower.Fullname,
			Username:             createdFollow.Follower.Username,
			Email:                createdFollow.Follower.Email,
			DateOfBirth:          createdFollow.Follower.DateOfBirth,
			ProfileUrl:           createdFollow.Follower.ProfileUrl,
			ProfileBackgroundUrl: createdFollow.Follower.ProfileBackgroundUrl,
			Info:                 createdFollow.Follower.Info,
			Role:                 createdFollow.Follower.Role,
			ProviderType:         createdFollow.Follower.ProviderType,
			Followers:            followersOfFollowerResp,
			CreatedAt:            createdFollow.Follower.CreatedAt,
			UpdatedAt:            createdFollow.Follower.UpdatedAt,
		}

		secureUserFollowingResp := &SecureUserFollow{
			ID:                   createdFollow.FollowingID,
			Fullname:             createdFollow.Following.Fullname,
			Username:             createdFollow.Following.Username,
			Email:                createdFollow.Following.Email,
			DateOfBirth:          createdFollow.Following.DateOfBirth,
			ProfileUrl:           createdFollow.Following.ProfileUrl,
			ProfileBackgroundUrl: createdFollow.Following.ProfileBackgroundUrl,
			Info:                 createdFollow.Following.Info,
			Role:                 createdFollow.Following.Role,
			ProviderType:         createdFollow.Following.ProviderType,
			Followers:            followersOfFollowingResp,
			CreatedAt:            createdFollow.Following.CreatedAt,
			UpdatedAt:            createdFollow.Following.UpdatedAt,
		}

		followResp := &Follow{
			ID:          createdFollow.ID,
			FollowerID:  createdFollow.FollowerID,
			Follower:    secureUserFollowerResp,
			FollowingID: createdFollow.FollowingID,
			Following:   secureUserFollowingResp,
			CreatedAt:   createdFollow.CreatedAt,
			UpdatedAt:   createdFollow.UpdatedAt,
		}
		return "Follow successfully", followResp, nil
	}

	// กรณี unfollow
	deletedFollow := &models.Follow{}
	deletedNotification := &models.Notification{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		deletedFollow, err = s.followRepository.DeleteOfFollow(
			ctx,
			tx,
			followerID,
			userByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete follow")
		}

		deletedNotification, err = s.notificationRepository.DeleteOfFollow(
			ctx,
			tx,
			followerID,
			userByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete notification of follow")
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return "", nil, err
	}

	followDTO := &socket.FollowDTO{
		ID:          deletedFollow.ID,
		FollowerID:  deletedFollow.FollowerID,
		FollowingID: deletedFollow.FollowingID,
	}
	go s.userSocket.EmitToggleFollow(followDTO)

	emitDeleteNotificationDTO := &socket.EmitDeleteNotificationDTO{
		ID:         deletedNotification.ID,
		ReceiverID: deletedNotification.ReceiverID,
	}
	go func() {
		err := s.notificationSocket.EmitDeleteNotification(emitDeleteNotificationDTO)
		if err != nil {
			logs.Error(err)
		}
	}()

	followResp := &Follow{
		ID:          deletedFollow.ID,
		FollowerID:  deletedFollow.FollowerID,
		FollowingID: deletedFollow.FollowingID,
	}
	return "Unfollow successfully", followResp, nil
}
