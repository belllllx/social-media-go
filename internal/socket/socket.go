package socket

import (
	"github.com/belllllx/social-media-go/internal/middlewares"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func NewSocketServer() *server.Server {
	io := server.NewServer(nil, nil)
	io.Use(middlewares.SocketRequireAuth)

	// io.On("connection", func(args ...any) {
	// 	socket := args[0].(*server.Socket)
	// 	fmt.Printf("Attached data: %v\n", socket.Data())
	// 	fmt.Printf("connected: %s\n", socket.Id())

	// 	socket.On("disconnect", func(args ...any) {
	// 		fmt.Printf("disconnected: %s\n", socket.Id())
	// 	})
	// })

	return io
}
