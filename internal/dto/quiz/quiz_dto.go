// Package quiz (dto) berisi bentuk request dan response JSON untuk fitur quiz.
//
// Di package lain, import dengan alias quizdto:
//
//	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
package quiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
)

// CreateQuizRequest adalah body JSON POST /api/v1/quizzes.
//
//	{
//	  "name": "Uji Kompetensi Matematika - Pecahan", "type": "uji_kom",
//	  "placement": {"education_level_id": "...", "grade_id": "...", "subject_id": "..."},
//	  "start_at": "2026-12-01T08:00",
//	  "duration_minutes": 60, "coin_cost": 0, "total_questions": 12,
//	  "bundle_quiz_ids": ["..."],
//	  "question_ids": ["...", "..."]
//	}
//
// Soal quiz = semua soal dari bundle_quiz_ids (urut bundle, lalu nomor soal di bundle), lalu
// question_ids. Soal yang muncul lebih dari sekali hanya diambil sekali (yang pertama).
// Minimal salah satu dari bundle_quiz_ids / question_ids diisi.
//
// Aturan yang bergantung pada field lain (total_questions = jumlah soal, kecocokan letak,
// syarat soal, kelas Uji Kom, waktu mulai) dicek di service.
type CreateQuizRequest struct {
	Name      string               `json:"name" binding:"required,max=200"`
	Type      string               `json:"type" binding:"required,oneof=uji_kom tryout"`
	Placement QuizPlacementRequest `json:"placement" binding:"required"`
	// Waktu mulai dalam WIB. Format "2026-12-01T08:00" / "2026-12-01 08:00" (dianggap WIB),
	// atau RFC3339 dengan zona waktu ("2026-12-01T08:00:00+07:00"). Harus setelah waktu sekarang.
	StartAt         string `json:"start_at" binding:"required"`
	DurationMinutes int    `json:"duration_minutes" binding:"required,min=1,max=600"` // menit
	CoinCost        *int   `json:"coin_cost" binding:"omitempty,min=0,max=1000000"`   // opsional, default 0 (gratis)
	TotalQuestions  int    `json:"total_questions" binding:"required,min=1,max=200"`
	// Quiz lain yang soalnya diambil semua (bundle). Maksimal 20 bundle.
	BundleQuizIDs []string `json:"bundle_quiz_ids" binding:"omitempty,max=20,unique,dive,uuid"`
	// Soal satuan. Urutan di array = urutan soal (setelah soal dari bundle).
	// Total soal maksimal 200 per quiz, termasuk soal dari bundle (contoh desain Tryout: 40).
	QuestionIDs []string `json:"question_ids" binding:"omitempty,max=200,unique,dive,uuid"`
}

// QuizPlacementRequest adalah letak quiz. Jenjang, kelas, mapel wajib; sisanya opsional
// tapi harus berurutan (bab butuh bidang materi, sub bab butuh bab, topik butuh sub bab).
type QuizPlacementRequest struct {
	EducationLevelID string  `json:"education_level_id" binding:"required,uuid"`
	GradeID          string  `json:"grade_id" binding:"required,uuid"`
	SubjectID        string  `json:"subject_id" binding:"required,uuid"`
	SubjectFieldID   *string `json:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID        *string `json:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID     *string `json:"subchapter_id" binding:"omitempty,uuid"`
	TopicID          *string `json:"topic_id" binding:"omitempty,uuid"`
}

// QuizDetailResponse adalah data quiz lengkap (response create).
type QuizDetailResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Placement       QuizPlacementResponse  `json:"placement"`
	StartAt         time.Time              `json:"start_at"` // WIB, contoh "2026-12-01T08:00:00+07:00"
	EndAt           time.Time              `json:"end_at"`   // start_at + durasi, WIB
	DurationMinutes int                    `json:"duration_minutes"`
	AccessCode      *string                `json:"access_code"` // berganti setiap jam selama quiz belum selesai
	CoinCost        int                    `json:"coin_cost"`
	TotalQuestions  int                    `json:"total_questions"`
	Status          string                 `json:"status"`
	Questions       []QuizQuestionResponse `json:"questions"`
	CreatedBy       RefResponse            `json:"created_by"`
	UpdatedBy       *RefResponse           `json:"updated_by"` // null kalau belum pernah diubah
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// QuizPlacementResponse adalah letak quiz dengan nama. Tingkat yang tidak diisi = null.
type QuizPlacementResponse struct {
	EducationLevel RefResponse  `json:"education_level"`
	Grade          RefResponse  `json:"grade"`
	Subject        RefResponse  `json:"subject"`
	SubjectField   *RefResponse `json:"subject_field"`
	Chapter        *RefResponse `json:"chapter"`
	Subchapter     *RefResponse `json:"subchapter"`
	Topic          *RefResponse `json:"topic"`
}

