// Package dailyquizreport (model) berisi struct data report Daily Quiz PER SISWA (Daily Quiz dibuat
// siswa sendiri, jadi tidak ada event): rekap dalam rentang tanggal, bisa difilter mapel, dan
// rincian per mapel untuk 1 siswa.
//
// Di package lain, import dengan alias dailyquizreportmodel:
//
//	dailyquizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquizreport"
package dailyquizreport

import (
	"errors"
	"time"
)

// Period adalah rentang tanggal submit [From, To) dalam WIB.
type Period struct {
	From, To time.Time
}

// ListFilter adalah filter list report per siswa. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Period
	SubjectID        string // mapel Daily Quiz; kosong = semua mapel digabung
	EducationLevelID string // jenjang akun siswa
	GradeID          string // kelas akun siswa
	Search           string // nama / kode siswa
	Page, PerPage    int
}

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Stats adalah rekap Daily Quiz yang sudah di-submit (per siswa, atau per mapel 1 siswa).
type Stats struct {
	TotalQuizzes           int
	Easy, Medium, Hard     int // jumlah Daily Quiz per level
	AverageScore           float64
	AverageDurationSeconds int
	TotalQuestions         int // soal yang dikerjakan (semua soal di Daily Quiz yang di-submit)
	Correct                int
	TotalPoints            int
	TotalDurationSeconds   int // penentu peringkat kalau point sama
}

// Wrong = soal yang tidak dijawab benar (salah + tidak dijawab).
func (s Stats) Wrong() int {
	return s.TotalQuestions - s.Correct
}

// CorrectPercent = persentase soal benar (0-100, 2 angka di belakang koma).
func (s Stats) CorrectPercent() float64 {
	if s.TotalQuestions == 0 {
		return 0
	}
	return float64(int(float64(s.Correct)*10000/float64(s.TotalQuestions)+0.5)) / 100
}

// StudentRow adalah 1 baris report: siswa + rekap + peringkat.
type StudentRow struct {
	Rank           int
	StudentID      string
	Name           string
	StudentCode    *string
	EducationLevel *Ref
	Grade          *Ref
	Stats
}

// SubjectRow adalah rekap 1 mapel untuk 1 siswa.
type SubjectRow struct {
	Subject Ref
	Stats
}

// Student adalah data siswa di detail report.
type Student struct {
	ID             string
	Name           string
	StudentCode    *string
	EducationLevel *Ref
	Grade          *Ref
}

var (
	ErrStudentNotFound = errors.New("siswa tidak ditemukan")
)

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
