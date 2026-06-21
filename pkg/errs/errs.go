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

func NewUnexpectedError() *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Message: "Unexpected error",
	}
}

func NewBadRequestError() *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Message: "Bad request error",
	}
}