// QuizQuestionResponse adalah 1 soal di quiz (ringkas, tanpa kunci jawaban).
type QuizQuestionResponse struct {
	No         int    `json:"no"` // nomor soal di quiz, mulai 1
	ID         string `json:"id"`
	Code       string `json:"code"`
	Type       string `json:"type"`
	Question   string `json:"question"`
	Difficulty string `json:"difficulty"`
}

// RefResponse adalah pasangan id + nama, contoh {"id": "...", "name": "Kelas 4"}.
type RefResponse struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

// ToQuizDetailResponse mengubah quizmodel.QuizDetail menjadi response.
func ToQuizDetailResponse(d quizmodel.QuizDetail) QuizDetailResponse {
	questions := make([]QuizQuestionResponse, 0, len(d.Questions))
	for _, q := range d.Questions {
		questions = append(questions, QuizQuestionResponse{
			No: q.SortOrder, ID: q.ID, Code: q.Code, Type: q.Type, Question: q.QuestionText, Difficulty: q.Difficulty,
		})
	}

	return QuizDetailResponse{
		ID:   d.ID,
		Name: d.Name,
		Type: d.Type,
		Placement: QuizPlacementResponse{
			EducationLevel: RefResponse{ID: d.EducationLevelID, Name: &d.EducationLevelName},
			Grade:          RefResponse{ID: d.GradeID, Name: &d.GradeName},
			Subject:        RefResponse{ID: d.SubjectID, Name: &d.SubjectName},
			SubjectField:   optionalRef(d.SubjectFieldID, d.SubjectFieldName),
			Chapter:        optionalRef(d.ChapterID, d.ChapterName),
			Subchapter:     optionalRef(d.SubchapterID, d.SubchapterName),
			Topic:          optionalRef(d.TopicID, d.TopicName),
		},
		StartAt:         d.StartAt.In(clock.Jakarta),
		EndAt:           d.StartAt.Add(time.Duration(d.DurationMinutes) * time.Minute).In(clock.Jakarta),
		DurationMinutes: d.DurationMinutes,
		AccessCode:      d.AccessCode,
		CoinCost:        d.CoinCost,
		TotalQuestions:  d.TotalQuestions,
		Status:          d.Status,
		Questions:       questions,
		CreatedBy:       RefResponse{ID: d.CreatedBy, Name: d.CreatedByName},
		UpdatedBy:       optionalRef(d.UpdatedBy, d.UpdatedByName),
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// optionalRef mengembalikan nil (JSON null) kalau id kosong.
func optionalRef(id, name *string) *RefResponse {
	if id == nil {
		return nil
	}
	return &RefResponse{ID: *id, Name: name}
}

// ListBundlesQuery adalah query string GET /api/v1/quiz-bundles (pilihan bundle saat membuat quiz).
// Semua opsional. Contoh: ?type=uji_kom&subject_id=...&grade_id=...&search=pecahan
type ListBundlesQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search           string `form:"search" binding:"omitempty,max=100"` // nama quiz
	Type             string `form:"type" binding:"omitempty,oneof=uji_kom tryout"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
}

// BundleResponse adalah 1 bundle (quiz yang soalnya bisa dipakai ulang).
// Isi soalnya: GET /api/v1/questions?quiz_id=<id>.
type BundleResponse struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	EducationLevel RefResponse `json:"education_level"`
	Grade          RefResponse `json:"grade"`
	Subject        RefResponse `json:"subject"`
	TotalQuestions int         `json:"total_questions"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ToBundleResponses mengubah daftar bundle menjadi response. Hasil kosong menjadi [] (bukan null).
func ToBundleResponses(bundles []quizmodel.Bundle) []BundleResponse {
	result := make([]BundleResponse, 0, len(bundles))
	for _, b := range bundles {
		result = append(result, BundleResponse{
			ID:             b.ID,
			Name:           b.Name,
			Type:           b.Type,
			EducationLevel: RefResponse{ID: b.EducationLevelID, Name: &b.EducationLevelName},
			Grade:          RefResponse{ID: b.GradeID, Name: &b.GradeName},
			Subject:        RefResponse{ID: b.SubjectID, Name: &b.SubjectName},
			TotalQuestions: b.TotalQuestions,
			CreatedAt:      b.CreatedAt,
		})
	}
	return result
}
