package notification

import (
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	CreateMany(notifications []user.Notification) error
	PreloadsRelation(notificationsID []uuid.UUID) ([]user.Notification, error)
}

type notificationRepositoryDB struct {
	db *gorm.DB
}

func NewNotificationRepositoryDB(db *gorm.DB) NotificationRepository {
	return &notificationRepositoryDB{db: db}
}

func (r *notificationRepositoryDB) CreateMany(notifications []user.Notification) error {
	return r.db.Create(&notifications).Error
}

func (r *notificationRepositoryDB) PreloadsRelation(notificationsID []uuid.UUID) ([]user.Notification, error) {
	notifications := &[]user.Notification{}
	err := r.db.Where("id IN ?", notificationsID).Preload("Sender", helpers.OmitUserPasswordHash).Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}
