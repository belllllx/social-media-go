package notification

import (
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/google/uuid"
)

type CreateNotificationsDTO struct {
	Type      user.NotificationType
	Message   string
	SenderID  uuid.UUID
	PostID    *uuid.UUID
	CommentID *uuid.UUID
}

type NotificationService interface {
	CreateNotifications(createNotificationsDTO *CreateNotificationsDTO) ([]user.Notification, error)
}

type notificationService struct {
	userRepository         user.UserRepository
	notificationRepository NotificationRepository
	userService            user.UserService
}

func NewNotificationService(
	userRepository user.UserRepository,
	notificationRepository NotificationRepository,
	userService user.UserService,
) NotificationService {
	return &notificationService{
		userRepository:         userRepository,
		notificationRepository: notificationRepository,
		userService:            userService,
	}
}

func (s *notificationService) CreateNotifications(createNotificationsDTO *CreateNotificationsDTO) ([]user.Notification, error) {
	usersExcept, err := s.userRepository.FindByIDExcept(createNotificationsDTO.SenderID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id except")
	}

	// ถ้าไม่มี users ก็ไม่ต้องสร้าง notifications
	if len(usersExcept) == 0 {
		return nil, nil
	}

	createNotifications := []user.Notification{}
	for _, userExcept := range usersExcept {
		createNotifications = append(createNotifications, user.Notification{
			Type:       createNotificationsDTO.Type,
			Message:    createNotificationsDTO.Message,
			SenderID:   createNotificationsDTO.SenderID,
			ReceiverID: userExcept.ID,
			PostID:     createNotificationsDTO.PostID,
			CommentID:  createNotificationsDTO.CommentID,
		})
	}
	err = s.notificationRepository.CreateMany(createNotifications)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create notificaions")
	}

	notificationsID := []uuid.UUID{}
	for _, createNotification := range createNotifications {
		notificationsID = append(notificationsID, createNotification.ID)
	}
	notifications, err := s.notificationRepository.PreloadsRelation(notificationsID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find notifications with relation")
	}

	for i := range notifications {
		err = s.userService.GetNotificationUserImage(&notifications[i])
		if err != nil {
			return nil, err
		}
	}

	return notifications, nil
}
