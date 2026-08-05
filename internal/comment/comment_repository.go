package comment

import (
	"context"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
