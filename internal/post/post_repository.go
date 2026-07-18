package post

import (
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type PostRepository interface {
	Create(post *user.Post) error
	PreloadRelations(postID uuid.UUID) (*user.Post, error)
	FindByIDPreloadRelation(postID uuid.UUID) (*user.Post, error)
	FindByIDCursor(postID uuid.UUID) (*Cursor, error)
	FindsCursorPagination(cursor *Cursor, limit int) ([]user.Post, error)
}

type postRepositoryDB struct {
	db *gorm.DB
}

func NewPostRepositoryDB(db *gorm.DB) PostRepository {
	return &postRepositoryDB{db: db}
}

func (r *postRepositoryDB) Create(post *user.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepositoryDB) PreloadRelations(postID uuid.UUID) (*user.Post, error) {
	post := &user.Post{}
	err := r.db.Where("id = ?", postID).
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

func (r *postRepositoryDB) FindByIDPreloadRelation(postID uuid.UUID) (*user.Post, error) {
	post := &user.Post{}
	err := r.db.Where("id = ?", postID).Preload("User", helpers.OmitUserPasswordHash).Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDCursor(postID uuid.UUID) (*Cursor, error) {
	post := &user.Post{}
	err := r.db.Where("id = ?", postID).Select("id", "created_at").Take(post).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        post.ID,
		CreatedAt: post.CreatedAt,
	}
	return cursor, nil
}

func (r *postRepositoryDB) FindsCursorPagination(cursor *Cursor, limit int) ([]user.Post, error) {
	posts := &[]user.Post{}
	db := r.db.
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
