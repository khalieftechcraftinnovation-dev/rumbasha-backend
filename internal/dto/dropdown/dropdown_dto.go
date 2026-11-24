// Package dropdown (dto) berisi bentuk request dan response JSON untuk fitur dropdown.
//
// Di package lain, import dengan alias dropdowndto:
//
//	dropdowndto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/dropdown"
package dropdown

import (
	dropdownmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dropdown"
)

// DropdownQuery adalah query string GET /api/v1/dropdowns/:type (filter induk).
// Hanya filter yang berlaku untuk type tersebut yang dipakai, sisanya diabaikan:
//
//	grade         → education_level_id (opsional)
//	subject_field → subject_id (wajib), grade_id (opsional)
//	chapter       → subject_field_id (wajib)
//	subchapter    → chapter_id (wajib)
//	topic         → subchapter_id (wajib)
type DropdownQuery struct {
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID   string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID        string `form:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID     string `form:"subchapter_id" binding:"omitempty,uuid"`
}

// OptionResponse adalah 1 pilihan dropdown.
// FE cukup memakai "value" (nilai yang dikirim ke API lain) dan "name" (teks yang ditampilkan).
type OptionResponse struct {
	Value    string  `json:"value"`
	ID       string  `json:"id"`
	Code     *string `json:"code"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

// ToOptionResponses mengubah daftar Option menjadi response. Hasil kosong menjadi [] (bukan null).
func ToOptionResponses(options []dropdownmodel.Option) []OptionResponse {
	result := make([]OptionResponse, 0, len(options))
	for _, o := range options {
		result = append(result, OptionResponse{
			Value:    o.Value,
			ID:       o.ID,
			Code:     o.Code,
			Name:     o.Name,
			ParentID: o.ParentID,
		})
	}
	return result
}
