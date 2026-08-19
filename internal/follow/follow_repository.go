package follow

import (
	"context"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		follow *models.Follow,
	) error
	FindIsFollowing(
		ctx context.Context,
		db *gorm.DB,
		followerID,
		followingID uuid.UUID,
	) (*models.Follow, error)
	FindByIDWithFollowingAndFollowerRelations(
		ctx context.Context,
		db *gorm.DB,
		followID int64,
	) (*models.Follow, error)
	DeleteOfFollow(
		ctx context.Context,
		db *gorm.DB,
		followerID,
		followingID uuid.UUID,
	) (*models.Follow, error)
}

type followRepositoryDB struct {
}

func NewFollowRepositoryDB() FollowRepository {
	return &followRepositoryDB{}
}

func (r *followRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	follow *models.Follow,
) error {
	return db.WithContext(ctx).Create(follow).Error
}

func (r *followRepositoryDB) FindIsFollowing(
	ctx context.Context,
	db *gorm.DB,
	followerID,
	followingID uuid.UUID,
) (*models.Follow, error) {
	follow := &models.Follow{}
	err := db.
		WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Take(follow).Error
	if err != nil {
		return nil, err
	}
	return follow, nil
}

func (r *followRepositoryDB) FindByIDWithFollowingAndFollowerRelations(
	ctx context.Context,
	db *gorm.DB,
	followID int64,
) (*models.Follow, error) {
	follow := &models.Follow{}
	err := db.
		WithContext(ctx).
		Where("id = ?", followID).
		Preload("Follower", helpers.OmitUserPasswordHash).
		Preload("Follower.Followers").
		Preload("Following", helpers.OmitUserPasswordHash).
		Preload("Following.Followers").
		Take(follow).Error
	if err != nil {
		return nil, err
	}
	return follow, nil
}

func (r *followRepositoryDB) DeleteOfFollow(
	ctx context.Context,
	db *gorm.DB,
	followerID,
	followingID uuid.UUID,
) (*models.Follow, error) {
	follow := &models.Follow{}
	err := db.
		WithContext(ctx).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "id"},
				{Name: "follower_id"},
				{Name: "following_id"},
			},
		}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(follow).Error
	if err != nil {
		return nil, err
	}
	return follow, nil
}
