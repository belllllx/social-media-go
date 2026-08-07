package socket

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type EmitNotificationDTO struct {
	ID         uuid.UUID               `json:"id"`
	Type       models.NotificationType `json:"type,omitempty"`
	Message    string                  `json:"message,omitempty"`
	IsRead     bool                    `json:"isRead,omitempty"`
	SenderID   uuid.UUID               `json:"senderId,omitempty"`
	Sender     *user.SecureUser        `json:"sender,omitempty"`
	ReceiverID uuid.UUID               `json:"receiverId,omitempty"`
	PostID     *uuid.UUID              `json:"postId,omitempty"`
	CommentID  *uuid.UUID              `json:"commentId,omitempty"`
	CreatedAt  time.Time               `json:"createdAt,omitempty"`
	UpdatedAt  time.Time               `json:"updatedAt,omitempty"`
}

type NotificationSocket interface {
	EmitNotification(emitNotificationDTO *EmitNotificationDTO)
	EmitNotifications(emitNotificationsDTO []EmitNotificationDTO)
}

type notificationSocket struct {
	socket *server.Server
}

func NewNotificationSocket(socket *server.Server) NotificationSocket {
	return &notificationSocket{socket: socket}
}

func (s *notificationSocket) EmitNotification(emitNotificationDTO *EmitNotificationDTO) {
	event := fmt.Sprintf("notification:%v", emitNotificationDTO.ReceiverID)
	s.socket.Emit(event, emitNotificationDTO)
}

func (s *notificationSocket) EmitNotifications(emitNotificationsDTO []EmitNotificationDTO) {
	for _, emitNotificationDTO := range emitNotificationsDTO {
		event := fmt.Sprintf("notification:%v", emitNotificationDTO.ReceiverID)
		s.socket.Emit(event, emitNotificationDTO)
	}
}
