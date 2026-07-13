package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type LikeDTO struct {
	ID        int64      `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	PostID    *uuid.UUID `json:"postId"`
	CommentID *uuid.UUID `json:"commentId"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CommentDTO struct {
	ID            uuid.UUID  `json:"id"`
	Message       *string    `json:"message"`
	UserID        uuid.UUID  `json:"userId"`
	PostID        uuid.UUID  `json:"postId"`
	ParentID      *uuid.UUID `json:"parentId"`
	ReplyID       *uuid.UUID `json:"replyId"`
	ReplyToUserID *uuid.UUID `json:"replyToUserId"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type BroadcastPostDTO struct {
	ID            uuid.UUID       `json:"id"`
	Message       *string         `json:"message"`
	UserID        uuid.UUID       `json:"userId"`
	User          user.SecureUser `json:"user"`
	ParentID      *uuid.UUID      `json:"parentId"`
	Likes         []LikeDTO       `json:"likes"`
	Comments      []CommentDTO    `json:"comments"`
	CommentsCount int             `json:"commentsCount"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type BroadcastPostWithFilesDTO struct {
	ID            uuid.UUID       `json:"id"`
	Message       *string         `json:"message"`
	UserID        uuid.UUID       `json:"userId"`
	User          user.SecureUser `json:"user"`
	ParentID      *uuid.UUID      `json:"parentId"`
	Likes         []LikeDTO       `json:"likes"`
	Comments      []CommentDTO    `json:"comments"`
	FilesURL      []string        `json:"filesUrl"`
	CommentsCount int             `json:"commentsCount"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type PostSocket interface {
	BroadcastPost(broadcastPostDTO *BroadcastPostDTO)
	BroadcastPostWithFiles(broadcastPostWithFilesDTO *BroadcastPostWithFilesDTO)
}

type postSocket struct {
	socket *server.Server
}

func NewPostSocket(socket *server.Server) PostSocket {
	return &postSocket{socket: socket}
}

func (s *postSocket) BroadcastPost(broadcastPostDTO *BroadcastPostDTO) {
	s.socket.Emit("newPost", broadcastPostDTO)
}

func (s *postSocket) BroadcastPostWithFiles(broadcastPostWithFilesDTO *BroadcastPostWithFilesDTO) {
	s.socket.Emit("newPost", broadcastPostWithFilesDTO)
}
