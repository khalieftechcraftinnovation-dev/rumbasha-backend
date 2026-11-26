// Package dropdown (model) berisi struct data untuk fitur dropdown (pilihan dari master_data_items).
//
// Di package lain, import dengan alias dropdownmodel:
//
//	dropdownmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dropdown"
package dropdown

import (
	"errors"
	"strings"
)

// Type dropdown. Nilainya sama persis dengan isi kolom master_data_items.type.
const (
	TypeEducationLevel = "education_level" // jenjang
	TypeGrade          = "grade"           // kelas
	TypeSubject        = "subject"         // mapel
	TypeSubjectField   = "subject_field"   // bidang materi
	TypeChapter        = "chapter"         // bab
	TypeSubchapter     = "subchapter"      // sub bab
	TypeTopic          = "topic"           // topik
	TypeQuestionType   = "question_type"   // tipe soal
	TypeQuizType       = "quiz_type"       // tipe quiz (peruntukan soal)
	TypeDifficulty     = "difficulty"      // level
	// Bukan master data: paket akses (tabel subscription_plans). parent_id = jenjang sasaran paket (null = semua).
	TypeSubscriptionPlan = "subscription_plan"
)

// Option adalah 1 pilihan dropdown.
// Value = nilai yang dikirim FE ke API lain:
//   - question_type, quiz_type, difficulty → code (contoh "easy"), karena tabel lain menyimpan teksnya.
//   - jenjang s/d topik                    → id (UUID), karena tabel lain menyimpan id-nya.
type Option struct {
	Value    string
	ID       string
	Code     *string
	Name     string
	ParentID *string
}

// Filter adalah kondisi query dropdown di repository.
type Filter struct {
	Type     string
	Root     bool   // true = hanya item tanpa induk (parent_id IS NULL)
	ParentID string // "" = tidak difilter
	GradeID  string // "" = tidak difilter (dipakai bidang materi)
}

// FieldError adalah 1 kesalahan query string, contoh field "subject_id" wajib diisi.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError berisi kesalahan query string dropdown (dibalas 422).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi dropdown gagal: " + strings.Join(parts, "; ")
}

// ErrUnknownType dikembalikan kalau type di URL tidak dikenal (dibalas 404).
var ErrUnknownType = errors.New("type dropdown tidak dikenal")
