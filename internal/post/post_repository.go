package post

import (
	"time"

	"github.com/belllllx/social-media-go/internal/comment"
	"github.com/belllllx/social-media-go/internal/like"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/google/uuid"
)

type Post struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Message       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UserID        uuid.UUID
	ParentID      *uuid.UUID
	SharePosts    []Post                      `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Likes         []like.Like                 `gorm:"constraint:OnDelete:CASCADE"`
	Comments      []comment.Comment           `gorm:"constraint:OnDelete:CASCADE"`
	Notifications []notification.Notification `gorm:"constraint:OnDelete:CASCADE"`
}
