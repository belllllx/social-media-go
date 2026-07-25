package notification

import (
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"gorm.io/gorm"
)

type NotificationService interface {
	CreateNotification(tx *gorm.DB, createNotificationDTO *models.Notification) error
	CreateNotifications(tx *gorm.DB, createNotificationsDTO []models.Notification) error
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

func (s *notificationService) CreateNotification(tx *gorm.DB, createNotificationDTO *models.Notification) error {
	err := s.notificationRepository.Create(tx, createNotificationDTO)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create notification")
	}

	return nil
}

func (s *notificationService) CreateNotifications(tx *gorm.DB, createNotificationsDTO []models.Notification) error {
	err := s.notificationRepository.CreateMany(tx, createNotificationsDTO)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to create notificaions")
	}

	return nil
}
