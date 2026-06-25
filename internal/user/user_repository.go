package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string
type ProviderType string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"

	ProviderTypeLocal  ProviderType = "LOCAL"
	ProviderTypeGoogle ProviderType = "GOOGLE"
	ProviderTypeGithub ProviderType = "GITHUB"
)

type User struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Fullname             string    `gorm:"type:varchar(30)"`
	Username             *string   `gorm:"type:varchar(15);unique"`
	Email                string    `gorm:"type:varchar(30);unique"`
	PasswordHash         *string
	DateOfBirth          *time.Time
	ProfileUrl           *string
	ProfileBackgroundUrl *string
	Info                 *string      `gorm:"type:varchar(30)"`
	Role                 Role         `gorm:"type:user_role;default:'USER'"`
	ProviderType         ProviderType `gorm:"type:provider_type;default:'LOCAL'"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type UserRepository interface {
	Create(user *User) error
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
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
