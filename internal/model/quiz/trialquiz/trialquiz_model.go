// Package trialquiz (model) berisi struct data untuk fitur Quiz Trial publik
// (tombol "Coba quiz gratis", BRD 1.3 & FR-AUTH-012).
//
// Di package lain, import dengan alias trialquizmodel:
//
//	trialquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/trialquiz"
package trialquiz

import (
	"encoding/json"
	"errors"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// MaxQuestionsPerGroup = jumlah soal maksimal 1 golongan (1 quiz trial = 10 soal).
const MaxQuestionsPerGroup = 10

// Golongan quiz trial. Nilainya sama persis dengan kolom grade_group.
const (
	GroupPraSD = "pra_sd" // Pra SD: Nol Kecil, Nol Besar
	GroupSD13  = "sd_1_3" // SD Kelas 1-3
	GroupSD46  = "sd_4_6" // SD Kelas 4-6
	GroupSMP   = "smp"    // SMP Kelas 7-9
	GroupSMA   = "sma"    // SMA Kelas 10-12
)

// Groups = semua golongan, urut tampil.
var Groups = []string{GroupPraSD, GroupSD13, GroupSD46, GroupSMP, GroupSMA}

// GroupNames = nama tampilan golongan.
var GroupNames = map[string]string{
	GroupPraSD: "Pra SD",
	GroupSD13:  "SD Kelas 1-3",
	GroupSD46:  "SD Kelas 4-6",
	GroupSMP:   "SMP",
	GroupSMA:   "SMA",
}

// groupByGradeCode = golongan menurut kode kelas bawaan (master data).
var groupByGradeCode = map[string]string{
	"NOL_KECIL": GroupPraSD, "NOL_BESAR": GroupPraSD,
	"GRADE_1": GroupSD13, "GRADE_2": GroupSD13, "GRADE_3": GroupSD13,
	"GRADE_4": GroupSD46, "GRADE_5": GroupSD46, "GRADE_6": GroupSD46,
	"GRADE_7": GroupSMP, "GRADE_8": GroupSMP, "GRADE_9": GroupSMP,
	"GRADE_10": GroupSMA, "GRADE_11": GroupSMA, "GRADE_12": GroupSMA,
}

// groupByLevelCode = golongan menurut kode jenjang, untuk kelas tambahan (bukan bawaan) yang
// kodenya tidak dikenal. SD tidak ada di sini: SD dibagi 2 golongan, jadi wajib kelas bawaan.
var groupByLevelCode = map[string]string{
	"PRA_SD": GroupPraSD,
	"SMP":    GroupSMP,
	"SMA":    GroupSMA,
}

// GroupForGrade mengembalikan golongan quiz trial sebuah kelas. "" = kelas ini tidak punya golongan.
func GroupForGrade(levelCode, gradeCode *string) string {
	if gradeCode != nil {
		if g, ok := groupByGradeCode[*gradeCode]; ok {
			return g
		}
	}
	if levelCode != nil {
		return groupByLevelCode[*levelCode]
	}
	return ""
}

// Question adalah 1 soal trial (tabel trial_quiz_questions). Isi soal sama dengan bank soal.
type Question struct {
	ID          string
	Code        string // otomatis, contoh TQ-3
	GradeGroup  string
	SubjectID   string
	SubjectName string
	SortOrder   int

	Type         string
	QuestionText string
	Formula      *string
	ImageID      *string // gambar Image Library
	ImageURL     *string // ATAU link gambar luar
	Table        *questionmodel.Table
	Explanation  string
	DisplayOrder []string
	Difficulty   string
	Answer       json.RawMessage

	CreatedBy     string
	CreatedByName *string
	UpdatedBy     *string
	UpdatedByName *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PublicQuestion adalah 1 soal untuk halaman trial publik: isi soal + kunci jawaban + pembahasan
// (dinilai & ditampilkan FE). Gambar Image Library sudah berupa link.
type PublicQuestion struct {
	ID, Code, Type          string
	SubjectID, SubjectName  string
	QuestionText            string
	Formula                 *string
	ImageURL                *string
	Table                   *questionmodel.Table
	DisplayOrder            []string
	Difficulty, Explanation string
	Answer                  json.RawMessage // opsi + kunci (bentuk sama dengan bank soal)
}

// GradeOption adalah 1 kelas yang bisa dipilih di form trial, beserta jenjang & golongannya.
type GradeOption struct {
	LevelID, LevelName string
	LevelCode          *string
	GradeID, GradeName string
	GradeCode          *string
	Group              string // diisi service (GroupForGrade)
}

// Attempt adalah 1 hasil trial yang disimpan.
type Attempt struct {
	ID               string
	Name             string
	WhatsApp         string // sudah dirapikan: 62xxxxxxxxxx
	EducationLevelID string
	GradeID          string
	GradeGroup       string
	Result           scoring.Result
	DurationSeconds  *int
	SubmittedAt      time.Time
}

// AttemptFilter adalah filter report trial. Field kosong = tidak difilter.
type AttemptFilter struct {
	From, To   *time.Time // rentang submitted_at [From, To)
	GradeGroup string
	Search     string // nama / no WhatsApp
	Page       int
	PerPage    int
}

// AttemptListItem adalah 1 baris report peserta trial.
type AttemptListItem struct {
	ID, Name           string
	WhatsApp           string `gorm:"column:whatsapp"` // tanpa tag, GORM mencari kolom "whats_app"
	EducationLevelID   string
	EducationLevelName string
	GradeID, GradeName string
	GradeGroup         string
	Score              float64
	CorrectCount       int
	TotalQuestions     int
	DurationSeconds    *int
	SubmittedAt        time.Time
}

// GroupStat = jumlah peserta & rata-rata nilai 1 golongan.
type GroupStat struct {
	GradeGroup   string
	Participants int
	AverageScore float64
}

// WrongQuestion = 1 soal yang paling sering dijawab salah.
type WrongQuestion struct {
	QuestionID, Code, QuestionText string
	SubjectName                    string
	GradeGroup                     string
	Answered, Wrong                int
}

// Summary adalah ringkasan report trial.
type Summary struct {
	Participants int
	AverageScore float64
	ByGroup      []GroupStat
	MostWrong    []WrongQuestion
}

var (
	ErrQuestionNotFound = errors.New("soal quiz trial tidak ditemukan")
	// ErrNoQuestions: golongan kelas yang dipilih belum punya soal trial.
	ErrNoQuestions = errors.New("quiz trial untuk kelas ini belum tersedia")
	// ErrGroupFull: golongan sudah berisi MaxQuestionsPerGroup soal.
	ErrGroupFull = errors.New("soal quiz trial golongan ini sudah penuh")
)
