package user

import (
	"time"

	"github.com/belllllx/social-media-go/pkg/helpers"
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
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_notifications_receiver_cursor,sort:desc"`
	Type       NotificationType `gorm:"type:notification_type"`
	Message    string
	IsRead     bool      `gorm:"default:false"`
	CreatedAt  time.Time `gorm:"index:idx_notifications_receiver_cursor,sort:desc"`
	UpdatedAt  time.Time
	SenderID   uuid.UUID
	Sender     User      `gorm:"foreignkey:SenderID;constraint:OnDelete:CASCADE"`
	ReceiverID uuid.UUID `gorm:"index:idx_notifications_receiver_cursor"`
	Receiver   User      `gorm:"foreignkey:ReceiverID;constraint:OnDelete:CASCADE"`
	PostID     *uuid.UUID
	Post       *Post `gorm:"constraint:OnDelete:CASCADE"`
	CommentID  *uuid.UUID
	Comment    *Comment `gorm:"constraint:OnDelete:CASCADE"`
}

type Post struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;index:idx_posts_cursor,sort:desc"`
	Message       *string
	CreatedAt     time.Time `gorm:"index:idx_posts_cursor,sort:desc"`
	UpdatedAt     time.Time
	UserID        uuid.UUID `gorm:"index"`
	User          User      `gorm:"constraint:OnDelete:CASCADE"`
	ParentID      *uuid.UUID
	Parent        *Post          `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	SharePosts    []Post         `gorm:"foreignkey:ParentID;constraint:OnDelete:CASCADE"`
	Likes         []Like         `gorm:"constraint:OnDelete:CASCADE"`
	Comments      []Comment      `gorm:"constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"constraint:OnDelete:CASCADE"`
}

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

type Follow struct {
	ID          int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FollowerID  uuid.UUID `gorm:"uniqueIndex:idx_followers_unique"`
	Follower    User      `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE"`
	FollowingID uuid.UUID `gorm:"uniqueIndex:idx_followers_unique"`
	Following   User      `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE"`
}

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

type UserRepository interface {
	Create(db *gorm.DB, user *User) error
	FindByUsername(db *gorm.DB, username string) (*User, error)
	FindByEmail(db *gorm.DB, email string) (*User, error)
	FindByID(db *gorm.DB, userID uuid.UUID) (*User, error)
	FindsByIDExcept(db *gorm.DB, userID uuid.UUID) ([]User, error)
	UpdatePassword(db *gorm.DB, email, passwordHash string) error
}

type userRepositoryDB struct {
}

func NewUserRepositoryDB(db *gorm.DB) UserRepository {
	db.AutoMigrate(
		&User{},
		&Follow{},
		&Like{},
		&Comment{},
		&Post{},
		&Notification{},
	)
	return &userRepositoryDB{}
}

func (r *userRepositoryDB) Create(db *gorm.DB, user *User) error {
	return db.Create(user).Error
}

func (r *userRepositoryDB) FindByUsername(db *gorm.DB, username string) (*User, error) {
	user := &User{}
	err := db.Where("username = ?", username).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByEmail(db *gorm.DB, email string) (*User, error) {
	user := &User{}
	err := db.Where("email = ?", email).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByID(db *gorm.DB, userID uuid.UUID) (*User, error) {
	user := &User{}
	err := db.
		Where("id = ?", userID).
		Preload("Followings.Following", helpers.OmitUserPasswordHash).
		Preload("Followings.Following.Followers").
		Preload("Followers.Follower", helpers.OmitUserPasswordHash).
		Preload("Followers.Follower.Followers").
		Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindsByIDExcept(db *gorm.DB, userID uuid.UUID) ([]User, error) {
	users := &[]User{}
	err := db.Where("id <> ?", userID).Select(
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

func (r *userRepositoryDB) UpdatePassword(db *gorm.DB, email, passwordHash string) error {
	user := &User{}
	return db.Model(user).Where("email = ?", email).Update("password_hash", passwordHash).Error
}
