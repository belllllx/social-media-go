package post

import (
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

type PostRepository interface {
	Create(db *gorm.DB, post *models.Post) error
	PreloadRelations(db *gorm.DB, postID uuid.UUID) (*models.Post, error)
	FindByID(db *gorm.DB, postID uuid.UUID) (*models.Post, error)
	FindByIDPreloadRelation(db *gorm.DB, postID uuid.UUID) (*models.Post, error)
	FindByIDPreloadRelations(db *gorm.DB, postID uuid.UUID) (*models.Post, error)
	FindByIDCursor(db *gorm.DB, postID uuid.UUID) (*Cursor, error)
	FindsCursorPagination(db *gorm.DB, cursor *Cursor, limit int) ([]models.Post, error)
	FindsByUserIDCursorPagination(db *gorm.DB, userID uuid.UUID, cursor *Cursor, limit int) ([]models.Post, error)
	Update(db *gorm.DB, postID uuid.UUID, updatePost *models.Post) error
}

type postRepositoryDB struct {
}

func NewPostRepositoryDB() PostRepository {
	return &postRepositoryDB{}
}

func (r *postRepositoryDB) Create(db *gorm.DB, post *models.Post) error {
	return db.Create(post).Error
}

func (r *postRepositoryDB) PreloadRelations(db *gorm.DB, postID uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.Where("id = ?", postID).
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("Likes").
		Preload("Comments").
		Limit(1).
		Find(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByID(db *gorm.DB, postID uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.Where("id = ?", postID).Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDPreloadRelation(db *gorm.DB, postID uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.Where("id = ?", postID).Preload("User", helpers.OmitUserPasswordHash).Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDPreloadRelations(db *gorm.DB, postID uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		Where("id = ?", postID).
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("Parent.User", helpers.OmitUserPasswordHash).
		Preload("Likes", func(db *gorm.DB) *gorm.DB {
			return db.Order("likes.created_at DESC")
		}).
		Preload("Likes.User", helpers.OmitUserPasswordHash).
		Preload("Comments").
		Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDCursor(db *gorm.DB, postID uuid.UUID) (*Cursor, error) {
	post := &models.Post{}
	err := db.Where("id = ?", postID).Select("id", "created_at").Take(post).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        post.ID,
		CreatedAt: post.CreatedAt,
	}
	return cursor, nil
}

func (r *postRepositoryDB) FindsCursorPagination(
	db *gorm.DB,
	cursor *Cursor,
	limit int,
) ([]models.Post, error) {
	posts := &[]models.Post{}
	db = db.
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("Parent.User", helpers.OmitUserPasswordHash).
		Preload("Likes", func(db *gorm.DB) *gorm.DB {
			return db.Order("likes.created_at DESC")
		}).
		Preload("Likes.User", helpers.OmitUserPasswordHash).
		Preload("Comments").
		Order("created_at DESC, id DESC").
		Limit(limit)

	if cursor != nil {
		db = db.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := db.Find(posts).Error
	if err != nil {
		return nil, err
	}
	return *posts, nil
}

func (r *postRepositoryDB) FindsByUserIDCursorPagination(
	db *gorm.DB,
	userID uuid.UUID,
	cursor *Cursor,
	limit int,
) ([]models.Post, error) {
	posts := &[]models.Post{}
	db = db.
		Where("user_id = ?", userID).
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("Parent.User", helpers.OmitUserPasswordHash).
		Preload("Likes", func(db *gorm.DB) *gorm.DB {
			return db.Order("likes.created_at DESC")
		}).
		Preload("Likes.User", helpers.OmitUserPasswordHash).
		Preload("Comments").
		Order("created_at DESC, id DESC").
		Limit(limit)

	if cursor != nil {
		db = db.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := db.Find(posts).Error
	if err != nil {
		return nil, err
	}
	return *posts, nil
}

func (r *postRepositoryDB) Update(db *gorm.DB, postID uuid.UUID, updatePost *models.Post) error {
	post := &models.Post{}
	err := db.Model(post).Where("id = ?", postID).Updates(*updatePost).Error
	if err != nil {
		return err
	}
	return nil
}
