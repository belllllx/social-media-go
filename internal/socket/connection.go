package socket

import (
	"context"
	"fmt"

	userSocket "github.com/belllllx/social-media-go/internal/socket/user"
	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func setupConnection(
	io *server.Server,
	socket *server.Socket,
	userSocketService userSocket.UserSocketService,
) {
	ctx, cancel := context.WithCancel(context.Background())

	userID, ok := socket.Data().(uuid.UUID)
	if !ok {
		cancel()
		socket.Disconnect(true)
		return
	}

	room := fmt.Sprintf("user:%v", userID)
	socket.Join(server.Room(room))

	userSocket.RegisterEvents(
		ctx,
		io,
		socket,
		userSocketService,
	)

	socket.On("disconnect", func(args ...any) {
		socket.Leave(server.Room(room))
		cancel()
	})
}
