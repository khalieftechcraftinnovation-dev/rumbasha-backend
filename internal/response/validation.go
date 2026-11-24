package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/validation"
)

// ValidationError dipanggil handler saat ShouldBindJSON/ShouldBindQuery gagal.
//   - Kalau ada field yang melanggar aturan validasi → 422 + daftar field-nya.
//   - Kalau body-nya bukan JSON yang valid → 400.
func ValidationError(c *gin.Context, err error) {
	// Error asli dicatat di log API (field "process"), contoh:
	// "Key: 'LoginRequest.Password' Error:Field validation for 'password' failed on the 'required' tag"
	_ = c.Error(err)

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		Error(c, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	fields := make([]FieldError, 0, len(validationErrors))
	for _, fe := range validationErrors {
		fields = append(fields, FieldError{
			Field:   validation.Path(fe),
			Message: validation.Message(fe),
		})
	}

	ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
}

// UseJSONFieldNames membuat error validasi Gin memakai nama field dari tag json/form
// (misalnya "per_page"), bukan nama field struct Go ("PerPage").
// Cukup dipanggil sekali saat router dibuat.
func UseJSONFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(validation.JSONTagName)
}
