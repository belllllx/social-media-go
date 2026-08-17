package auth

import (
	"net/http"
	"time"

	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ResetPasswordRequest struct {
	Password        string `json:"password" binding:"required,min=6,max=20"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,min=6,max=20,eqfield=Password"`
}

type SendEmailForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=30"`
}

type SendEmailRegisterRequest struct {
	Fullname string `json:"fullname" binding:"required,max=30"`
	Username string `json:"username" binding:"required,max=15"`
	Email    string `json:"email" binding:"required,email,max=30"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type VerifyOTPRequest struct {
	OTP string `json:"otp" binding:"required,len=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthHandler interface {
	SendEmailRegister(c *gin.Context)
	SendEmailForgotPassword(c *gin.Context)
	ResendEmailRegister(c *gin.Context)
	ResendEmailForgotPassword(c *gin.Context)
	VerifyOTPRegister(c *gin.Context)
	VerifyOTPForgotPassword(c *gin.Context)
	Login(c *gin.Context)
	Profile(c *gin.Context)
	Refresh(c *gin.Context)
	Logout(c *gin.Context)
	ResetPassword(c *gin.Context)
	GoogleLogin(c *gin.Context)
	FacebookLogin(c *gin.Context)
	GithubLogin(c *gin.Context)
	GoogleCallback(c *gin.Context)
	FacebookCallback(c *gin.Context)
	GithubCallback(c *gin.Context)
}

type authHandler struct {
	authService  AuthService
	emailService email.EmailService
	userService  user.UserService
}

func NewAuthHandler(
	authService AuthService,
	emailService email.EmailService,
	userService user.UserService,
) AuthHandler {
	return &authHandler{
		authService:  authService,
		emailService: emailService,
		userService:  userService,
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
	ctx := c.Request.Context()

	sendEmailRegisterRequest := &SendEmailRegisterRequest{}
	err := c.ShouldBind(sendEmailRegisterRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	sendEmailRegisterDTO := &SendEmailRegisterDTO{
		Fullname: sendEmailRegisterRequest.Fullname,
		Username: sendEmailRegisterRequest.Username,
		Email:    sendEmailRegisterRequest.Email,
		Password: sendEmailRegisterRequest.Password,
	}

	token, err := h.authService.SendEmailRegister(ctx, sendEmailRegisterDTO)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "register_token",
		Value:  token,
		MaxAge: time.Minute * 15,
	})

	response.Ok(c, "Send email successfully", nil)
}

// SendEmailForgotPassword godoc
//
//	@Description	send email to verify otp and set cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		SendEmailForgotPasswordRequest	true	"send email forgot password payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		404		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/forgot-password/send-email [post]
func (h *authHandler) SendEmailForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()

	sendEmailForgotPasswordRequest := &SendEmailForgotPasswordRequest{}
	err := c.ShouldBind(sendEmailForgotPasswordRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	token, err := h.authService.SendEmailForgotPassword(ctx, sendEmailForgotPasswordRequest.Email)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "forgot_password_token",
		Value:  token,
		MaxAge: time.Minute * 15,
	})

	response.Ok(c, "Send email successfully", nil)
}

