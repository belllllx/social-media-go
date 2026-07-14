package post

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/socket"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
)

type CreatePostDTO struct {
	Message  string
	FilesURL []string
	UserID   uuid.UUID
}

type CreatedPost struct {
	ID            uuid.UUID           `json:"id"`
	Message       *string             `json:"message"`
	UserID        uuid.UUID           `json:"userId"`
	User          user.SecureUser     `json:"user"`
	ParentID      *uuid.UUID          `json:"parentId"`
	Likes         []socket.LikeDTO    `json:"likes"`
	Comments      []socket.CommentDTO `json:"comments"`
	FilesURL      []string            `json:"filesUrl,omitempty"`
	CommentsCount int                 `json:"commentsCount"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}

type PostService interface {
	CreatePost(createPostDTO *CreatePostDTO) (*CreatedPost, error)
}

type postService struct {
	postRepository      PostRepository
	userRepository      user.UserRepository
	fileRepository      file.FileRepository
	userService         user.UserService
	notificationService notification.NotificationService
	fileService         file.FileService
	notificationSocket  socket.NotificationSocket
	postSocket          socket.PostSocket
}

func NewPostService(
	postRepository PostRepository,
	userRepository user.UserRepository,
	fileRepository file.FileRepository,
	userService user.UserService,
	notificationService notification.NotificationService,
	fileService file.FileService,
	notificationSocket socket.NotificationSocket,
	postSocket socket.PostSocket,
) PostService {
	return &postService{
		postRepository:      postRepository,
		userRepository:      userRepository,
		fileRepository:      fileRepository,
		userService:         userService,
		notificationService: notificationService,
		fileService:         fileService,
		notificationSocket:  notificationSocket,
		postSocket:          postSocket,
	}
}

func (s *postService) CreatePost(createPostDTO *CreatePostDTO) (*CreatedPost, error) {
	if createPostDTO.Message == "" && len(createPostDTO.FilesURL) == 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Create post must contains with message or files")
	}

	userByID, err := s.userRepository.FindByID(createPostDTO.UserID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User id %v is not found", createPostDTO.UserID))
	}

	createPost := &user.Post{
		Message: &createPostDTO.Message,
		UserID:  userByID.ID,
	}
	err = s.postRepository.Create(createPost)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create post")
	}

	post, err := s.postRepository.PreloadRelations(createPost.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post with relations")
	}

	err = s.userService.GetPostUserImage(post)
	if err != nil {
		return nil, err
	}

	createNotificationsDTO := &notification.CreateNotificationsDTO{
		Type:     user.NotificationTypePost,
		Message:  "Create a new post",
		SenderID: userByID.ID,
		PostID:   &createPost.ID,
	}
	notifications, err := s.notificationService.CreateNotifications(createNotificationsDTO)
	if err != nil {
		return nil, err
	}

	if notifications != nil {
		broadcastNotificationsDTO := []socket.BroadcastNotificationsDTO{}
		for _, notification := range notifications {
			broadcastNotificationsDTO = append(broadcastNotificationsDTO, socket.BroadcastNotificationsDTO{
				ID:         notification.ID,
				Type:       notification.Type,
				Message:    notification.Message,
				IsRead:     notification.IsRead,
				SenderID:   notification.SenderID,
				ReceiverID: notification.ReceiverID,
				PostID:     notification.PostID,
				CommentID:  notification.CommentID,
				CreatedAt:  notification.CreatedAt,
				UpdatedAt:  notification.UpdatedAt,
			})
		}
		go s.notificationSocket.BroadcastNotifications(broadcastNotificationsDTO)
	}

	secureUser := user.SecureUser{
		ID:                   post.User.ID,
		Fullname:             post.User.Fullname,
		Username:             post.User.Username,
		Email:                post.User.Email,
		DateOfBirth:          post.User.DateOfBirth,
		ProfileUrl:           post.User.ProfileUrl,
		ProfileBackgroundUrl: post.User.ProfileBackgroundUrl,
		Info:                 post.User.Info,
		Role:                 post.User.Role,
		ProviderType:         post.User.ProviderType,
		CreatedAt:            post.User.CreatedAt,
		UpdatedAt:            post.User.UpdatedAt,
	}
	likesDTO := []socket.LikeDTO{}
	for _, like := range post.Likes {
		likesDTO = append(likesDTO, socket.LikeDTO{
			ID:        like.ID,
			UserID:    like.UserID,
			PostID:    like.PostID,
			CommentID: like.CommentID,
			CreatedAt: like.CreatedAt,
			UpdatedAt: like.UpdatedAt,
		})
	}
	commentsDTO := []socket.CommentDTO{}
	for _, comment := range post.Comments {
		commentsDTO = append(commentsDTO, socket.CommentDTO{
			ID:            comment.ID,
			Message:       comment.Message,
			UserID:        comment.UserID,
			PostID:        comment.PostID,
			ParentID:      comment.ParentID,
			ReplyID:       comment.ReplyID,
			ReplyToUserID: comment.ReplyToUserID,
			CreatedAt:     comment.CreatedAt,
			UpdatedAt:     comment.UpdatedAt,
		})
	}

	// กรณีไม่มี files
	if len(createPostDTO.FilesURL) == 0 {
		broadcastPostDTO := &socket.BroadcastPostDTO{
			ID:            post.ID,
			Message:       post.Message,
			UserID:        post.UserID,
			User:          secureUser,
			ParentID:      post.ParentID,
			Likes:         likesDTO,
			Comments:      commentsDTO,
			CommentsCount: len(post.Comments),
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
		}
		go s.postSocket.BroadcastPost(broadcastPostDTO)

		respPost := &CreatedPost{
			ID:            post.ID,
			Message:       post.Message,
			UserID:        post.UserID,
			User:          secureUser,
			ParentID:      post.ParentID,
			Likes:         likesDTO,
			Comments:      commentsDTO,
			CommentsCount: len(post.Comments),
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
		}
		return respPost, nil
	}

	// กรณีมี files
	for _, fileURL := range createPostDTO.FilesURL {
		fileDIR, filename, err := helpers.SplitPresignedURL(fileURL)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
		}
		filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
		err = s.fileRepository.UpdateContentID(createPost.ID, filePath, file.FileTypePost)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to update file of post")
		}
	}

	files, err := s.fileRepository.FindsByContentID(createPost.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find files of post")
	}

	filesURL := []string{}
	for _, file := range files {
		fileURL, err := s.fileService.PresignGetFile(file.Filename)
		if err != nil {
			return nil, err
		}

		filesURL = append(filesURL, fileURL)
	}

	broadcastPostWithFilesDTO := &socket.BroadcastPostWithFilesDTO{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         likesDTO,
		Comments:      commentsDTO,
		FilesURL:      filesURL,
		CommentsCount: len(post.Comments),
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}
	go s.postSocket.BroadcastPostWithFiles(broadcastPostWithFilesDTO)

	respPostWithFiles := &CreatedPost{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         likesDTO,
		Comments:      commentsDTO,
		FilesURL:      filesURL,
		CommentsCount: len(post.Comments),
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}
	return respPostWithFiles, nil
}
