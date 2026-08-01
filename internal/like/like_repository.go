package like

import (
	"context"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		like *models.Like,
	) error
	FindOfPost(
		ctx context.Context,
		db *gorm.DB,
		userID,
		postID uuid.UUID,
	) (*models.Like, error)
	FindOfPostWithUserRelation(
		ctx context.Context,
		db *gorm.DB,
		userID,
		postID uuid.UUID,
	) (*models.Like, error)
	DeleteOfPost(
		ctx context.Context,
		db *gorm.DB,
		userID,
		postID uuid.UUID,
	) (*models.Like, error)
}

type likeRepositoryDB struct {
}

func NewLikeRepositoryDB() LikeRepository {
	return &likeRepositoryDB{}
}

func (r *likeRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	like *models.Like,
) error {
	return db.WithContext(ctx).Create(like).Error
}

func (r *likeRepositoryDB) FindOfPost(
	ctx context.Context,
	db *gorm.DB,
	userID,
	postID uuid.UUID,
) (*models.Like, error) {
	like := &models.Like{}
	err := db.
		WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Take(like).Error
	if err != nil {
		return nil, err
	}
	return like, nil
}

func (r *likeRepositoryDB) FindOfPostWithUserRelation(
	ctx context.Context,
	db *gorm.DB,
	userID,
	postID uuid.UUID,
) (*models.Like, error) {
	like := &models.Like{}
	err := db.
		WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Preload("User", helpers.OmitUserPasswordHash).
		Take(like).Error
	if err != nil {
		return nil, err
	}
	return like, nil
}

func (r *likeRepositoryDB) DeleteOfPost(
	ctx context.Context,
	db *gorm.DB,
	userID,
	postID uuid.UUID,
) (*models.Like, error) {
	like := &models.Like{}
	err := db.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(like).Error
	if err != nil {
		return nil, err
	}
	return like, nil
}
