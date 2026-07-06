package comment

import (
	"time"

	"github.com/belllllx/social-media-go/internal/like"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/google/uuid"
)

type Comment struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Message       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PostID        uuid.UUID
	UserID        uuid.UUID
	ParentID      *uuid.UUID
	ReplyID       *uuid.UUID
	ReplyToUserID *uuid.UUID
	Likes         []like.Like                 `gorm:"constraint:OnDelete:CASCADE"`
	Replies       []Comment                   `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Tags          []Comment                   `gorm:"foreignkey:ReplyID;constraint:OnDelete:CASCADE"`
	Notifications []notification.Notification `gorm:"constraint:OnDelete:CASCADE"`
}
