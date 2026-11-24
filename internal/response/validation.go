package response

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
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
			Field:   fieldPath(fe),
			Message: validationMessage(fe),
		})
	}

	ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
}

// fieldPath mengembalikan lokasi lengkap field dengan nama JSON, tanpa nama struct paling luar.
// Contoh: "CreateQuestionRequest.placement.chapter_id" → "placement.chapter_id",
// "CreateQuestionRequest.used_for[1]" → "used_for[1]". Field biasa tetap: "username".
func fieldPath(fe validator.FieldError) string {
	_, path, found := strings.Cut(fe.Namespace(), ".")
	if !found {
		return fe.Field()
	}
	return path
}

// validationMessage menerjemahkan aturan validasi yang gagal menjadi pesan bahasa Indonesia.
// Tambahkan case baru kalau memakai aturan validasi lain.
func validationMessage(fe validator.FieldError) string {
	isList := fe.Kind() == reflect.Slice || fe.Kind() == reflect.Array

	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("maksimal %s karakter", fe.Param())
		}
		if isList {
			return fmt.Sprintf("maksimal %s item", fe.Param())
		}
		return fmt.Sprintf("maksimal %s", fe.Param())
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("minimal %s karakter", fe.Param())
		}
		if isList {
			return fmt.Sprintf("minimal %s item", fe.Param())
		}
		return fmt.Sprintf("minimal %s", fe.Param())
	case "oneof":
		return "harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "uuid":
		return "format UUID tidak valid"
	case "unique":
		return "tidak boleh ada yang sama"
	case "email":
		return "format email tidak valid"
	default:
		return "tidak valid"
	}
}

// UseJSONFieldNames membuat error validasi memakai nama field dari tag json/form
// (misalnya "per_page"), bukan nama field struct Go ("PerPage").
// Cukup dipanggil sekali saat router dibuat.
func UseJSONFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form"} {
			name, _, _ := strings.Cut(field.Tag.Get(tag), ",")
			if name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})
}