// ResendEmailRegister godoc
//
//	@Description	authentication and resend email register
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponse
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/register/resend-email [post]
func (h *authHandler) ResendEmailRegister(c *gin.Context) {
	ctx := c.Request.Context()

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	err := h.authService.ResendEmail(
		ctx,
		email,
		"register",
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Send email successfully", nil)
}

// ResendEmailForgotPassword godoc
//
//	@Description	authentication and resend email forgot password
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponse
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/forgot-password/resend-email [post]
func (h *authHandler) ResendEmailForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	err := h.authService.ResendEmail(
		ctx,
		email,
		"reset password",
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.Ok(c, "Send email successfully", nil)
}

// VerifyOTPRegister godoc
//
//	@Description	authentication	verify otp and create user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		VerifyOTPRequest	true	"verify otp register payload"
//	@Success		201		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/register/verify-otp [post]
func (h *authHandler) VerifyOTPRegister(c *gin.Context) {
	ctx := c.Request.Context()

	verifyOTPRequest := &VerifyOTPRequest{}
	err := c.ShouldBind(verifyOTPRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	err = h.authService.VerifyOTPRegister(
		ctx,
		email,
		verifyOTPRequest.OTP,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.ClearCookie(c, "register_token")
	response.Created(c, "Register user successfully", nil)
}

// VerifyOTPForgotPassword godoc
//
//	@Description	authentication verify otp and set cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		VerifyOTPRequest	true	"verify otp forgot password payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/forgot-password/verify-otp [post]
func (h *authHandler) VerifyOTPForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()

	verifyOTPRequest := &VerifyOTPRequest{}
	err := c.ShouldBind(verifyOTPRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	token, err := h.authService.VerifyOTPForgotPassword(
		ctx,
		email,
		verifyOTPRequest.OTP,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "reset_password_token",
		Value:  token,
		MaxAge: time.Minute * 10,
	})
	response.Ok(c, "Verify otp successfully", nil)
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
	userID, ok := c.MustGet("userID").(*uuid.UUID)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	tokens, err := h.authService.CreateTokens(*userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "access_token",
		Value:  tokens.AccessToken,
		MaxAge: time.Minute * 10,
	})
	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "refresh_token",
		Value:  tokens.RefreshToken,
		MaxAge: time.Hour * 72,
	})
	response.Ok(c, "Login successfully", nil)
}

// Profile godoc
//
//	@Description	authentication and set cookie
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponseWithData{data=user.SecureUserWithFollowingRelation}
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/profile [get]
func (h *authHandler) Profile(c *gin.Context) {
	user, ok := c.MustGet("user").(*user.SecureUserWithFollowingRelation)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	response.Ok(c, "User retrive successfully", user)
}

// Refresh godoc
//
//	@Description	authentication refresh token
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponseWithData{data=Tokens}
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/refresh-token [post]
func (h *authHandler) Refresh(c *gin.Context) {
	userID, ok := c.MustGet("userID").(uuid.UUID)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	tokens, err := h.authService.CreateTokens(userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "access_token",
		Value:  tokens.AccessToken,
		MaxAge: time.Minute * 10,
	})
	response.SetSecureCookie(c, response.CookieOptions{
		Key:    "refresh_token",
		Value:  tokens.RefreshToken,
		MaxAge: time.Hour * 72,
	})

	response.Ok(c, "Refresh token successfully", tokens)
}

// Logout godoc
//
//	@Description	authentication and clear cookies
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.SwaggerResponse
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/logout [post]
func (h *authHandler) Logout(c *gin.Context) {
	response.ClearCookie(c, "access_token")
	response.ClearCookie(c, "refresh_token")
	response.Ok(c, "Logout successfully", nil)
}

// ResetPassword godoc
//
//	@Description	authentication and reset password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ResetPasswordRequest	true	"reset password payload"
//	@Success		200		{object}	response.SwaggerResponse
//	@Failure		400		{object}	response.SwaggerBadRequestResponse
//	@Failure		401		{object}	response.SwaggerResponse
//	@Failure		500		{object}	response.SwaggerResponse
//	@Router			/auth/forgot-password/reset-password [patch]
func (h *authHandler) ResetPassword(c *gin.Context) {
	ctx := c.Request.Context()

	resetPasswordRequest := &ResetPasswordRequest{}
	err := c.ShouldBind(resetPasswordRequest)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	email, ok := c.MustGet("email").(string)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	err = h.userService.ResetPassword(
		ctx,
		email,
		resetPasswordRequest.ConfirmPassword,
	)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response.ClearCookie(c, "forgot_password_token")
	response.ClearCookie(c, "reset_password_token")
	response.Ok(c, "Reset password successfully", nil)
}

