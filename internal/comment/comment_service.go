package comment

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/file"
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

type CreatedComment struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	PostID        uuid.UUID        `json:"postId"`
	UserID        uuid.UUID        `json:"userId"`
	ParentID      *uuid.UUID       `json:"parentId,omitempty"`
	ReplyID       *uuid.UUID       `json:"replyId,omitempty"`
	ReplyToUserID *uuid.UUID       `json:"replyToUserId,omitempty"`
	ReplyToUser   *user.SecureUser `json:"replyToUser,omitempty"`
	User          *user.SecureUser `json:"user"`
	FileURL       string           `json:"fileUrl,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type CommentService interface {
	CreateComment(ctx context.Context, createCommentDTO *CreateCommentDTO) (*CreatedComment, error)
	CreateReplyComment(ctx context.Context, createReplyCommentDTO *CreateReplyCommentDTO) (*CreatedComment, error)
	CreateTagReply(ctx context.Context, createTagReplyDTO *CreateTagReplyDTO) (*CreatedComment, error)
}

type commentService struct {
	db                     *gorm.DB
	redisClient            *redis.Client
	s3Client               *s3.Client
	commentRepository      CommentRepository
	userRepository         user.UserRepository
	postRepository         post.PostRepository
	fileRepository         file.FileRepository
	notificationRepository notification.NotificationRepository
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
	commentRepository CommentRepository,
	userRepository user.UserRepository,
	postRepository post.PostRepository,
	fileRepository file.FileRepository,
	notificationRepository notification.NotificationRepository,
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
		commentRepository:      commentRepository,
		userRepository:         userRepository,
		postRepository:         postRepository,
		fileRepository:         fileRepository,
		notificationRepository: notificationRepository,
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

	comment, err := s.commentRepository.FindByIDWithUserRelation(
		ctx,
		s.db,
		createComment.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find comment by id with user relation")
	}

	err = s.userService.GetUserImage(ctx, &comment.User)
	if err != nil {
		return nil, err
	}

	secureUserComment := &user.SecureUser{
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
	commentDTO := &socket.CommentDTO{
		ID:        comment.ID,
		Message:   comment.Message,
		PostID:    comment.PostID,
		UserID:    comment.UserID,
		User:      secureUserComment,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
	commentResp := &CreatedComment{
		ID:        comment.ID,
		Message:   comment.Message,
		PostID:    comment.PostID,
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

		commentDTO.FileURL = fileURL
		commentResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(commentDTO)

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

		err = s.userService.GetUserImage(ctx, &notification.Sender)
		if err != nil {
			return nil, err
		}

		secureSenderNotification := &user.SecureUser{
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

	reply, err := s.commentRepository.FindByIDWithUserRelation(
		ctx,
		s.db,
		createReplyComment.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find reply by id with user relation")
	}

	err = s.userService.GetUserImage(ctx, &reply.User)
	if err != nil {
		return nil, err
	}

	secureUserReply := &user.SecureUser{
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
	replyDTO := &socket.CommentDTO{
		ID:        reply.ID,
		Message:   reply.Message,
		PostID:    reply.PostID,
		UserID:    reply.UserID,
		ParentID:  reply.ParentID,
		User:      secureUserReply,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
	}
	replyResp := &CreatedComment{
		ID:        reply.ID,
		Message:   reply.Message,
		PostID:    reply.PostID,
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

		replyDTO.FileURL = fileURL
		replyResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(replyDTO)

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

		err = s.userService.GetUserImage(ctx, &notification.Sender)
		if err != nil {
			return nil, err
		}

		secureSenderNotification := &user.SecureUser{
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

	tag, err := s.commentRepository.FindByIDWithUserAndReplyToUserRelations(
		ctx,
		s.db,
		createTagReply.ID,
	)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewInternalServerErrorWithMessage("Failed to find tag by id with user and reply to user relations")
	}

	err = s.userService.GetUserImage(ctx, &tag.User)
	if err != nil {
		return nil, err
	}

	err = s.userService.GetUserImage(ctx, tag.ReplyToUser)
	if err != nil {
		return nil, err
	}

	secureUserTag := &user.SecureUser{
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
	secureReplyToUser := &user.SecureUser{
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
	tagDTO := &socket.CommentDTO{
		ID:            tag.ID,
		Message:       tag.Message,
		PostID:        tag.PostID,
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
		ID:            tag.ID,
		Message:       tag.Message,
		PostID:        tag.PostID,
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

		tagDTO.FileURL = fileURL
		tagResp.FileURL = fileURL
	}

	go s.commentSocket.EmitCreate(tagDTO)

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

		err = s.userService.GetUserImage(ctx, &notification.Sender)
		if err != nil {
			return nil, err
		}

		secureSenderNotification := &user.SecureUser{
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
