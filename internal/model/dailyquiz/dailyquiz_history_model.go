package dailyquiz

import "time"

// Halaman Daily Quiz siswa: riwayat & statistik (BRD FR-QUIZ-019).

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// HistoryFilter adalah filter riwayat Daily Quiz siswa. From/To nil = semua waktu.
type HistoryFilter struct {
	From, To      *time.Time // tanggal submit [From, To), WIB
	SubjectID     string
	Difficulty    string
	Page, PerPage int
}

// HistoryItem = 1 Daily Quiz yang sudah dikumpulkan siswa.
type HistoryItem struct {
	ID                                   string
	Subject                              Ref
	SubjectField                         Ref
	Chapter, Subchapter, Topic           *Ref
	Grade                                Ref
	Difficulty                           string
	TotalQuestions, CorrectCount         int
	Score                                float64
	Points, DurationSeconds, CoinCharged int
	SubmittedAt                          time.Time
}

// Summary = jumlah, rata-rata nilai, dan point Daily Quiz dalam 1 rentang.
type Summary struct {
	Quizzes      int
	AverageScore *float64 // nil = belum ada
	Points       int
}

// HistoryStats = statistik halaman Daily Quiz siswa.
type HistoryStats struct {
	Month              string // bulan WIB, contoh 2026-11
	ThisMonth, AllTime Summary
	Rank               *int // peringkat Daily Quiz bulan itu di antara siswa sekelas; nil = belum ikut
	RankTotal          int
}
