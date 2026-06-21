package auth

import (
	"time"

	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	Register(c *gin.Context)
}

type authHandler struct {
	authService AuthService
}

func NewAuthHandler(authService AuthService) AuthHandler {
	return &authHandler{authService: authService}
}

// Register godoc
// @Description  create user pending and send email to verify otp
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param payload body RegisterRequest true "register payload"
// @Success      200  {object}  response.RegisterResponse
// @Failure 400 {object} response.RegisterResponse
// @Failure 500 {object} response.RegisterResponse
// @Router       /auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	registerRequest := &RegisterRequest{}
	err := c.ShouldBind(registerRequest)
	if err != nil {
		c.Error(err)
		return
	}

	result, token, err := h.authService.Register(registerRequest)
	if err != nil {
		helpers.HandleError(c, err, result)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "register_token",
		Value:  token,
		MaxAge: time.Minute * 5,
	})

	response.Ok(c, result, nil)
}
