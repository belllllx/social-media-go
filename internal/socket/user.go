package socket

import (
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

type FollowerDataDTO struct {
	ID          int64     `json:"id"`
	FollowerID  uuid.UUID `json:"followerId"`
	FollowingID uuid.UUID `json:"followingId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SecureUserFollowDTO struct {
	ID                   uuid.UUID           `json:"id"`
	Fullname             string              `json:"fullname"`
	Username             *string             `json:"username"`
	Email                string              `json:"email"`
	DateOfBirth          *time.Time          `json:"dateOfBirth"`
	ProfileUrl           *string             `json:"profileUrl"`
	ProfileBackgroundUrl *string             `json:"profileBackgroundUrl"`
	Info                 *string             `json:"info"`
	Role                 models.Role         `json:"role"`
	ProviderType         models.ProviderType `json:"providerType"`
	Followers            []FollowerDataDTO   `json:"followers"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type FollowDTO struct {
	ID          int64                `json:"id"`
	FollowerID  uuid.UUID            `json:"followerId"`
	Follower    *SecureUserFollowDTO `json:"follower,omitempty"`
	FollowingID uuid.UUID            `json:"followingId"`
	Following   *SecureUserFollowDTO `json:"following,omitempty"`
	CreatedAt   time.Time            `json:"createdAt,omitzero"`
	UpdatedAt   time.Time            `json:"updatedAt,omitzero"`
}

type UserSocket interface {
	EmitToggleFollow(followDTO *FollowDTO)
}

type userSocket struct {
	io *server.Server
}

func NewUserSocket(io *server.Server) UserSocket {
	return &userSocket{io: io}
}

func (s *userSocket) EmitToggleFollow(followDTO *FollowDTO) {
	s.io.Emit("toggleFollow", followDTO)
}
