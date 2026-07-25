package notification

import (
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(db *gorm.DB, notification *models.Notification) error
	CreateMany(db *gorm.DB, notifications []models.Notification) error
	PreloadRelation(db *gorm.DB, notificationID uuid.UUID) (*models.Notification, error)
	PreloadsRelation(db *gorm.DB, notificationsID []uuid.UUID) ([]models.Notification, error)
}

type notificationRepositoryDB struct {
}

func NewNotificationRepositoryDB() NotificationRepository {
	return &notificationRepositoryDB{}
}

func (r *notificationRepositoryDB) Create(db *gorm.DB, notification *models.Notification) error {
	return db.Create(notification).Error
}

func (r *notificationRepositoryDB) CreateMany(db *gorm.DB, notifications []models.Notification) error {
	return db.Create(&notifications).Error
}

func (r *notificationRepositoryDB) PreloadRelation(db *gorm.DB, notificationID uuid.UUID) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.Where("id = ?", notificationID).Preload("Sender", helpers.OmitUserPasswordHash).Limit(1).Find(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) PreloadsRelation(db *gorm.DB, notificationsID []uuid.UUID) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.Where("id IN ?", notificationsID).Preload("Sender", helpers.OmitUserPasswordHash).Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}
