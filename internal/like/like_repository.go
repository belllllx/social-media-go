package like

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID  `gorm:"uniqueIndex:idx_user_post;uniqueIndex:idx_user_comment"`
	PostID    *uuid.UUID `gorm:"uniqueIndex:idx_user_post"`
	CommentID *uuid.UUID `gorm:"uniqueIndex:idx_user_comment"`
}
