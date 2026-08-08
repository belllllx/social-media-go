package comment

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

type CommentRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		comment *models.Comment,
	) error
	FindByID(
		ctx context.Context,
		db *gorm.DB,
		commentID uuid.UUID,
	) (*models.Comment, error)
	FindByIDWithUserRelation(
		ctx context.Context,
		db *gorm.DB,
		commentID uuid.UUID,
	) (*models.Comment, error)
	FindByIDWithUserAndReplyToUserRelations(
		ctx context.Context,
		db *gorm.DB,
		commentID uuid.UUID,
	) (*models.Comment, error)
	FindByIDCursor(
		ctx context.Context,
		db *gorm.DB,
		commentID uuid.UUID,
	) (*Cursor, error)
	FindsByPostIDCursorPaginationWithCommentRelations(
		ctx context.Context,
		db *gorm.DB,
		postID uuid.UUID,
		cursor *Cursor,
		limit int,
	) ([]models.Comment, error)
	Update(
		ctx context.Context,
		db *gorm.DB,
		commentID uuid.UUID,
		updateComment *models.Comment,
	) (*models.Comment, error)
}

type commentRepositoryDB struct {
}

func NewCommentRepositoryDB() CommentRepository {
	return &commentRepositoryDB{}
}

func (r *commentRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	comment *models.Comment,
) error {
	return db.WithContext(ctx).Create(comment).Error
}

func (r *commentRepositoryDB) FindByID(
	ctx context.Context,
	db *gorm.DB,
	commentID uuid.UUID,
) (*models.Comment, error) {
	comment := &models.Comment{}
	err := db.
		WithContext(ctx).
		Where("id = ?", commentID).
		Take(comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *commentRepositoryDB) FindByIDWithUserRelation(
	ctx context.Context,
	db *gorm.DB,
	commentID uuid.UUID,
) (*models.Comment, error) {
	comment := &models.Comment{}
	err := db.
		WithContext(ctx).
		Where("id = ?", commentID).
		Preload("User", helpers.OmitUserPasswordHash).
		Take(comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *commentRepositoryDB) FindByIDWithUserAndReplyToUserRelations(
	ctx context.Context,
	db *gorm.DB,
	commentID uuid.UUID,
) (*models.Comment, error) {
	comment := &models.Comment{}
	err := db.
		WithContext(ctx).
		Where("id = ?", commentID).
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("ReplyToUser", helpers.OmitUserPasswordHash).
		Take(comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *commentRepositoryDB) FindByIDCursor(
	ctx context.Context,
	db *gorm.DB,
	commentID uuid.UUID,
) (*Cursor, error) {
	comment := &models.Comment{}
	err := db.
		WithContext(ctx).
		Where("id = ?", commentID).
		Select("id", "created_at").
		Take(comment).Error
	if err != nil {
		return nil, err
	}
	cursor := &Cursor{
		ID:        comment.ID,
		CreatedAt: comment.CreatedAt,
	}
	return cursor, nil
}

func (r *commentRepositoryDB) FindsByPostIDCursorPaginationWithCommentRelations(
	ctx context.Context,
	db *gorm.DB,
	postID uuid.UUID,
	cursor *Cursor,
	limit int,
) ([]models.Comment, error) {
	comments := &[]models.Comment{}
	db = db.
		WithContext(ctx).
		Where("(post_id = ? AND parent_id IS NULL)", postID).
		Preload("User", helpers.OmitUserPasswordHash).
		Preload("Likes", func(db *gorm.DB) *gorm.DB {
			return db.Order("likes.created_at DESC")
		}).
		Preload("Likes.User", helpers.OmitUserPasswordHash).
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("comments.created_at DESC")
		}).
		Preload("Replies.Likes", func(db *gorm.DB) *gorm.DB {
			return db.Order("likes.created_at DESC")
		}).
		Preload("Replies.Likes.User", helpers.OmitUserPasswordHash).
		Preload("Replies.User", helpers.OmitUserPasswordHash).
		Preload("Replies.ReplyToUser", helpers.OmitUserPasswordHash).
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

	err := db.Find(comments).Error
	if err != nil {
		return nil, err
	}
	return *comments, nil
}

func (r *commentRepositoryDB) Update(
	ctx context.Context,
	db *gorm.DB,
	commentID uuid.UUID,
	updateComment *models.Comment,
) (*models.Comment, error) {
	comment := &models.Comment{}
	err := db.
		WithContext(ctx).
		Model(comment).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "id"},
				{Name: "message"},
				{Name: "post_id"},
			},
		}).
		Where("id = ?", commentID).
		Updates(*updateComment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}
