package notification

import (
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type NotificationHandler interface {
	FindsWithReceiverIDCursorPagination(c *gin.Context)
}

type notificationHandler struct {
	notificationService NotificationService
}

func NewNotificationHandler(notificationService NotificationService) NotificationHandler {
	return &notificationHandler{notificationService: notificationService}
}

// FindsWithReceiverIDCursorPagination godoc
//
//	@Description	authentication and find notifications with receiver id cursor pagination
//	@Tags			notification
//	@Produce		json
//	@Param			cursor	query		string	false	"cursor uuid for notification id"
//	@Param			limit	query		int		true	"limit for notifications cursor pagination"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=NotificationCursorPagination}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/notification/finds [get]
func (h *notificationHandler) FindsWithReceiverIDCursorPagination(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowRelations)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	cursor := c.Query("cursor")
	limit := c.Query("limit")
	notificationCursorPagination, err := h.notificationService.FindsWithReceiverIDCursorPagination(
		ctx,
		user.ID,
		cursor,
		limit,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Notifications retrive successfully", notificationCursorPagination)
}
