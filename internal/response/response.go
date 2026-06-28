package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type GlobalResponse struct {
	Status      int               `json:"status"`
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Data        any               `json:"data,omitempty"`
	ErrorFields map[string]string `json:"errorFields,omitempty"`
}

type SwaggerResponse struct {
	Status  int    `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type SwaggerResponseWithData struct {
	Status  int    `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type SwaggerBadRequestResponse struct {
	Status      int               `json:"status"`
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	ErrorFields map[string]string `json:"errorFields"`
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

func ClearCookie(c *gin.Context, key string) {
	c.SetCookie(key, "", -1, "/", "localhost", true, true)
}

func Ok(c *gin.Context, msg string, result any) {
	c.JSON(http.StatusOK, GlobalResponse{
		Status:  http.StatusOK,
		Success: true,
		Message: msg,
		Data:    result,
	})
}

func Created(c *gin.Context, msg string, result any) {
	c.JSON(http.StatusCreated, GlobalResponse{
		Status:  http.StatusCreated,
		Success: true,
		Message: msg,
		Data:    result,
	})
}

func BadRequest(c *gin.Context, errorField map[string]string) {
	c.JSON(http.StatusBadRequest, GlobalResponse{
		Status:      http.StatusBadRequest,
		Success:     false,
		Message:     "Invalid input",
		ErrorFields: errorField,
	})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, GlobalResponse{
		Status:  http.StatusUnauthorized,
		Success: false,
		Message: "Unauthorized",
	})
}

func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, GlobalResponse{
		Status:  http.StatusNotFound,
		Success: false,
		Message: "Not found",
	})
}

func InternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, GlobalResponse{
		Status:  http.StatusInternalServerError,
		Success: false,
		Message: err.Error(),
	})
}
