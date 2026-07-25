package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string
type ProviderType string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"

	ProviderTypeLocal    ProviderType = "LOCAL"
	ProviderTypeGoogle   ProviderType = "GOOGLE"
	ProviderTypeFacebook ProviderType = "FACEBOOK"
	ProviderTypeGithub   ProviderType = "GITHUB"
)

type User struct {
	ID                    uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_users_cursor,sort:desc;index:idx_users_fullname_cursor,sort:desc"`
	Fullname              string    `gorm:"type:varchar(30);index:idx_users_fullname_cursor"`
	Username              *string   `gorm:"type:varchar(15);uniqueIndex"`
	Email                 string    `gorm:"type:varchar(30);uniqueIndex"`
	PasswordHash          *string
	DateOfBirth           *time.Time
	ProfileUrl            *string
	ProfileBackgroundUrl  *string
	Info                  *string      `gorm:"type:varchar(30)"`
	Role                  Role         `gorm:"type:user_role;default:'USER'"`
	ProviderType          ProviderType `gorm:"type:provider_type;default:'LOCAL'"`
	CreatedAt             time.Time    `gorm:"index:idx_users_cursor,sort:desc;index:idx_users_fullname_cursor,sort:desc"`
	UpdatedAt             time.Time
	Posts                 []Post         `gorm:"constraint:OnDelete:CASCADE"`
	Likes                 []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Comments              []Comment      `gorm:"constraint:OnDelete:CASCADE"`
	Replies               []Comment      `gorm:"foreignKey:ReplyToUserID"`
	SenderNotifications   []Notification `gorm:"foreignKey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverNotifications []Notification `gorm:"foreignKey:ReceiverID;constraint:OnDelete:CASCADE"`
	Followings            []Follow       `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	Followers             []Follow       `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
}
