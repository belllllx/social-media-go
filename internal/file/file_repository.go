package file

import (
	"context"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		file *models.File,
	) error
	CreateMany(
		ctx context.Context,
		db *gorm.DB,
		files []models.File,
	) error
	FindByContentID(
		ctx context.Context,
		db *gorm.DB,
		contentID uuid.UUID,
	) (*models.File, error)
	FindsByContentID(
		ctx context.Context,
		db *gorm.DB,
		contentID uuid.UUID,
	) ([]models.File, error)
	FindByFilenameType(
		ctx context.Context,
		db *gorm.DB,
		filename string,
		fileType models.FileType,
	) (*models.File, error)
	UpdateContentID(
		ctx context.Context,
		db *gorm.DB,
		contentID uuid.UUID,
		filename string,
		fileType models.FileType,
	) error
	Delete(
		ctx context.Context,
		db *gorm.DB,
		id int64,
		filename string,
		fileType models.FileType,
	) error
	DeleteMany(
		ctx context.Context,
		db *gorm.DB,
		files []models.File,
	) error
}

type fileRepositoryDB struct {
}

func NewFileRepositoryDB() FileRepository {
	return &fileRepositoryDB{}
}

func (r *fileRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	file *models.File,
) error {
	return db.WithContext(ctx).Create(file).Error
}

func (r *fileRepositoryDB) CreateMany(
	ctx context.Context,
	db *gorm.DB,
	files []models.File,
) error {
	return db.WithContext(ctx).Create(&files).Error
}

func (r *fileRepositoryDB) FindByContentID(
	ctx context.Context,
	db *gorm.DB,
	contentID uuid.UUID,
) (*models.File, error) {
	file := &models.File{}
	err := db.
		WithContext(ctx).
		Where("content_id = ?", contentID).
		Take(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepositoryDB) FindsByContentID(
	ctx context.Context,
	db *gorm.DB,
	contentID uuid.UUID,
) ([]models.File, error) {
	files := &[]models.File{}
	err := db.
		WithContext(ctx).
		Where("content_id = ?", contentID).
		Find(files).Error
	if err != nil {
		return nil, err
	}
	return *files, nil
}

func (r *fileRepositoryDB) FindByFilenameType(
	ctx context.Context,
	db *gorm.DB,
	filename string,
	fileType models.FileType,
) (*models.File, error) {
	file := &models.File{}
	err := db.
		WithContext(ctx).
		Where("filename = ? AND file_type = ?", filename, fileType).
		Take(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepositoryDB) UpdateContentID(
	ctx context.Context,
	db *gorm.DB,
	contentID uuid.UUID,
	filename string,
	fileType models.FileType,
) error {
	file := &models.File{}
	return db.
		WithContext(ctx).
		Model(file).
		Where("filename = ? AND file_type = ?", filename, fileType).
		Update("content_id", contentID).Error
}

func (r *fileRepositoryDB) Delete(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	filename string,
	fileType models.FileType,
) error {
	file := &models.File{}
	return db.
		WithContext(ctx).
		Where("id = ? AND filename = ? AND file_type = ?", id, filename, fileType).
		Delete(file).Error
}

func (r *fileRepositoryDB) DeleteMany(
	ctx context.Context,
	db *gorm.DB,
	files []models.File,
) error {
	return db.WithContext(ctx).Delete(&files).Error
}
