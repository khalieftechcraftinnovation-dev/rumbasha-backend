// Package question (model) berisi struct data untuk fitur bank soal.
//
// Di package lain, import dengan alias questionmodel:
//
//	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
package question

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Tipe soal. Nilainya sama persis dengan isi kolom questions.type.
const (
	TypeSingleChoice   = "single_choice"   // pilihan ganda, 1 jawaban benar
	TypeMultipleChoice = "multiple_choice" // pilihan ganda, bisa lebih dari 1 jawaban benar
	TypeTrueFalse      = "true_false"      // beberapa pernyataan, masing-masing benar/salah
	TypeFreeText       = "free_text"       // isian (angka atau teks)
)

// Bagian soal yang bisa diatur urutan tampilnya (display_order).
const (
	PartQuestion = "question"
	PartFormula  = "formula"
	PartImage    = "image"
	PartTable    = "table"
)

// Question mewakili satu baris di tabel "questions".
// Kolom JSONB (table_data, display_order, used_for, answer) diubah ke/dari JSON di repository.
type Question struct {
	ID           string
	Code         string // otomatis dari database, contoh Q-12
	Type         string
	QuestionText string
	Formula      *string
	ImageID      *string
	Table        *Table
	Explanation  *string
	DisplayOrder []string

	EducationLevelID string
	GradeID          string
	SubjectID        string
	SubjectFieldID   string
	ChapterID        string
	SubchapterID     *string
	TopicID          *string

	Difficulty string
	UsedFor    []string
	Status     string
	Answer     json.RawMessage // isi sesuai Type, lihat struct *Answer di bawah

	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Table adalah tabel di dalam soal. Headers boleh kosong.
type Table struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ----------------------------------------------------------------------------
// Bentuk jawaban per tipe soal (disimpan di kolom questions.answer).
// ----------------------------------------------------------------------------

// ChoiceOption adalah satu opsi jawaban (A, B, C, ...). Boleh teks, gambar, atau keduanya.
type ChoiceOption struct {
	Key     string  `json:"key"`
	Content string  `json:"content"`
	ImageID *string `json:"image_id"`
}

// SingleChoiceAnswer: pilihan ganda dengan 1 kunci jawaban.
type SingleChoiceAnswer struct {
	Options    []ChoiceOption `json:"options"`
	CorrectKey string         `json:"correct_key"`
}

// Cara menilai soal yang punya lebih dari satu kunci.
const (
	ScoringAllOrNothing = "all_or_nothing" // benar semua baru dapat nilai
	ScoringPartial      = "partial"        // nilai sebagian sesuai jumlah yang benar
)

// MultipleChoiceAnswer: pilihan ganda dengan beberapa kunci jawaban.
type MultipleChoiceAnswer struct {
	Options     []ChoiceOption `json:"options"`
	CorrectKeys []string       `json:"correct_keys"`
	Scoring     string         `json:"scoring"`
}

// TrueFalseAnswer: setiap opsi adalah pernyataan yang dinilai benar/salah.
type TrueFalseAnswer struct {
	Options     []ChoiceOption `json:"options"`
	CorrectKeys []TrueFalseKey `json:"correct_keys"`
}

// TrueFalseKey adalah kunci untuk satu pernyataan.
// Correct memakai pointer supaya bisa dibedakan antara "false" dan "tidak diisi".
type TrueFalseKey struct {
	Key     string `json:"key"`
	Correct *bool  `json:"correct"`
}

// Jenis isian pada soal free_text.
const (
	InputTypeNumber = "number"
	InputTypeString = "string"
)

// FreeTextAnswer: soal isian.
//   - number: jawaban siswa dibandingkan sebagai angka (boleh pecahan "3/4"), dengan selisih maksimal Tolerance.
//   - string: jawaban siswa dianggap benar jika MENGANDUNG salah satu kata kunci di AcceptedAnswers.
//
// omitempty: pengaturan yang tidak berlaku untuk input_type-nya tidak ikut disimpan
// (tolerance hanya untuk number, case_sensitive hanya untuk string).
type FreeTextAnswer struct {
	InputType       string   `json:"input_type"`
	AcceptedAnswers []string `json:"accepted_answers"`
	Tolerance       *float64 `json:"tolerance,omitempty"`
	CaseSensitive   *bool    `json:"case_sensitive,omitempty"`
}

// ----------------------------------------------------------------------------
// Data master yang dipakai untuk mengecek placement (letak soal).
// ----------------------------------------------------------------------------

// MasterDataItem adalah kolom master_data_items yang dibutuhkan untuk validasi placement.
type MasterDataItem struct {
	ID               string
	Type             string
	ParentID         *string
	SubjectID        *string
	EducationLevelID *string
	GradeID          *string
	Status           string
}

// ----------------------------------------------------------------------------
// Error
// ----------------------------------------------------------------------------

// FieldError menjelaskan kesalahan pada satu field request, contoh:
// {Field: "answer.options[1].content", Message: "wajib diisi"}.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError berisi semua kesalahan isi soal. Handler mengubahnya menjadi response 422.
type ValidationError struct {
	Fields []FieldError
}

// Error membuat ValidationError memenuhi interface error.
// Teksnya ikut tercatat di log API (field "process"), contoh:
// "validasi soal gagal: answer.correct_key: harus salah satu key opsi".
func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi soal gagal: " + strings.Join(parts, "; ")
}

// ErrQuestionNotFound dipakai saat soal tidak ada (untuk endpoint get/update nanti).
var ErrQuestionNotFound = errors.New("soal tidak ditemukan")
