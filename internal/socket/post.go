package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type PostParentDTO struct {
	ID        uuid.UUID        `json:"id"`
	Message   *string          `json:"message"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user"`
	ParentID  *uuid.UUID       `json:"parentId"`
	FilesURL  []string         `json:"filesUrl"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type LikeDTO struct {
	ID        int64            `json:"id"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user"`
	PostID    *uuid.UUID       `json:"postId"`
	CommentID *uuid.UUID       `json:"commentId"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type PostDTO struct {
	ID            uuid.UUID        `json:"id"`
	Message       *string          `json:"message"`
	UserID        uuid.UUID        `json:"userId"`
	User          *user.SecureUser `json:"user"`
	ParentID      *uuid.UUID       `json:"parentId"`
	Parent        *PostParentDTO   `json:"parent,omitempty"`
	Likes         []LikeDTO        `json:"likes,omitempty"`
	FilesURL      []string         `json:"filesUrl,omitempty"`
	CommentsCount int              `json:"commentsCount,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type PostSocket interface {
	EmitCreate(postDTO *PostDTO)
	EmitUpdate(postDTO *PostDTO)
}

type postSocket struct {
	socket *server.Server
}

func NewPostSocket(socket *server.Server) PostSocket {
	return &postSocket{socket: socket}
}

func (s *postSocket) EmitCreate(postDTO *PostDTO) {
	s.socket.Emit("newPost", postDTO)
}

func (s *postSocket) EmitUpdate(postDTO *PostDTO) {
	s.socket.Emit("updatePost", postDTO)
}
