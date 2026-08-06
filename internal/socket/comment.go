package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type CommentLikeDTO struct {
	ID        int64            `json:"id"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user,omitempty"`
	CommentID uuid.UUID        `json:"commentId"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type CommentDTO struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	PostID        uuid.UUID        `json:"postId"`
	UserID        uuid.UUID        `json:"userId"`
	ParentID      *uuid.UUID       `json:"parentId,omitempty"`
	ReplyID       *uuid.UUID       `json:"replyId,omitempty"`
	ReplyToUserID *uuid.UUID       `json:"replyToUserId,omitempty"`
	ReplyToUser   *user.SecureUser `json:"replyToUser,omitempty"`
	User          *user.SecureUser `json:"user"`
	FileURL       string           `json:"fileUrl,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type CommentSocket interface {
	EmitCreate(commentDTO *CommentDTO)
}

type commentSocket struct {
	socket *server.Server
}

func NewCommentSocket(socket *server.Server) CommentSocket {
	return &commentSocket{socket: socket}
}

func (s *commentSocket) EmitCreate(commentDTO *CommentDTO) {
	s.socket.Emit("newComment", commentDTO)
}
