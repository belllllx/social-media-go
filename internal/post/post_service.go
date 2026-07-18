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

type CreateSharePostDTO struct {
	Message  string
	UserID   uuid.UUID
	ParentID string
}

type CreatePostDTO struct {
	Message  string
	FilesURL []string
	UserID   uuid.UUID
}

type Like struct {
	ID        int64      `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	PostID    *uuid.UUID `json:"postId"`
	CommentID *uuid.UUID `json:"commentId"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Comment struct {
	ID            uuid.UUID  `json:"id"`
	Message       *string    `json:"message"`
	UserID        uuid.UUID  `json:"userId"`
	PostID        uuid.UUID  `json:"postId"`
	ParentID      *uuid.UUID `json:"parentId"`
	ReplyID       *uuid.UUID `json:"replyId"`
	ReplyToUserID *uuid.UUID `json:"replyToUserId"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type PostParent struct {
	ID        uuid.UUID        `json:"id"`
	Message   *string          `json:"message"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user"`
	ParentID  *uuid.UUID       `json:"parentId"`
	FilesURL  []string         `json:"filesUrl"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Post struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	UserID        uuid.UUID        `json:"userId"`
	User          *user.SecureUser `json:"user"`
	ParentID      *uuid.UUID       `json:"parentId"`
	Parent        *PostParent      `json:"parent"`
	Likes         []Like           `json:"likes"`
	FilesURL      []string         `json:"filesUrl,omitempty"`
	CommentsCount int              `json:"commentsCount"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type CreatedSharePost struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	UserID        uuid.UUID        `json:"userId"`
	User          *user.SecureUser `json:"user"`
	ParentID      *uuid.UUID       `json:"parentId"`
	Parent        *PostParent      `json:"parent"`
	Likes         []Like           `json:"likes"`
	Comments      []Comment        `json:"comments"`
	CommentsCount int              `json:"commentsCount"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type CreatedPost struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	UserID        uuid.UUID        `json:"userId"`
	User          *user.SecureUser `json:"user"`
	ParentID      *uuid.UUID       `json:"parentId"`
	Likes         []Like           `json:"likes"`
	Comments      []Comment        `json:"comments"`
	FilesURL      []string         `json:"filesUrl,omitempty"`
	CommentsCount int              `json:"commentsCount"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type PostService interface {
	CreatePost(createPostDTO *CreatePostDTO) (*CreatedPost, error)
	CreateSharePost(createSharePostDTO *CreateSharePostDTO) (*CreatedSharePost, error)
	FindsCursorPagination(cursor *uuid.UUID, limit int) ([]Post, error)
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

	userByID, err := s.userService.SecureFindWithID(createPostDTO.UserID)
	if err != nil {
		return nil, err
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

	usersExcept, err := s.userRepository.FindsByIDExcept(userByID.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find users by id except")
	}

	filesURL := []string{}

	// กรณีมี files
	if len(createPostDTO.FilesURL) > 0 {
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

		for _, file := range files {
			fileURL, err := s.fileService.PresignGetFile(file.Filename)
			if err != nil {
				return nil, err
			}

			filesURL = append(filesURL, fileURL)
		}
	}

	secureUser := &user.SecureUser{
		ID:                   post.UserID,
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

	// มี user ถึงสร้างและ ส่ง notification กับ post
	if len(usersExcept) > 0 {
		createNotificationsDTO := []user.Notification{}
		for _, userExcept := range usersExcept {
			createNotificationsDTO = append(createNotificationsDTO, user.Notification{
				Type:       user.NotificationTypePost,
				Message:    "Create a new post",
				SenderID:   userByID.ID,
				ReceiverID: userExcept.ID,
				PostID:     &createPost.ID,
			})
		}

		notifications, err := s.notificationService.CreateNotifications(createNotificationsDTO)
		if err != nil {
			return nil, err
		}

		broadcastNotificationsDTO := []socket.BroadcastNotificationDTO{}
		for _, notification := range notifications {
			notificationSender := &user.SecureUser{
				ID:                   notification.SenderID,
				Fullname:             notification.Sender.Fullname,
				Username:             notification.Sender.Username,
				Email:                notification.Sender.Email,
				DateOfBirth:          notification.Sender.DateOfBirth,
				ProfileUrl:           notification.Sender.ProfileUrl,
				ProfileBackgroundUrl: notification.Sender.ProfileBackgroundUrl,
				Info:                 notification.Sender.Info,
				Role:                 notification.Sender.Role,
				ProviderType:         notification.Sender.ProviderType,
				CreatedAt:            notification.Sender.CreatedAt,
				UpdatedAt:            notification.Sender.UpdatedAt,
			}

			broadcastNotificationsDTO = append(broadcastNotificationsDTO, socket.BroadcastNotificationDTO{
				ID:         notification.ID,
				Type:       notification.Type,
				Message:    notification.Message,
				IsRead:     notification.IsRead,
				SenderID:   notification.SenderID,
				Sender:     notificationSender,
				ReceiverID: notification.ReceiverID,
				PostID:     notification.PostID,
				CommentID:  notification.CommentID,
				CreatedAt:  notification.CreatedAt,
				UpdatedAt:  notification.UpdatedAt,
			})
		}
		go s.notificationSocket.BroadcastNotifications(broadcastNotificationsDTO)

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
		} else {
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
		}
	}

	likes := []Like{}
	for _, like := range post.Likes {
		likes = append(likes, Like{
			ID:        like.ID,
			UserID:    like.UserID,
			PostID:    like.PostID,
			CommentID: like.CommentID,
			CreatedAt: like.CreatedAt,
			UpdatedAt: like.UpdatedAt,
		})
	}
	comments := []Comment{}
	for _, comment := range post.Comments {
		comments = append(comments, Comment{
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
		respPost := &CreatedPost{
			ID:            post.ID,
			Message:       post.Message,
			UserID:        post.UserID,
			User:          secureUser,
			ParentID:      post.ParentID,
			Likes:         likes,
			Comments:      comments,
			CommentsCount: len(post.Comments),
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
		}
		return respPost, nil
	}

	respPostWithFiles := &CreatedPost{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         likes,
		Comments:      comments,
		FilesURL:      filesURL,
		CommentsCount: len(post.Comments),
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}
	return respPostWithFiles, nil
}

func (s *postService) CreateSharePost(createSharePostDTO *CreateSharePostDTO) (*CreatedSharePost, error) {
	userByID, err := s.userService.SecureFindWithID(createSharePostDTO.UserID)
	if err != nil {
		return nil, err
	}

	parentID, err := helpers.ParseUUID(createSharePostDTO.ParentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	post, err := s.postRepository.FindByIDPreloadRelation(*parentID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post id %v to share is not found", parentID))
	}

	createSharePost := &user.Post{
		Message:  &createSharePostDTO.Message,
		UserID:   userByID.ID,
		ParentID: &post.ID,
	}
	err = s.postRepository.Create(createSharePost)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to create share post")
	}

	sharePost, err := s.postRepository.PreloadRelations(createSharePost.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post with relations")
	}

	err = s.userService.GetPostUserImage(sharePost)
	if err != nil {
		return nil, err
	}

	err = s.userService.GetPostUserImage(post)
	if err != nil {
		return nil, err
	}

	// ต้องไม่แชร์โพสต์ตัวเองถึงสร้าง notification
	if userByID.ID != post.UserID {
		createNotificationDTO := &user.Notification{
			Type:       user.NotificationTypePost,
			Message:    "Share your post",
			SenderID:   userByID.ID,
			ReceiverID: post.UserID,
			PostID:     &createSharePost.ID,
		}
		notification, err := s.notificationService.CreateNotification(createNotificationDTO)
		if err != nil {
			return nil, err
		}

		notificationSender := &user.SecureUser{
			ID:                   notification.SenderID,
			Fullname:             notification.Sender.Fullname,
			Username:             notification.Sender.Username,
			Email:                notification.Sender.Email,
			DateOfBirth:          notification.Sender.DateOfBirth,
			ProfileUrl:           notification.Sender.ProfileUrl,
			ProfileBackgroundUrl: notification.Sender.ProfileBackgroundUrl,
			Info:                 notification.Sender.Info,
			Role:                 notification.Sender.Role,
			ProviderType:         notification.Sender.ProviderType,
			CreatedAt:            notification.Sender.CreatedAt,
			UpdatedAt:            notification.Sender.UpdatedAt,
		}
		broadcastNotificationsDTO := &socket.BroadcastNotificationDTO{
			ID:         notification.ID,
			Type:       notification.Type,
			Message:    notification.Message,
			IsRead:     notification.IsRead,
			SenderID:   notification.SenderID,
			Sender:     notificationSender,
			ReceiverID: notification.ReceiverID,
			PostID:     notification.PostID,
			CommentID:  notification.CommentID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}
		go s.notificationSocket.BroadcastNotification(broadcastNotificationsDTO)
	}

	files, err := s.fileRepository.FindsByContentID(post.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find files of post")
	}

	filesURL := []string{}
	if len(files) > 0 {
		for _, file := range files {
			fileURL, err := s.fileService.PresignGetFile(file.Filename)
			if err != nil {
				return nil, err
			}

			filesURL = append(filesURL, fileURL)
		}
	}

	postParentSecureUser := &user.SecureUser{
		ID:                   post.UserID,
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
	postParentDTO := &socket.PostParentDTO{
		ID:        post.ID,
		Message:   post.Message,
		UserID:    post.UserID,
		User:      postParentSecureUser,
		ParentID:  post.ParentID,
		FilesURL:  filesURL,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}
	secureUser := &user.SecureUser{
		ID:                   sharePost.UserID,
		Fullname:             sharePost.User.Fullname,
		Username:             sharePost.User.Username,
		Email:                sharePost.User.Email,
		DateOfBirth:          sharePost.User.DateOfBirth,
		ProfileUrl:           sharePost.User.ProfileUrl,
		ProfileBackgroundUrl: sharePost.User.ProfileBackgroundUrl,
		Info:                 sharePost.User.Info,
		Role:                 sharePost.User.Role,
		ProviderType:         sharePost.User.ProviderType,
		CreatedAt:            sharePost.User.CreatedAt,
		UpdatedAt:            sharePost.User.UpdatedAt,
	}
	likesDTO := []socket.LikeDTO{}
	for _, like := range sharePost.Likes {
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
	for _, comment := range sharePost.Comments {
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
	broadcastSharePostDTO := &socket.BroadcastSharePostDTO{
		ID:            sharePost.ID,
		Message:       sharePost.Message,
		UserID:        sharePost.UserID,
		User:          secureUser,
		ParentID:      sharePost.ParentID,
		Parent:        postParentDTO,
		Likes:         likesDTO,
		Comments:      commentsDTO,
		CommentsCount: len(sharePost.Comments),
		CreatedAt:     sharePost.CreatedAt,
		UpdatedAt:     sharePost.UpdatedAt,
	}
	go s.postSocket.BroadcastSharePost(broadcastSharePostDTO)

	postParent := &PostParent{
		ID:        post.ID,
		Message:   post.Message,
		UserID:    post.UserID,
		User:      postParentSecureUser,
		ParentID:  post.ParentID,
		FilesURL:  filesURL,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}
	likes := []Like{}
	for _, like := range sharePost.Likes {
		likes = append(likes, Like{
			ID:        like.ID,
			UserID:    like.UserID,
			PostID:    like.PostID,
			CommentID: like.CommentID,
			CreatedAt: like.CreatedAt,
			UpdatedAt: like.UpdatedAt,
		})
	}
	comments := []Comment{}
	for _, comment := range sharePost.Comments {
		comments = append(comments, Comment{
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

	createdSharePost := &CreatedSharePost{
		ID:            sharePost.ID,
		Message:       sharePost.Message,
		UserID:        sharePost.UserID,
		User:          secureUser,
		ParentID:      sharePost.ParentID,
		Parent:        postParent,
		Likes:         likes,
		Comments:      comments,
		CommentsCount: len(sharePost.Comments),
		CreatedAt:     sharePost.CreatedAt,
		UpdatedAt:     sharePost.UpdatedAt,
	}
	return createdSharePost, nil
}

func (s *postService) FindsCursorPagination(cursor *uuid.UUID, limit int) ([]Post, error) {
	return nil, nil
}
