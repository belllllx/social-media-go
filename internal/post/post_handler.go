package post

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type CreatePostRequest struct {
	Message  string   `json:"message"`
	FilesURL []string `json:"filesUrl"`
}

type PostHandler interface {
	UploadFiles(c *gin.Context)
	CreatePost(c *gin.Context)
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

// CreatePost godoc
//
//	@Description	authentication create post, notifications and socket broadcast post, notification to client
//	@Tags			post
//	@Accept			json
//	@Produce		json
//	@Param			userID	path		string				true	"uuid for user id"
//	@Param			payload	body		CreatePostRequest	false	"create post payload"
//	@Success		201		{object}	response.SwaggerResponseWithData{data=post.CreatedPost}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/post/create/{userID} [post]
func (h *postHandler) CreatePost(c *gin.Context) {
	createPostRequest := &CreatePostRequest{}
	err := c.ShouldBind(createPostRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	userID := c.Param("userID")
	createPostDTO := &CreatePostDTO{
		Message:  createPostRequest.Message,
		FilesURL: createPostRequest.FilesURL,
		UserID:   userID,
	}
	createdPost, err := h.postService.CreatePost(createPostDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Created(c, "Post create successfully", createdPost)
}
