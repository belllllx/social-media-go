package socket

import (
	"fmt"

	"github.com/google/uuid"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func NewSocketServer(
	authMiddleware func(*server.Socket, func(*server.ExtendedError)),
) *server.Server {
	io := server.NewServer(nil, nil)
	io.Use(authMiddleware)

	io.On("connection", func(args ...any) {
		socket := args[0].(*server.Socket)

		userID, ok := socket.Data().(uuid.UUID)
		if !ok {
			socket.Disconnect(true)
			return
		}

		room := fmt.Sprintf("user:%v", userID)
		socket.Join(server.Room(room))

		socket.On("disconnect", func(args ...any) {
			socket.Leave(server.Room(room))
		})
	})

	return io
}
