package file

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
)

const maxFileSize = 30 << 20 // 30 MB

var allowedContentTypesCreateFile = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

var allowedContentTypesCreateFiles = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"video/mp4":  true,
}

type FileURL struct {
	FileURL string `json:"fileUrl"`
}

type FilesURL struct {
	FilesURL []string `json:"filesUrl"`
}

type FileData struct {
	Filename    string
	ContentType string
	Body        io.Reader
	Size        int64
}

type FileService interface {
	UploadFile(fileData FileData, fileType FileType) (*FileURL, error)
	UploadFiles(filesData []FileData, fileType FileType) (*FilesURL, error)
	PresignGetFile(filename string) (string, error)
}

type fileService struct {
	fileRepository FileRepository
	s3Client       *s3.Client
	presignClient  *s3.PresignClient
}

func NewFileService(
	fileRepository FileRepository,
	s3Client *s3.Client,
	presignClient *s3.PresignClient,
) FileService {
	return &fileService{
		fileRepository: fileRepository,
		s3Client:       s3Client,
		presignClient:  presignClient,
	}
}

func (s *fileService) UploadFile(fileData FileData, fileType FileType) (*FileURL, error) {
	if fileType != FileTypeComment && fileType != FileTypeReply {
		logs.Warn("Failed to upload file invalid file type")
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to upload file invalid file type")
	}

	if !allowedContentTypesCreateFile[fileData.ContentType] {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid file type")
	}

	if fileData.Size > maxFileSize {
		return nil, errs.NewBadRequestErrorWithMessage("File size exceeds 30 mb")
	}

	newFileName := helpers.GenerateFilename(fileData.Filename)
	key := ""
	switch fileType {
	case FileTypeComment:
		key = fmt.Sprintf("%s/%s", "comment-image", newFileName)
	case FileTypeReply:
		key = fmt.Sprintf("%s/%s", "reply-image", newFileName)
	}

	ctx := context.Background()
	_, err := helpers.PutObject(
		s.s3Client,
		ctx,
		key,
		fileData.Body,
		fileData.ContentType,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to upload file to bucket")
	}

	req, err := helpers.PresignGetObject(s.presignClient, ctx, key)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
	}

	createFile := &File{
		Filename: key,
		FileType: fileType,
	}
	err = s.fileRepository.Create(createFile)
	if err != nil {
		logs.Error(err)

		if _, err := helpers.DeleteObject(s.s3Client, ctx, key); err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to create file and delete object")
		}
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create file")
	}

	fileURL := &FileURL{
		FileURL: req.URL,
	}
	return fileURL, nil
}

func (s *fileService) UploadFiles(filesData []FileData, fileType FileType) (*FilesURL, error) {
	if fileType != FileTypePost {
		logs.Warn("Failed to upload files invalid file type")
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to upload files invalid file type")
	}

	for _, fileData := range filesData {
		if !allowedContentTypesCreateFiles[fileData.ContentType] {
			return nil, errs.NewBadRequestErrorWithMessage("Invalid file type")
		}

		if fileData.Size > maxFileSize {
			return nil, errs.NewBadRequestErrorWithMessage("File size exceeds 30 mb")
		}
	}

	keys := []string{}
	filesURL := []string{}
	createFiles := []File{}
	ctx := context.Background()

	for _, fileData := range filesData {
		newFileName := helpers.GenerateFilename(fileData.Filename)
		key := ""
		if strings.Contains(fileData.ContentType, "image") {
			key = fmt.Sprintf("%s/%s", "post-image", newFileName)
		} else {
			key = fmt.Sprintf("%s/%s", "post-video", newFileName)
		}

		_, err := helpers.PutObject(
			s.s3Client,
			ctx,
			key,
			fileData.Body,
			fileData.ContentType,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to upload files to bucket")
		}

		req, err := helpers.PresignGetObject(s.presignClient, ctx, key)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
		}

		keys = append(keys, key)
		filesURL = append(filesURL, req.URL)
		createFiles = append(createFiles, File{
			Filename: key,
			FileType: fileType,
		})
	}

	err := s.fileRepository.CreateMany(createFiles)
	if err != nil {
		logs.Error(err)

		for _, key := range keys {
			if _, err := helpers.DeleteObject(s.s3Client, ctx, key); err != nil {
				logs.Error(err)
				return nil, errs.NewInternalServerErrorWithMessage("Failed to create files and delete object")
			}
		}
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create files")
	}

	filesURLData := &FilesURL{
		FilesURL: filesURL,
	}

	return filesURLData, nil
}

func (s *fileService) PresignGetFile(filename string) (string, error) {
	ctx := context.Background()
	req, err := helpers.PresignGetObject(s.presignClient, ctx, filename)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
	}

	return req.URL, nil
}
