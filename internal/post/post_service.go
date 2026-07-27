package post

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/socket"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UpdatePostDTO struct {
	PostID                   string
	Message                  *string
	FilesURL                 []string
	ShouldDeleteCurrentFiles bool
	IsSharePost              bool
}

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
	ID        int64            `json:"id"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user,omitempty"`
	PostID    *uuid.UUID       `json:"postId"`
	CommentID *uuid.UUID       `json:"commentId"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
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

type PostCursorPagination struct {
	Posts      []Post     `json:"posts"`
	NextCursor *uuid.UUID `json:"nextCursor"`
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
	FindsCursorPagination(cursor, limit string) (*PostCursorPagination, error)
	FindsWithUserIDCursorPagination(userID, cursor, limit string) (*PostCursorPagination, error)
	FindWithID(postID string) (*Post, error)
	UpdatePost(updatePostDTO *UpdatePostDTO) (*Post, error)
}

type postService struct {
	db                     *gorm.DB
	redisClient            *redis.Client
	s3Client               *s3.Client
	postRepository         PostRepository
	userRepository         user.UserRepository
	fileRepository         file.FileRepository
	notificationRepository notification.NotificationRepository
	userService            user.UserService
	notificationService    notification.NotificationService
	fileService            file.FileService
	notificationSocket     socket.NotificationSocket
	postSocket             socket.PostSocket
}

func NewPostService(
	db *gorm.DB,
	redisClient *redis.Client,
	s3Client *s3.Client,
	postRepository PostRepository,
	userRepository user.UserRepository,
	fileRepository file.FileRepository,
	notificationRepository notification.NotificationRepository,
	userService user.UserService,
	notificationService notification.NotificationService,
	fileService file.FileService,
	notificationSocket socket.NotificationSocket,
	postSocket socket.PostSocket,
) PostService {
	return &postService{
		db:                     db,
		redisClient:            redisClient,
		s3Client:               s3Client,
		postRepository:         postRepository,
		userRepository:         userRepository,
		fileRepository:         fileRepository,
		notificationRepository: notificationRepository,
		userService:            userService,
		notificationService:    notificationService,
		fileService:            fileService,
		notificationSocket:     notificationSocket,
		postSocket:             postSocket,
	}
}

func (s *postService) CreatePost(createPostDTO *CreatePostDTO) (*CreatedPost, error) {
	if createPostDTO.Message == "" && len(createPostDTO.FilesURL) == 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Create post must contains with message or files")
	}

	createPost := &models.Post{
		Message: &createPostDTO.Message,
		UserID:  createPostDTO.UserID,
	}

	usersExcept, err := s.userRepository.FindsByIDExcept(s.db, createPostDTO.UserID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find users by id except")
	}

	createNotificationsDTO := []models.Notification{}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err := s.postRepository.Create(tx, createPost)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create post")
		}

		// กรณีมี files ผูกเข้ากับโพสต์
		if len(createPostDTO.FilesURL) > 0 {
			for _, fileURL := range createPostDTO.FilesURL {
				fileDIR, filename, err := helpers.SplitPresignedURL(fileURL)
				if err != nil {
					logs.Error(err)
					return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
				}
				filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
				err = s.fileRepository.UpdateContentID(tx, createPost.ID, filePath, models.FileTypePost)
				if err != nil {
					logs.Error(err)
					return errs.NewInternalServerErrorWithMessage("Failed to update file of post")
				}
			}
		}

		// มี users ถึงสร้าง notifications
		if len(usersExcept) > 0 {
			for _, userExcept := range usersExcept {
				createNotificationsDTO = append(createNotificationsDTO, models.Notification{
					Type:       models.NotificationTypePost,
					Message:    "Create a new post",
					SenderID:   createPostDTO.UserID,
					ReceiverID: userExcept.ID,
					PostID:     &createPost.ID,
				})
			}

			err = s.notificationService.CreateNotifications(tx, createNotificationsDTO)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return nil, err
	}

	post, err := s.postRepository.PreloadRelations(s.db, createPost.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post with relations")
	}

	err = s.userService.GetUserImage(&post.User)
	if err != nil {
		return nil, err
	}

	filesURL := []string{}

	// กรณีมี files
	if len(createPostDTO.FilesURL) > 0 {
		filesURL, err = s.fileService.PresignGetFiles(createPost.ID)
		if err != nil {
			return nil, err
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

	// มี user ถึง broadcast notifications กับ post
	if len(usersExcept) > 0 {
		notificationsID := []uuid.UUID{}
		for _, createNotificationDTO := range createNotificationsDTO {
			notificationsID = append(notificationsID, createNotificationDTO.ID)
		}
		notifications, err := s.notificationRepository.PreloadsRelation(s.db, notificationsID)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notifications with relation")
		}

		for i := range notifications {
			err = s.userService.GetUserImage(&notifications[i].Sender)
			if err != nil {
				return nil, err
			}
		}

		emitNotificationsDTO := []socket.EmitNotificationDTO{}
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

			emitNotificationsDTO = append(emitNotificationsDTO, socket.EmitNotificationDTO{
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
		go s.notificationSocket.EmitNotifications(emitNotificationsDTO)

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

		postDTO := &socket.PostDTO{
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
		if len(createPostDTO.FilesURL) > 0 {
			postDTO.FilesURL = filesURL
		}
		go s.postSocket.EmitCreate(postDTO)
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

	if len(createPostDTO.FilesURL) > 0 {
		respPost.FilesURL = filesURL
	}

	return respPost, nil
}

func (s *postService) CreateSharePost(createSharePostDTO *CreateSharePostDTO) (*CreatedSharePost, error) {
	err := helpers.ValidateUUID(createSharePostDTO.ParentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	parentID, err := helpers.ParseUUID(createSharePostDTO.ParentID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	post, err := s.postRepository.FindByIDPreloadRelation(s.db, *parentID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post id %v to share is not found", parentID))
	}

	createSharePost := &models.Post{
		Message:  &createSharePostDTO.Message,
		UserID:   createSharePostDTO.UserID,
		ParentID: &post.ID,
	}

	var notificationID uuid.UUID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.postRepository.Create(tx, createSharePost)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create share post")
		}

		// ต้องไม่แชร์โพสต์ตัวเองถึงสร้าง notification
		if createSharePostDTO.UserID != post.UserID {
			createNotificationDTO := &models.Notification{
				Type:       models.NotificationTypeShare,
				Message:    "Share your post",
				SenderID:   createSharePostDTO.UserID,
				ReceiverID: post.UserID,
				PostID:     &createSharePost.ID,
			}
			err = s.notificationService.CreateNotification(tx, createNotificationDTO)
			if err != nil {
				return err
			}

			notificationID = createNotificationDTO.ID
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return nil, err
	}

	sharePost, err := s.postRepository.PreloadRelations(s.db, createSharePost.ID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post with relations")
	}

	err = s.userService.GetUserImage(&sharePost.User)
	if err != nil {
		return nil, err
	}

	err = s.userService.GetUserImage(&post.User)
	if err != nil {
		return nil, err
	}

	// ต้องไม่แชร์โพสต์ตัวเองถึง broadcast notification
	if createSharePostDTO.UserID != post.UserID {
		notification, err := s.notificationRepository.PreloadRelation(s.db, notificationID)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification with relation")
		}

		err = s.userService.GetUserImage(&notification.Sender)
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
		emitNotificationsDTO := &socket.EmitNotificationDTO{
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
		go s.notificationSocket.EmitNotification(emitNotificationsDTO)
	}

	filesURL, err := s.fileService.PresignGetFiles(post.ID)
	if err != nil {
		return nil, err
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
	postDTO := &socket.PostDTO{
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
	go s.postSocket.EmitCreate(postDTO)

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

func (s *postService) FindsCursorPagination(cursor, limit string) (*PostCursorPagination, error) {
	var nextCursor *uuid.UUID
	var cursorID *uuid.UUID

	if cursor != "" {
		err := helpers.ValidateUUID(cursor)
		if err != nil {
			logs.Warn(err)
			return nil, err
		}

		cursorID, err = helpers.ParseUUID(cursor)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	posts := []models.Post{}
	postsCursorPagination := []Post{}
	if cursor == "" {
		posts, err = s.postRepository.FindsCursorPagination(s.db, nil, limitInt)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find posts cursor pagination")
		}
	} else {
		postCursor, err := s.postRepository.FindByIDCursor(s.db, *cursorID)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find post cursor by post id")
		}

		if helpers.IsErrRecordNotFound(err) {
			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by post id %v is not found", *cursorID))
		}

		posts, err = s.postRepository.FindsCursorPagination(s.db, postCursor, limitInt)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find posts cursor pagination")
		}
	}

	if len(posts) > 0 {
		for _, post := range posts {
			err = s.userService.GetUserImage(&post.User)
			if err != nil {
				return nil, err
			}

			comments := []models.Comment{}
			for _, comment := range post.Comments {
				if comment.ParentID == nil {
					comments = append(comments, comment)
				}
			}
			commentsCount := len(comments)

			filesURL, err := s.fileService.PresignGetFiles(post.ID)
			if err != nil {
				return nil, err
			}

			updateLikes := []Like{}
			for _, like := range post.Likes {
				err = s.userService.GetUserImage(&like.User)
				if err != nil {
					return nil, err
				}

				secureUser := &user.SecureUser{
					ID:                   like.UserID,
					Fullname:             like.User.Fullname,
					Username:             like.User.Username,
					Email:                like.User.Email,
					DateOfBirth:          like.User.DateOfBirth,
					ProfileUrl:           like.User.ProfileUrl,
					ProfileBackgroundUrl: like.User.ProfileBackgroundUrl,
					Info:                 like.User.Info,
					Role:                 like.User.Role,
					ProviderType:         like.User.ProviderType,
					CreatedAt:            like.User.CreatedAt,
					UpdatedAt:            like.User.UpdatedAt,
				}
				updateLikes = append(updateLikes, Like{
					ID:        like.ID,
					UserID:    like.UserID,
					User:      secureUser,
					PostID:    like.PostID,
					CommentID: like.CommentID,
					CreatedAt: like.CreatedAt,
					UpdatedAt: like.UpdatedAt,
				})
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
			postCursorPagination := Post{
				ID:            post.ID,
				Message:       post.Message,
				UserID:        post.UserID,
				User:          secureUser,
				ParentID:      post.ParentID,
				Likes:         updateLikes,
				FilesURL:      filesURL,
				CommentsCount: commentsCount,
				CreatedAt:     post.CreatedAt,
				UpdatedAt:     post.UpdatedAt,
			}

			if post.ParentID != nil {
				err = s.userService.GetUserImage(&post.Parent.User)
				if err != nil {
					return nil, err
				}

				filesURL, err = s.fileService.PresignGetFiles(*post.ParentID)
				if err != nil {
					return nil, err
				}

				postParentSecureUser := &user.SecureUser{
					ID:                   post.Parent.UserID,
					Fullname:             post.Parent.User.Fullname,
					Username:             post.Parent.User.Username,
					Email:                post.Parent.User.Email,
					DateOfBirth:          post.Parent.User.DateOfBirth,
					ProfileUrl:           post.Parent.User.ProfileUrl,
					ProfileBackgroundUrl: post.Parent.User.ProfileBackgroundUrl,
					Info:                 post.Parent.User.Info,
					Role:                 post.Parent.User.Role,
					ProviderType:         post.Parent.User.ProviderType,
					CreatedAt:            post.Parent.User.CreatedAt,
					UpdatedAt:            post.Parent.User.UpdatedAt,
				}
				postParent := &PostParent{
					ID:        *post.ParentID,
					Message:   post.Parent.Message,
					UserID:    post.Parent.UserID,
					User:      postParentSecureUser,
					ParentID:  post.ParentID,
					FilesURL:  filesURL,
					CreatedAt: post.Parent.CreatedAt,
					UpdatedAt: post.Parent.UpdatedAt,
				}
				postCursorPagination.Parent = postParent
			}

			postsCursorPagination = append(postsCursorPagination, postCursorPagination)
		}
	}

	if len(postsCursorPagination) > 0 {
		nextCursor = &postsCursorPagination[len(postsCursorPagination)-1].ID
	}
	postCursorPagination := &PostCursorPagination{
		Posts:      postsCursorPagination,
		NextCursor: nextCursor,
	}
	return postCursorPagination, nil
}

func (s *postService) FindsWithUserIDCursorPagination(userID, cursor, limit string) (*PostCursorPagination, error) {
	var nextCursor *uuid.UUID
	var cursorID *uuid.UUID

	err := helpers.ValidateUUID(userID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	userIDParse, err := helpers.ParseUUID(userID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	userByID, err := s.userService.SecureFindWithID(*userIDParse)
	if err != nil {
		return nil, err
	}

	if cursor != "" {
		err := helpers.ValidateUUID(cursor)
		if err != nil {
			logs.Warn(err)
			return nil, err
		}

		cursorID, err = helpers.ParseUUID(cursor)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	posts := []models.Post{}
	postsCursorPagination := []Post{}
	if cursor == "" {
		posts, err = s.postRepository.FindsByUserIDCursorPagination(s.db, userByID.ID, nil, limitInt)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find posts by user id cursor pagination")
		}
	} else {
		postCursor, err := s.postRepository.FindByIDCursor(s.db, *cursorID)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find post cursor by post id")
		}

		if helpers.IsErrRecordNotFound(err) {
			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by post id %v is not found", *cursorID))
		}

		posts, err = s.postRepository.FindsByUserIDCursorPagination(s.db, userByID.ID, postCursor, limitInt)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find posts by user id cursor pagination")
		}
	}

	if len(posts) > 0 {
		for _, post := range posts {
			err = s.userService.GetUserImage(&post.User)
			if err != nil {
				return nil, err
			}

			comments := []models.Comment{}
			for _, comment := range post.Comments {
				if comment.ParentID == nil {
					comments = append(comments, comment)
				}
			}
			commentsCount := len(comments)

			filesURL, err := s.fileService.PresignGetFiles(post.ID)
			if err != nil {
				return nil, err
			}

			updateLikes := []Like{}
			for _, like := range post.Likes {
				err = s.userService.GetUserImage(&like.User)
				if err != nil {
					return nil, err
				}

				secureUser := &user.SecureUser{
					ID:                   like.UserID,
					Fullname:             like.User.Fullname,
					Username:             like.User.Username,
					Email:                like.User.Email,
					DateOfBirth:          like.User.DateOfBirth,
					ProfileUrl:           like.User.ProfileUrl,
					ProfileBackgroundUrl: like.User.ProfileBackgroundUrl,
					Info:                 like.User.Info,
					Role:                 like.User.Role,
					ProviderType:         like.User.ProviderType,
					CreatedAt:            like.User.CreatedAt,
					UpdatedAt:            like.User.UpdatedAt,
				}
				updateLikes = append(updateLikes, Like{
					ID:        like.ID,
					UserID:    like.UserID,
					User:      secureUser,
					PostID:    like.PostID,
					CommentID: like.CommentID,
					CreatedAt: like.CreatedAt,
					UpdatedAt: like.UpdatedAt,
				})
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
			postCursorPagination := Post{
				ID:            post.ID,
				Message:       post.Message,
				UserID:        post.UserID,
				User:          secureUser,
				ParentID:      post.ParentID,
				Likes:         updateLikes,
				FilesURL:      filesURL,
				CommentsCount: commentsCount,
				CreatedAt:     post.CreatedAt,
				UpdatedAt:     post.UpdatedAt,
			}

			if post.ParentID != nil {
				err = s.userService.GetUserImage(&post.Parent.User)
				if err != nil {
					return nil, err
				}

				filesURL, err = s.fileService.PresignGetFiles(*post.ParentID)
				if err != nil {
					return nil, err
				}

				postParentSecureUser := &user.SecureUser{
					ID:                   post.Parent.UserID,
					Fullname:             post.Parent.User.Fullname,
					Username:             post.Parent.User.Username,
					Email:                post.Parent.User.Email,
					DateOfBirth:          post.Parent.User.DateOfBirth,
					ProfileUrl:           post.Parent.User.ProfileUrl,
					ProfileBackgroundUrl: post.Parent.User.ProfileBackgroundUrl,
					Info:                 post.Parent.User.Info,
					Role:                 post.Parent.User.Role,
					ProviderType:         post.Parent.User.ProviderType,
					CreatedAt:            post.Parent.User.CreatedAt,
					UpdatedAt:            post.Parent.User.UpdatedAt,
				}
				postParent := &PostParent{
					ID:        *post.ParentID,
					Message:   post.Parent.Message,
					UserID:    post.Parent.UserID,
					User:      postParentSecureUser,
					ParentID:  post.ParentID,
					FilesURL:  filesURL,
					CreatedAt: post.Parent.CreatedAt,
					UpdatedAt: post.Parent.UpdatedAt,
				}
				postCursorPagination.Parent = postParent
			}

			postsCursorPagination = append(postsCursorPagination, postCursorPagination)
		}
	}

	if len(postsCursorPagination) > 0 {
		nextCursor = &postsCursorPagination[len(postsCursorPagination)-1].ID
	}
	postCursorPagination := &PostCursorPagination{
		Posts:      postsCursorPagination,
		NextCursor: nextCursor,
	}
	return postCursorPagination, nil
}

func (s *postService) FindWithID(postID string) (*Post, error) {
	err := helpers.ValidateUUID(postID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	postIDParse, err := helpers.ParseUUID(postID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	ctx := context.Background()
	key := fmt.Sprintf("post:find:%v", *postIDParse)
	value, err := helpers.RedisGet(s.redisClient, ctx, key)
	if err != nil && err != redis.Nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to get post from redis")
	}

	if err == nil {
		post := &Post{}
		err = json.Unmarshal([]byte(value), post)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to unmarshal json post")
		}

		return post, nil
	}

	post, err := s.postRepository.FindByIDPreloadRelations(s.db, *postIDParse)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post with relations")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	err = s.userService.GetUserImage(&post.User)
	if err != nil {
		return nil, err
	}

	comments := []models.Comment{}
	for _, comment := range post.Comments {
		if comment.ParentID == nil {
			comments = append(comments, comment)
		}
	}
	commentsCount := len(comments)

	filesURL, err := s.fileService.PresignGetFiles(post.ID)
	if err != nil {
		return nil, err
	}

	updateLikes := []Like{}
	for _, like := range post.Likes {
		err = s.userService.GetUserImage(&like.User)
		if err != nil {
			return nil, err
		}

		secureUser := &user.SecureUser{
			ID:                   like.UserID,
			Fullname:             like.User.Fullname,
			Username:             like.User.Username,
			Email:                like.User.Email,
			DateOfBirth:          like.User.DateOfBirth,
			ProfileUrl:           like.User.ProfileUrl,
			ProfileBackgroundUrl: like.User.ProfileBackgroundUrl,
			Info:                 like.User.Info,
			Role:                 like.User.Role,
			ProviderType:         like.User.ProviderType,
			CreatedAt:            like.User.CreatedAt,
			UpdatedAt:            like.User.UpdatedAt,
		}
		updateLikes = append(updateLikes, Like{
			ID:        like.ID,
			UserID:    like.UserID,
			User:      secureUser,
			PostID:    like.PostID,
			CommentID: like.CommentID,
			CreatedAt: like.CreatedAt,
			UpdatedAt: like.UpdatedAt,
		})
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
	postResp := &Post{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         updateLikes,
		FilesURL:      filesURL,
		CommentsCount: commentsCount,
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}

	if post.ParentID != nil {
		err = s.userService.GetUserImage(&post.Parent.User)
		if err != nil {
			return nil, err
		}

		filesURL, err = s.fileService.PresignGetFiles(*post.ParentID)
		if err != nil {
			return nil, err
		}

		postParentSecureUser := &user.SecureUser{
			ID:                   post.Parent.UserID,
			Fullname:             post.Parent.User.Fullname,
			Username:             post.Parent.User.Username,
			Email:                post.Parent.User.Email,
			DateOfBirth:          post.Parent.User.DateOfBirth,
			ProfileUrl:           post.Parent.User.ProfileUrl,
			ProfileBackgroundUrl: post.Parent.User.ProfileBackgroundUrl,
			Info:                 post.Parent.User.Info,
			Role:                 post.Parent.User.Role,
			ProviderType:         post.Parent.User.ProviderType,
			CreatedAt:            post.Parent.User.CreatedAt,
			UpdatedAt:            post.Parent.User.UpdatedAt,
		}
		postParent := &PostParent{
			ID:        *post.ParentID,
			Message:   post.Parent.Message,
			UserID:    post.Parent.UserID,
			User:      postParentSecureUser,
			ParentID:  post.ParentID,
			FilesURL:  filesURL,
			CreatedAt: post.Parent.CreatedAt,
			UpdatedAt: post.Parent.UpdatedAt,
		}
		postResp.Parent = postParent
	}

	data, err := json.Marshal(postResp)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to marshal json post")
	}

	err = helpers.RedisSet(s.redisClient, ctx, key, data, time.Minute*10)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to set post from redis")
	}

	return postResp, nil
}

func (s *postService) UpdatePost(updatePostDTO *UpdatePostDTO) (*Post, error) {
	if updatePostDTO.Message == nil && len(updatePostDTO.FilesURL) == 0 && !updatePostDTO.IsSharePost {
		return nil, errs.NewBadRequestErrorWithMessage("Update post must contains with message or files")
	}

	err := helpers.ValidateUUID(updatePostDTO.PostID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	postID, err := helpers.ParseUUID(updatePostDTO.PostID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	postByID, err := s.postRepository.FindByID(s.db, *postID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postID))
	}

	updatePost := &models.Post{
		Message: updatePostDTO.Message,
	}

	files := []models.File{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.postRepository.Update(
			tx,
			postByID.ID,
			updatePost,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to update post")
		}

		if len(updatePostDTO.FilesURL) > 0 && updatePostDTO.ShouldDeleteCurrentFiles {
			files, err = s.fileRepository.FindsByContentID(tx, postByID.ID)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to find files of post")
			}

			if len(files) > 0 {
				err = s.fileRepository.DeleteMany(tx, files)
				if err != nil {
					logs.Error(err)
					return errs.NewInternalServerErrorWithMessage("Failed to delete files of post")
				}
			}

			for _, fileURL := range updatePostDTO.FilesURL {
				fileDIR, filename, err := helpers.SplitPresignedURL(fileURL)
				if err != nil {
					logs.Error(err)
					return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
				}
				filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
				err = s.fileRepository.UpdateContentID(tx, postByID.ID, filePath, models.FileTypePost)
				if err != nil {
					logs.Error(err)
					return errs.NewInternalServerErrorWithMessage("Failed to update file of post")
				}
			}
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return nil, err
	}

	post, err := s.postRepository.FindByIDPreloadRelations(s.db, postByID.ID)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id with relations")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", postByID.ID))
	}

	comments := []models.Comment{}
	for _, comment := range post.Comments {
		if comment.ParentID == nil {
			comments = append(comments, comment)
		}
	}
	commentsCount := len(comments)

	err = s.userService.GetUserImage(&post.User)
	if err != nil {
		return nil, err
	}

	updateLikes := []Like{}
	for _, like := range post.Likes {
		err = s.userService.GetUserImage(&like.User)
		if err != nil {
			return nil, err
		}

		secureUser := &user.SecureUser{
			ID:                   like.UserID,
			Fullname:             like.User.Fullname,
			Username:             like.User.Username,
			Email:                like.User.Email,
			DateOfBirth:          like.User.DateOfBirth,
			ProfileUrl:           like.User.ProfileUrl,
			ProfileBackgroundUrl: like.User.ProfileBackgroundUrl,
			Info:                 like.User.Info,
			Role:                 like.User.Role,
			ProviderType:         like.User.ProviderType,
			CreatedAt:            like.User.CreatedAt,
			UpdatedAt:            like.User.UpdatedAt,
		}
		updateLikes = append(updateLikes, Like{
			ID:        like.ID,
			UserID:    like.UserID,
			User:      secureUser,
			PostID:    like.PostID,
			CommentID: like.CommentID,
			CreatedAt: like.CreatedAt,
			UpdatedAt: like.UpdatedAt,
		})
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

	postDTO := &socket.PostDTO{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         likesDTO,
		CommentsCount: len(post.Comments),
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}

	postResp := &Post{
		ID:            post.ID,
		Message:       post.Message,
		UserID:        post.UserID,
		User:          secureUser,
		ParentID:      post.ParentID,
		Likes:         updateLikes,
		CommentsCount: commentsCount,
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}

	if post.ParentID != nil {
		err = s.userService.GetUserImage(&post.Parent.User)
		if err != nil {
			return nil, err
		}

		postParentSecureUser := &user.SecureUser{
			ID:                   post.Parent.UserID,
			Fullname:             post.Parent.User.Fullname,
			Username:             post.Parent.User.Username,
			Email:                post.Parent.User.Email,
			DateOfBirth:          post.Parent.User.DateOfBirth,
			ProfileUrl:           post.Parent.User.ProfileUrl,
			ProfileBackgroundUrl: post.Parent.User.ProfileBackgroundUrl,
			Info:                 post.Parent.User.Info,
			Role:                 post.Parent.User.Role,
			ProviderType:         post.Parent.User.ProviderType,
			CreatedAt:            post.Parent.User.CreatedAt,
			UpdatedAt:            post.Parent.User.UpdatedAt,
		}
		postParent := &PostParent{
			ID:        *post.ParentID,
			Message:   post.Parent.Message,
			UserID:    post.Parent.UserID,
			User:      postParentSecureUser,
			ParentID:  post.ParentID,
			CreatedAt: post.Parent.CreatedAt,
			UpdatedAt: post.Parent.UpdatedAt,
		}

		postParentDTO := &socket.PostParentDTO{
			ID:        *post.ParentID,
			Message:   post.Parent.Message,
			UserID:    post.Parent.UserID,
			User:      postParentSecureUser,
			ParentID:  post.ParentID,
			CreatedAt: post.Parent.CreatedAt,
			UpdatedAt: post.Parent.UpdatedAt,
		}

		postDTO.Parent = postParentDTO
		postResp.Parent = postParent
	}

	if len(updatePostDTO.FilesURL) == 0 || !updatePostDTO.ShouldDeleteCurrentFiles {
		go s.postSocket.EmitUpdate(postDTO)
		return postResp, nil
	}

	ctx := context.Background()
	for _, file := range files {
		_, err = helpers.DeleteObject(s.s3Client, ctx, file.Filename)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to delete file from bucket")
		}
	}

	filesURL, err := s.fileService.PresignGetFiles(post.ID)
	if err != nil {
		return nil, err
	}

	postDTO.FilesURL = filesURL
	postResp.FilesURL = filesURL

	go s.postSocket.EmitUpdate(postDTO)
	return postResp, nil
}
