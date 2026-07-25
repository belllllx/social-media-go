package file

import (
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileRepository interface {
	Create(db *gorm.DB, file *models.File) error
	CreateMany(db *gorm.DB, files []models.File) error
	UpdateContentID(db *gorm.DB, contentID uuid.UUID, filename string, fileType models.FileType) error
	FindByContentID(db *gorm.DB, contentID uuid.UUID) (*models.File, error)
	FindsByContentID(db *gorm.DB, contentID uuid.UUID) ([]models.File, error)
	FindByFilenameType(db *gorm.DB, filename string, fileType models.FileType) (*models.File, error)
	Delete(db *gorm.DB, id int64, filename string, fileType models.FileType) error
}

type fileRepositoryDB struct {
}

func NewFileRepositoryDB() FileRepository {
	return &fileRepositoryDB{}
}

func (r *fileRepositoryDB) Create(db *gorm.DB, file *models.File) error {
	return db.Create(file).Error
}

func (r *fileRepositoryDB) CreateMany(db *gorm.DB, files []models.File) error {
	return db.Create(&files).Error
}

func (r *fileRepositoryDB) UpdateContentID(
	db *gorm.DB,
	contentID uuid.UUID,
	filename string,
	fileType models.FileType,
) error {
	file := &models.File{}
	return db.Model(file).Where("filename = ? AND file_type = ?", filename, fileType).Update("content_id", contentID).Error
}

func (r *fileRepositoryDB) FindByContentID(db *gorm.DB, contentID uuid.UUID) (*models.File, error) {
	file := &models.File{}
	err := db.Where("id = ?", contentID).Take(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepositoryDB) FindsByContentID(db *gorm.DB, contentID uuid.UUID) ([]models.File, error) {
	files := &[]models.File{}
	err := db.Where("content_id = ?", contentID).Find(files).Error
	if err != nil {
		return nil, err
	}
	return *files, nil
}

func (r *fileRepositoryDB) FindByFilenameType(
	db *gorm.DB,
	filename string,
	fileType models.FileType,
) (*models.File, error) {
	file := &models.File{}
	err := db.Where("filename = ? AND file_type = ?", filename, fileType).Take(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepositoryDB) Delete(
	db *gorm.DB,
	id int64,
	filename string,
	fileType models.FileType,
) error {
	file := &models.File{}
	return db.Where("id = ? AND filename = ? AND file_type = ?", id, filename, fileType).Delete(file).Error
}
