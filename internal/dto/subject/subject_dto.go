// Package subject (dto) berisi bentuk request dan response JSON untuk fitur subject.
// Dipisah dari model supaya format API bisa berubah tanpa mengubah struktur tabel.
//
// Di package lain, import dengan alias subjectdto:
//
//	subjectdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subject"
package subject

import (
	"time"

	subjectmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subject"
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

// ToSubjectResponse mengubah subjectmodel.Subject menjadi SubjectResponse.
func ToSubjectResponse(subject subjectmodel.Subject) SubjectResponse {
	return SubjectResponse{
		ID:        subject.ID,
		Code:      subject.Code,
		Name:      subject.Name,
		CreatedAt: subject.CreatedAt,
		UpdatedAt: subject.UpdatedAt,
	}
}

// ToSubjectResponses mengubah daftar subject menjadi daftar SubjectResponse.
func ToSubjectResponses(subjects []subjectmodel.Subject) []SubjectResponse {
	// make(..., 0, n) membuat slice kosong (bukan nil), supaya JSON-nya [] bukan null.
	result := make([]SubjectResponse, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, ToSubjectResponse(subject))
	}
	return result
}
