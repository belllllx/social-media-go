package models

import (
	"time"

	"github.com/google/uuid"
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
