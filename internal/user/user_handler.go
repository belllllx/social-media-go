package user

import (
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	FindsWithFullnameCursorPagination(c *gin.Context)
	FindsCursorPaginationWithFollowerRelation(c *gin.Context)
	FindByIDWithFollowRelations(c *gin.Context)
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

	user, ok := c.MustGet("user").(*SecureUserWithFollowingRelation)
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

	user, ok := c.MustGet("user").(*SecureUserWithFollowingRelation)
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
