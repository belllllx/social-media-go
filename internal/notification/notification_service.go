package notification

import (
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/google/uuid"
)

type NotificationService interface {
	CreateNotification(createNotificationDTO *user.Notification) (*user.Notification, error)
	CreateNotifications(createNotificationsDTO []user.Notification) ([]user.Notification, error)
}

type notificationService struct {
	notificationRepository NotificationRepository
	userService            user.UserService
}

func NewNotificationService(
	notificationRepository NotificationRepository,
	userService user.UserService,
) NotificationService {
	return &notificationService{
		notificationRepository: notificationRepository,
		userService:            userService,
	}
}

func (s *notificationService) CreateNotification(createNotificationDTO *user.Notification) (*user.Notification, error) {
	err := s.notificationRepository.Create(createNotificationDTO)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create notification")
	}

	notification, err := s.notificationRepository.PreloadRelation(createNotificationDTO.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification with relation")
	}

	err = s.userService.GetNotificationUserImage(notification)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *notificationService) CreateNotifications(createNotificationsDTO []user.Notification) ([]user.Notification, error) {
	err := s.notificationRepository.CreateMany(createNotificationsDTO)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create notificaions")
	}

	notificationsID := []uuid.UUID{}
	for _, createNotificationDTO := range createNotificationsDTO {
		notificationsID = append(notificationsID, createNotificationDTO.ID)
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
