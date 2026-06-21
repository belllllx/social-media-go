package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ResponseShape struct {
	Status      int               `json:"status"`
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Data        interface{}       `json:"data,omitempty"`
	ErrorFields map[string]string `json:"error_fields,omitempty"`
}

type RegisterResponse struct {
	Status      int               `json:"status"`
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	ErrorFields map[string]string `json:"error_fields,omitempty"`
}

type CookieOptions struct {
	Key    string
	Value  string
	MaxAge time.Duration
}

func SetSecureCookie(c *gin.Context, cookieOptions CookieOptions) {
	c.SetCookieData(&http.Cookie{
		Name:     cookieOptions.Key,
		Value:    cookieOptions.Value,
		MaxAge:   int(cookieOptions.MaxAge.Seconds()),
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
	})
}

func Ok(c *gin.Context, msg string, result interface{}) {
	c.JSON(http.StatusOK, ResponseShape{
		Status:  http.StatusOK,
		Success: true,
		Message: msg,
		Data:    result,
	})
}

func Created(c *gin.Context, msg string, result interface{}) {
	c.JSON(http.StatusCreated, ResponseShape{
		Status:  http.StatusCreated,
		Success: true,
		Message: msg,
		Data:    result,
	})
}

func BadRequest(c *gin.Context, errorField map[string]string) {
	c.JSON(http.StatusBadRequest, ResponseShape{
		Status:      http.StatusBadRequest,
		Success:     false,
		Message:     "Invalid input",
		ErrorFields: errorField,
	})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, ResponseShape{
		Status:  http.StatusUnauthorized,
		Success: false,
		Message: "Unauthorized",
	})
}

func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ResponseShape{
		Status:  http.StatusNotFound,
		Success: false,
		Message: "Not found",
	})
}

func InternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, ResponseShape{
		Status:  http.StatusInternalServerError,
		Success: false,
		Message: err.Error(),
	})
}
