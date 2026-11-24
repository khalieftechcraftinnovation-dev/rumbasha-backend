// Package question (dto) berisi bentuk request dan response JSON untuk fitur bank soal.
//
// Di package lain, import dengan alias questiondto:
//
//	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
package question

import (
	"encoding/json"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// CreateQuestionRequest adalah body JSON untuk POST /api/v1/questions.
//
// Aturan di tag binding hanya mengecek bentuk dasar. Aturan yang bergantung pada
// field lain (isi answer per type, display_order, kecocokan placement) dicek di service.
//
// Field "answer" dibiarkan mentah (json.RawMessage) karena bentuknya berbeda per type;
// service yang membacanya sesuai type.
type CreateQuestionRequest struct {
	Type         string           `json:"type" binding:"required,oneof=single_choice multiple_choice true_false free_text"`
	Question     string           `json:"question" binding:"required,max=5000"`
	Formula      *string          `json:"formula" binding:"omitempty,max=2000"`
	ImageID      *string          `json:"image_id" binding:"omitempty,uuid"`
	Table        *TableRequest    `json:"table"`
	Explanation  *string          `json:"explanation" binding:"omitempty,max=5000"`
	DisplayOrder []string         `json:"display_order" binding:"omitempty,unique,dive,oneof=question formula image table"`
	Placement    PlacementRequest `json:"placement" binding:"required"`
	Difficulty   string           `json:"difficulty" binding:"required,oneof=easy medium hard"`
	UsedFor      []string         `json:"used_for" binding:"required,min=1,unique,dive,oneof=daily_quiz uji_kom tryout rumath"`
	Status       string           `json:"status" binding:"omitempty,oneof=active inactive"`
	Answer       json.RawMessage  `json:"answer" binding:"required"`
}

// TableRequest adalah tabel di dalam soal: maksimal 10 kolom dan 50 baris.
// Jumlah kolom tiap baris harus sama (dicek di service).
type TableRequest struct {
	Headers []string   `json:"headers" binding:"omitempty,max=10,dive,max=200"`
	Rows    [][]string `json:"rows" binding:"required,min=1,max=50,dive,min=1,max=10,dive,max=500"`
}

// PlacementRequest adalah letak soal. Wajib sampai bab, sub bab & topik opsional.
type PlacementRequest struct {
	EducationLevelID string  `json:"education_level_id" binding:"required,uuid"`
	GradeID          string  `json:"grade_id" binding:"required,uuid"`
	SubjectID        string  `json:"subject_id" binding:"required,uuid"`
	SubjectFieldID   string  `json:"subject_field_id" binding:"required,uuid"`
	ChapterID        string  `json:"chapter_id" binding:"required,uuid"`
	SubchapterID     *string `json:"subchapter_id" binding:"omitempty,uuid"`
	TopicID          *string `json:"topic_id" binding:"omitempty,uuid"`
}

// QuestionResponse adalah bentuk data soal yang dikirim ke client.
type QuestionResponse struct {
	ID           string               `json:"id"`
	Code         string               `json:"code"`
	Type         string               `json:"type"`
	Question     string               `json:"question"`
	Formula      *string              `json:"formula"`
	ImageID      *string              `json:"image_id"`
	Table        *questionmodel.Table `json:"table"`
	Explanation  *string              `json:"explanation"`
	DisplayOrder []string             `json:"display_order"`
	Placement    PlacementResponse    `json:"placement"`
	Difficulty   string               `json:"difficulty"`
	UsedFor      []string             `json:"used_for"`
	Status       string               `json:"status"`
	Answer       json.RawMessage      `json:"answer"`
	CreatedBy    string               `json:"created_by"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// PlacementResponse adalah letak soal di response.
type PlacementResponse struct {
	EducationLevelID string  `json:"education_level_id"`
	GradeID          string  `json:"grade_id"`
	SubjectID        string  `json:"subject_id"`
	SubjectFieldID   string  `json:"subject_field_id"`
	ChapterID        string  `json:"chapter_id"`
	SubchapterID     *string `json:"subchapter_id"`
	TopicID          *string `json:"topic_id"`
}

// ToQuestionResponse mengubah questionmodel.Question menjadi QuestionResponse.
func ToQuestionResponse(q questionmodel.Question) QuestionResponse {
	return QuestionResponse{
		ID:           q.ID,
		Code:         q.Code,
		Type:         q.Type,
		Question:     q.QuestionText,
		Formula:      q.Formula,
		ImageID:      q.ImageID,
		Table:        q.Table,
		Explanation:  q.Explanation,
		DisplayOrder: q.DisplayOrder,
		Placement: PlacementResponse{
			EducationLevelID: q.EducationLevelID,
			GradeID:          q.GradeID,
			SubjectID:        q.SubjectID,
			SubjectFieldID:   q.SubjectFieldID,
			ChapterID:        q.ChapterID,
			SubchapterID:     q.SubchapterID,
			TopicID:          q.TopicID,
		},
		Difficulty: q.Difficulty,
		UsedFor:    q.UsedFor,
		Status:     q.Status,
		Answer:     q.Answer,
		CreatedBy:  q.CreatedBy,
		CreatedAt:  q.CreatedAt,
		UpdatedAt:  q.UpdatedAt,
	}
}
