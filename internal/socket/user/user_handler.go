package user

import (
	"context"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func RegisterEvents(
	ctx context.Context,
	io *server.Server,
	socket *server.Socket,
	userSocketService UserSocketService,
) {
	socket.On("connected", func(args ...any) {
		userID, err := helpers.ParseUserID(args)
		if err != nil {
			logs.Warn(err)

			err = socket.Emit("error", err.Error())
			if err != nil {
				logs.Error(err)
			}
			return
		}

		activeUsers, err := userSocketService.Connected(
			ctx,
			string(socket.Id()),
			userID,
		)
		if err != nil {
			if helpers.IsErrContextCanceled(err) {
				return
			}

			err = socket.Emit("error", err.Error())
			if err != nil {
				logs.Error(err)
			}
			return
		}

		io.Emit("activeUsers", activeUsers)
	})
}
