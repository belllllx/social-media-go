package post

import (
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository interface {
	Create(post *user.Post) error
	PreloadRelations(postID uuid.UUID) (*user.Post, error)
	FindByIDPreloadRelation(postID uuid.UUID) (*user.Post, error)
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
