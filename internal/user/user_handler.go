package user

import (
	"github.com/belllllx/social-media-go/internal/dto"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type UpdatesUserInfoRequest struct {
	Fullname    *string `json:"fullname"`
	DateOfBirth *string `json:"dateOfBirth" binding:"omitempty,dateofbirth"`
	Info        *string `json:"info"`
}

type ClearUserImagesRequest struct {
	FileURL string `json:"fileUrl" binding:"presignedurl"`
}

type UserHandler interface {
	FindsWithFullnameCursorPagination(c *gin.Context)
	FindsCursorPaginationWithFollowerRelation(c *gin.Context)
	FindByIDWithFollowRelations(c *gin.Context)
	ToggleFollow(c *gin.Context)
	UploadEditUserAvatar(c *gin.Context)
	UploadEditUserBackground(c *gin.Context)
	ClearUserAvatar(c *gin.Context)
	ClearUserBackground(c *gin.Context)
	UpdatesInfo(c *gin.Context)
}

type userHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) UserHandler {
	return &userHandler{userService: userService}
}

// FindsWithFullnameCursorPagination godoc
//
//	@Description	authentication and find users with fullname cursor pagination
//	@Tags			user
//	@Produce		json
//	@Param			fullname	query		string	true	"search user with fullname"
//	@Param			cursor		query		string	false	"cursor uuid for user id"
//	@Param			limit		query		int		true	"limit for users cursor pagination"
//	@Success		200			{object}	response.SwaggerResponseWithData{data=UserCursorPagination}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/user/finds-with-fullname [get]
func (h *userHandler) FindsWithFullnameCursorPagination(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	fullname := c.Query("fullname")
	cursor := c.Query("cursor")
	limit := c.Query("limit")
	findsWithFullnameCursorPaginationDTO := &FindsWithFullnameCursorPaginationDTO{
		UserID:   user.ID,
		Fullname: fullname,
		Cursor:   cursor,
		Limit:    limit,
	}
	userCursorPagination, err := h.userService.FindsWithFullnameCursorPagination(ctx, findsWithFullnameCursorPaginationDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Users retrive successfully", userCursorPagination)
}

// FindsCursorPaginationWithFollowerRelation godoc
//
//	@Description	authentication and find users follower relation cursor pagination
//	@Tags			user
//	@Produce		json
//	@Param			cursor	query		string	false	"cursor uuid for user id"
//	@Param			limit	query		int		true	"limit for users follower relation cursor pagination"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=UserWithFollowerRelationCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/finds [get]
func (h *userHandler) FindsCursorPaginationWithFollowerRelation(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	cursor := c.Query("cursor")
	limit := c.Query("limit")
	userWithFollowerRelationCursorPagination, err := h.userService.FindsCursorPaginationWithFollowerRelation(
		ctx,
		user.ID,
		cursor,
		limit,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Users retrive successfully", userWithFollowerRelationCursorPagination)
}

// FindByIDWithFollowRelations godoc
//
//	@Description	authentication and find user by id with follow relations
//	@Tags			user
//	@Produce		json
//	@Param			userID	path		string	true	"uuid for user id"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=SecureUserWithFollowRelations}
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/find/{userID} [get]
func (h *userHandler) FindByIDWithFollowRelations(c *gin.Context) {
	ctx := c.Request.Context()

	userID := c.Param("userID")
	secureUserWithFollowRelations, err := h.userService.FindByIDWithFollowRelations(ctx, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "User retrive successfully", secureUserWithFollowRelations)
}

// ToggleFollow godoc
//
//	@Description	authentication follow or unfollow and emit follow or unfollow, notification to client
//	@Tags			user
//	@Produce		json
//	@Param			followingID	path		string	true	"uuid for user id following"
//	@Success		200			{object}	response.SwaggerResponseWithData{data=Follow}
//	@Failure		400			{object}	response.SwaggerBadRequestResponse
//	@Failure		401			{object}	response.SwaggerResponse
//	@Failure		404			{object}	response.SwaggerResponse
//	@Failure		500			{object}	response.SwaggerResponse
//	@Router			/user/toggle-follow/{followingID} [post]
func (h *userHandler) ToggleFollow(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	followingID := c.Param("followingID")
	message, follow, err := h.userService.ToggleFollow(
		ctx,
		user.ID,
		followingID,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, message, follow)
}

// UploadEditUserAvatar godoc
//
//	@Description	authentication upload file and edit user avatar image
//	@Tags			user
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"single image file"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=FileURL}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/avatar/upload-file [patch]
func (h *userHandler) UploadEditUserAvatar(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

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

	fileDataDTO := &FileDataDTO{
		Filename:    form.Filename,
		ContentType: form.Header.Get("Content-Type"),
		Body:        fileOpen,
		Size:        form.Size,
	}
	fileURL, err := h.userService.UploadEditUserFile(
		ctx,
		user,
		fileDataDTO,
		EditFileTypeAvatar,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "User avatar upload successfully", fileURL)
}

// UploadEditUserBackground godoc
//
//	@Description	authentication upload file and edit user background image
//	@Tags			user
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"single image file"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=FileURL}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/background/upload-file [patch]
func (h *userHandler) UploadEditUserBackground(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

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

	fileDataDTO := &FileDataDTO{
		Filename:    form.Filename,
		ContentType: form.Header.Get("Content-Type"),
		Body:        fileOpen,
		Size:        form.Size,
	}
	fileURL, err := h.userService.UploadEditUserFile(
		ctx,
		user,
		fileDataDTO,
		EditFileTypeBackground,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "User background upload successfully", fileURL)
}

// ClearUserAvatar godoc
//
//	@Description	authentication and clear user avatar image
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ClearUserImagesRequest	true	"clear user avatar payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/avatar/delete-file [patch]
func (h *userHandler) ClearUserAvatar(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	clearUserImagesRequest := &ClearUserImagesRequest{}
	err := c.ShouldBind(clearUserImagesRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	err = h.userService.ClearUserImages(
		ctx,
		user.ID,
		clearUserImagesRequest.FileURL,
		EditFileTypeAvatar,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Clear user avatar successfully", nil)
}

// ClearUserBackground godoc
//
//	@Description	authentication and clear user background image
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ClearUserImagesRequest	true	"clear user background payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/background/delete-file [patch]
func (h *userHandler) ClearUserBackground(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	clearUserImagesRequest := &ClearUserImagesRequest{}
	err := c.ShouldBind(clearUserImagesRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	err = h.userService.ClearUserImages(
		ctx,
		user.ID,
		clearUserImagesRequest.FileURL,
		EditFileTypeBackground,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Clear user background successfully", nil)
}

// UpdatesInfo godoc
//
//	@Description	authentication and updates user info
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UpdatesUserInfoRequest	true	"updates user info payload"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=UpdatedUserInfo}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/user/edit-info [put]
func (h *userHandler) UpdatesInfo(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*dto.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	updatesUserInfoRequest := &UpdatesUserInfoRequest{}
	err := c.ShouldBind(updatesUserInfoRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	updatesInfoDTO := &UpdatesInfoDTO{
		UserID:      user.ID,
		Fullname:    updatesUserInfoRequest.Fullname,
		DateOfBirth: updatesUserInfoRequest.DateOfBirth,
		Info:        updatesUserInfoRequest.Info,
	}
	updatedUserInfo, err := h.userService.UpdatesInfo(ctx, updatesInfoDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Updates user info successfully", updatedUserInfo)
}
