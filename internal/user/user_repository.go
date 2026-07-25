package user

import (
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(db *gorm.DB, user *models.User) error
	FindByUsername(db *gorm.DB, username string) (*models.User, error)
	FindByEmail(db *gorm.DB, email string) (*models.User, error)
	FindByID(db *gorm.DB, userID uuid.UUID) (*models.User, error)
	FindsByIDExcept(db *gorm.DB, userID uuid.UUID) ([]models.User, error)
	UpdatePassword(db *gorm.DB, email, passwordHash string) error
}

type userRepositoryDB struct {
}

func NewUserRepositoryDB() UserRepository {
	return &userRepositoryDB{}
}

func (r *userRepositoryDB) Create(db *gorm.DB, user *models.User) error {
	return db.Create(user).Error
}

func (r *userRepositoryDB) FindByUsername(db *gorm.DB, username string) (*models.User, error) {
	user := &models.User{}
	err := db.Where("username = ?", username).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByEmail(db *gorm.DB, email string) (*models.User, error) {
	user := &models.User{}
	err := db.Where("email = ?", email).Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByID(db *gorm.DB, userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
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

func (r *userRepositoryDB) FindsByIDExcept(db *gorm.DB, userID uuid.UUID) ([]models.User, error) {
	users := &[]models.User{}
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

func (r *userRepositoryDB) UpdatePassword(
	db *gorm.DB,
	email,
	passwordHash string,
) error {
	user := &models.User{}
	return db.Model(user).Where("email = ?", email).Update("password_hash", passwordHash).Error
}
