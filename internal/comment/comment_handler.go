package comment

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type UpdateCommentRequest struct {
	Message                 *string `json:"message"`
	FileURL                 string  `json:"fileUrl" binding:"omitempty,presignedurl"`
	ShouldDeleteCurrentFile bool    `json:"shouldDeleteCurrentFile"`
}

type CreateCommentRequest struct {
	Message string `json:"message"`
	FileURL string `json:"fileUrl" binding:"omitempty,presignedurl"`
}

type CommentHandler interface {
	UploadFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	CreateComment(c *gin.Context)
	CreateReplyComment(c *gin.Context)
	CreateTagReply(c *gin.Context)
	FindsWithPostIDCursorPagination(c *gin.Context)
	UpdateComment(c *gin.Context)
	DeleteComment(c *gin.Context)
	ToggleLike(c *gin.Context)
}

type commentHandler struct {
	commentService CommentService
	fileService    file.FileService
}

func NewCommentHandler(commentService CommentService, fileService file.FileService) CommentHandler {
	return &commentHandler{
		commentService: commentService,
		fileService:    fileService,
	}
}

// UploadFile godoc
//
//	@Description	authentication and upload file
//	@Tags			comment
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"single image file"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=file.FileURL}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/comment/upload-file [post]
func (h *commentHandler) UploadFile(c *gin.Context) {
	ctx := c.Request.Context()

	errField := map[string]string{}
	form, err := c.FormFile("file")
	if err != nil {
		errField["file"] = "This field is required"
		response.AbortWithBadRequestErrorFields(c, errField)
		return
	}

	fileOpen, err := form.Open()
	if err != nil {
		response.AbortWithInternalServerError(c, err)
		return
	}

	fileDataDTO := &file.FileDataDTO{
		Filename:    form.Filename,
		ContentType: form.Header.Get("Content-Type"),
		Body:        fileOpen,
		Size:        form.Size,
	}
	fileURL, err := h.fileService.UploadFile(
		ctx,
		fileDataDTO,
		models.FileTypeComment,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Upload file successfully", fileURL)
}

// DeleteFile godoc
//
//	@Description	authentication and delete file
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		file.DeleteFileRequest	true	"delete file payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/comment/delete-file [delete]
func (h *commentHandler) DeleteFile(c *gin.Context) {
	ctx := c.Request.Context()

	deleteFileRequest := &file.DeleteFileRequest{}
	err := c.ShouldBind(deleteFileRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	deleteFileDTO := &file.DeleteFileDTO{
		FileURL:  deleteFileRequest.FileURL,
		FileType: models.FileTypeComment,
	}
	err = h.fileService.DeleteFile(ctx, deleteFileDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Delete file successfully", nil)
}

// CreateComment godoc
//
//	@Description	authentication create comment and socket emit comment and notification to client
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		string					true	"uuid for post id"
//	@Param			payload	body		CreateCommentRequest	false	"create comment payload"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=CreatedComment}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/comment/create/{postID} [post]
func (h *commentHandler) CreateComment(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	createCommentRequest := &CreateCommentRequest{}
	err := c.ShouldBind(createCommentRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	postID := c.Param("postID")
	createCommentDTO := &CreateCommentDTO{
		Message: createCommentRequest.Message,
		FileURL: createCommentRequest.FileURL,
		PostID:  postID,
		UserID:  user.ID,
	}
	createdComment, err := h.commentService.CreateComment(ctx, createCommentDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Create comment successfully", createdComment)
}

// CreateReplyComment godoc
//
//	@Description	authentication create reply comment and socket emit reply and notification to client
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			postID		path		string					true	"uuid for post id"
//	@Param			parentID	path		string					true	"uuid for comment id"
//	@Param			payload		body		CreateCommentRequest	false	"create reply comment payload"
//	@Success		201			{object}	response.SwaggerResponseWithData{data=CreatedComment}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/comment/reply/create/{postID}/{parentID} [post]
func (h *commentHandler) CreateReplyComment(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	createCommentRequest := &CreateCommentRequest{}
	err := c.ShouldBind(createCommentRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	postID := c.Param("postID")
	parentID := c.Param("parentID")
	createReplyCommentDTO := &CreateReplyCommentDTO{
		Message:  createCommentRequest.Message,
		FileURL:  createCommentRequest.FileURL,
		PostID:   postID,
		UserID:   user.ID,
		ParentID: parentID,
	}
	createdReply, err := h.commentService.CreateReplyComment(ctx, createReplyCommentDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Create reply comment successfully", createdReply)
}

// CreateTagReply godoc
//
//	@Description	authentication create tag reply and socket emit tag and notification to client
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			postID		path		string					true	"uuid for post id"
//	@Param			parentID	path		string					true	"uuid for comment id"
//	@Param			replyID		path		string					true	"uuid for reply id"
//	@Param			payload		body		CreateCommentRequest	false	"create tag reply payload"
//	@Success		201			{object}	response.SwaggerResponseWithData{data=CreatedComment}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/comment/tag/create/{postID}/{parentID}/{replyID} [post]
func (h *commentHandler) CreateTagReply(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	createCommentRequest := &CreateCommentRequest{}
	err := c.ShouldBind(createCommentRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	postID := c.Param("postID")
	parentID := c.Param("parentID")
	replyID := c.Param("replyID")
	createTagReplyDTO := &CreateTagReplyDTO{
		Message:  createCommentRequest.Message,
		FileURL:  createCommentRequest.FileURL,
		PostID:   postID,
		UserID:   user.ID,
		ParentID: parentID,
		ReplyID:  replyID,
	}
	createdTag, err := h.commentService.CreateTagReply(ctx, createTagReplyDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Create tag reply successfully", createdTag)
}

// FindsWithPostIDCursorPagination godoc
//
//	@Description	authentication and find comments cursor pagination
//	@Tags			comment
//	@Produce		json
//	@Param			postID	path		string	true	"uuid for post id"
//	@Param			cursor	query		string	false	"cursor uuid for comment id"
//	@Param			limit	query		int		true	"limit for comments cursor pagination"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=CommentCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/comment/finds/{postID} [get]
func (h *commentHandler) FindsWithPostIDCursorPagination(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("postID")
	cursor := c.Query("cursor")
	limit := c.Query("limit")
	commentCursorPagination, err := h.commentService.FindsWithPostIDCursorPagination(
		ctx,
		postID,
		cursor,
		limit,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Comments retrive successfully", commentCursorPagination)
}

// UpdateComment godoc
//
//	@Description	authentication update comment and socket emit comment to client
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			commentID	path		string					true	"uuid for comment id"
//	@Param			payload		body		UpdateCommentRequest	false	"update comment payload"
//	@Success		200			{object}	response.SwaggerResponseWithData{data=UpdatedComment}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/comment/update/{commentID} [patch]
func (h *commentHandler) UpdateComment(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	updateCommentRequest := &UpdateCommentRequest{}
	err := c.ShouldBind(updateCommentRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	commentID := c.Param("commentID")
	updateCommentDTO := &UpdateCommentDTO{
		UserID:                  user.ID,
		Message:                 updateCommentRequest.Message,
		FileURL:                 updateCommentRequest.FileURL,
		CommentID:               commentID,
		ShouldDeleteCurrentFile: updateCommentRequest.ShouldDeleteCurrentFile,
	}
	updatedComment, err := h.commentService.UpdateComment(ctx, updateCommentDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Update comment successfully", updatedComment)
}

// DeleteComment godoc
//
//	@Description	authentication delete comment and socket emit comment and notification to client
//	@Tags			comment
//	@Produce		json
//	@Param			commentID	path		string	true	"uuid for comment id"
//	@Success		200			{object}	response.SwaggerResponseWithData{data=DeletedComment}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/comment/delete/{commentID} [delete]
func (h *commentHandler) DeleteComment(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	commentID := c.Param("commentID")
	deletedComment, err := h.commentService.DeleteComment(
		ctx,
		user.ID,
		commentID,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Delete comment successfully", deletedComment)
}

// ToggleLike godoc
//
//	@Description	authentication toggle like or unlike comment and socket emit like or unlike comment, notification to client
//	@Tags			comment
//	@Produce		json
//	@Param			postID		path		string	true	"uuid for post id"
//	@Param			commentID	path		string	true	"uuid for comment id"
//	@Success		200			{object}	response.SwaggerResponseWithData{data=Like}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/comment/toggle-like/{postID}/{commentID} [post]
func (h *commentHandler) ToggleLike(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	postID := c.Param("postID")
	commentID := c.Param("commentID")
	message, like, err := h.commentService.ToggleLike(
		ctx,
		user.ID,
		postID,
		commentID,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, message, like)
}
