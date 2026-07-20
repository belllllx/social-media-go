package post

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

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
//	@Param			files	formData	file	true	"Multiple image files"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=file.FilesURL}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/upload-files [post]
func (h *postHandler) UploadFiles(c *gin.Context) {
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

	filesData := []file.FileData{}
	for _, fileHeader := range filesHeader {
		fileOpen, err := fileHeader.Open()
		if err != nil {
			response.AbortWithInternalServerError(c, err)
			return
		}

		filesData = append(filesData, file.FileData{
			Filename:    fileHeader.Filename,
			ContentType: fileHeader.Header.Get("Content-Type"),
			Body:        fileOpen,
			Size:        fileHeader.Size,
		})
	}

	filesURL, err := h.fileService.UploadFiles(filesData, file.FileTypePost)
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
//	@Router			/post/delete/file [delete]
func (h *postHandler) DeleteFile(c *gin.Context) {
	deleteFileRequest := &file.DeleteFileRequest{}
	err := c.ShouldBind(deleteFileRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	deleteFileDTO := &file.DeleteFileDTO{
		FileURL:  deleteFileRequest.FileURL,
		FileType: file.FileTypePost,
	}
	err = h.fileService.DeleteFile(deleteFileDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Delete file successfully", nil)
}

// CreatePost godoc
//
//	@Description	authentication create post, notifications and socket broadcast post, notifications to client
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreatePostRequest	false	"create post payload"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=post.CreatedPost}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/create [post]
func (h *postHandler) CreatePost(c *gin.Context) {
	user, ok := c.MustGet("user").(*user.SecureUser)
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
	createdPost, err := h.postService.CreatePost(createPostDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Post create successfully", createdPost)
}

// CreateSharePost godoc
//
//	@Description	authentication create share post, notification and socket broadcast share post, notification to client
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			parentID	path		string					true	"uuid for parent post id"
//	@Param			payload		body		CreateSharePostRequest	false	"create share post payload"
//	@Success		201			{object}	response.SwaggerResponseWithData{data=post.CreatedSharePost}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/post/share/create/{parentID} [post]
func (h *postHandler) CreateSharePost(c *gin.Context) {
	user, ok := c.MustGet("user").(*user.SecureUser)
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
	createdSharePost, err := h.postService.CreateSharePost(createSharePostDTO)
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
//	@Success		200		{object}	response.SwaggerResponseWithData{data=post.PostCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/finds [get]
func (h *postHandler) FindsCursorPagination(c *gin.Context) {
	cursor := c.Query("cursor")
	limit := c.Query("limit")
	postCursorPagination, err := h.postService.FindsCursorPagination(cursor, limit)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Posts retrive successfully", postCursorPagination)
}
