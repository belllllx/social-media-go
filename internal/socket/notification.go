package socket

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/dto"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type DeleteNotification struct {
	ID uuid.UUID `json:"id"`
}

type EmitDeleteNotificationDTO struct {
	ID         uuid.UUID `json:"id"`
	ReceiverID uuid.UUID `json:"receiverId"`
}

type EmitNotificationDTO struct {
	ID         uuid.UUID               `json:"id"`
	Type       models.NotificationType `json:"type"`
	Message    string                  `json:"message"`
	IsRead     bool                    `json:"isRead"`
	SenderID   uuid.UUID               `json:"senderId"`
	Sender     *dto.SecureUser         `json:"sender"`
	ReceiverID uuid.UUID               `json:"receiverId"`
	PostID     *uuid.UUID              `json:"postId"`
	CommentID  *uuid.UUID              `json:"commentId,omitempty"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

type NotificationSocket interface {
	EmitNotification(emitNotificationDTO *EmitNotificationDTO)
	EmitNotifications(emitNotificationsDTO []EmitNotificationDTO)
	EmitDeleteNotification(emitDeleteNotificationDTO *EmitDeleteNotificationDTO)
	EmitDeleteNotifications(emitDeleteNotificationsDTO []EmitDeleteNotificationDTO)
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

func (s *notificationSocket) EmitDeleteNotification(emitDeleteNotificationDTO *EmitDeleteNotificationDTO) {
	event := fmt.Sprintf("notification:%v", emitDeleteNotificationDTO.ReceiverID)
	s.socket.Emit(event, DeleteNotification{ID: emitDeleteNotificationDTO.ID})
}

func (s *notificationSocket) EmitDeleteNotifications(emitDeleteNotificationsDTO []EmitDeleteNotificationDTO) {
	for _, emitDeleteNotificationDTO := range emitDeleteNotificationsDTO {
		event := fmt.Sprintf("notification:%v", emitDeleteNotificationDTO.ReceiverID)
		s.socket.Emit(event, DeleteNotification{ID: emitDeleteNotificationDTO.ID})
	}
}
