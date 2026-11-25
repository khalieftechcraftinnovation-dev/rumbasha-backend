// Package quizreport (dto) berisi bentuk request dan response JSON report quiz Uji Kom & Tryout.
//
// Di package lain, import dengan alias quizreportdto:
//
//	quizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quizreport"
package quizreport

import (
	"fmt"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quizreport"
)

// ListQuery adalah query string GET /api/v1/quiz-reports. Semua opsional.
//
//	?type=tryout&search=matematika&date_from=2026-11-01&date_to=2026-11-30&grade_id=...&page=1&per_page=20
type ListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Type             string `form:"type" binding:"omitempty,oneof=uji_kom tryout"`
	Search           string `form:"search" binding:"omitempty,max=100"` // nama quiz
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	DateFrom         string `form:"date_from" binding:"omitempty,max=10"` // tanggal mulai event >= (WIB)
	DateTo           string `form:"date_to" binding:"omitempty,max=10"`   // tanggal mulai event <= (WIB)
}

// DetailQuery adalah query string GET /api/v1/quiz-reports/:id (peserta).
type DetailQuery struct {
	Search  string `form:"search" binding:"omitempty,max=100"` // nama / kode siswa
	Page    int    `form:"page" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SummaryResponse adalah ringkasan 1 event quiz (baris list report & kepala detail report).
type SummaryResponse struct {
	QuizID         string      `json:"quiz_id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"` // uji_kom / tryout
	EducationLevel RefResponse `json:"education_level"`
	Grade          RefResponse `json:"grade"`
	Subject        RefResponse `json:"subject"`
	StartAt        time.Time   `json:"start_at"` // tanggal event (WIB)
	EndAt          time.Time   `json:"end_at"`
	TotalQuestions int         `json:"total_questions"`
	Participants   int         `json:"participants"`  // siswa yang sudah submit
	AverageScore   *float64    `json:"average_score"` // null kalau belum ada peserta
	HighestScore   *float64    `json:"highest_score"`
	LowestScore    *float64    `json:"lowest_score"`
}

// ToSummaryResponse mengubah ringkasan menjadi response.
func ToSummaryResponse(s quizreportmodel.Summary) SummaryResponse {
	return SummaryResponse{
		QuizID: s.QuizID, Name: s.Name, Type: s.Type,
		EducationLevel: RefResponse(s.EducationLevel), Grade: RefResponse(s.Grade), Subject: RefResponse(s.Subject),
		StartAt: s.StartAt.In(clock.Jakarta), EndAt: s.EndAt.In(clock.Jakarta),
		TotalQuestions: s.TotalQuestions, Participants: s.Participants,
		AverageScore: s.AverageScore, HighestScore: s.HighestScore, LowestScore: s.LowestScore,
	}
}

// ToSummaryResponses mengubah list ringkasan menjadi response. Kosong = [].
func ToSummaryResponses(items []quizreportmodel.Summary) []SummaryResponse {
	result := make([]SummaryResponse, 0, len(items))
	for _, s := range items {
		result = append(result, ToSummaryResponse(s))
	}
	return result
}

// DetailResponse adalah data GET /api/v1/quiz-reports/:id: ringkasan event + 1 halaman peserta.
type DetailResponse struct {
	Quiz         SummaryResponse       `json:"quiz"`
	Participants []ParticipantResponse `json:"participants"`
}

// ParticipantResponse adalah hasil 1 peserta.
type ParticipantResponse struct {
	Rank            int       `json:"rank"`
	Student         Student   `json:"student"`
	CorrectCount    int       `json:"correct_count"`
	TotalQuestions  int       `json:"total_questions"`
	CorrectLabel    string    `json:"correct_label"` // contoh "9/10"
	Score           float64   `json:"score"`         // nilai 0-100
	Points          int       `json:"points"`
	DurationSeconds int       `json:"duration_seconds"` // waktu penyelesaian
	SubmittedAt     time.Time `json:"submitted_at"`     // WIB
}

// Student adalah data siswa peserta.
type Student struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	StudentCode *string `json:"student_code"`
}

// ToDetailResponse mengubah detail report menjadi response.
func ToDetailResponse(s quizreportmodel.Summary, participants []quizreportmodel.Participant) DetailResponse {
	r := DetailResponse{Quiz: ToSummaryResponse(s), Participants: make([]ParticipantResponse, 0, len(participants))}
	for _, p := range participants {
		r.Participants = append(r.Participants, ParticipantResponse{
			Rank:         p.Rank,
			Student:      Student{ID: p.StudentID, Name: p.Name, StudentCode: p.StudentCode},
			CorrectCount: p.CorrectCount, TotalQuestions: s.TotalQuestions,
			CorrectLabel: fmt.Sprintf("%d/%d", p.CorrectCount, s.TotalQuestions),
			Score:        p.Score, Points: p.Points, DurationSeconds: p.DurationSeconds,
			SubmittedAt: p.SubmittedAt.In(clock.Jakarta),
		})
	}
	return r
}

// QuestionsResponse adalah data GET /api/v1/quiz-reports/:id/questions.
type QuestionsResponse struct {
	Quiz      SummaryResponse    `json:"quiz"`
	Questions []QuestionResponse `json:"questions"` // paling sering salah dulu
}

// QuestionResponse adalah statistik 1 soal.
type QuestionResponse struct {
	No              int     `json:"no"` // nomor soal di quiz
	ID              string  `json:"id"`
	Code            string  `json:"code"`
	Type            string  `json:"type"`
	Question        string  `json:"question"`
	Difficulty      string  `json:"difficulty"`
	Participants    int     `json:"participants"`
	CorrectCount    int     `json:"correct_count"`
	WrongCount      int     `json:"wrong_count"`      // menjawab tapi salah
	UnansweredCount int     `json:"unanswered_count"` // tidak dijawab
	CorrectPercent  float64 `json:"correct_percent"`  // 0-100, contoh 13.33
	CorrectLabel    string  `json:"correct_label"`    // contoh "13%" (dibulatkan)
}

// ToQuestionsResponse mengubah statistik soal menjadi response (urutan dari service).
func ToQuestionsResponse(s quizreportmodel.Summary, stats []quizreportmodel.QuestionStat) QuestionsResponse {
	r := QuestionsResponse{Quiz: ToSummaryResponse(s), Questions: make([]QuestionResponse, 0, len(stats))}
	for _, q := range stats {
		percent := q.CorrectPercent()
		r.Questions = append(r.Questions, QuestionResponse{
			No: q.No, ID: q.QuestionID, Code: q.Code, Type: q.Type, Question: q.QuestionText, Difficulty: q.Difficulty,
			Participants: q.Participants, CorrectCount: q.Correct, WrongCount: q.Wrong, UnansweredCount: q.Unanswered,
			CorrectPercent: percent, CorrectLabel: fmt.Sprintf("%.0f%%", percent),
		})
	}
	return r
}
