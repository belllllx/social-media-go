package errs

import "net/http"

type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewInternalServerError() *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Message: "Internal server error",
	}
}

func NewInternalServerErrorWithMessage(msg string) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Message: msg,
	}
}

func NewUnexpectedError() *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Message: "Unexpected error",
	}
}

func NewUnexpectedErrorWithMessage(msg string) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Message: msg,
	}
}

func NewBadRequestError() *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Message: "Bad request error",
	}
}

func NewBadRequestErrorWithMessage(msg string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Message: msg,
	}
}

func NewUnauthorizedError() *AppError {
	return &AppError{
		Status:  http.StatusUnauthorized,
		Message: "Unauthorized",
	}
}

func NewNotFoundError() *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Message: "Not found",
	}
}

func NewNotFoundErrorWithMessage(msg string) *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Message: msg,
	}
}
