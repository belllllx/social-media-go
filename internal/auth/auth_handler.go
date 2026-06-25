package auth

import (
	"time"

	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type SendEmailRegisterRequest struct {
	Fullname string `json:"fullname" binding:"required,max=30"`
	Username string `json:"username" binding:"required,max=15"`
	Email    string `json:"email" binding:"required,email,max=30"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type VerifyOTPRegisterRequest struct {
	OTP string `json:"otp" binding:"required,len=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthHandler interface {
	SendEmailRegister(c *gin.Context)
	ResendEmailRegister(c *gin.Context)
	VerifyOTPRegister(c *gin.Context)
	Login(c *gin.Context)
}

type authHandler struct {
	authService  AuthService
	emailService email.EmailService
}

func NewAuthHandler(authService AuthService, emailService email.EmailService) AuthHandler {
	return &authHandler{
		authService:  authService,
		emailService: emailService,
	}
}

// SendEmailRegister godoc
//
//	@Description	create user store send email to verify otp and set cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		SendEmailRegisterRequest	true	"send email register payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/register/send-email [post]
func (h *authHandler) SendEmailRegister(c *gin.Context) {
	sendEmailRegisterRequest := &SendEmailRegisterRequest{}
	err := c.ShouldBind(sendEmailRegisterRequest)
	if err != nil {
		c.Error(err)
		return
	}

	result, token, err := h.authService.SendEmailRegister(sendEmailRegisterRequest)
	if err != nil {
		helpers.HandleError(c, err, result)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "register_token",
		Value:  token,
		MaxAge: time.Minute * 10,
	})

	response.Ok(c, result, nil)
}

// ResendEmailRegister godoc
//
//	@Description	resend email register
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponse
//	@Failure		400	{object}	response.SwaggerBadRequestResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/register/resend-email [post]
func (h *authHandler) ResendEmailRegister(c *gin.Context) {
	email, ok := c.MustGet("email").(string)
	if !ok {
		response.Unauthorized(c)
		return
	}

	result, err := h.emailService.SendEmailRegister(email)
	if err != nil {
		helpers.HandleError(c, err, result)
		return
	}

	response.Ok(c, result, nil)
}

// VerifyOTPRegister godoc
//
//	@Description	verify otp and create user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		VerifyOTPRegisterRequest	true	"verify otp register payload"
//	@Success		201		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/register/verify-otp [post]
func (h *authHandler) VerifyOTPRegister(c *gin.Context) {
	verifyOTPRegisterRequest := &VerifyOTPRegisterRequest{}
	err := c.ShouldBind(verifyOTPRegisterRequest)
	if err != nil {
		c.Error(err)
		return
	}

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.Unauthorized(c)
		return
	}

	result, err := h.authService.VerifyOTPRegister(email, verifyOTPRegisterRequest.OTP)
	if err != nil {
		helpers.HandleError(c, err, result)
		return
	}

	response.ClearCookie(c, "register_token")
	response.Created(c, result, nil)
}

// Login godoc
//
//	@Description	authentication and set cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		LoginRequest	true	"login payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	secureUser, ok := c.MustGet("user").(*SecureUser)
	if !ok {
		response.Unauthorized(c)
		return
	}

	result, tokens, err := h.authService.Login(secureUser.ID)
	if err != nil {
		helpers.HandleError(c, err, result)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "access_token",
		Value:  tokens.accessToken,
		MaxAge: time.Minute * 10,
	})
	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "refresh_token",
		Value:  tokens.refreshToken,
		MaxAge: time.Hour * 72,
	})
	response.Ok(c, result, nil)
}
