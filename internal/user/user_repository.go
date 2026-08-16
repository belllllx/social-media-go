package user

import (
	"context"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type UserRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		user *models.User,
	) error
	FindByUsername(
		ctx context.Context,
		db *gorm.DB,
		username string,
	) (*models.User, error)
	FindByEmail(
		ctx context.Context,
		db *gorm.DB,
		email string,
	) (*models.User, error)
	FindByIDWithFollowRelations(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) (*models.User, error)
	FindsByIDExcept(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) ([]models.User, error)
	FindByID(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) (*models.User, error)
	FindByIDCursor(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) (*Cursor, error)
	FindsByFullnameCursorPagination(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
		fullname string,
		cursor *Cursor,
		limit int,
	) ([]models.User, error)
	UpdatePassword(
		ctx context.Context,
		db *gorm.DB,
		email,
		passwordHash string,
	) error
}

type userRepositoryDB struct {
}

func NewUserRepositoryDB() UserRepository {
	return &userRepositoryDB{}
}

func (r *userRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	user *models.User,
) error {
	return db.WithContext(ctx).Create(user).Error
}

func (r *userRepositoryDB) FindByUsername(
	ctx context.Context,
	db *gorm.DB,
	username string,
) (*models.User, error) {
	user := &models.User{}
	err := db.
		WithContext(ctx).
		Where("username = ?", username).
		Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByEmail(
	ctx context.Context,
	db *gorm.DB,
	email string,
) (*models.User, error) {
	user := &models.User{}
	err := db.
		WithContext(ctx).
		Where("email = ?", email).
		Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByIDWithFollowRelations(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
) (*models.User, error) {
	user := &models.User{}
	err := db.
		WithContext(ctx).
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

func (r *userRepositoryDB) FindsByIDExcept(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
) ([]models.User, error) {
	users := &[]models.User{}
	err := db.
		WithContext(ctx).
		Where("id <> ?", userID).
		Select(
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
		).
		Find(users).Error
	if err != nil {
		return nil, err
	}
	return *users, nil
}

func (r *userRepositoryDB) FindByID(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
) (*models.User, error) {
	user := &models.User{}
	err := db.
		WithContext(ctx).
		Where("id = ?", userID).
		Take(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositoryDB) FindByIDCursor(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
) (*Cursor, error) {
	user := &models.User{}
	err := db.
		WithContext(ctx).
		Where("id = ?", userID).
		Select("id", "created_at").
		Take(user).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
	}
	return cursor, nil
}

func (r *userRepositoryDB) FindsByFullnameCursorPagination(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
	fullname string,
	cursor *Cursor,
	limit int,
) ([]models.User, error) {
	users := &[]models.User{}
	db = db.
		WithContext(ctx).
		Select(
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
		).
		Where("id <> ? AND fullname ILIKE ?", userID, "%"+fullname+"%").
		Order("created_at DESC, id DESC").
		Limit(limit)

	if cursor != nil {
		db = db.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := db.Find(users).Error
	if err != nil {
		return nil, err
	}
	return *users, nil
}

func (r *userRepositoryDB) UpdatePassword(
	ctx context.Context,
	db *gorm.DB,
	email,
	passwordHash string,
) error {
	user := &models.User{}
	return db.
		WithContext(ctx).
		Model(user).
		Where("email = ?", email).
		Update("password_hash", passwordHash).Error
}
