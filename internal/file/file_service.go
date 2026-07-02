package file

import (
	"mime/multipart"

	"github.com/belllllx/social-media-go/pkg/errs"
)

var allowedFileTypes = map[string]bool{
	"image/png":  true,
	"image/jpg":  true,
	"image/jpeg": true,
	"image/webp": true,
	"video/mp4":  true,
}

type FileService interface {
	CreateFile(file *multipart.FileHeader, fileType FileType) (fileURL string, err error)
	CreateFiles(files []*multipart.FileHeader, fileType FileType) (filesURL []string, err error)
}

type fileService struct {
	fileRepository FileRepository
}

func NewFileService(fileRepository FileRepository) FileService {
	return &fileService{fileRepository: fileRepository}
}

func (s *fileService) CreateFile(file *multipart.FileHeader, fileType FileType) (string, error) {
	if !allowedFileTypes[file.Header.Get("Content-Type")] {
		return "", errs.NewBadRequestErrorWithMessage("Invalid file type")
	}

	panic("")
}

func (s *fileService) CreateFiles(files []*multipart.FileHeader, fileType FileType) ([]string, error) {
	panic("")
}
