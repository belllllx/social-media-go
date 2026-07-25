package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_comments_post_cursor,sort:desc"`
	Message       *string
	CreatedAt     time.Time `gorm:"index:idx_comments_post_cursor,sort:desc"`
	UpdatedAt     time.Time
	PostID        uuid.UUID `gorm:"index:idx_comments_post_cursor"`
	Post          Post      `gorm:"constraint:OnDelete:CASCADE"`
	UserID        uuid.UUID
	User          User `gorm:"constraint:OnDelete:CASCADE"`
	ParentID      *uuid.UUID
	Parent        *Comment `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	ReplyID       *uuid.UUID
	Reply         *Comment `gorm:"foreignkey:ReplyID;constraint:OnDelete:CASCADE"`
	ReplyToUserID *uuid.UUID
	ReplyToUser   *User          `gorm:"foreignkey:ReplyToUserID;constraint:OnDelete:CASCADE"`
	Likes         []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Replies       []Comment      `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Tags          []Comment      `gorm:"foreignkey:ReplyID;constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"constraint:OnDelete:CASCADE"`
}
