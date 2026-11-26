// Package quiz (model) berisi struct data untuk fitur quiz (Uji Kom & Tryout).
//
// Di package lain, import dengan alias quizmodel:
//
//	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
package quiz

import (
	"errors"
	"strings"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
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

// Status jadwal event, dihitung dari start_at & end_at terhadap waktu sekarang (tidak disimpan).
const (
	ScheduleUpcoming = "upcoming" // akan dimulai: sekarang < start_at
	ScheduleOngoing  = "ongoing"  // sedang berjalan: start_at <= sekarang < end_at
	ScheduleFinished = "finished" // sudah lewat: sekarang >= end_at
)

// ScheduleStatus menghitung status jadwal event pada waktu now.
func ScheduleStatus(startAt, endAt, now time.Time) string {
	switch {
	case now.Before(startAt):
		return ScheduleUpcoming
	case now.Before(endAt):
		return ScheduleOngoing
	default:
		return ScheduleFinished
	}
}

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

	StartAt         time.Time // mulai event
	EndAt           time.Time // selesai event (boleh beda hari)
	DurationMinutes int       // lama mengerjakan 1 kali, di dalam jendela StartAt..EndAt
	CoinCost        int       // 0 = gratis
	TotalQuestions  int       // selalu = len(QuestionIDs)
	Status          string

	AccessCode          *string    // 6 karakter A-Z/0-9, wajib untuk Uji Kom & Tryout
	AccessCodeUpdatedAt *time.Time // terakhir dibuat/diganti (scheduler per jam)

	Points              Points     // point 1 soal benar per level (diatur admin)
	Prizes              Prizes     // hadiah coin juara 1-3 leaderboard
	PrizesDistributedAt *time.Time // kapan hadiah dibagikan scheduler; nil = belum

	QuestionIDs []string // urutan = nomor soal di quiz (sort_order mulai 1)

	CreatedBy string
	UpdatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Points adalah point leaderboard 1 soal yang dijawab BENAR, per level soal. Diatur admin per quiz;
// default = scoring.PointsByDifficulty (10 / 15 / 20). Salah / tidak dijawab tetap 0.
type Points struct {
	Easy, Medium, Hard int
}

// DefaultPoints = point bawaan kalau admin tidak mengisi (sama dengan Daily Quiz).
var DefaultPoints = Points{
	Easy:   scoring.PointsByDifficulty[questionmodel.DifficultyEasy],
	Medium: scoring.PointsByDifficulty[questionmodel.DifficultyMedium],
	Hard:   scoring.PointsByDifficulty[questionmodel.DifficultyHard],
}

// Table mengubah Points menjadi map level → point (dipakai scoring.Grade).
func (p Points) Table() map[string]int {
	return map[string]int{
		questionmodel.DifficultyEasy:   p.Easy,
		questionmodel.DifficultyMedium: p.Medium,
		questionmodel.DifficultyHard:   p.Hard,
	}
}

// Prizes adalah hadiah coin juara 1, 2, 3 leaderboard quiz. 0 = tanpa hadiah.
type Prizes struct {
	Rank1, Rank2, Rank3 int
}

// ForRank = hadiah coin untuk peringkat rank (1-3); peringkat lain 0.
func (p Prizes) ForRank(rank int) int {
	switch rank {
	case 1:
		return p.Rank1
	case 2:
		return p.Rank2
	case 3:
		return p.Rank3
	}
	return 0
}

// PrizeAward adalah 1 siswa penerima hadiah juara (tabel quiz_prize_awards).
type PrizeAward struct {
	Rank            int
	StudentID       string
	StudentName     *string
	Points          int
	DurationSeconds int
	CoinAmount      int
	CreatedAt       time.Time
}

// PrizeRunResult adalah hasil 1 kali scheduler membagikan hadiah.
type PrizeRunResult struct {
	Quizzes int // quiz yang selesai diproses (ditandai sudah dibagikan)
	Awards  int // jumlah siswa yang menerima coin
	Coins   int // total coin yang dibagikan
	Failed  int // quiz yang gagal (dicoba lagi di jadwal berikutnya)
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

	Questions   []QuizQuestion
	PrizeAwards []PrizeAward // penerima hadiah (terisi di detail admin setelah dibagikan)
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

// ErrScheduleConflict dikembalikan repository kalau database menolak jadwal karena beririsan
// dengan quiz lain bermapel & berkelas sama (exclusion constraint quizzes_no_overlap).
// Biasanya sudah dicegah service; ini terjadi kalau 2 request masuk bersamaan.
var ErrScheduleConflict = errors.New("jadwal quiz beririsan dengan quiz lain")

// ErrQuizStarted dikembalikan saat mengubah quiz yang event-nya sudah dimulai.
var ErrQuizStarted = errors.New("quiz sudah dimulai, tidak bisa diubah")

// ScheduleConflict adalah quiz lain yang jadwalnya beririsan (untuk pesan error).
type ScheduleConflict struct {
	ID, Name       string
	StartAt, EndAt time.Time
}

// ListFilter adalah filter list quiz. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Search                               string // nama quiz
	Type                                 string
	Status                               string // draft / published
	Schedule                             string // upcoming / ongoing / finished
	EducationLevelID, GradeID, SubjectID string
	Page, PerPage                        int
}

// ScheduleCounts adalah jumlah quiz per status jadwal (angka di tab).
type ScheduleCounts struct {
	Upcoming, Ongoing, Finished int
}

// Attempt adalah catatan siswa membuka quiz Uji Kom / Tryout (tabel quiz_attempts).
type Attempt struct {
	ID        string
	StartedAt time.Time

	// Terisi setelah siswa submit jawaban (nil = belum submit).
	SubmittedAt     *time.Time
	Score           float64
	CorrectCount    int
	Answers         []scoring.Answer
	DurationSeconds int // lama mengerjakan total (submit - mulai)
	Points          int // total point leaderboard
}

// QuizReview adalah hasil & review 1 quiz setelah siswa submit: nilai + setiap soal (urutan
// sama dengan saat dikerjakan) dengan kunci jawaban, pembahasan, dan jawaban siswa.
type QuizReview struct {
	QuizDetail
	Attempt
	Items []scoring.ReviewItem
}

// OpenedQuiz adalah quiz yang dibuka siswa: info quiz, waktu mulai mengerjakan & batasnya,
// dan soal lengkap dengan kunci jawaban (tanpa pembahasan).
type OpenedQuiz struct {
	QuizDetail
	StartedAt  time.Time // pertama kali siswa membuka quiz
	DeadlineAt time.Time // batas mengerjakan = min(StartedAt + durasi, EndAt)
	Content    []questionmodel.AttemptQuestion
}

// ClosedError dikembalikan saat quiz dibuka di luar jadwal event (belum mulai / sudah selesai).
// Message berisi kalimat untuk siswa, contoh "Quiz belum dimulai. Mulai 01-12-2026 08:00 WIB".
type ClosedError struct {
	Message string
}

func (e *ClosedError) Error() string {
	return e.Message
}
