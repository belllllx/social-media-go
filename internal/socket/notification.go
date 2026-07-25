package socket

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type BroadcastNotificationDTO struct {
	ID         uuid.UUID               `json:"id"`
	Type       models.NotificationType `json:"type"`
	Message    string                  `json:"message"`
	IsRead     bool                    `json:"isRead"`
	SenderID   uuid.UUID               `json:"senderId"`
	Sender     *user.SecureUser        `json:"sender"`
	ReceiverID uuid.UUID               `json:"receiverId"`
	PostID     *uuid.UUID              `json:"postId"`
	CommentID  *uuid.UUID              `json:"commentId"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

type NotificationSocket interface {
	BroadcastNotification(broadcastNotificationDTO *BroadcastNotificationDTO)
	BroadcastNotifications(broadcastNotificationsDTO []BroadcastNotificationDTO)
}

type notificationSocket struct {
	socket *server.Server
}

func NewNotificationSocket(socket *server.Server) NotificationSocket {
	return &notificationSocket{socket: socket}
}

func (s *notificationSocket) BroadcastNotification(broadcastNotificationDTO *BroadcastNotificationDTO) {
	event := fmt.Sprintf("notification:%v", broadcastNotificationDTO.ReceiverID)
	s.socket.Emit(event, broadcastNotificationDTO)
}

func (s *notificationSocket) BroadcastNotifications(broadcastNotificationsDTO []BroadcastNotificationDTO) {
	for _, broadcastNotificationDTO := range broadcastNotificationsDTO {
		event := fmt.Sprintf("notification:%v", broadcastNotificationDTO.ReceiverID)
		s.socket.Emit(event, broadcastNotificationDTO)
	}
}
