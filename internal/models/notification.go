package models

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
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_notifications_receiver_cursor,sort:desc"`
	Type       NotificationType `gorm:"type:notification_type"`
	Message    string
	IsRead     bool      `gorm:"default:false"`
	CreatedAt  time.Time `gorm:"index:idx_notifications_receiver_cursor,sort:desc"`
	UpdatedAt  time.Time
	SenderID   uuid.UUID
	Sender     User      `gorm:"foreignkey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverID uuid.UUID `gorm:"index:idx_notifications_receiver_cursor"`
	Receiver   User      `gorm:"foreignkey:ReceiverID;constraint:OnDelete:CASCADE"`
	PostID     *uuid.UUID
	Post       *Post `gorm:"constraint:OnDelete:CASCADE"`
	CommentID  *uuid.UUID
	Comment    *Comment `gorm:"constraint:OnDelete:CASCADE"`
}
