package user

import (
	"time"

	"github.com/belllllx/social-media-go/internal/comment"
	"github.com/belllllx/social-media-go/internal/like"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/post"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

type Follower struct {
	ID          int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FollowerID  uuid.UUID `gorm:"uniqueIndex:idx_follower_unique"`
	FollowingID uuid.UUID `gorm:"uniqueIndex:idx_follower_unique"`
}

type User struct {
	ID                    uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Fullname              string    `gorm:"type:varchar(30)"`
	Username              *string   `gorm:"type:varchar(15);unique"`
	Email                 string    `gorm:"type:varchar(30);unique"`
	PasswordHash          *string
	DateOfBirth           *time.Time
	ProfileUrl            *string
	ProfileBackgroundUrl  *string
	Info                  *string      `gorm:"type:varchar(30)"`
	Role                  Role         `gorm:"type:user_role;default:'USER'"`
	ProviderType          ProviderType `gorm:"type:provider_type;default:'LOCAL'"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Posts                 []post.Post                 `gorm:"constraint:OnDelete:CASCADE"`
	Likes                 []like.Like                 `gorm:"constraint:OnDelete:CASCADE"`
	Comments              []comment.Comment           `gorm:"constraint:OnDelete:CASCADE"`
	Replies               []comment.Comment           `gorm:"foreignKey:ReplyToUserID"`
	SenderNotifications   []notification.Notification `gorm:"foreignKey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverNotifications []notification.Notification `gorm:"foreignKey:ReceiverID;constraint:OnDelete:CASCADE"`
	Followings            []Follower                  `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	Followers             []Follower                  `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
}

type UserRepository interface {
	Create(user *User) error
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByID(ID uuid.UUID) (*User, error)
	UpdatePassword(email, passwordHash string) error
}

type userRepositoryDB struct {
	db *gorm.DB
}

func NewUserRepositoryDB(db *gorm.DB) UserRepository {
	db.AutoMigrate(&User{})
	return &userRepositoryDB{db: db}
}

func (r *userRepositoryDB) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *userRepositoryDB) FindByUsername(username string) (*User, error) {
	user := &User{}
	err := r.db.Where("username = ?", username).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByEmail(email string) (*User, error) {
	user := &User{}
	err := r.db.Where("email = ?", email).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByID(ID uuid.UUID) (*User, error) {
	user := &User{}
	err := r.db.Where("id = ?", ID).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) UpdatePassword(email, passwordHash string) error {
	user := &User{}
	return r.db.Model(user).Where("email = ?", email).Update("password_hash", passwordHash).Error
}
