package models

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID  `gorm:"uniqueIndex:idx_user_post;uniqueIndex:idx_user_comment"`
	User      User       `gorm:"constraint:OnDelete:CASCADE"`
	PostID    *uuid.UUID `gorm:"uniqueIndex:idx_user_post"`
	Post      *Post      `gorm:"constraint:OnDelete:CASCADE"`
	CommentID *uuid.UUID `gorm:"uniqueIndex:idx_user_comment"`
	Comment   *Comment   `gorm:"constraint:OnDelete:CASCADE"`
}
