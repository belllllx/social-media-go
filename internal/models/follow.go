package models

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	ID          int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FollowerID  uuid.UUID `gorm:"uniqueIndex:idx_followers_unique"`
	Follower    User      `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	FollowingID uuid.UUID `gorm:"uniqueIndex:idx_followers_unique"`
	Following   User      `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
}
