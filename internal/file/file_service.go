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
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeleteFileRequest struct {
	FileURL string `json:"fileUrl" binding:"required,presignedurl"`
}

type DeleteFileDTO struct {
	FileURL  string
	FileType FileType
}

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
	DeleteFile(deleteFileDTO *DeleteFileDTO) error
	PresignGetFile(contentID uuid.UUID) (string, error)
	PresignGetFiles(contentID uuid.UUID) ([]string, error)
}

type fileService struct {
	db             *gorm.DB
	fileRepository FileRepository
	s3Client       *s3.Client
	presignClient  *s3.PresignClient
}

func NewFileService(
	db *gorm.DB,
	fileRepository FileRepository,
	s3Client *s3.Client,
	presignClient *s3.PresignClient,
) FileService {
	return &fileService{
		db:             db,
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
	err = s.fileRepository.Create(s.db, createFile)
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

	err := s.fileRepository.CreateMany(s.db, createFiles)
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

func (s *fileService) DeleteFile(deleteFileDTO *DeleteFileDTO) error {
	fileDIR, filename, err := helpers.SplitPresignedURL(deleteFileDTO.FileURL)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
	}
	filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
	file, err := s.fileRepository.FindByFilenameType(s.db, filePath, deleteFileDTO.FileType)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to find file")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return errs.NewNotFoundErrorWithMessage(fmt.Sprintf("File %s is not found", deleteFileDTO.FileURL))
	}

	_, err = helpers.DeleteObject(s.s3Client, context.Background(), file.Filename)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to delete file object")
	}

	err = s.fileRepository.Delete(
		s.db,
		file.ID,
		file.Filename,
		file.FileType,
	)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to delete file")
	}

	return nil
}

func (s *fileService) PresignGetFile(contentID uuid.UUID) (string, error) {
	file, err := s.fileRepository.FindByContentID(s.db, contentID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to find file")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", errs.NewNotFoundErrorWithMessage(fmt.Sprintf("File by content id %v not found", contentID))
	}

	ctx := context.Background()
	req, err := helpers.PresignGetObject(s.presignClient, ctx, file.Filename)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
	}

	return req.URL, nil
}

func (s *fileService) PresignGetFiles(contentID uuid.UUID) ([]string, error) {
	files, err := s.fileRepository.FindsByContentID(s.db, contentID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find files")
	}

	filesURL := []string{}
	ctx := context.Background()
	if len(files) > 0 {
		for _, file := range files {
			req, err := helpers.PresignGetObject(s.presignClient, ctx, file.Filename)
			if err != nil {
				logs.Error(err)
				return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
			}
			filesURL = append(filesURL, req.URL)
		}
	}

	return filesURL, nil
}
