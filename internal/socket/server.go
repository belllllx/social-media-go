package socket

import (
	userSocket "github.com/belllllx/social-media-go/internal/socket/user"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func NewSocketServer(
	authMiddleware func(*server.Socket, func(*server.ExtendedError)),
	userSocketService userSocket.UserSocketService,
) *server.Server {
	io := server.NewServer(nil, nil)
	io.Use(authMiddleware)

	io.On("connection", func(args ...any) {
		socket := args[0].(*server.Socket)

		setupConnection(
			io,
			socket,
			userSocketService,
		)
	})

	return io
}
