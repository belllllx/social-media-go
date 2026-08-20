package middlewares

import (
	"errors"

	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/response"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func GlobalErrorsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			var validateErrs validator.ValidationErrors

			if errors.As(ginErr.Err, &validateErrs) {
				errFields := map[string]string{}
				for _, e := range validateErrs {
					errFields[e.Field()] = helpers.GetErrorMessages(e)
				}
				response.AbortWithBadRequestErrorFields(c, errFields)
				return
			}
		}

		if len(c.Errors) > 0 {
			logs.Error(c.Errors.Last().Err)
			response.AbortWithInternalServerError(c, c.Errors.Last().Err)
		}
	}
}
