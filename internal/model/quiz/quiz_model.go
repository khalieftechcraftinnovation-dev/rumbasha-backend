// Package quiz (model) berisi struct data untuk fitur quiz (Uji Kom & Tryout).
//
// Di package lain, import dengan alias quizmodel:
//
//	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
package quiz

import (
	"errors"
	"strings"
	"time"
)

// Tipe quiz. Nilainya sama dengan code quiz_type di master data & isi questions.used_for.
const (
	TypeUjiKom = "uji_kom"
	TypeTryout = "tryout"
)

// TypeNames dipakai di pesan error, contoh "soal tidak ditandai untuk Uji Kom".
var TypeNames = map[string]string{
	TypeUjiKom: "Uji Kom",
	TypeTryout: "Tryout",
}

// UjiKomGradeCodes adalah kelas yang boleh untuk Uji Kom (code master data kelas): Kelas 6, 9, 12.
var UjiKomGradeCodes = map[string]bool{
	"GRADE_6":  true,
	"GRADE_9":  true,
	"GRADE_12": true,
}

// MaxQuestions adalah batas soal per quiz, termasuk soal yang diambil dari bundle.
const MaxQuestions = 200

// Status quiz. Quiz baru selalu draft; publish diatur di langkah jadwal & kode akses (nanti).
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Quiz adalah 1 baris tabel quizzes + ID soal yang dipilih (urut = nomor soal).
type Quiz struct {
	ID   string
	Name string
	Type string

	EducationLevelID string
	GradeID          string
	SubjectID        string
	SubjectFieldID   *string // opsional
	ChapterID        *string // opsional, butuh SubjectFieldID
	SubchapterID     *string // opsional, butuh ChapterID
	TopicID          *string // opsional, butuh SubchapterID

	StartAt         time.Time // waktu mulai; selesai = StartAt + DurationMinutes
	DurationMinutes int
	CoinCost        int // 0 = gratis
	TotalQuestions  int // selalu = len(QuestionIDs)
	Status          string

	AccessCode          *string    // 6 karakter A-Z/0-9, wajib untuk Uji Kom & Tryout
	AccessCodeUpdatedAt *time.Time // terakhir dibuat/diganti (scheduler per jam)

	QuestionIDs []string // urutan = nomor soal di quiz (sort_order mulai 1)

	CreatedBy string
	UpdatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MasterDataItem adalah item master data yang dibutuhkan untuk mengecek letak quiz.
type MasterDataItem struct {
	ID       string
	Type     string
	Code     *string // contoh GRADE_6 (dipakai cek kelas Uji Kom)
	ParentID *string
	GradeID  *string
	Status   string
}

// QuestionCandidate adalah soal yang dipilih admin, berisi data yang dicek sebelum
// dimasukkan ke quiz (status, peruntukan, letak).
type QuestionCandidate struct {
	ID             string
	Code           string
	Deleted        bool // true = soal sudah dihapus (bisa terjadi pada soal dari bundle lama)
	Status         string
	UsedFor        []string
	SubjectID      string
	GradeID        string
	SubjectFieldID string
	ChapterID      string
	SubchapterID   *string
	TopicID        *string
}

// QuizDetail adalah quiz lengkap dengan nama letak, nama pembuat, dan daftar soalnya.
type QuizDetail struct {
	Quiz

	EducationLevelName string
	GradeName          string
	SubjectName        string
	SubjectFieldName   *string
	ChapterName        *string
	SubchapterName     *string
	TopicName          *string
	CreatedByName      *string
	UpdatedByName      *string

	Questions []QuizQuestion
}

// QuizQuestion adalah 1 soal di dalam quiz (ringkas, tanpa kunci jawaban).
type QuizQuestion struct {
	SortOrder    int
	ID           string
	Code         string
	Type         string
	QuestionText string
	Difficulty   string
}

// Bundle adalah quiz yang sudah dibuat, dipakai sebagai sumber soal ("bundle") saat membuat quiz baru.
type Bundle struct {
	ID, Name, Type                       string
	EducationLevelID, EducationLevelName string
	GradeID, GradeName                   string
	SubjectID, SubjectName               string
	TotalQuestions                       int
	CreatedAt                            time.Time
}

// BundleFilter adalah filter list bundle. Field kosong ("") = tidak difilter.
type BundleFilter struct {
	Search                               string // nama quiz
	Type                                 string
	EducationLevelID, GradeID, SubjectID string
	Page, PerPage                        int
}

// FieldError adalah 1 kesalahan pada field request, contoh field "question_ids[2]".
type FieldError struct {
	Field   string
	Message string
}

// ValidationError berisi semua kesalahan data quiz (dibalas 422).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi quiz gagal: " + strings.Join(parts, "; ")
}

var ErrQuizNotFound = errors.New("quiz tidak ditemukan")

// ErrAccessCodeTaken dikembalikan repository kalau kode akses sudah dipakai quiz lain;
// service membuat kode baru lalu mencoba lagi.
var ErrAccessCodeTaken = errors.New("kode akses sudah dipakai quiz lain")
