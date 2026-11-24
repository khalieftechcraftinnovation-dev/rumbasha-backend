// Package validation berisi helper validasi struct yang dipakai bersama:
//   - response: mengubah error binding Gin (ShouldBindJSON) menjadi pesan per field.
//   - service: memvalidasi struct yang TIDAK datang dari JSON (contoh: baris Excel yang
//     sudah diubah menjadi request create), dengan aturan tag "binding" yang sama persis.
package validation

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// JSONTagName membuat nama field di error memakai tag json/form (contoh "per_page"),
// bukan nama field struct Go ("PerPage").
func JSONTagName(field reflect.StructField) string {
	for _, tag := range []string{"json", "form"} {
		name, _, _ := strings.Cut(field.Tag.Get(tag), ",")
		if name != "" && name != "-" {
			return name
		}
	}
	return field.Name
}

// structValidator dibuat sekali saja (validator menyimpan cache aturan per struct).
var structValidator = sync.OnceValue(func() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding") // aturan sama dengan yang dipakai Gin saat binding JSON
	v.RegisterTagNameFunc(JSONTagName)
	return v
})

// Struct memvalidasi struct dengan aturan tag "binding" (sama dengan Gin).
// Mengembalikan validator.ValidationErrors (atau nil kalau valid).
func Struct(s any) error {
	return structValidator().Struct(s)
}

// Path mengembalikan lokasi lengkap field dengan nama JSON, tanpa nama struct paling luar.
// Contoh: "CreateQuestionRequest.placement.chapter_id" → "placement.chapter_id",
// "CreateQuestionRequest.used_for[1]" → "used_for[1]". Field biasa tetap: "username".
func Path(fe validator.FieldError) string {
	_, path, found := strings.Cut(fe.Namespace(), ".")
	if !found {
		return fe.Field()
	}
	return path
}

// Message menerjemahkan aturan validasi yang gagal menjadi pesan bahasa Indonesia.
// Tambahkan case baru kalau memakai aturan validasi lain.
func Message(fe validator.FieldError) string {
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
