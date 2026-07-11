package socket

import socketio "github.com/googollee/go-socket.io"

func NewSocketServer() *socketio.Server {
	return socketio.NewServer(nil)
}
