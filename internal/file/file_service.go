package file

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
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
	FileType models.FileType
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

type FileDataDTO struct {
	Filename    string
	ContentType string
	Body        io.Reader
	Size        int64
}

type FileService interface {
	UploadFile(
		ctx context.Context,
		fileDataDTO *FileDataDTO,
		fileType models.FileType,
	) (*FileURL, error)
	UploadFiles(
		ctx context.Context,
		filesDataDTO []FileDataDTO,
		fileType models.FileType,
	) (*FilesURL, error)
	DeleteFile(ctx context.Context, deleteFileDTO *DeleteFileDTO) error
	PresignGetFile(ctx context.Context, contentID uuid.UUID) (string, error)
	PresignGetFiles(ctx context.Context, contentID uuid.UUID) ([]string, error)
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

func (s *fileService) UploadFile(
	ctx context.Context,
	fileDataDTO *FileDataDTO,
	fileType models.FileType,
) (*FileURL, error) {
	if fileType != models.FileTypeComment && fileType != models.FileTypeReply {
		logs.Warn("Failed to upload file invalid file type")
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to upload file invalid file type")
	}

	if !allowedContentTypesCreateFile[fileDataDTO.ContentType] {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid file type")
	}

	if fileDataDTO.Size > maxFileSize {
		return nil, errs.NewBadRequestErrorWithMessage("File size exceeds 30 mb")
	}

	newFileName := helpers.GenerateFilename(fileDataDTO.Filename)
	key := ""
	switch fileType {
	case models.FileTypeComment:
		key = fmt.Sprintf("%s/%s", "comment-image", newFileName)
	case models.FileTypeReply:
		key = fmt.Sprintf("%s/%s", "reply-image", newFileName)
	}

	_, err := helpers.PutObject(
		ctx,
		s.s3Client,
		key,
		fileDataDTO.Body,
		fileDataDTO.ContentType,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to upload file to bucket")
	}

	req, err := helpers.PresignGetObject(
		ctx,
		s.presignClient,
		key,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
	}

	createFile := &models.File{
		Filename: key,
		FileType: fileType,
	}
	err = s.fileRepository.Create(
		ctx,
		s.db,
		createFile,
	)
	if err != nil {
		logs.Error(err)

		if _, err := helpers.DeleteObject(
			ctx,
			s.s3Client,
			key,
		); err != nil {
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

func (s *fileService) UploadFiles(
	ctx context.Context,
	filesDataDTO []FileDataDTO,
	fileType models.FileType,
) (*FilesURL, error) {
	if fileType != models.FileTypePost {
		logs.Warn("Failed to upload files invalid file type")
		return nil, errs.NewUnexpectedErrorWithMessage("Failed to upload files invalid file type")
	}

	for _, fileDataDTO := range filesDataDTO {
		if !allowedContentTypesCreateFiles[fileDataDTO.ContentType] {
			return nil, errs.NewBadRequestErrorWithMessage("Invalid file type")
		}

		if fileDataDTO.Size > maxFileSize {
			return nil, errs.NewBadRequestErrorWithMessage("File size exceeds 30 mb")
		}
	}

	keys := []string{}
	filesURL := []string{}
	createFiles := []models.File{}

	for _, fileDataDTO := range filesDataDTO {
		newFileName := helpers.GenerateFilename(fileDataDTO.Filename)
		key := ""
		if strings.Contains(fileDataDTO.ContentType, "image") {
			key = fmt.Sprintf("%s/%s", "post-image", newFileName)
		} else {
			key = fmt.Sprintf("%s/%s", "post-video", newFileName)
		}

		_, err := helpers.PutObject(
			ctx,
			s.s3Client,
			key,
			fileDataDTO.Body,
			fileDataDTO.ContentType,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to upload files to bucket")
		}

		req, err := helpers.PresignGetObject(
			ctx,
			s.presignClient,
			key,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
		}

		keys = append(keys, key)
		filesURL = append(filesURL, req.URL)
		createFiles = append(createFiles, models.File{
			Filename: key,
			FileType: fileType,
		})
	}

	err := s.fileRepository.CreateMany(
		ctx,
		s.db,
		createFiles,
	)
	if err != nil {
		logs.Error(err)

		if _, err := helpers.DeleteObjects(
			ctx,
			s.s3Client,
			keys,
		); err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to create files and delete object")
		}
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create files")
	}

	filesURLData := &FilesURL{
		FilesURL: filesURL,
	}

	return filesURLData, nil
}

func (s *fileService) DeleteFile(ctx context.Context, deleteFileDTO *DeleteFileDTO) error {
	fileDIR, filename, err := helpers.SplitPresignedURL(deleteFileDTO.FileURL)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
	}

	filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
	file, err := s.fileRepository.FindByFilenameType(
		ctx,
		s.db,
		filePath,
		deleteFileDTO.FileType,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to find file")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return errs.NewNotFoundErrorWithMessage(fmt.Sprintf("File %s is not found", deleteFileDTO.FileURL))
	}

	_, err = helpers.DeleteObject(
		ctx,
		s.s3Client,
		file.Filename,
	)
	if err != nil {
		logs.Error(err)
		return errs.NewInternalServerErrorWithMessage("Failed to delete file object")
	}

	err = s.fileRepository.Delete(
		ctx,
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

func (s *fileService) PresignGetFile(ctx context.Context, contentID uuid.UUID) (string, error) {
	file, err := s.fileRepository.FindByContentID(
		ctx,
		s.db,
		contentID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to find file")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", errs.NewNotFoundErrorWithMessage(fmt.Sprintf("File by content id %v not found", contentID))
	}

	req, err := helpers.PresignGetObject(
		ctx,
		s.presignClient,
		file.Filename,
	)
	if err != nil {
		logs.Error(err)
		return "", errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
	}

	return req.URL, nil
}

func (s *fileService) PresignGetFiles(ctx context.Context, contentID uuid.UUID) ([]string, error) {
	files, err := s.fileRepository.FindsByContentID(
		ctx,
		s.db,
		contentID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find files")
	}

	filesURL := []string{}
	if len(files) > 0 {
		for _, file := range files {
			req, err := helpers.PresignGetObject(
				ctx,
				s.presignClient,
				file.Filename,
			)
			if err != nil {
				logs.Error(err)
				return nil, errs.NewInternalServerErrorWithMessage("Failed to presign get file object")
			}
			filesURL = append(filesURL, req.URL)
		}
	}

	return filesURL, nil
}
