package notification

import (
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateNotificationRequest struct {
	NotificationsID uuid.UUIDs `json:"notificationsId" binding:"required"`
}

type NotificationHandler interface {
	FindsWithReceiverIDCursorPagination(c *gin.Context)
	UpdateIsRead(c *gin.Context)
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

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
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

// UpdateIsRead godoc
//
//	@Description	authentication and update read notifications
//	@Tags			notification
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UpdateNotificationRequest	true	"update notification payload"
//	@Success		200		{object}	response.SwaggerResponseWithData{data=[]UpdatedNotification}
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/notification/read-all [patch]
func (h *notificationHandler) UpdateIsRead(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	updateNotificationRequest := &UpdateNotificationRequest{}
	err := c.ShouldBind(updateNotificationRequest)
	if err != nil {
		errFields := map[string]string{
			"notificationsId": "Invalid uuid",
		}
		response.AbortWithBadRequestErrorFields(c, errFields)
		return
	}

	updatedNotifications, err := h.notificationService.UpdateIsRead(
		ctx,
		user.ID,
		updateNotificationRequest.NotificationsID,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Update notifications successfully", updatedNotifications)
}
