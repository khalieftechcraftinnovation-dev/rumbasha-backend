// Package quizreport (model) berisi struct data report quiz Uji Kom & Tryout: ringkasan per event,
// hasil setiap peserta, dan analisis soal (paling sering salah).
//
// Di package lain, import dengan alias quizreportmodel:
//
//	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/quizreport"
package quizreport

import (
	"errors"
	"time"
)

// ListFilter adalah filter list report. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Type                                 string // uji_kom / tryout
	Search                               string // nama quiz
	EducationLevelID, GradeID, SubjectID string
	From, To                             *time.Time // event dimulai di rentang [From, To)
	Page, PerPage                        int
}

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Summary adalah ringkasan 1 event quiz (1 baris list report).
type Summary struct {
	QuizID         string
	Name           string
	Type           string
	EducationLevel Ref
	Grade          Ref
	Subject        Ref
	StartAt, EndAt time.Time
	TotalQuestions int
	Participants   int      // siswa yang sudah submit
	AverageScore   *float64 // nil kalau belum ada peserta
	HighestScore   *float64
	LowestScore    *float64
}

// Participant adalah hasil 1 peserta di detail report.
type Participant struct {
	Rank            int
	StudentID       string
	Name            string
	StudentCode     *string
	CorrectCount    int
	Score           float64
	Points          int
	DurationSeconds int
	SubmittedAt     time.Time
}

// ParticipantFilter adalah pencarian & halaman detail report.
type ParticipantFilter struct {
	Search        string // nama siswa / kode siswa
	Page, PerPage int
}

// QuestionStat adalah statistik 1 soal di quiz: berapa peserta yang menjawab benar / salah / tidak
// menjawab.
type QuestionStat struct {
	No           int // nomor soal di quiz
	QuestionID   string
	Code         string
	Type         string
	QuestionText string
	Difficulty   string
	Correct      int
	Wrong        int // menjawab tapi salah
	Unanswered   int // tidak dijawab
	Participants int
}

// CorrectPercent = persentase peserta yang menjawab benar (0-100, 2 angka di belakang koma).
func (q QuestionStat) CorrectPercent() float64 {
	if q.Participants == 0 {
		return 0
	}
	return float64(int(float64(q.Correct)*10000/float64(q.Participants)+0.5)) / 100
}

// ErrQuizNotFound: quiz tidak ada / dihapus / bukan Uji Kom-Tryout.
var ErrQuizNotFound = errors.New("quiz tidak ditemukan")

// FieldError & ValidationError: kesalahan query (dibalas 422).
type FieldError struct {
	Field, Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validasi report gagal"
	}
	return "validasi report gagal: " + e.Fields[0].Field + ": " + e.Fields[0].Message
}
