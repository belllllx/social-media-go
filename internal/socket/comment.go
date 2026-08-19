package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/dto"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type CommentLikeDTO struct {
	ID        int64           `json:"id"`
	UserID    uuid.UUID       `json:"userId"`
	User      *dto.SecureUser `json:"user,omitempty"`
	CommentID uuid.UUID       `json:"commentId"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type PostDTO struct {
	UserID uuid.UUID `json:"userId"`
}

type DeleteCommentDTO struct {
	ID       uuid.UUID  `json:"id"`
	PostID   uuid.UUID  `json:"postId"`
	Post     *PostDTO   `json:"post"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
}

type UpdateCommentDTO struct {
	ID      uuid.UUID `json:"id"`
	Message *string   `json:"message"`
	PostID  uuid.UUID `json:"postId"`
	FileURL string    `json:"fileUrl"`
}

type CreateCommentDTO struct {
	ID            uuid.UUID       `json:"id"`
	Message       *string         `json:"message"`
	PostID        uuid.UUID       `json:"postId"`
	Post          *PostDTO        `json:"post"`
	UserID        uuid.UUID       `json:"userId"`
	ParentID      *uuid.UUID      `json:"parentId,omitempty"`
	ReplyID       *uuid.UUID      `json:"replyId,omitempty"`
	ReplyToUserID *uuid.UUID      `json:"replyToUserId,omitempty"`
	ReplyToUser   *dto.SecureUser `json:"replyToUser,omitempty"`
	User          *dto.SecureUser `json:"user"`
	FileURL       string          `json:"fileUrl,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type CommentSocket interface {
	EmitCreate(createCommentDTO *CreateCommentDTO)
	EmitUpdate(updateCommentDTO *UpdateCommentDTO)
	EmitDelete(deleteCommentDTO *DeleteCommentDTO)
	EmitToggleLike(commentLikeDTO *CommentLikeDTO)
}

type commentSocket struct {
	socket *server.Server
}

func NewCommentSocket(socket *server.Server) CommentSocket {
	return &commentSocket{socket: socket}
}

func (s *commentSocket) EmitCreate(createCommentDTO *CreateCommentDTO) {
	s.socket.Emit("newComment", createCommentDTO)
}

func (s *commentSocket) EmitUpdate(updateCommentDTO *UpdateCommentDTO) {
	s.socket.Emit("updateComment", updateCommentDTO)
}

func (s *commentSocket) EmitDelete(deleteCommentDTO *DeleteCommentDTO) {
	s.socket.Emit("deleteComment", deleteCommentDTO)
}

func (s *commentSocket) EmitToggleLike(commentLikeDTO *CommentLikeDTO) {
	s.socket.Emit("toggleLikeComment", commentLikeDTO)
}
