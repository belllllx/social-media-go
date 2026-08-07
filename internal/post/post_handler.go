package post

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type UpdatePostRequest struct {
	Message                  *string  `json:"message"`
	FilesURL                 []string `json:"filesUrl" binding:"dive,presignedurl"`
	ShouldDeleteCurrentFiles bool     `json:"shouldDeleteCurrentFiles"`
	IsSharePost              bool     `json:"isSharePost"`
}

type CreateSharePostRequest struct {
	Message string `json:"message"`
}

type CreatePostRequest struct {
	Message  string   `json:"message"`
	FilesURL []string `json:"filesUrl" binding:"dive,presignedurl"`
}

type PostHandler interface {
	UploadFiles(c *gin.Context)
	DeleteFile(c *gin.Context)
	CreatePost(c *gin.Context)
	CreateSharePost(c *gin.Context)
	FindsCursorPagination(c *gin.Context)
	FindsWithUserIDCursorPagination(c *gin.Context)
	FindWithID(c *gin.Context)
	UpdatePost(c *gin.Context)
	DeletePost(c *gin.Context)
	ToggleLike(c *gin.Context)
}

type postHandler struct {
	fileService file.FileService
	postService PostService
}

func NewPostHandler(fileService file.FileService, postService PostService) PostHandler {
	return &postHandler{
		fileService: fileService,
		postService: postService,
	}
}

