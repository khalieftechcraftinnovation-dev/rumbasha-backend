// Package leaderboard (model) berisi struct data untuk fitur leaderboard (peringkat siswa).
//
// Di package lain, import dengan alias leaderboardmodel:
//
//	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/leaderboard"
package leaderboard

import (
	"errors"
	"time"
)

// Entry adalah 1 baris leaderboard (1 siswa).
// Urutan: point tertinggi; point sama → total waktu pengerjaan paling singkat. Point & waktu sama
// persis → peringkat sama (RANK).
type Entry struct {
	Rank            int
	StudentID       string
	Name            string
	StudentCode     *string
	SchoolOrigin    *string
	EducationLevel  *Ref // jenjang siswa (akun siswa)
	Grade           *Ref // kelas siswa (akun siswa)
	Points          int  // total point
	DurationSeconds int  // total lama mengerjakan (detik)
	QuizCount       int  // Daily Quiz: jumlah Daily Quiz yang dikerjakan bulan itu
	Score           *float64
}

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// DailyFilter adalah filter leaderboard Daily Quiz: 1 bulan (WIB) + jenjang / kelas siswa.
type DailyFilter struct {
	From, To                  time.Time // rentang submit [From, To)
	EducationLevelID, GradeID string    // "" = tidak difilter
	Page, PerPage             int
}

// Board adalah 1 halaman leaderboard + total peserta + posisi siswa yang melihat (Me, nil kalau
// bukan siswa / belum ikut).
type Board struct {
	Entries []Entry
	Total   int
	Me      *Entry
	Prize   *DailyPrize // Daily Quiz: hadiah juara bulanan (nil untuk leaderboard quiz)
}

// DailyPrize adalah hadiah coin juara 1-3 bulanan Daily Quiz (per kelas) untuk 1 bulan: snapshot
// saat dibagikan, atau pengaturan saat ini kalau belum dibagikan.
type DailyPrize struct {
	PrizeRank1, PrizeRank2, PrizeRank3 int
	DistributedAt                      *time.Time // nil = belum dibagikan
}

// ForRank = hadiah coin untuk peringkat rank (1-3); peringkat lain 0.
func (p DailyPrize) ForRank(rank int) int {
	switch rank {
	case 1:
		return p.PrizeRank1
	case 2:
		return p.PrizeRank2
	case 3:
		return p.PrizeRank3
	}
	return 0
}

// Quiz adalah info quiz Uji Kom / Tryout untuk leaderboard per quiz.
type Quiz struct {
	ID, Name, Type, Status, GradeID    string
	StartAt, EndAt                     time.Time
	PrizeRank1, PrizeRank2, PrizeRank3 int        // hadiah coin juara 1-3 (0 = tanpa hadiah)
	PrizesDistributedAt                *time.Time // nil = belum dibagikan
}

// FieldError & ValidationError: kesalahan query string (dibalas 422).
type FieldError struct {
	Field, Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validasi leaderboard gagal"
	}
	return "validasi leaderboard gagal: " + e.Fields[0].Field + ": " + e.Fields[0].Message
}

// ErrQuizNotFound: quiz tidak ada / bukan Uji Kom-Tryout / tidak boleh dilihat siswa ini.
var ErrQuizNotFound = errors.New("quiz tidak ditemukan")

// Papan peringkat (kolom leaderboard_entries.board).
const (
	BoardDailyQuiz = "daily_quiz"
	BoardUjiKom    = "uji_kom"
	BoardTryout    = "tryout"
)

// Asal point (kolom leaderboard_point_logs.source_type).
const (
	SourceDailyQuiz   = "daily_quiz"   // daily_quizzes.id
	SourceQuizAttempt = "quiz_attempt" // quiz_attempts.id
)

// PeriodLayout adalah format period Daily Quiz: bulan WIB, contoh "2026-11".
const PeriodLayout = "2006-01"

// PointRecord adalah point dari 1 pengerjaan quiz yang dicatat ke leaderboard saat submit.
type PointRecord struct {
	Board           string  // daily_quiz / uji_kom / tryout
	Period          *string // Daily Quiz: bulan WIB "YYYY-MM"
	QuizID          *string // Uji Kom / Tryout
	StudentID       string
	Points          int
	DurationSeconds int
	Score           *float64 // Uji Kom / Tryout: nilai (info)
	SubmittedAt     time.Time
	SourceType      string // daily_quiz / quiz_attempt
	SourceID        string
}

// PrizeForRank = hadiah coin quiz untuk peringkat rank (1-3); peringkat lain 0.
func (q Quiz) PrizeForRank(rank int) int {
	switch rank {
	case 1:
		return q.PrizeRank1
	case 2:
		return q.PrizeRank2
	case 3:
		return q.PrizeRank3
	}
	return 0
}
