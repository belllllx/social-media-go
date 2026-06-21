package auth

import (
	"errors"

	"github.com/belllllx/social-media-go/internal/email"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/user"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
)

type RegisterRequest struct {
	Fullname string `json:"fullname" binding:"required,max=30"`
	Username string `json:"username" binding:"required,max=15"`
	Email    string `json:"email" binding:"required,email,max=30"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type AuthService interface {
	Register(registerRequest *RegisterRequest) (result, token string, err error)
}

type authService struct {
	userRepository user.UserRepository
	emailService   email.EmailService
}

func NewAuthService(userRepository user.UserRepository, emailService email.EmailService) AuthService {
	return &authService{
		userRepository: userRepository,
		emailService:   emailService,
	}
}

func (s *authService) Register(registerRequest *RegisterRequest) (string, string, error) {
	userExist, err := s.userRepository.FindByUsername(registerRequest.Username)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to register", "", errs.NewInternalServerError()
	}
	if userExist != nil {
		logs.Warn(errors.New("Username is already exist"))
		return "Username is already exist", "", errs.NewBadRequestError()
	}

	userExist, err = s.userRepository.FindByEmail(registerRequest.Email)
	if err != nil && !helpers.IsErrRecordNotFound(err) {
		logs.Error(err)
		return "Failed to register", "", errs.NewInternalServerError()
	}
	if userExist != nil {
		logs.Warn(errors.New("Email is already exist"))
		return "Email is already exist", "", errs.NewBadRequestError()
	}

	SendEmailRegisterDTO := &email.SendEmailRegisterDTO{
		Fullname: registerRequest.Fullname,
		Username: registerRequest.Username,
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
	}
	result, token, err := s.emailService.SendEmailRegister(SendEmailRegisterDTO)
	if err != nil {
		return "Failed to send email", "", err
	}
	return result, token, nil
}