// GoogleLogin godoc
//
//	@Summary		login with google
//	@Description	redirect to google login
//	@Tags			auth
//	@Success		307
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/google [get]
func (h *authHandler) GoogleLogin(c *gin.Context) {
	ctx := c.Request.Context()

	url, err := h.authService.SocialLogin(ctx, models.ProviderTypeGoogle)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// FacebookLogin godoc
//
//	@Summary		login with facebook
//	@Description	redirect to facebook login
//	@Tags			auth
//	@Success		307
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/facebook [get]
func (h *authHandler) FacebookLogin(c *gin.Context) {
	ctx := c.Request.Context()

	url, err := h.authService.SocialLogin(ctx, models.ProviderTypeFacebook)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GithubLogin godoc
//
//	@Summary		login with github
//	@Description	redirect to github login
//	@Tags			auth
//	@Success		307
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/github [get]
func (h *authHandler) GithubLogin(c *gin.Context) {
	ctx := c.Request.Context()

	url, err := h.authService.SocialLogin(ctx, models.ProviderTypeGithub)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback godoc
//
//	@Summary		google login callback
//	@Description	authentications set cookies and redirect
//	@Tags			auth
//	@Success		308
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/google/callback [get]
func (h *authHandler) GoogleCallback(c *gin.Context) {
	ctx := c.Request.Context()

	socialUser, ok := c.MustGet("socialUser").(*SocialUserDTO)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	tokens, url, err := h.authService.SocialLoginCallback(ctx, socialUser)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	if tokens != nil {
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "access_token",
			Value:  tokens.AccessToken,
			MaxAge: time.Minute * 10,
		})
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "refresh_token",
			Value:  tokens.RefreshToken,
			MaxAge: time.Hour * 72,
		})

		c.Redirect(http.StatusPermanentRedirect, url)
	}

	c.Redirect(http.StatusPermanentRedirect, url)
}

// FacebookCallback godoc
//
//	@Summary		facebook login callback
//	@Description	authentications set cookies and redirect
//	@Tags			auth
//	@Success		308
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/facebook/callback [get]
func (h *authHandler) FacebookCallback(c *gin.Context) {
	ctx := c.Request.Context()

	socialUser, ok := c.MustGet("socialUser").(*SocialUserDTO)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	tokens, url, err := h.authService.SocialLoginCallback(ctx, socialUser)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	if tokens != nil {
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "access_token",
			Value:  tokens.AccessToken,
			MaxAge: time.Minute * 10,
		})
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "refresh_token",
			Value:  tokens.RefreshToken,
			MaxAge: time.Hour * 72,
		})

		c.Redirect(http.StatusPermanentRedirect, url)
	}

	c.Redirect(http.StatusPermanentRedirect, url)
}

// GithubCallback godoc
//
//	@Summary		github login callback
//	@Description	authentications set cookies and redirect
//	@Tags			auth
//	@Success		308
//	@Failure		401	{object}	response.SwaggerResponse
//	@Failure		500	{object}	response.SwaggerResponse
//	@Router			/auth/github/callback [get]
func (h *authHandler) GithubCallback(c *gin.Context) {
	ctx := c.Request.Context()

	socialUser, ok := c.MustGet("socialUser").(*SocialUserDTO)
	if !ok {
		response.AbortWithUnauthorized(c)
		return
	}

	tokens, url, err := h.authService.SocialLoginCallback(ctx, socialUser)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	if tokens != nil {
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "access_token",
			Value:  tokens.AccessToken,
			MaxAge: time.Minute * 10,
		})
		response.SetSecureCookie(c, response.CookieOptions{
			Key:    "refresh_token",
			Value:  tokens.RefreshToken,
			MaxAge: time.Hour * 72,
		})

		c.Redirect(http.StatusPermanentRedirect, url)
	}

	c.Redirect(http.StatusPermanentRedirect, url)
}
