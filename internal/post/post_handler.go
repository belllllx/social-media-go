package post

import (
	"github.com/belllllx/social-media-go/internal/file"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type PostHandler interface {
	UploadFiles(c *gin.Context)
}

type postHandler struct {
	fileService file.FileService
}

func NewPostHandler(fileService file.FileService) PostHandler {
	return &postHandler{fileService: fileService}
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
