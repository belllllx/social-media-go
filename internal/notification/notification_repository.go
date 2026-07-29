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
	FindWithSenderRelation(db *gorm.DB, notificationID uuid.UUID) (*models.Notification, error)
	FindsWithSenderRelation(db *gorm.DB, notificationsID []uuid.UUID) ([]models.Notification, error)
	FindOfPost(db *gorm.DB, senderID, receiverID, postID uuid.UUID) (*models.Notification, error)
	FindsOfPost(db *gorm.DB, senderID, receiverID, postID uuid.UUID) ([]models.Notification, error)
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

func (r *notificationRepositoryDB) FindWithSenderRelation(db *gorm.DB, notificationID uuid.UUID) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.Where("id = ?", notificationID).Preload("Sender", helpers.OmitUserPasswordHash).Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindsWithSenderRelation(db *gorm.DB, notificationsID []uuid.UUID) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.Where("id IN ?", notificationsID).Preload("Sender", helpers.OmitUserPasswordHash).Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}

func (r *notificationRepositoryDB) FindOfPost(
	db *gorm.DB,
	senderID,
	receiverID,
	postID uuid.UUID,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.Where("sender_id = ? AND receiver_id = ? AND post_id = ?", senderID, receiverID, postID).Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindsOfPost(
	db *gorm.DB,
	senderID,
	receiverID,
	postID uuid.UUID,
) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.Where("sender_id = ? AND receiver_id = ? AND post_id = ?", senderID, receiverID, postID).Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}
