// Package dailyquiz (model) berisi struct data untuk fitur Daily Quiz: quiz latihan yang
// di-generate siswa dari soal acak sesuai pilihannya (tanpa kode akses & timer).
//
// Di package lain, import dengan alias dailyquizmodel:
//
//	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
package dailyquiz

import (
	"errors"
	"fmt"
	"strings"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// sdGradeGroups membagi kelas SD menjadi 2 kelompok: siswa hanya boleh memilih kelas dalam
// kelompoknya (contoh siswa Kelas 5 → Kelas 4, 5, 6). Jenjang lain: 1 kelompok per jenjang.
var sdGradeGroups = map[string]string{
	"GRADE_1": "SD_1_3", "GRADE_2": "SD_1_3", "GRADE_3": "SD_1_3",
	"GRADE_4": "SD_4_6", "GRADE_5": "SD_4_6", "GRADE_6": "SD_4_6",
}

// GradeGroup mengembalikan kelompok kelas: kelompok SD untuk Kelas 1-6, selain itu jenjangnya.
func GradeGroup(gradeCode *string, educationLevelID string) string {
	if gradeCode != nil {
		if group, ok := sdGradeGroups[*gradeCode]; ok {
			return group
		}
	}
	return "LEVEL:" + educationLevelID
}

// Student adalah data siswa yang dibutuhkan Daily Quiz (jenjang & kelasnya; nil = belum diisi).
type Student struct {
	EducationLevelID *string
	GradeID          *string
}

// GradeOption adalah 1 kelas yang boleh dipilih siswa.
type GradeOption struct {
	ID        string
	Code      *string
	Name      string
	ParentID  string // jenjang
	IsDefault bool   // kelas siswa sendiri
}

// MasterDataItem adalah item master data yang dicek saat generate (letak soal).
type MasterDataItem struct {
	ID       string
	Type     string
	Name     string
	ParentID *string
	GradeID  *string
	Status   string
}

// QuestionFilter adalah syarat soal yang diambil acak untuk Daily Quiz.
type QuestionFilter struct {
	GradeID, SubjectFieldID          string
	ChapterID, SubchapterID, TopicID *string
	Difficulty                       string
	Limit                            int
}

// DailyQuiz adalah 1 baris daily_quizzes + ID soal (urut = nomor soal).
type DailyQuiz struct {
	ID        string
	StudentID string

	EducationLevelID, GradeID, SubjectID, SubjectFieldID string
	ChapterID, SubchapterID, TopicID                     *string

	Difficulty     string
	TotalQuestions int
	QuestionIDs    []string
	CreatedAt      time.Time
}

// StudentQuestion adalah 1 soal untuk dikerjakan siswa: TANPA kunci jawaban & pembahasan.
// Gambar dari Image Library sudah diubah menjadi link (ImageURL).
type StudentQuestion struct {
	ID, Type, QuestionText string
	Formula                *string
	ImageURL               *string
	Table                  *questionmodel.Table
	DisplayOrder           []string
	Options                []Option // pilihan jawaban (single/multiple choice, true/false); nil untuk isian
	InputType              *string  // isian: "number" / "text"; nil untuk tipe lain
}

// Option adalah 1 pilihan jawaban yang ditampilkan ke siswa (tanpa tanda benar/salah).
type Option struct {
	Key      string
	Content  string
	ImageURL *string
}

// DailyQuizDetail adalah Daily Quiz lengkap dengan nama letak & soal-soalnya (untuk siswa).
type DailyQuizDetail struct {
	DailyQuiz
	EducationLevelName, GradeName, SubjectName, SubjectFieldName string
	ChapterName, SubchapterName, TopicName                       *string
	Questions                                                    []StudentQuestion
}

// FieldError adalah 1 kesalahan field request, contoh "placement.grade_id".
type FieldError struct {
	Field   string
	Message string
}

// ValidationError berisi semua kesalahan request generate (dibalas 422).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi daily quiz gagal: " + strings.Join(parts, "; ")
}

// ErrStudentGradeMissing: kelas/jenjang siswa belum diisi admin, jadi pilihan kelas tidak bisa dibuat.
var ErrStudentGradeMissing = errors.New("kelas siswa belum diisi")

// StateError dikembalikan kalau aksi tidak sesuai tahap Daily Quiz (contoh submit sebelum dibuka,
// atau submit 2x). Message = kalimat untuk siswa (dibalas 409).
type StateError struct {
	Message string
}

func (e *StateError) Error() string {
	return e.Message
}

// Submission adalah tahap & hasil Daily Quiz (opened_at, submitted_at, nilai, jawaban).
type Submission struct {
	OpenedAt        *time.Time
	SubmittedAt     *time.Time
	Score           float64
	CorrectCount    int
	Answers         []scoring.Answer
	DurationSeconds int // lama mengerjakan total (submit - dibuka)
	Points          int // total point leaderboard
}

// DailyQuizReview adalah hasil & review Daily Quiz setelah submit.
type DailyQuizReview struct {
	DailyQuizDetail
	Submission
	Items []scoring.ReviewItem
}

// ErrDailyQuizNotFound: Daily Quiz tidak ada atau bukan milik siswa ini.
var ErrDailyQuizNotFound = errors.New("daily quiz tidak ditemukan")

// InsufficientCoinError: jatah gratis hari ini habis dan coin siswa tidak cukup (dibalas 402).
type InsufficientCoinError struct {
	Need, Balance int
}

func (e *InsufficientCoinError) Error() string {
	return fmt.Sprintf("Coin tidak cukup: jatah gratis hari ini sudah habis, butuh %d coin, saldo %d coin", e.Need, e.Balance)
}

// OpenResult adalah hasil transaksi buka Daily Quiz.
type OpenResult struct {
	OpenedAt      time.Time
	CoinCharged   int  // coin yang dipotong (0 = gratis)
	CoinBalance   int  // saldo coin siswa setelah dibuka
	OpenedToday   int  // jumlah Daily Quiz yang sudah dibuka hari ini (WIB), termasuk yang ini
	AlreadyOpened bool // true = sudah pernah dibuka sebelumnya (tidak dihitung & tidak dipotong lagi)
}

// OpenedDailyQuiz adalah Daily Quiz yang dibuka siswa: info + pemakaian jatah/coin + soal
// lengkap dengan kunci jawaban (tanpa pembahasan).
type OpenedDailyQuiz struct {
	DailyQuizDetail
	OpenResult
	FreePerDay int
	Content    []questionmodel.AttemptQuestion
}
