package post

import (
	"context"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type PostRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		post *models.Post,
	) error
	FindByID(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
	) (*models.Post, error)
	FindByIDWithUserRelation(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
	) (*models.Post, error)
	FindByIDWithParentRelation(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
	) (*models.Post, error)
	FindByIDWithPostRelations(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
	) (*models.Post, error)
	FindByIDCursor(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
	) (*Cursor, error)
	FindsCursorPaginationWithPostRelations(
		ctx context.Context,
		db *gorm.DB,
		cursor *Cursor,
		limit int,
	) ([]models.Post, error)
	FindsByUserIDCursorPaginationWithPostRelations(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
		cursor *Cursor,
		limit int,
	) ([]models.Post, error)
	Update(
		ctx context.Context,
		db *gorm.DB,
		userID,
		postID uuid.UUID,
		updatePost *models.Post,
	) (*models.Post, error)
	Delete(
		ctx context.Context,
		db *gorm.DB,
		userID,
		postID uuid.UUID,
	) (*models.Post, error)
}

type postRepositoryDB struct {
}

func NewPostRepositoryDB() PostRepository {
	return &postRepositoryDB{}
}

func (r *postRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	post *models.Post,
) error {
	return db.WithContext(ctx).Create(post).Error
}

func (r *postRepositoryDB) FindByID(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Where("id = ?", postID).
		Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDWithUserRelation(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Where("id = ?", postID).
		Preload("User", helpers.OmitUserPasswordHash).
		Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDWithParentRelation(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Where("id = ?", postID).
		Preload("Parent").
		Take(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) FindByIDWithPostRelations(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
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

func (r *postRepositoryDB) FindByIDCursor(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
) (*Cursor, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Where("id = ?", postID).
		Select("id", "created_at").
		Take(post).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        post.ID,
		CreatedAt: post.CreatedAt,
	}
	return cursor, nil
}

func (r *postRepositoryDB) FindsCursorPaginationWithPostRelations(
	ctx context.Context,
	db *gorm.DB,
	cursor *Cursor,
	limit int,
) ([]models.Post, error) {
	posts := &[]models.Post{}
	db = db.
		WithContext(ctx).
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
			"created_at < ? OR (created_at = ? AND id < ?)",
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

func (r *postRepositoryDB) FindsByUserIDCursorPaginationWithPostRelations(
	ctx context.Context,
	db *gorm.DB,
	userID uuid.UUID,
	cursor *Cursor,
	limit int,
) ([]models.Post, error) {
	posts := &[]models.Post{}
	db = db.
		WithContext(ctx).
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
			"(created_at < ? OR (created_at = ? AND id < ?))",
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

func (r *postRepositoryDB) Update(
	ctx context.Context,
	db *gorm.DB,
	userID,
	postID uuid.UUID,
	updatePost *models.Post,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Model(post).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "id"},
				{Name: "message"},
			},
		}).
		Where("id = ? AND user_id = ?", postID, userID).
		Updates(*updatePost).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *postRepositoryDB) Delete(
	ctx context.Context,
	db *gorm.DB,
	userID,
	postID uuid.UUID,
) (*models.Post, error) {
	post := &models.Post{}
	err := db.
		WithContext(ctx).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "id"},
			},
		}).
		Where("id = ? AND user_id = ?", postID, userID).
		Delete(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}
