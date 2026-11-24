// Package question (model) berisi struct data untuk fitur bank soal.
//
// Di package lain, import dengan alias questionmodel:
//
//	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
package question

import (
	"encoding/json"
	"errors"
	"fmt"
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
	ImageID      *string // gambar dari Image Library (media_assets)
	ImageURL     *string // ATAU link gambar dari luar (http/https); tidak boleh dua-duanya
	Table        *Table
	Explanation  string // pembahasan, wajib
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
	UpdatedBy *string // NULL kalau belum pernah diubah
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
// Gambar opsi: image_id (Image Library) ATAU image_url (link luar), tidak boleh dua-duanya.
// image_url memakai omitempty supaya opsi tanpa link luar tidak menampilkan field ini.
type ChoiceOption struct {
	Key      string  `json:"key"`
	Content  string  `json:"content"`
	ImageID  *string `json:"image_id"`
	ImageURL *string `json:"image_url,omitempty"`
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
	InputTypeText   = "text"
)

// FreeTextAnswer: soal isian.
//   - number: jawaban siswa dibandingkan sebagai angka (boleh pecahan "3/4"), dengan selisih maksimal Tolerance.
//   - text: jawaban siswa dianggap benar jika MENGANDUNG salah satu kata kunci di AcceptedAnswers.
//
// omitempty: pengaturan yang tidak berlaku untuk input_type-nya tidak ikut disimpan
// (tolerance hanya untuk number, case_sensitive hanya untuk text).
type FreeTextAnswer struct {
	InputType       string   `json:"input_type"`
	AcceptedAnswers []string `json:"accepted_answers"`
	Tolerance       *float64 `json:"tolerance,omitempty"`
	CaseSensitive   *bool    `json:"case_sensitive,omitempty"`
}

// ----------------------------------------------------------------------------
// Detail soal (endpoint GET /questions/:id)
// ----------------------------------------------------------------------------

// QuestionDetail adalah soal lengkap dengan NAMA placement dan nama pembuatnya.
type QuestionDetail struct {
	Question

	EducationLevelName string
	GradeName          string
	SubjectName        string
	SubjectFieldName   string
	ChapterName        string
	SubchapterName     *string // NULL kalau soal tidak punya sub bab
	TopicName          *string // NULL kalau soal tidak punya topik
	CreatedByName      *string // NULL kalau user pembuat sudah tidak ada
	UpdatedByName      *string // NULL kalau belum pernah diubah
}

// CanonicalAnswer merapikan JSON answer dari database supaya urutan field-nya sama
// dengan saat create. (PostgreSQL JSONB mengurutkan ulang key object, contoh
// "correct_key" bisa muncul sebelum "options".)
func CanonicalAnswer(questionType string, raw []byte) (json.RawMessage, error) {
	var target any
	switch questionType {
	case TypeSingleChoice:
		target = &SingleChoiceAnswer{}
	case TypeMultipleChoice:
		target = &MultipleChoiceAnswer{}
	case TypeTrueFalse:
		target = &TrueFalseAnswer{}
	case TypeFreeText:
		target = &FreeTextAnswer{}
	default:
		return nil, errors.New("type soal tidak dikenal: " + questionType)
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return nil, err
	}
	return json.Marshal(target)
}

// ----------------------------------------------------------------------------
// Data master yang dipakai untuk mengecek placement (letak soal).
// ----------------------------------------------------------------------------

// MasterDataItem adalah kolom master_data_items yang dibutuhkan untuk validasi placement.
type MasterDataItem struct {
	ID               string
	Type             string
	Name             string // hanya diisi saat mencari berdasarkan nama (upload Excel)
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
	Row     int    // khusus upload Excel: nomor baris (mulai 1, baris 1 = header)
	Column  string // khusus upload Excel: huruf kolom, contoh "AB"
}

// ValidationError berisi semua kesalahan isi soal. Handler mengubahnya menjadi response 422.
// Message boleh kosong (handler memakai "Validasi gagal").
type ValidationError struct {
	Message string
	Fields  []FieldError
}

// Error membuat ValidationError memenuhi interface error.
// Teksnya ikut tercatat di log API (field "process"), contoh:
// "validasi soal gagal: answer.correct_key: harus salah satu key opsi"
// atau untuk Excel: "validasi soal gagal: AB3 correct_key: harus salah satu key di options".
func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		location := f.Field
		if f.Row > 0 {
			location = fmt.Sprintf("%s%d %s", f.Column, f.Row, f.Field)
		}
		parts = append(parts, location+": "+f.Message)
	}
	return "validasi soal gagal: " + strings.Join(parts, "; ")
}

// ImportedQuestion adalah 1 soal hasil cek file Excel (belum disimpan).
// Placement sudah berisi ID + nama (nama diambil dari database, bukan dari Excel).
type ImportedQuestion struct {
	Row int // nomor baris di Excel
	QuestionDetail
}

var ErrQuestionNotFound = errors.New("soal tidak ditemukan")

// ----------------------------------------------------------------------------
// List soal (endpoint GET /questions)
// ----------------------------------------------------------------------------

// ListFilter adalah filter list Bank Soal. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Type       string
	Difficulty string

	EducationLevelID string
	GradeID          string
	SubjectID        string
	SubjectFieldID   string
	ChapterID        string
	SubchapterID     string
	TopicID          string

	UsedFor string // peruntukan, contoh "uji_kom" (soal yang used_for-nya berisi nilai ini)
	Status  string // active / inactive
	QuizID  string // isi 1 bundle: hanya soal yang ada di quiz ini

	Search string // dicari di kode soal & teks soal (tidak peka huruf besar/kecil)

	Page    int
	PerPage int
}

// QuestionListItem adalah 1 baris tabel Bank Soal. Tanpa kunci jawaban & pembahasan
// (lihat detail untuk itu), supaya response list tetap kecil.
type QuestionListItem struct {
	ID, Code, Type, QuestionText string

	// Media di soal (bukan di opsi jawaban), untuk label "+N media" di tabel.
	HasFormula, HasImage, HasTable bool

	EducationLevelID, EducationLevelName string
	GradeID, GradeName                   string
	SubjectID, SubjectName               string
	SubjectFieldID, SubjectFieldName     string
	ChapterID, ChapterName               string
	SubchapterID, SubchapterName         *string
	TopicID, TopicName                   *string

	Difficulty string
	UsedFor    []string
	Status     string

	CreatedBy                string
	CreatedByName            *string
	UpdatedBy, UpdatedByName *string // NULL kalau belum pernah diubah
	CreatedAt, UpdatedAt     time.Time

	Quizzes []QuizRef // quiz yang sudah memakai soal ini ("bundle"), terbaru dulu
}

// QuizRef adalah quiz yang memakai sebuah soal (tag json karena dibaca dari json_agg).
type QuizRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
