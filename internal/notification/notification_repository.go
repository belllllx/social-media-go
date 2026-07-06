package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypePost    NotificationType = "POST"
	NotificationTypeShare   NotificationType = "SHARE"
	NotificationTypeComment NotificationType = "COMMENT"
	NotificationTypeReply   NotificationType = "REPLY"
	NotificationTypeTag     NotificationType = "TAG"
	NotificationTypeLike    NotificationType = "LIKE"
	NotificationTypeFollow  NotificationType = "FOLLOW"
)

type Notification struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Type       NotificationType `gorm:"type:notification_type"`
	Message    string
	IsRead     bool `gorm:"default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SenderID   uuid.UUID
	ReceiverID uuid.UUID
	PostID     *uuid.UUID
	CommentID  *uuid.UUID
}
