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
	Filename  string     `gorm:"uniqueIndex:idx_file_unique"`
	ContentID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_file_unique"`
	FileType  FileType   `gorm:"type:file_type;uniqueIndex:idx_file_unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FileRepository interface {
	Create(file *File) error
	CreateMany(files []File) error
}

type fileRepositoryDB struct {
	db *gorm.DB
}

func NewFileRepositoryDB(db *gorm.DB) FileRepository {
	db.AutoMigrate(File{})
	return &fileRepositoryDB{db: db}
}

func (r *fileRepositoryDB) Create(file *File) error {
	return r.db.Create(file).Error
}

func (r *fileRepositoryDB) CreateMany(files []File) error {
	return r.db.Create(&files).Error
}
