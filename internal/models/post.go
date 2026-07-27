package models

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_posts_cursor,sort:desc;index:idx_posts_user_cursor,sort:desc"`
	Message       *string
	CreatedAt     time.Time `gorm:"index:idx_posts_cursor,sort:desc;index:idx_posts_user_cursor,sort:desc"`
	UpdatedAt     time.Time
	UserID        uuid.UUID `gorm:"index:idx_posts_user_cursor"`
	User          User      `gorm:"constraint:OnDelete:CASCADE"`
	ParentID      *uuid.UUID
	Parent        *Post          `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	SharePosts    []Post         `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Likes         []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Comments      []Comment      `gorm:"constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"constraint:OnDelete:CASCADE"`
}
