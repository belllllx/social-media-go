package notification

import (
	"context"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

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
	FindByIDWithSenderRelation(
		ctx context.Context,
		db *gorm.DB,
		notificationID uuid.UUID,
	) (*models.Notification, error)
	FindsByIDWithSenderRelation(
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
	FindOfLikePost(
		ctx context.Context,
		db *gorm.DB,
		senderID,
		receiverID,
		postID uuid.UUID,
	) (*models.Notification, error)
	FindOfComment(
		ctx context.Context,
		db *gorm.DB,
		senderID,
		receiverID,
		commentID uuid.UUID,
		notificationType models.NotificationType,
	) (*models.Notification, error)
	FindOfLikeComment(
		ctx context.Context,
		db *gorm.DB,
		senderID,
		receiverID,
		postID,
		commentID uuid.UUID,
	) (*models.Notification, error)
	FindByIDCursor(
		ctx context.Context,
		db *gorm.DB,
		notificationID uuid.UUID,
	) (*Cursor, error)
	FindsByReceiverIDCursorPaginationWithSenderRelation(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
		cursor *Cursor,
		limit int,
	) ([]models.Notification, error)
	UpdateIsRead(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
		notificationsID []uuid.UUID,
	) ([]models.Notification, error)
	Delete(
		ctx context.Context,
		db *gorm.DB,
		notificationID uuid.UUID,
	) error
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

func (r *notificationRepositoryDB) FindByIDWithSenderRelation(
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

func (r *notificationRepositoryDB) FindsByIDWithSenderRelation(
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
		Select("id", "receiver_id").
		Where(
			"sender_id = ? AND receiver_id = ? AND post_id = ? AND type = ?",
			senderID,
			receiverID,
			postID,
			models.NotificationTypeShare,
		).
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
		Select("id", "receiver_id").
		Where(
			"sender_id = ? AND receiver_id = ? AND post_id = ? AND type = ?",
			senderID,
			receiverID,
			postID,
			models.NotificationTypePost,
		).
		Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}

func (r *notificationRepositoryDB) FindOfLikePost(
	ctx context.Context,
	db *gorm.DB,
	senderID,
	receiverID,
	postID uuid.UUID,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Select("id", "receiver_id").
		Where(
			"type = ? AND sender_id = ? AND receiver_id = ? AND post_id = ?",
			models.NotificationTypeLike,
			senderID,
			receiverID,
			postID,
		).
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindOfComment(
	ctx context.Context,
	db *gorm.DB,
	senderID,
	receiverID,
	commentID uuid.UUID,
	notificationType models.NotificationType,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Select("id", "receiver_id").
		Where(
			"sender_id = ? AND receiver_id = ? AND comment_id = ? AND type = ?",
			senderID,
			receiverID,
			commentID,
			notificationType,
		).
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindOfLikeComment(
	ctx context.Context,
	db *gorm.DB,
	senderID,
	receiverID,
	postID,
	commentID uuid.UUID,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Select("id", "receiver_id").
		Where(
			"type = ? AND sender_id = ? AND receiver_id = ? AND post_id = ? AND comment_id = ?",
			models.NotificationTypeLike,
			senderID,
			receiverID,
			postID,
			commentID,
		).
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationRepositoryDB) FindByIDCursor(
	ctx context.Context,
	db *gorm.DB,
	notificationID uuid.UUID,
) (*Cursor, error) {
	notification := &models.Notification{}
	err := db.
		WithContext(ctx).
		Where("id = ?", notificationID).
		Select("id", "created_at").
		Take(notification).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        notification.ID,
		CreatedAt: notification.CreatedAt,
	}
	return cursor, nil
}

func (r *notificationRepositoryDB) FindsByReceiverIDCursorPaginationWithSenderRelation(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
	cursor *Cursor,
	limit int,
) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	db = db.
		WithContext(ctx).
		Where("receiver_id = ?", userID).
		Preload("Sender", helpers.OmitUserPasswordHash).
		Order("created_at DESC, id DESC").
		Limit(limit)

	if cursor != nil {
		db = db.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := db.Find(notifications).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}

func (r *notificationRepositoryDB) UpdateIsRead(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
	notificationsID []uuid.UUID,
) ([]models.Notification, error) {
	notifications := &[]models.Notification{}
	err := db.
		WithContext(ctx).
		Model(notifications).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "id"},
				{Name: "is_read"},
			},
		}).
		Where(
			"id IN ? AND receiver_id = ? AND is_read = ?",
			notificationsID,
			userID,
			false,
		).
		Update("is_read", true).Error
	if err != nil {
		return nil, err
	}
	return *notifications, nil
}

func (r *notificationRepositoryDB) Delete(
	ctx context.Context,
	db *gorm.DB,
	notificationID uuid.UUID,
) error {
	notification := &models.Notification{}
	return db.
		WithContext(ctx).
		Where("id = ?", notificationID).
		Delete(notification).Error
}
