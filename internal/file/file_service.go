package file

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/configs"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/spf13/viper"
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
}

type fileService struct {
	fileRepository FileRepository
	s3Client       *s3.Client
	presignClient  *s3.PresignClient
}

func NewFileService(fileRepository FileRepository) FileService {
	s3Client := configs.InitS3Client()
	presignClient := s3.NewPresignClient(s3Client)

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
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(viper.GetString("app.aws_bucket_name")),
		Key:         aws.String(key),
		Body:        fileData.Body,
		ContentType: aws.String(fileData.ContentType),
	})
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to upload file to bucket")
	}

	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24
	})
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to presign object")
	}

	createFile := &File{
		Filename: key,
		FileType: fileType,
	}
	err = s.fileRepository.Create(createFile)
	if err != nil {
		logs.Error(err)

		if _, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
			Key:    aws.String(key),
		}); err != nil {
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

		_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(viper.GetString("app.aws_bucket_name")),
			Key:         aws.String(key),
			Body:        fileData.Body,
			ContentType: aws.String(fileData.ContentType),
		})
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to upload files to bucket")
		}

		req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = time.Hour * 24
		})
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to presign object")
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
			if _, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(viper.GetString("app.aws_bucket_name")),
				Key:    aws.String(key),
			}); err != nil {
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
