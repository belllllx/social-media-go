package notification

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID         uuid.UUID               `json:"id"`
	Type       models.NotificationType `json:"type"`
	Message    string                  `json:"message"`
	IsRead     bool                    `json:"isRead"`
	SenderID   uuid.UUID               `json:"senderId"`
	Sender     *user.SecureUser        `json:"sender"`
	ReceiverID uuid.UUID               `json:"receiverId"`
	PostID     *uuid.UUID              `json:"postId,omitempty"`
	CommentID  *uuid.UUID              `json:"commentId,omitempty"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

type NotificationCursorPagination struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *uuid.UUID     `json:"nextCursor"`
}

type NotificationService interface {
	CreateNotification(
		ctx context.Context,
		tx *gorm.DB,
		createNotificationDTO *models.Notification,
	) error
	CreateNotifications(
		ctx context.Context,
		tx *gorm.DB,
		createNotificationsDTO []models.Notification,
	) error
	FindsWithReceiverIDCursorPagination(
		ctx context.Context,
		userID uuid.UUID,
		cursor,
		limit string,
	) (*NotificationCursorPagination, error)
}

type notificationService struct {
	db                     *gorm.DB
	notificationRepository NotificationRepository
	userService            user.UserService
}

func NewNotificationService(
	db *gorm.DB,
	notificationRepository NotificationRepository,
	userService user.UserService,
) NotificationService {
	return &notificationService{
		db:                     db,
		notificationRepository: notificationRepository,
		userService:            userService,
	}
}

func (s *notificationService) CreateNotification(
	ctx context.Context,
	tx *gorm.DB,
	createNotificationDTO *models.Notification,
) error {
	err := s.notificationRepository.Create(
		ctx,
		tx,
		createNotificationDTO,
	)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create notification")
	}

	return nil
}

func (s *notificationService) CreateNotifications(
	ctx context.Context,
	tx *gorm.DB,
	createNotificationsDTO []models.Notification,
) error {
	err := s.notificationRepository.CreateMany(
		ctx,
		tx,
		createNotificationsDTO,
	)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create notificaions")
	}

	return nil
}

func (s *notificationService) FindsWithReceiverIDCursorPagination(
	ctx context.Context,
	userID uuid.UUID,
	cursor,
	limit string,
) (*NotificationCursorPagination, error) {
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
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	notifications := []models.Notification{}
	notificationsCursorPagination := []Notification{}
	if cursor == "" {
		notifications, err = s.notificationRepository.FindsByReceiverIDCursorPaginationWithSenderRelation(
			ctx,
			s.db,
			userID,
			nil,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notifications by receiver id cursor pagination with sender relation")
		}
	} else {
		notificationCursor, err := s.notificationRepository.FindByIDCursor(
			ctx,
			s.db,
			*cursorID,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification cursor by notification id")
		}

		if helpers.IsErrRecordNotFound(err) {
			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by notification id %v is not found", *cursorID))
		}

		notifications, err = s.notificationRepository.FindsByReceiverIDCursorPaginationWithSenderRelation(
			ctx,
			s.db,
			userID,
			notificationCursor,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notifications by receiver id cursor pagination with sender relation")
		}
	}

	for _, notification := range notifications {
		err = s.userService.GetUserImage(ctx, &notification.Sender)
		if err != nil {
			return nil, err
		}

		secureSenderNotification := &user.SecureUser{
			ID:                   notification.SenderID,
			Fullname:             notification.Sender.Fullname,
			Username:             notification.Sender.Username,
			Email:                notification.Sender.Email,
			DateOfBirth:          notification.Sender.DateOfBirth,
			ProfileUrl:           notification.Sender.ProfileUrl,
			ProfileBackgroundUrl: notification.Sender.ProfileBackgroundUrl,
			Info:                 notification.Sender.Info,
			Role:                 notification.Sender.Role,
			ProviderType:         notification.Sender.ProviderType,
			CreatedAt:            notification.Sender.CreatedAt,
			UpdatedAt:            notification.Sender.UpdatedAt,
		}
		notificationsCursorPagination = append(notificationsCursorPagination, Notification{
			ID:         notification.ID,
			Type:       notification.Type,
			Message:    notification.Message,
			IsRead:     notification.IsRead,
			SenderID:   notification.SenderID,
			Sender:     secureSenderNotification,
			ReceiverID: notification.ReceiverID,
			PostID:     notification.PostID,
			CommentID:  notification.CommentID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		})
	}

	if len(notificationsCursorPagination) == limitInt {
		nextCursor = &notificationsCursorPagination[len(notificationsCursorPagination)-1].ID
	}

	notificationCursorPagination := &NotificationCursorPagination{
		Notifications: notificationsCursorPagination,
		NextCursor:    nextCursor,
	}

	return notificationCursorPagination, nil
}
