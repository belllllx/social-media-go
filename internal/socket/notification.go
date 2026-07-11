package socket

import (
	"fmt"
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
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
	socket *socketio.Server
}

func NewNotificationSocket(socket *socketio.Server) NotificationSocket {
	return &notificationSocket{socket: socket}
}

func (s *notificationSocket) BroadcastNotifications(broadcastNotificationsDTO []BroadcastNotificationsDTO) {
	for _, broadcastNotificationDTO := range broadcastNotificationsDTO {
		event := fmt.Sprintf("notification:%v", broadcastNotificationDTO.ReceiverID)
		s.socket.BroadcastToNamespace("/", event, broadcastNotificationDTO)
	}
}
