// Package dto (Data Transfer Object) berisi bentuk request dan response JSON.
// Dipisah dari model supaya format API bisa berubah tanpa mengubah struktur tabel.
package dto

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
)

// CreateSubjectRequest adalah body JSON untuk POST /api/v1/subjects.
//
// Catatan Go: teks di dalam backtick (`...`) adalah "tag":
//   - json:"code"      → nama field di JSON
//   - binding:"..."    → aturan validasi (required = wajib, max=20 = maksimal 20 karakter)
type CreateSubjectRequest struct {
	Code string `json:"code" binding:"required,max=20"`
	Name string `json:"name" binding:"required,max=100"`
}

// UpdateSubjectRequest adalah body JSON untuk PUT /api/v1/subjects/:id.
// Isinya sama dengan Create untuk saat ini, tapi dibuat terpisah karena
// aturan create dan update biasanya akan berbeda seiring waktu.
type UpdateSubjectRequest struct {
	Code string `json:"code" binding:"required,max=20"`
	Name string `json:"name" binding:"required,max=100"`
}

// SubjectResponse adalah bentuk data subject yang dikirim ke client.
type SubjectResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToSubjectResponse mengubah model.Subject menjadi SubjectResponse.
func ToSubjectResponse(subject model.Subject) SubjectResponse {
	return SubjectResponse{
		ID:        subject.ID,
		Code:      subject.Code,
		Name:      subject.Name,
		CreatedAt: subject.CreatedAt,
		UpdatedAt: subject.UpdatedAt,
	}
}

// ToSubjectResponses mengubah daftar model.Subject menjadi daftar SubjectResponse.
func ToSubjectResponses(subjects []model.Subject) []SubjectResponse {
	// make(..., 0, n) membuat slice kosong (bukan nil), supaya JSON-nya [] bukan null.
	result := make([]SubjectResponse, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, ToSubjectResponse(subject))
	}
	return result
}

// PaginationQuery adalah query string ?page=1&per_page=20 untuk endpoint list.
// Tag form:"..." dipakai untuk query string (bukan body JSON).
type PaginationQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// SetDefaults mengisi nilai default kalau client tidak mengirim page/per_page.
func (q *PaginationQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}
