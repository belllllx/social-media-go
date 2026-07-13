package socket

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type BroadcastNotificationsDTO struct {
	ID         uuid.UUID             `json:"id"`
	Type       user.NotificationType `json:"type"`
	Message    string                `json:"message"`
	IsRead     bool                  `json:"isRead"`
	SenderID   uuid.UUID             `json:"senderId"`
	ReceiverID uuid.UUID             `json:"receiverId"`
	PostID     *uuid.UUID            `json:"postId"`
	CommentID  *uuid.UUID            `json:"commentId"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

type NotificationSocket interface {
	BroadcastNotifications(broadcastNotificationsDTO []BroadcastNotificationsDTO)
}

type notificationSocket struct {
	socket *server.Server
}

func NewNotificationSocket(socket *server.Server) NotificationSocket {
	return &notificationSocket{socket: socket}
}

func (s *notificationSocket) BroadcastNotifications(broadcastNotificationsDTO []BroadcastNotificationsDTO) {
	for _, broadcastNotificationDTO := range broadcastNotificationsDTO {
		event := fmt.Sprintf("notification:%v", broadcastNotificationDTO.ReceiverID)
		s.socket.Emit(event, broadcastNotificationDTO)
	}
}
