package file

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileType string

const (
	FileTypePost    FileType = "POST"
	FileTypeComment FileType = "COMMENT"
	FileTypeReply   FileType = "REPLY"
)

type File struct {
	ID        int64
	Filename  string     `gorm:"uniqueIndex:idx_files_unique"`
	ContentID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_files_unique"`
	FileType  FileType   `gorm:"type:file_type;uniqueIndex:idx_files_unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FileRepository interface {
	Create(db *gorm.DB, file *File) error
	CreateMany(db *gorm.DB, files []File) error
	UpdateContentID(db *gorm.DB, contentID uuid.UUID, filename string, fileType FileType) error
	FindByContentID(db *gorm.DB, contentID uuid.UUID) (*File, error)
	FindsByContentID(db *gorm.DB, contentID uuid.UUID) ([]File, error)
	FindByFilenameType(db *gorm.DB, filename string, fileType FileType) (*File, error)
	Delete(db *gorm.DB, id int64, filename string, fileType FileType) error
}

type fileRepositoryDB struct {
}

func NewFileRepositoryDB(db *gorm.DB) FileRepository {
	db.AutoMigrate(File{})
	return &fileRepositoryDB{}
}

func (r *fileRepositoryDB) Create(db *gorm.DB, file *File) error {
	return db.Create(file).Error
}

func (r *fileRepositoryDB) CreateMany(db *gorm.DB, files []File) error {
	return db.Create(&files).Error
}

func (r *fileRepositoryDB) UpdateContentID(
	db *gorm.DB,
	contentID uuid.UUID,
	filename string,
	fileType FileType,
) error {
	file := &File{}
	return db.Model(file).Where("filename = ? AND file_type = ?", filename, fileType).Update("content_id", contentID).Error
}

func (r *fileRepositoryDB) FindByContentID(db *gorm.DB, contentID uuid.UUID) (*File, error) {
	file := &File{}
	err := db.Where("id = ?", contentID).Take(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepositoryDB) FindsByContentID(db *gorm.DB, contentID uuid.UUID) ([]File, error) {
	files := &[]File{}
	err := db.Where("content_id = ?", contentID).Find(files).Error
	if err != nil {
		return nil, err
	}
	return *files, nil
}

func (r *fileRepositoryDB) FindByFilenameType(
	db *gorm.DB,
	filename string,
	fileType FileType,
) (*File, error) {
	file := &File{}
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
	fileType FileType,
) error {
	file := &File{}
	return db.Where("id = ? AND filename = ? AND file_type = ?", id, filename, fileType).Delete(file).Error
}
