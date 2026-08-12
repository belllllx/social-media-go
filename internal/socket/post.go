package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type PostLikeDTO struct {
	ID        int64            `json:"id"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user,omitempty"`
	PostID    uuid.UUID        `json:"postId"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type DeletePostDTO struct {
	ID uuid.UUID `json:"id"`
}

type UpdatePostDTO struct {
	ID       uuid.UUID `json:"id"`
	Message  *string   `json:"message"`
	FilesURL []string  `json:"filesUrl"`
}

type PostParentDTO struct {
	ID        uuid.UUID        `json:"id"`
	Message   *string          `json:"message"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user"`
	FilesURL  []string         `json:"filesUrl"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type CreatePostDTO struct {
	ID        uuid.UUID        `json:"id"`
	Message   *string          `json:"message"`
	UserID    uuid.UUID        `json:"userId"`
	User      *user.SecureUser `json:"user"`
	ParentID  *uuid.UUID       `json:"parentId"`
	Parent    *PostParentDTO   `json:"parent,omitempty"`
	FilesURL  []string         `json:"filesUrl,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type PostSocket interface {
	EmitCreate(createPostDTO *CreatePostDTO)
	EmitUpdate(updatePostDTO *UpdatePostDTO)
	EmitDelete(deletePostDTO *DeletePostDTO)
	EmitLikeOrUnlike(postLikeDTO *PostLikeDTO)
}

type postSocket struct {
	socket *server.Server
}

func NewPostSocket(socket *server.Server) PostSocket {
	return &postSocket{socket: socket}
}

func (s *postSocket) EmitCreate(createPostDTO *CreatePostDTO) {
	s.socket.Emit("newPost", createPostDTO)
}

func (s *postSocket) EmitUpdate(updatePostDTO *UpdatePostDTO) {
	s.socket.Emit("updatePost", updatePostDTO)
}

func (s *postSocket) EmitDelete(deletePostDTO *DeletePostDTO) {
	s.socket.Emit("deletePost", deletePostDTO)
}

func (s *postSocket) EmitLikeOrUnlike(postLikeDTO *PostLikeDTO) {
	s.socket.Emit("newLikePost", postLikeDTO)
}
