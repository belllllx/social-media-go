package comment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/dto"
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/like"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/notification"
	"github.com/belllllx/social-media-go/internal/post"
	"github.com/belllllx/social-media-go/internal/socket"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UpdateCommentDTO struct {
	UserID                  uuid.UUID
	Message                 *string
	FileURL                 string
	CommentID               string
	ShouldDeleteCurrentFile bool
}

type CreateTagReplyDTO struct {
	Message  string
	FileURL  string
	PostID   string
	ParentID string
	ReplyID  string
	UserID   uuid.UUID
}

type CreateReplyCommentDTO struct {
	Message  string
	FileURL  string
	PostID   string
	ParentID string
	UserID   uuid.UUID
}

type CreateCommentDTO struct {
	Message string
	FileURL string
	PostID  string
	UserID  uuid.UUID
}

type Post struct {
	UserID uuid.UUID `json:"userId"`
}

type DeletedComment struct {
	ID       uuid.UUID  `json:"id"`
	PostID   uuid.UUID  `json:"postId"`
	Post     *Post      `json:"post"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
}

type UpdatedComment struct {
	ID      uuid.UUID `json:"id"`
	Message *string   `json:"message"`
	PostID  uuid.UUID `json:"postId"`
	FileURL string    `json:"fileUrl"`
}

type Like struct {
	ID        int64           `json:"id"`
	UserID    uuid.UUID       `json:"userId"`
	User      *dto.SecureUser `json:"user,omitempty"`
	CommentID uuid.UUID       `json:"commentId"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type ReplyOrTag struct {
	ID            uuid.UUID       `json:"id"`
	Message       *string         `json:"message"`
	PostID        uuid.UUID       `json:"postId"`
	UserID        uuid.UUID       `json:"userId"`
	User          *dto.SecureUser `json:"user"`
	ParentID      uuid.UUID       `json:"parentId"`
	ReplyID       *uuid.UUID      `json:"replyId,omitempty"`
	ReplyToUserID *uuid.UUID      `json:"replyToUserId,omitempty"`
	ReplyToUser   *dto.SecureUser `json:"replyToUser,omitempty"`
	Likes         []Like          `json:"likes"`
	FileURL       string          `json:"fileUrl,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type Comment struct {
	ID           uuid.UUID       `json:"id"`
	Message      *string         `json:"message"`
	PostID       uuid.UUID       `json:"postId"`
	UserID       uuid.UUID       `json:"userId"`
	User         *dto.SecureUser `json:"user"`
	Likes        []Like          `json:"likes"`
	Replies      []ReplyOrTag    `json:"replies"`
	FileURL      string          `json:"fileUrl,omitempty"`
	RepliesCount int             `json:"repliesCount"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type CommentCursorPagination struct {
	Comments   []Comment  `json:"comments"`
	NextCursor *uuid.UUID `json:"nextCursor"`
}

type CreatedComment struct {
	ID            uuid.UUID       `json:"id"`
	Message       *string         `json:"message"`
	PostID        uuid.UUID       `json:"postId"`
	Post          *Post           `json:"post"`
	UserID        uuid.UUID       `json:"userId"`
	ParentID      *uuid.UUID      `json:"parentId,omitempty"`
	ReplyID       *uuid.UUID      `json:"replyId,omitempty"`
	ReplyToUserID *uuid.UUID      `json:"replyToUserId,omitempty"`
	ReplyToUser   *dto.SecureUser `json:"replyToUser,omitempty"`
	User          *dto.SecureUser `json:"user"`
	FileURL       string          `json:"fileUrl,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type CommentService interface {
	CreateComment(ctx context.Context, createCommentDTO *CreateCommentDTO) (*CreatedComment, error)
	CreateReplyComment(ctx context.Context, createReplyCommentDTO *CreateReplyCommentDTO) (*CreatedComment, error)
	CreateTagReply(ctx context.Context, createTagReplyDTO *CreateTagReplyDTO) (*CreatedComment, error)
	FindsWithPostIDCursorPagination(
		ctx context.Context,
		postID,
		cursor,
		limit string,
	) (*CommentCursorPagination, error)
	UpdateComment(ctx context.Context, updateCommentDTO *UpdateCommentDTO) (*UpdatedComment, error)
	DeleteComment(
		ctx context.Context,
		userID uuid.UUID,
		commentID string,
	) (*DeletedComment, error)
	ToggleLike(
		ctx context.Context,
		userID uuid.UUID,
		postID,
		commentID string,
	) (string, *Like, error)
}

type commentService struct {
	db                     *gorm.DB
	redisClient            *redis.Client
	s3Client               *s3.Client
	presignClient          *s3.PresignClient
	commentRepository      CommentRepository
	userRepository         user.UserRepository
	postRepository         post.PostRepository
	fileRepository         file.FileRepository
	notificationRepository notification.NotificationRepository
	likeRepository         like.LikeRepository
	notificationService    notification.NotificationService
	userService            user.UserService
	fileService            file.FileService
	commentSocket          socket.CommentSocket
	notificationSocket     socket.NotificationSocket
}

func NewCommentService(
	db *gorm.DB,
	redisClient *redis.Client,
	s3Client *s3.Client,
	presignClient *s3.PresignClient,
	commentRepository CommentRepository,
	userRepository user.UserRepository,
	postRepository post.PostRepository,
	fileRepository file.FileRepository,
	notificationRepository notification.NotificationRepository,
	likeRepository like.LikeRepository,
	notificationService notification.NotificationService,
	userService user.UserService,
	fileService file.FileService,
	commentSocket socket.CommentSocket,
	notificationSocket socket.NotificationSocket,
) CommentService {
	return &commentService{
		db:                     db,
		redisClient:            redisClient,
		s3Client:               s3Client,
		presignClient:          presignClient,
		commentRepository:      commentRepository,
		userRepository:         userRepository,
		postRepository:         postRepository,
		fileRepository:         fileRepository,
		notificationRepository: notificationRepository,
		likeRepository:         likeRepository,
		notificationService:    notificationService,
		userService:            userService,
		fileService:            fileService,
		commentSocket:          commentSocket,
		notificationSocket:     notificationSocket,
	}
}

func (s *commentService) CreateComment(ctx context.Context, createCommentDTO *CreateCommentDTO) (*CreatedComment, error) {
	if createCommentDTO.Message == "" && createCommentDTO.FileURL == "" {
		return nil, errs.NewBadRequestErrorWithMessage("Create comment must contains with message or file")
	}

	err := helpers.ValidateUUID(createCommentDTO.PostID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	postIDParse, err := helpers.ParseUUID(createCommentDTO.PostID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	postByID, err := s.postRepository.FindByID(
		ctx,
		s.db,
		*postIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	createComment := &models.Comment{
		Message: &createCommentDTO.Message,
		PostID:  postByID.ID,
		UserID:  createCommentDTO.UserID,
	}
	var notificationID uuid.UUID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.commentRepository.Create(
			ctx,
			tx,
			createComment,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create comment")
		}

		// กรณีมีไฟล์ ผูกเข้ากับ comment
		if createCommentDTO.FileURL != "" {
			fileDIR, filename, err := helpers.SplitPresignedURL(createCommentDTO.FileURL)
			if err != nil {
				logs.Error(err)
				return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
			}

			filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
			err = s.fileRepository.UpdateContentID(
				ctx,
				tx,
				createComment.ID,
				filePath,
				models.FileTypeComment,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to update file of comment")
			}
		}

		// ต้องไม่ comment post ตัวเองถึงสร้าง notification
		if postByID.UserID != createCommentDTO.UserID {
			createNotificationDTO := &models.Notification{
				Type:       models.NotificationTypeComment,
				Message:    "Comment on your post",
				SenderID:   createCommentDTO.UserID,
				ReceiverID: postByID.UserID,
				PostID:     &postByID.ID,
				CommentID:  &createComment.ID,
			}
			err = s.notificationService.CreateNotification(
				ctx,
				tx,
				createNotificationDTO,
			)
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

	comment, err := s.commentRepository.FindByIDWithUserAndPostRelations(
		ctx,
		s.db,
		createComment.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id with user and post relations")
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		&comment.User,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	secureUserComment := &dto.SecureUser{
		ID:                   comment.UserID,
		Fullname:             comment.User.Fullname,
		Username:             comment.User.Username,
		Email:                comment.User.Email,
		DateOfBirth:          comment.User.DateOfBirth,
		ProfileUrl:           comment.User.ProfileUrl,
		ProfileBackgroundUrl: comment.User.ProfileBackgroundUrl,
		Info:                 comment.User.Info,
		Role:                 comment.User.Role,
		ProviderType:         comment.User.ProviderType,
		CreatedAt:            comment.User.CreatedAt,
		UpdatedAt:            comment.User.UpdatedAt,
	}
	createCommentDTOSocket := &socket.CreateCommentDTO{
		ID:      comment.ID,
		Message: comment.Message,
		PostID:  comment.PostID,
		Post: &socket.PostDTO{
			UserID: comment.Post.UserID,
		},
		UserID:    comment.UserID,
		User:      secureUserComment,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
	commentResp := &CreatedComment{
		ID:      comment.ID,
		Message: comment.Message,
		PostID:  comment.PostID,
		Post: &Post{
			UserID: comment.Post.UserID,
		},
		UserID:    comment.UserID,
		User:      secureUserComment,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}

	// กรณีมีไฟล์
	if createCommentDTO.FileURL != "" {
		fileURL, err := s.fileService.PresignGetFile(ctx, comment.ID)
		if err != nil {
			return nil, err
		}

		createCommentDTOSocket.FileURL = fileURL
		commentResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(createCommentDTOSocket)

	// ต้องไม่ comment post ตัวเองถึง emit notification
	if postByID.UserID != createCommentDTO.UserID {
		notification, err := s.notificationRepository.FindByIDWithSenderRelation(
			ctx,
			s.db,
			notificationID,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification by id with sender relation")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&notification.Sender,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		secureSenderNotification := &dto.SecureUser{
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
		emitNotificationDTO := &socket.EmitNotificationDTO{
			ID:         notification.ID,
			Type:       notification.Type,
			Message:    notification.Message,
			IsRead:     notification.IsRead,
			SenderID:   notification.SenderID,
			Sender:     secureSenderNotification,
			ReceiverID: notification.ReceiverID,
			PostID:     notification.PostID,
			CommentID:  notification.CommentID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}

		go s.notificationSocket.EmitNotification(emitNotificationDTO)
	}

	return commentResp, nil
}

func (s *commentService) CreateReplyComment(ctx context.Context, createReplyCommentDTO *CreateReplyCommentDTO) (*CreatedComment, error) {
	if createReplyCommentDTO.Message == "" && createReplyCommentDTO.FileURL == "" {
		return nil, errs.NewBadRequestErrorWithMessage("Create reply comment must contains with message or file")
	}

	err := helpers.ValidateUUID(createReplyCommentDTO.PostID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	postIDParse, err := helpers.ParseUUID(createReplyCommentDTO.PostID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	err = helpers.ValidateUUID(createReplyCommentDTO.ParentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	parentIDParse, err := helpers.ParseUUID(createReplyCommentDTO.ParentID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	postByID, err := s.postRepository.FindByID(
		ctx,
		s.db,
		*postIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	commentByID, err := s.commentRepository.FindByID(
		ctx,
		s.db,
		*parentIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Comment by id %v is not found", *parentIDParse))
	}

	createReplyComment := &models.Comment{
		Message:  &createReplyCommentDTO.Message,
		PostID:   postByID.ID,
		UserID:   createReplyCommentDTO.UserID,
		ParentID: &commentByID.ID,
	}
	var notificationID uuid.UUID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.commentRepository.Create(
			ctx,
			tx,
			createReplyComment,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create reply comment")
		}

		// กรณีมีไฟล์ ผูกเข้ากับ reply
		if createReplyCommentDTO.FileURL != "" {
			fileDIR, filename, err := helpers.SplitPresignedURL(createReplyCommentDTO.FileURL)
			if err != nil {
				logs.Error(err)
				return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
			}

			filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
			err = s.fileRepository.UpdateContentID(
				ctx,
				tx,
				createReplyComment.ID,
				filePath,
				models.FileTypeComment,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to update file of reply")
			}
		}

		// ต้องไม่ reply comment ตัวเองถึงสร้าง notification
		if commentByID.UserID != createReplyCommentDTO.UserID {
			createNotificationDTO := &models.Notification{
				Type:       models.NotificationTypeReply,
				Message:    "Reply on your comment",
				SenderID:   createReplyCommentDTO.UserID,
				ReceiverID: commentByID.UserID,
				PostID:     &postByID.ID,
				CommentID:  &createReplyComment.ID,
			}
			err = s.notificationService.CreateNotification(
				ctx,
				tx,
				createNotificationDTO,
			)
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

	reply, err := s.commentRepository.FindByIDWithUserAndPostRelations(
		ctx,
		s.db,
		createReplyComment.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find reply by id with user and post relations")
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		&reply.User,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	secureUserReply := &dto.SecureUser{
		ID:                   reply.UserID,
		Fullname:             reply.User.Fullname,
		Username:             reply.User.Username,
		Email:                reply.User.Email,
		DateOfBirth:          reply.User.DateOfBirth,
		ProfileUrl:           reply.User.ProfileUrl,
		ProfileBackgroundUrl: reply.User.ProfileBackgroundUrl,
		Info:                 reply.User.Info,
		Role:                 reply.User.Role,
		ProviderType:         reply.User.ProviderType,
		CreatedAt:            reply.User.CreatedAt,
		UpdatedAt:            reply.User.UpdatedAt,
	}
	createReplyDTOSocket := &socket.CreateCommentDTO{
		ID:      reply.ID,
		Message: reply.Message,
		PostID:  reply.PostID,
		Post: &socket.PostDTO{
			UserID: reply.Post.UserID,
		},
		UserID:    reply.UserID,
		ParentID:  reply.ParentID,
		User:      secureUserReply,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
	}
	replyResp := &CreatedComment{
		ID:      reply.ID,
		Message: reply.Message,
		PostID:  reply.PostID,
		Post: &Post{
			UserID: reply.Post.UserID,
		},
		UserID:    reply.UserID,
		ParentID:  reply.ParentID,
		User:      secureUserReply,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
	}

	// กรณีมีไฟล์
	if createReplyCommentDTO.FileURL != "" {
		fileURL, err := s.fileService.PresignGetFile(ctx, reply.ID)
		if err != nil {
			return nil, err
		}

		createReplyDTOSocket.FileURL = fileURL
		replyResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(createReplyDTOSocket)

	// ต้องไม่ reply comment ตัวเองถึง emit notification
	if commentByID.UserID != createReplyCommentDTO.UserID {
		notification, err := s.notificationRepository.FindByIDWithSenderRelation(
			ctx,
			s.db,
			notificationID,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification by id with sender relation")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&notification.Sender,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		secureSenderNotification := &dto.SecureUser{
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
		emitNotificationDTO := &socket.EmitNotificationDTO{
			ID:         notification.ID,
			Type:       notification.Type,
			Message:    notification.Message,
			IsRead:     notification.IsRead,
			SenderID:   notification.SenderID,
			Sender:     secureSenderNotification,
			ReceiverID: notification.ReceiverID,
			PostID:     notification.PostID,
			CommentID:  notification.CommentID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}

		go s.notificationSocket.EmitNotification(emitNotificationDTO)
	}

	return replyResp, nil
}

func (s *commentService) CreateTagReply(ctx context.Context, createTagReplyDTO *CreateTagReplyDTO) (*CreatedComment, error) {
	if createTagReplyDTO.Message == "" && createTagReplyDTO.FileURL == "" {
		return nil, errs.NewBadRequestErrorWithMessage("Create tag reply must contains with message or file")
	}

	err := helpers.ValidateUUID(createTagReplyDTO.PostID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	postIDParse, err := helpers.ParseUUID(createTagReplyDTO.PostID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	err = helpers.ValidateUUID(createTagReplyDTO.ParentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	parentIDParse, err := helpers.ParseUUID(createTagReplyDTO.ParentID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	err = helpers.ValidateUUID(createTagReplyDTO.ReplyID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	replyIDParse, err := helpers.ParseUUID(createTagReplyDTO.ReplyID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	postByID, err := s.postRepository.FindByID(
		ctx,
		s.db,
		*postIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	commentByID, err := s.commentRepository.FindByID(
		ctx,
		s.db,
		*parentIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Comment by id %v is not found", *parentIDParse))
	}

	replyByID, err := s.commentRepository.FindByID(
		ctx,
		s.db,
		*replyIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find reply by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Reply by id %v is not found", *replyIDParse))
	}

	createTagReply := &models.Comment{
		Message:       &createTagReplyDTO.Message,
		PostID:        postByID.ID,
		UserID:        createTagReplyDTO.UserID,
		ParentID:      &commentByID.ID,
		ReplyID:       &replyByID.ID,
		ReplyToUserID: &replyByID.UserID,
	}
	var notificationID uuid.UUID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.commentRepository.Create(
			ctx,
			tx,
			createTagReply,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to create tag reply")
		}

		// กรณีมีไฟล์ ผูกเข้ากับ tag
		if createTagReplyDTO.FileURL != "" {
			fileDIR, filename, err := helpers.SplitPresignedURL(createTagReplyDTO.FileURL)
			if err != nil {
				logs.Error(err)
				return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
			}

			filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
			err = s.fileRepository.UpdateContentID(
				ctx,
				tx,
				createTagReply.ID,
				filePath,
				models.FileTypeComment,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to update file of tag")
			}
		}

		// ต้องไม่ tag reply ตัวเองถึงสร้าง notification
		if replyByID.UserID != createTagReplyDTO.UserID {
			createNotificationDTO := &models.Notification{
				Type:       models.NotificationTypeTag,
				Message:    "Tag on your reply",
				SenderID:   createTagReplyDTO.UserID,
				ReceiverID: replyByID.UserID,
				PostID:     &postByID.ID,
				CommentID:  &createTagReply.ID,
			}
			err = s.notificationService.CreateNotification(
				ctx,
				tx,
				createNotificationDTO,
			)
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

	tag, err := s.commentRepository.FindByIDWithCommentRelations(
		ctx,
		s.db,
		createTagReply.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find tag by id with comment relations")
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		&tag.User,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		tag.ReplyToUser,
	)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	secureUserTag := &dto.SecureUser{
		ID:                   tag.UserID,
		Fullname:             tag.User.Fullname,
		Username:             tag.User.Username,
		Email:                tag.User.Email,
		DateOfBirth:          tag.User.DateOfBirth,
		ProfileUrl:           tag.User.ProfileUrl,
		ProfileBackgroundUrl: tag.User.ProfileBackgroundUrl,
		Info:                 tag.User.Info,
		Role:                 tag.User.Role,
		ProviderType:         tag.User.ProviderType,
		CreatedAt:            tag.User.CreatedAt,
		UpdatedAt:            tag.User.UpdatedAt,
	}
	secureReplyToUser := &dto.SecureUser{
		ID:                   *tag.ReplyToUserID,
		Fullname:             tag.ReplyToUser.Fullname,
		Username:             tag.ReplyToUser.Username,
		Email:                tag.ReplyToUser.Email,
		DateOfBirth:          tag.ReplyToUser.DateOfBirth,
		ProfileUrl:           tag.ReplyToUser.ProfileUrl,
		ProfileBackgroundUrl: tag.ReplyToUser.ProfileBackgroundUrl,
		Info:                 tag.ReplyToUser.Info,
		Role:                 tag.ReplyToUser.Role,
		ProviderType:         tag.ReplyToUser.ProviderType,
		CreatedAt:            tag.ReplyToUser.CreatedAt,
		UpdatedAt:            tag.ReplyToUser.UpdatedAt,
	}
	createTagDTOSocket := &socket.CreateCommentDTO{
		ID:      tag.ID,
		Message: tag.Message,
		PostID:  tag.PostID,
		Post: &socket.PostDTO{
			UserID: tag.Post.UserID,
		},
		UserID:        tag.UserID,
		ParentID:      tag.ParentID,
		ReplyID:       tag.ReplyID,
		ReplyToUserID: tag.ReplyToUserID,
		ReplyToUser:   secureReplyToUser,
		User:          secureUserTag,
		CreatedAt:     tag.CreatedAt,
		UpdatedAt:     tag.UpdatedAt,
	}
	tagResp := &CreatedComment{
		ID:      tag.ID,
		Message: tag.Message,
		PostID:  tag.PostID,
		Post: &Post{
			UserID: tag.Post.UserID,
		},
		UserID:        tag.UserID,
		ParentID:      tag.ParentID,
		ReplyID:       tag.ReplyID,
		ReplyToUserID: tag.ReplyToUserID,
		ReplyToUser:   secureReplyToUser,
		User:          secureUserTag,
		CreatedAt:     tag.CreatedAt,
		UpdatedAt:     tag.UpdatedAt,
	}

	// กรณีมีไฟล์
	if createTagReplyDTO.FileURL != "" {
		fileURL, err := s.fileService.PresignGetFile(ctx, tag.ID)
		if err != nil {
			return nil, err
		}

		createTagDTOSocket.FileURL = fileURL
		tagResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(createTagDTOSocket)

	// ต้องไม่ tag reply ตัวเองถึง emit notification
	if replyByID.UserID != createTagReplyDTO.UserID {
		notification, err := s.notificationRepository.FindByIDWithSenderRelation(
			ctx,
			s.db,
			notificationID,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification by id with sender relation")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&notification.Sender,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		secureSenderNotification := &dto.SecureUser{
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
		emitNotificationDTO := &socket.EmitNotificationDTO{
			ID:         notification.ID,
			Type:       notification.Type,
			Message:    notification.Message,
			IsRead:     notification.IsRead,
			SenderID:   notification.SenderID,
			Sender:     secureSenderNotification,
			ReceiverID: notification.ReceiverID,
			PostID:     notification.PostID,
			CommentID:  notification.CommentID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}

		go s.notificationSocket.EmitNotification(emitNotificationDTO)
	}

	return tagResp, nil
}

func (s *commentService) FindsWithPostIDCursorPagination(
	ctx context.Context,
	postID,
	cursor,
	limit string,
) (*CommentCursorPagination, error) {
	var nextCursor *uuid.UUID
	var cursorID *uuid.UUID

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
		logs.Warn(err)
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be string integer")
	}

	if limitInt <= 0 {
		return nil, errs.NewBadRequestErrorWithMessage("Invalid limit must be greater than 0")
	}

	postByID, err := s.postRepository.FindByID(
		ctx,
		s.db,
		*postIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	comments := []models.Comment{}
	commentsCursorPagination := []Comment{}
	if cursor == "" {
		comments, err = s.commentRepository.FindsByPostIDCursorPaginationWithCommentRelations(
			ctx,
			s.db,
			postByID.ID,
			nil,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find comments cursor pagination with comment relations")
		}
	} else {
		cursor, err := s.commentRepository.FindByIDCursor(
			ctx,
			s.db,
			*cursorID,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment cursor by comment id")
		}

		if helpers.IsErrRecordNotFound(err) {
			logs.Warn(err)
			return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Cursor by comment id %v is not found", *cursorID))
		}

		comments, err = s.commentRepository.FindsByPostIDCursorPaginationWithCommentRelations(
			ctx,
			s.db,
			postByID.ID,
			cursor,
			limitInt,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find comments cursor pagination with comment relations")
		}
	}

	for _, comment := range comments {
		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&comment.User,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}

		updateUserLikes := []Like{}
		for _, like := range comment.Likes {
			err = helpers.GetUserImage(
				ctx,
				s.presignClient,
				&like.User,
			)
			if err != nil {
				logs.Error(err)
				return nil, err
			}

			secureUserLike := &dto.SecureUser{
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
			updateUserLikes = append(updateUserLikes, Like{
				ID:        like.ID,
				UserID:    like.UserID,
				User:      secureUserLike,
				CommentID: *like.CommentID,
				CreatedAt: like.CreatedAt,
				UpdatedAt: like.UpdatedAt,
			})
		}

		fileURL, err := s.fileService.PresignGetFile(ctx, comment.ID)
		if err != nil {
			return nil, err
		}

		secureUserComment := &dto.SecureUser{
			ID:                   comment.UserID,
			Fullname:             comment.User.Fullname,
			Username:             comment.User.Username,
			Email:                comment.User.Email,
			DateOfBirth:          comment.User.DateOfBirth,
			ProfileUrl:           comment.User.ProfileUrl,
			ProfileBackgroundUrl: comment.User.ProfileBackgroundUrl,
			Info:                 comment.User.Info,
			Role:                 comment.User.Role,
			ProviderType:         comment.User.ProviderType,
			CreatedAt:            comment.User.CreatedAt,
			UpdatedAt:            comment.User.UpdatedAt,
		}

		replyOrTags := []ReplyOrTag{}
		for _, replyOrTag := range comment.Replies {
			err = helpers.GetUserImage(
				ctx,
				s.presignClient,
				&replyOrTag.User,
			)
			if err != nil {
				logs.Error(err)
				return nil, err
			}

			updateUserLikes := []Like{}
			for _, like := range replyOrTag.Likes {
				err = helpers.GetUserImage(
					ctx,
					s.presignClient,
					&like.User,
				)
				if err != nil {
					logs.Error(err)
					return nil, err
				}

				secureUserLike := &dto.SecureUser{
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
				updateUserLikes = append(updateUserLikes, Like{
					ID:        like.ID,
					UserID:    like.UserID,
					User:      secureUserLike,
					CommentID: *like.CommentID,
					CreatedAt: like.CreatedAt,
					UpdatedAt: like.UpdatedAt,
				})
			}

			fileURL, err := s.fileService.PresignGetFile(ctx, replyOrTag.ID)
			if err != nil {
				return nil, err
			}

			secureUserReplyOrTag := &dto.SecureUser{
				ID:                   replyOrTag.UserID,
				Fullname:             replyOrTag.User.Fullname,
				Username:             replyOrTag.User.Username,
				Email:                replyOrTag.User.Email,
				DateOfBirth:          replyOrTag.User.DateOfBirth,
				ProfileUrl:           replyOrTag.User.ProfileUrl,
				ProfileBackgroundUrl: replyOrTag.User.ProfileBackgroundUrl,
				Info:                 replyOrTag.User.Info,
				Role:                 replyOrTag.User.Role,
				ProviderType:         replyOrTag.User.ProviderType,
				CreatedAt:            replyOrTag.User.CreatedAt,
				UpdatedAt:            replyOrTag.User.UpdatedAt,
			}
			updateReplyOrTag := ReplyOrTag{
				ID:            replyOrTag.ID,
				Message:       replyOrTag.Message,
				PostID:        replyOrTag.PostID,
				UserID:        replyOrTag.UserID,
				User:          secureUserReplyOrTag,
				ParentID:      *replyOrTag.ParentID,
				ReplyID:       replyOrTag.ReplyID,
				ReplyToUserID: replyOrTag.ReplyToUserID,
				Likes:         updateUserLikes,
				FileURL:       fileURL,
				CreatedAt:     replyOrTag.CreatedAt,
				UpdatedAt:     replyOrTag.UpdatedAt,
			}
			if replyOrTag.ReplyToUserID != nil {
				err = helpers.GetUserImage(
					ctx,
					s.presignClient,
					replyOrTag.ReplyToUser,
				)
				if err != nil {
					logs.Error(err)
					return nil, err
				}

				secureReplyToUser := &dto.SecureUser{
					ID:                   *replyOrTag.ReplyToUserID,
					Fullname:             replyOrTag.ReplyToUser.Fullname,
					Username:             replyOrTag.ReplyToUser.Username,
					Email:                replyOrTag.ReplyToUser.Email,
					DateOfBirth:          replyOrTag.ReplyToUser.DateOfBirth,
					ProfileUrl:           replyOrTag.ReplyToUser.ProfileUrl,
					ProfileBackgroundUrl: replyOrTag.ReplyToUser.ProfileBackgroundUrl,
					Info:                 replyOrTag.ReplyToUser.Info,
					Role:                 replyOrTag.ReplyToUser.Role,
					ProviderType:         replyOrTag.ReplyToUser.ProviderType,
					CreatedAt:            replyOrTag.ReplyToUser.CreatedAt,
					UpdatedAt:            replyOrTag.ReplyToUser.UpdatedAt,
				}
				updateReplyOrTag.ReplyToUser = secureReplyToUser
			}
			replyOrTags = append(replyOrTags, updateReplyOrTag)
		}

		commentCursorPagination := Comment{
			ID:           comment.ID,
			Message:      comment.Message,
			PostID:       comment.PostID,
			UserID:       comment.UserID,
			User:         secureUserComment,
			Likes:        updateUserLikes,
			Replies:      replyOrTags,
			FileURL:      fileURL,
			RepliesCount: len(comment.Replies),
			CreatedAt:    comment.CreatedAt,
			UpdatedAt:    comment.UpdatedAt,
		}
		commentsCursorPagination = append(commentsCursorPagination, commentCursorPagination)
	}

	if len(commentsCursorPagination) == limitInt {
		nextCursor = &commentsCursorPagination[len(commentsCursorPagination)-1].ID
	}

	commentCursorPagination := &CommentCursorPagination{
		Comments:   commentsCursorPagination,
		NextCursor: nextCursor,
	}

	return commentCursorPagination, nil
}

func (s *commentService) UpdateComment(ctx context.Context, updateCommentDTO *UpdateCommentDTO) (*UpdatedComment, error) {
	if updateCommentDTO.Message == nil && updateCommentDTO.FileURL == "" {
		return nil, errs.NewBadRequestErrorWithMessage("Update comment must contains with message or file")
	}

	err := helpers.ValidateUUID(updateCommentDTO.CommentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	commentIDParse, err := helpers.ParseUUID(updateCommentDTO.CommentID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	commentByID, err := s.commentRepository.FindByID(
		ctx,
		s.db,
		*commentIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Comment by id %v is not found", *commentIDParse))
	}

	// ถ้าไม่ใช่คอมเมนท์ตัวเองห้าม
	if commentByID.UserID != updateCommentDTO.UserID {
		return nil, errs.NewBadRequestErrorWithMessage("You can only update your own comments")
	}

	updatedComment := &models.Comment{}
	var file *models.File
	err = s.db.Transaction(func(tx *gorm.DB) error {
		updateComment := &models.Comment{
			Message: updateCommentDTO.Message,
		}
		updatedComment, err = s.commentRepository.Update(
			ctx,
			tx,
			updateCommentDTO.UserID,
			commentByID.ID,
			updateComment,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to update comment")
		}

		// กรณีมีไฟล์และลบรูปปัจจุบัน
		if updateCommentDTO.FileURL != "" && updateCommentDTO.ShouldDeleteCurrentFile {
			file, err = s.fileRepository.FindByContentID(
				ctx,
				tx,
				updatedComment.ID,
			)
			if err != nil && !helpers.IsErrRecordNotFound(err) {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to find file of comment")
			}

			if file != nil {
				err = s.fileRepository.Delete(
					ctx,
					tx,
					file.ID,
					file.Filename,
					file.FileType,
				)
				if err != nil {
					logs.Error(err)
					return errs.NewInternalServerErrorWithMessage("Failed to delete file of comment")
				}
			}

			fileDIR, filename, err := helpers.SplitPresignedURL(updateCommentDTO.FileURL)
			if err != nil {
				logs.Error(err)
				return errs.NewUnexpectedErrorWithMessage("Failed to split presigned url")
			}

			filePath := fmt.Sprintf("%s/%s", fileDIR, filename)
			err = s.fileRepository.UpdateContentID(
				ctx,
				tx,
				updatedComment.ID,
				filePath,
				models.FileTypeComment,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to update file of comment")
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

	if file != nil {
		_, err = helpers.DeleteObject(
			ctx,
			s.s3Client,
			file.Filename,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to delete file from bucket")
		}
	}

	fileURL, err := s.fileService.PresignGetFile(ctx, updatedComment.ID)
	if err != nil {
		return nil, err
	}

	updateCommentSocketDTO := &socket.UpdateCommentDTO{
		ID:      updatedComment.ID,
		Message: updatedComment.Message,
		PostID:  updatedComment.PostID,
		FileURL: fileURL,
	}
	updateCommentResp := &UpdatedComment{
		ID:      updatedComment.ID,
		Message: updatedComment.Message,
		PostID:  updatedComment.PostID,
		FileURL: fileURL,
	}

	go s.commentSocket.EmitUpdate(updateCommentSocketDTO)
	return updateCommentResp, nil
}

func (s *commentService) DeleteComment(
	ctx context.Context,
	userID uuid.UUID,
	commentID string,
) (*DeletedComment, error) {
	err := helpers.ValidateUUID(commentID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	commentIDParse, err := helpers.ParseUUID(commentID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	commentByID, err := s.commentRepository.FindByIDWithPostAndParentRelations(
		ctx,
		s.db,
		*commentIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id with post and parent relations")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Comment by id %v is not found", *commentIDParse))
	}

	// ถ้าไม่ใช่คอมเมนท์ตัวเองห้าม
	if commentByID.UserID != userID {
		return nil, errs.NewBadRequestErrorWithMessage("You can only delete your own comments")
	}

	file, err := s.fileRepository.FindByContentID(
		ctx,
		s.db,
		commentByID.ID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find file of comment")
	}

	var notification *models.Notification

	// กรณี comment
	if commentByID.ParentID == nil {
		notification, err = s.notificationRepository.FindOfComment(
			ctx,
			s.db,
			userID,
			commentByID.Post.UserID,
			commentByID.ID,
			models.NotificationTypeComment,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification of comment")
		}
	} else if commentByID.ParentID != nil && commentByID.ReplyID == nil && commentByID.ReplyToUserID == nil {
		// กรณี reply
		notification, err = s.notificationRepository.FindOfComment(
			ctx,
			s.db,
			userID,
			commentByID.Parent.UserID,
			commentByID.ID,
			models.NotificationTypeReply,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification of reply")
		}
	} else {
		// กรณี tag
		notification, err = s.notificationRepository.FindOfComment(
			ctx,
			s.db,
			userID,
			*commentByID.ReplyToUserID,
			commentByID.ID,
			models.NotificationTypeTag,
		)
		if err != nil && !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find notification of tag")
		}
	}

	deletedComment := &models.Comment{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// กรณีมีไฟล์
		if file != nil {
			err = s.fileRepository.Delete(
				ctx,
				tx,
				file.ID,
				file.Filename,
				file.FileType,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to delete file of comment")
			}
		}

		deletedComment, err = s.commentRepository.Delete(
			ctx,
			tx,
			userID,
			commentByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete comment")
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

	if file != nil {
		_, err = helpers.DeleteObject(
			ctx,
			s.s3Client,
			file.Filename,
		)
		if err != nil {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to delete file from bucket")
		}
	}

	deleteCommentDTOSocket := &socket.DeleteCommentDTO{
		ID:     deletedComment.ID,
		PostID: deletedComment.PostID,
		Post: &socket.PostDTO{
			UserID: commentByID.Post.UserID,
		},
		ParentID: deletedComment.ParentID,
	}
	go s.commentSocket.EmitDelete(deleteCommentDTOSocket)

	if notification != nil {
		emitDeleteNotificationDTO := &socket.EmitDeleteNotificationDTO{
			ID:         notification.ID,
			ReceiverID: notification.ReceiverID,
		}
		go s.notificationSocket.EmitDeleteNotification(emitDeleteNotificationDTO)
	}

	deleteCommentResp := &DeletedComment{
		ID:     deletedComment.ID,
		PostID: deletedComment.PostID,
		Post: &Post{
			UserID: commentByID.Post.UserID,
		},
		ParentID: deletedComment.ParentID,
	}
	return deleteCommentResp, nil
}

func (s *commentService) ToggleLike(
	ctx context.Context,
	userID uuid.UUID,
	postID,
	commentID string,
) (string, *Like, error) {
	err := helpers.ValidateUUID(postID)
	if err != nil {
		logs.Warn(err)
		return "", nil, err
	}

	postIDParse, err := helpers.ParseUUID(postID)
	if err != nil {
		logs.Error(err)
		return "", nil, err
	}

	err = helpers.ValidateUUID(commentID)
	if err != nil {
		logs.Warn(err)
		return "", nil, err
	}

	commentIDParse, err := helpers.ParseUUID(commentID)
	if err != nil {
		logs.Error(err)
		return "", nil, err
	}

	postByID, err := s.postRepository.FindByID(
		ctx,
		s.db,
		*postIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find post by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Post by id %v is not found", *postIDParse))
	}

	commentByID, err := s.commentRepository.FindByID(
		ctx,
		s.db,
		*commentIDParse,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id")
	}

	if helpers.IsErrRecordNotFound(err) {
		logs.Warn(err)
		return "", nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("Comment by id %v is not found", *commentIDParse))
	}

	_, err = s.likeRepository.FindOfComment(
		ctx,
		s.db,
		userID,
		commentByID.ID,
	)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find like of comment")
	}

	// กรณี like
	if helpers.IsErrRecordNotFound(err) {
		var createNotificationDTO *models.Notification

		if commentByID.UserID != userID {
			createNotificationDTO = &models.Notification{
				Type:       models.NotificationTypeLike,
				SenderID:   userID,
				ReceiverID: commentByID.UserID,
				PostID:     &postByID.ID,
				CommentID:  &commentByID.ID,
				Message:    "Like your comment",
			}
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			createLike := &models.Like{
				UserID:    userID,
				CommentID: &commentByID.ID,
			}
			err = s.likeRepository.Create(
				ctx,
				tx,
				createLike,
			)
			if err != nil {
				logs.Error(err)
				return errs.NewInternalServerErrorWithMessage("Failed to create like of comment")
			}

			// ต้องไม่ like comment ตัวเองถึงสร้าง notification
			if commentByID.UserID != userID {
				err = s.notificationService.CreateNotification(
					ctx,
					tx,
					createNotificationDTO,
				)
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
			return "", nil, err
		}

		createdLike, err := s.likeRepository.FindOfCommentWithUserRelation(
			ctx,
			s.db,
			userID,
			commentByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find like of comment with user relation")
		}

		err = helpers.GetUserImage(
			ctx,
			s.presignClient,
			&createdLike.User,
		)
		if err != nil {
			logs.Error(err)
			return "", nil, err
		}

		secureUserLike := &dto.SecureUser{
			ID:                   createdLike.UserID,
			Fullname:             createdLike.User.Fullname,
			Username:             createdLike.User.Username,
			Email:                createdLike.User.Email,
			DateOfBirth:          createdLike.User.DateOfBirth,
			ProfileUrl:           createdLike.User.ProfileUrl,
			ProfileBackgroundUrl: createdLike.User.ProfileBackgroundUrl,
			Info:                 createdLike.User.Info,
			Role:                 createdLike.User.Role,
			ProviderType:         createdLike.User.ProviderType,
			CreatedAt:            createdLike.User.CreatedAt,
			UpdatedAt:            createdLike.User.UpdatedAt,
		}
		commentLikeDTO := &socket.CommentLikeDTO{
			ID:        createdLike.ID,
			UserID:    createdLike.UserID,
			User:      secureUserLike,
			CommentID: *createdLike.CommentID,
			CreatedAt: createdLike.CreatedAt,
			UpdatedAt: createdLike.UpdatedAt,
		}
		go s.commentSocket.EmitToggleLike(commentLikeDTO)

		// ต้องไม่ like comment ตัวเองถึง emit notification
		if commentByID.UserID != userID {
			createdNotification, err := s.notificationRepository.FindByIDWithSenderRelation(
				ctx,
				s.db,
				createNotificationDTO.ID,
			)
			if err != nil {
				logs.Error(err)
				return "", nil, errs.NewInternalServerErrorWithMessage("Failed to find notification by id with sender relation")
			}

			err = helpers.GetUserImage(
				ctx,
				s.presignClient,
				&createdNotification.Sender,
			)
			if err != nil {
				logs.Error(err)
				return "", nil, err
			}

			secureUserNotification := &dto.SecureUser{
				ID:                   createdNotification.SenderID,
				Fullname:             createdNotification.Sender.Fullname,
				Username:             createdNotification.Sender.Username,
				Email:                createdNotification.Sender.Email,
				DateOfBirth:          createdNotification.Sender.DateOfBirth,
				ProfileUrl:           createdNotification.Sender.ProfileUrl,
				ProfileBackgroundUrl: createdNotification.Sender.ProfileBackgroundUrl,
				Info:                 createdNotification.Sender.Info,
				Role:                 createdNotification.Sender.Role,
				ProviderType:         createdNotification.Sender.ProviderType,
				CreatedAt:            createdNotification.Sender.CreatedAt,
				UpdatedAt:            createdNotification.Sender.UpdatedAt,
			}
			emitNotificationDTO := &socket.EmitNotificationDTO{
				ID:         createdNotification.ID,
				Type:       createdNotification.Type,
				Message:    createdNotification.Message,
				IsRead:     createdNotification.IsRead,
				SenderID:   createdNotification.SenderID,
				Sender:     secureUserNotification,
				ReceiverID: createdNotification.ReceiverID,
				PostID:     createdNotification.PostID,
				CommentID:  createdNotification.CommentID,
				CreatedAt:  createdNotification.CreatedAt,
				UpdatedAt:  createdNotification.UpdatedAt,
			}
			go s.notificationSocket.EmitNotification(emitNotificationDTO)
		}

		likeResp := &Like{
			ID:        createdLike.ID,
			UserID:    createdLike.UserID,
			User:      secureUserLike,
			CommentID: *createdLike.CommentID,
			CreatedAt: createdLike.CreatedAt,
			UpdatedAt: createdLike.UpdatedAt,
		}
		return "Like successfully", likeResp, nil
	}

	// กรณี unlike
	deletedLike := &models.Like{}
	deletedNotification := &models.Notification{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		deletedLike, err = s.likeRepository.DeleteOfComment(
			ctx,
			tx,
			userID,
			commentByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete like of comment")
		}

		deletedNotification, err = s.notificationRepository.DeleteOfLikeComment(
			ctx,
			tx,
			userID,
			commentByID.UserID,
			postByID.ID,
			commentByID.ID,
		)
		if err != nil {
			logs.Error(err)
			return errs.NewInternalServerErrorWithMessage("Failed to delete notification of like comment")
		}

		return nil
	})
	if err != nil {
		_, ok := err.(*errs.AppError)
		if !ok {
			logs.Error(err)
		}
		return "", nil, err
	}

	commentLikeDTO := &socket.CommentLikeDTO{
		ID:        deletedLike.ID,
		UserID:    deletedLike.UserID,
		CommentID: *deletedLike.CommentID,
		CreatedAt: deletedLike.CreatedAt,
		UpdatedAt: deletedLike.UpdatedAt,
	}
	go s.commentSocket.EmitToggleLike(commentLikeDTO)

	if deletedNotification != nil {
		emitDeleteNotificationDTO := &socket.EmitDeleteNotificationDTO{
			ID:         deletedNotification.ID,
			ReceiverID: deletedNotification.ReceiverID,
		}
		go s.notificationSocket.EmitDeleteNotification(emitDeleteNotificationDTO)
	}

	likeResp := &Like{
		ID:        deletedLike.ID,
		UserID:    deletedLike.UserID,
		CommentID: *deletedLike.CommentID,
		CreatedAt: deletedLike.CreatedAt,
		UpdatedAt: deletedLike.UpdatedAt,
	}
	return "Unlike successfully", likeResp, nil
}