// UploadFiles godoc
//
//	@Description	authentication and upload files
//	@Tags			post
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			files	formData	file	true	"multiple image files"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=file.FilesURL}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/upload-files [post]
func (h *postHandler) UploadFiles(c *gin.Context) {
	ctx := c.Request.Context()

	errFields := map[string]string{}
	form, err := c.MultipartForm()
	if err != nil {
		errFields["files"] = "This field is required"
		response.AbortWithBadRequestErrorFields(c, errFields)
		return
	}

	filesHeader := form.File["files"]
	if len(filesHeader) == 0 {
		errFields["files"] = "This field is not empty"
		response.AbortWithBadRequestErrorFields(c, errFields)
		return
	} else if len(filesHeader) > 10 {
		response.AbortWithBadRequestMessage(c, "Upload file exceeds 10")
		return
	}

	filesDataDTO := []file.FileDataDTO{}
	for _, fileHeader := range filesHeader {
		fileOpen, err := fileHeader.Open()
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		filesDataDTO = append(filesDataDTO, file.FileDataDTO{
			Filename:    fileHeader.Filename,
			ContentType: fileHeader.Header.Get("Content-Type"),
			Body:        fileOpen,
			Size:        fileHeader.Size,
		})
	}

	filesURL, err := h.fileService.UploadFiles(
		ctx,
		filesDataDTO,
		models.FileTypePost,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Upload files successfully", filesURL)
}

// DeleteFile godoc
//
//	@Description	authentication and delete file
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		file.DeleteFileRequest	true	"delete file payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/delete-file [delete]
func (h *postHandler) DeleteFile(c *gin.Context) {
	ctx := c.Request.Context()

	deleteFileRequest := &file.DeleteFileRequest{}
	err := c.ShouldBind(deleteFileRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	deleteFileDTO := &file.DeleteFileDTO{
		FileURL:  deleteFileRequest.FileURL,
		FileType: models.FileTypePost,
	}
	err = h.fileService.DeleteFile(ctx, deleteFileDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Delete file successfully", nil)
}

// CreatePost godoc
//
//	@Description	authentication create post, notifications and socket emit post, notifications to client
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreatePostRequest	false	"create post payload"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=CreatedPost}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/create [post]
func (h *postHandler) CreatePost(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowRelations)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	createPostRequest := &CreatePostRequest{}
	err := c.ShouldBind(createPostRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	createPostDTO := &CreatePostDTO{
		Message:  createPostRequest.Message,
		FilesURL: createPostRequest.FilesURL,
		UserID:   user.ID,
	}
	createdPost, err := h.postService.CreatePost(ctx, createPostDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Post create successfully", createdPost)
}

// CreateSharePost godoc
//
//	@Description	authentication create share post, notification and socket emit share post, notification to client
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			parentID	path		string					true	"uuid for parent post id"
//	@Param			payload		body		CreateSharePostRequest	false	"create share post payload"
//	@Success		201			{object}	response.SwaggerResponseWithData{data=CreatedSharePost}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/post/share/create/{parentID} [post]
func (h *postHandler) CreateSharePost(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowRelations)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	createSharePostRequest := &CreateSharePostRequest{}
	err := c.ShouldBind(createSharePostRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	parentID := c.Param("parentID")
	createSharePostDTO := &CreateSharePostDTO{
		Message:  createSharePostRequest.Message,
		UserID:   user.ID,
		ParentID: parentID,
	}
	createdSharePost, err := h.postService.CreateSharePost(ctx, createSharePostDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Share post create successfully", createdSharePost)
}

// FindsCursorPagination godoc
//
//	@Description	authentication and find posts cursor pagination
//	@Tags			post
//	@Produce		json
//	@Param			cursor	query		string	false	"cursor uuid for post id"
//	@Param			limit	query		int		true	"limit for posts cursor pagination"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=PostCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/finds [get]
func (h *postHandler) FindsCursorPagination(c *gin.Context) {
	ctx := c.Request.Context()

	cursor := c.Query("cursor")
	limit := c.Query("limit")
	postCursorPagination, err := h.postService.FindsCursorPagination(
		ctx,
		cursor,
		limit,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Posts retrive successfully", postCursorPagination)
}

// FindsWithUserIDCursorPagination godoc
//
//	@Description	authentication and find posts with user id cursor pagination
//	@Tags			post
//	@Produce		json
//	@Param			userID	path		string	true	"uuid for user id"
//	@Param			cursor	query		string	false	"cursor uuid for post id"
//	@Param			limit	query		int		true	"limit for posts cursor pagination"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=PostCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/finds/{userID} [get]
func (h *postHandler) FindsWithUserIDCursorPagination(c *gin.Context) {
	ctx := c.Request.Context()

	userID := c.Param("userID")
	cursor := c.Query("cursor")
	limit := c.Query("limit")
	postCursorPagination, err := h.postService.FindsWithUserIDCursorPagination(
		ctx,
		userID,
		cursor,
		limit,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Posts by user id retrive successfully", postCursorPagination)
}

// FindWithID godoc
//
//	@Description	authentication and find post with id
//	@Tags			post
//	@Produce		json
//	@Param			postID	path		string	true	"uuid for post id"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=Post}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/find/{postID} [get]
func (h *postHandler) FindWithID(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("postID")
	post, err := h.postService.FindWithID(ctx, postID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Post retrive successfully", post)
}

// UpdatePost godoc
//
//	@Description	authentication update post and socket emit to clients
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		string				true	"uuid for post id"
//	@Param			payload	body		UpdatePostRequest	false	"update post payload"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=UpdatedPost}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/update/{postID} [patch]
func (h *postHandler) UpdatePost(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("postID")
	updatePostRequest := &UpdatePostRequest{}
	err := c.ShouldBind(updatePostRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	updatePostDTO := &UpdatePostDTO{
		PostID:                   postID,
		Message:                  updatePostRequest.Message,
		FilesURL:                 updatePostRequest.FilesURL,
		ShouldDeleteCurrentFiles: updatePostRequest.ShouldDeleteCurrentFiles,
		IsSharePost:              updatePostRequest.IsSharePost,
	}
	post, err := h.postService.UpdatePost(ctx, updatePostDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Update post successfully", post)
}

// DeletePost godoc
//
//	@Description	authentication delete post and socket emit notifications to client
//	@Tags			post
//	@Produce		json
//	@Param			postID	path		string	true	"uuid for post id"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=DeletedPost}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/delete/{postID} [delete]
func (h *postHandler) DeletePost(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("postID")
	deletedPost, err := h.postService.DeletePost(ctx, postID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Post delete successfully", deletedPost)
}

// ToggleLike godoc
//
//	@Description	authentication toggle like or unlike post and socket emit like or unlike post, notification to client
//	@Tags			post
//	@Produce		json
//	@Param			postID	path		string	true	"uuid for post id"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=Like}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/toggle-like/{postID} [post]
func (h *postHandler) ToggleLike(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowRelations)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	postID := c.Param("postID")
	message, like, err := h.postService.ToggleLike(
		ctx,
		user.ID,
		postID,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, message, like)
}
