package socket

import (
	"net/http"

	"github.com/belllllx/social-media-go/internal/auth"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/spf13/viper"
	server "github.com/zishang520/socket.io/servers/socket/v3"
)

func NewSocketServer() *server.Server {
	io := server.NewServer(nil, nil)

	io.Use(func(socket *server.Socket, next func(*server.ExtendedError)) {
		headers := socket.Handshake().Headers
		req := &http.Request{
			Header: headers.Header(),
		}
		accessToken, err := req.Cookie("access_token")
		if err != nil {
			next(&server.ExtendedError{
				Message: "Unauthorized",
			})
			return
		}

		token, err := helpers.VerifyJWT(accessToken.Value, &auth.UserAccessTokenClaims{}, viper.GetString("app.access_token_secret"))
		if err != nil {
			next(&server.ExtendedError{
				Message: "Unauthorized",
			})
			return
		}

		claims, ok := token.Claims.(*auth.UserAccessTokenClaims)
		if !ok {
			next(&server.ExtendedError{
				Message: "Unauthorized",
			})
			return
		}

		socket.SetData(claims.ID)
		next(nil)
	})

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
