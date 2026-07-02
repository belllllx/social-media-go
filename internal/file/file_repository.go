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
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Filename  string     `gorm:"unique"`
	ContentID *uuid.UUID `gorm:"type:uuid;unique"`
	FileType  FileType   `gorm:"type:file_type;unique"`
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
