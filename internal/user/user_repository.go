package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string
type ProviderType string
type NotificationType string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"

	ProviderTypeLocal    ProviderType = "LOCAL"
	ProviderTypeGoogle   ProviderType = "GOOGLE"
	ProviderTypeFacebook ProviderType = "FACEBOOK"
	ProviderTypeGithub   ProviderType = "GITHUB"

	NotificationTypePost    NotificationType = "POST"
	NotificationTypeShare   NotificationType = "SHARE"
	NotificationTypeComment NotificationType = "COMMENT"
	NotificationTypeReply   NotificationType = "REPLY"
	NotificationTypeTag     NotificationType = "TAG"
	NotificationTypeLike    NotificationType = "LIKE"
	NotificationTypeFollow  NotificationType = "FOLLOW"
)

type Notification struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Type       NotificationType `gorm:"type:notification_type"`
	Message    string
	IsRead     bool `gorm:"default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SenderID   uuid.UUID
	Sender     User `gorm:"foreignkey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverID uuid.UUID
	Receiver   User `gorm:"foreignkey:ReceiverID;constraint:OnDelete:CASCADE"`
	PostID     *uuid.UUID
	Post       *Post `gorm:"constraint:OnDelete:CASCADE"`
	CommentID  *uuid.UUID
	Comment    *Comment `gorm:"constraint:OnDelete:CASCADE"`
}

type Post struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Message       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UserID        uuid.UUID
	User          User `gorm:"constraint:OnDelete:CASCADE"`
	ParentID      *uuid.UUID
	Parent        *Post          `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	SharePosts    []Post         `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Likes         []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Comments      []Comment      `gorm:"constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"constraint:OnDelete:CASCADE"`
}

type Comment struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Message       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PostID        uuid.UUID
	Post          Post `gorm:"constraint:OnDelete:CASCADE"`
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

type Follower struct {
	ID          int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FollowerID  uuid.UUID `gorm:"uniqueIndex:idx_follower_unique"`
	Follower    User      `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	FollowingID uuid.UUID `gorm:"uniqueIndex:idx_follower_unique"`
	Following   User      `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
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
	Posts                 []Post         `gorm:"constraint:OnDelete:CASCADE"`
	Likes                 []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Comments              []Comment      `gorm:"constraint:OnDelete:CASCADE"`
	Replies               []Comment      `gorm:"foreignKey:ReplyToUserID"`
	SenderNotifications   []Notification `gorm:"foreignKey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverNotifications []Notification `gorm:"foreignKey:ReceiverID;constraint:OnDelete:CASCADE"`
	Followings            []Follower     `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	Followers             []Follower     `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
}

type UserRepository interface {
	Create(user *User) error
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByID(ID uuid.UUID) (*User, error)
	FindByIDExcept(ID uuid.UUID) ([]User, error)
	UpdatePassword(email, passwordHash string) error
}

type userRepositoryDB struct {
	db *gorm.DB
}

func NewUserRepositoryDB(db *gorm.DB) UserRepository {
	db.AutoMigrate(
		&User{},
		&Follower{},
		&Like{},
		&Comment{},
		&Post{},
		&Notification{},
	)
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

func (r *userRepositoryDB) FindByIDExcept(ID uuid.UUID) ([]User, error) {
	users := &[]User{}
	err := r.db.Where("id <> ?", ID).Select(
		"id",
		"fullname",
		"username",
		"email",
		"date_of_birth",
		"profile_url",
		"profile_background_url",
		"info",
		"role",
		"provider_type",
		"created_at",
		"updated_at",
	).Find(users).Error
	if err != nil {
		return nil, err
	}
	return *users, nil
}

func (r *userRepositoryDB) UpdatePassword(email, passwordHash string) error {
	user := &User{}
	return r.db.Model(user).Where("email = ?", email).Update("password_hash", passwordHash).Error
}
