package notification

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
	EmitNotification(emitNotificationDTO *EmitNotificationDTO) error
	EmitNotifications(emitNotificationsDTO []EmitNotificationDTO) error
	EmitDeleteNotification(emitDeleteNotificationDTO *EmitDeleteNotificationDTO) error
	EmitDeleteNotifications(emitDeleteNotificationsDTO []EmitDeleteNotificationDTO) error
}

type notificationSocket struct {
	io *server.Server
}

func NewNotificationSocket(io *server.Server) NotificationSocket {
	return &notificationSocket{io: io}
}

func (s *notificationSocket) EmitNotification(emitNotificationDTO *EmitNotificationDTO) error {
	room := fmt.Sprintf("user:%v", emitNotificationDTO.ReceiverID)
	return s.io.To(server.Room(room)).Emit("notification", emitNotificationDTO)
}

func (s *notificationSocket) EmitNotifications(emitNotificationsDTO []EmitNotificationDTO) error {
	for _, emitNotificationDTO := range emitNotificationsDTO {
		room := fmt.Sprintf("user:%v", emitNotificationDTO.ReceiverID)
		err := s.io.To(server.Room(room)).Emit("notification", emitNotificationDTO)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationSocket) EmitDeleteNotification(emitDeleteNotificationDTO *EmitDeleteNotificationDTO) error {
	room := fmt.Sprintf("user:%v", emitDeleteNotificationDTO.ReceiverID)
	return s.io.To(server.Room(room)).Emit("notification", DeleteNotification{ID: emitDeleteNotificationDTO.ID})
}

func (s *notificationSocket) EmitDeleteNotifications(emitDeleteNotificationsDTO []EmitDeleteNotificationDTO) error {
	for _, emitDeleteNotificationDTO := range emitDeleteNotificationsDTO {
		room := fmt.Sprintf("user:%v", emitDeleteNotificationDTO.ReceiverID)
		err := s.io.To(server.Room(room)).Emit("notification", DeleteNotification{ID: emitDeleteNotificationDTO.ID})
		if err != nil {
			return err
		}
	}
	return nil
}
