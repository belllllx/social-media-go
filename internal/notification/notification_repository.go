package notification

import (
	"context"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		notification *models.Notification,
	) error
	CreateMany(
		ctx context.Context,
		db *gorm.DB,
		notifications []models.Notification,
	) error
	FindWithSenderRelation(
		ctx context.Context,
		db *gorm.DB,
		notificationID uuid.UUID,
	) (*models.Notification, error)
	FindsWithSenderRelation(
		ctx context.Context,
		db *gorm.DB,
		notificationsID []uuid.UUID,
	) ([]models.Notification, error)
	FindOfPost(
		ctx context.Context,
		db *gorm.DB,
		senderID,
		receiverID,
		postID uuid.UUID,
	) (*models.Notification, error)
	FindsOfPost(
		ctx context.Context,
		db *gorm.DB,
		senderID,
		receiverID,
		postID uuid.UUID,
	) ([]models.Notification, error)
}

type notificationRepositoryDB struct {
}

func NewNotificationRepositoryDB() NotificationRepository {
	return &notificationRepositoryDB{}
}

func (r *notificationRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	notification *models.Notification,
) error {
	return db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepositoryDB) CreateMany(
	ctx context.Context,
	db *gorm.DB,
	notifications []models.Notification,
) error {
	return db.WithContext(ctx).Create(&notifications).Error
}

func (r *notificationRepositoryDB) FindWithSenderRelation(
	ctx context.Context,
	db *gorm.DB,
	notificationID uuid.UUID,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Where("id = ?", notificationID).
		Preload("Sender", helpers.OmitUserPasswordHash).
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindsWithSenderRelation(
	ctx context.Context,
	db *gorm.DB,
	notificationsID []uuid.UUID,
) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.
		WithContext(ctx).
		Where("id IN ?", notificationsID).
		Preload("Sender", helpers.OmitUserPasswordHash).
		Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}

func (r *notificationRepositoryDB) FindOfPost(
	ctx context.Context,
	db *gorm.DB,
	senderID,
	receiverID,
	postID uuid.UUID,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Where("sender_id = ? AND receiver_id = ? AND post_id = ?", senderID, receiverID, postID).
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindsOfPost(
	ctx context.Context,
	db *gorm.DB,
	senderID,
	receiverID,
	postID uuid.UUID,
) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.
		WithContext(ctx).
		Where("sender_id = ? AND receiver_id = ? AND post_id = ?", senderID, receiverID, postID).
		Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}
