package comment

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type CommentHandler interface {
	UploadFile(c *gin.Context)
	DeleteFile(c *gin.Context)
}

type commentHandler struct {
	fileService file.FileService
}

func NewCommentHandler(fileService file.FileService) CommentHandler {
	return &commentHandler{
		fileService: fileService,
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
//	@Router			/comment/delete/file [delete]
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
