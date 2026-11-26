package dailyquiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
)

// HistoryQuery adalah query string GET /api/v1/student/daily-quizzes (riwayat).
//
//	?month=2026-11&subject_id=...&difficulty=easy&page=1&per_page=20
type HistoryQuery struct {
	Month      string `form:"month" binding:"omitempty,datetime=2006-01"` // kosong = semua waktu
	SubjectID  string `form:"subject_id" binding:"omitempty,uuid"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=easy medium hard"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PerPage    int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// HistoryStatsQuery adalah query string GET /api/v1/student/daily-quizzes/stats.
type HistoryStatsQuery struct {
	Month string `form:"month" binding:"omitempty,datetime=2006-01"` // kosong = bulan ini (WIB)
}

// DifficultyLabels = nama level Daily Quiz untuk ditampilkan.
var DifficultyLabels = map[string]string{"easy": "Mudah", "medium": "Sedang", "hard": "Sulit"}

// HistoryItemResponse = 1 Daily Quiz yang sudah dikumpulkan (kolom Riwayat: materi, level, waktu,
// nilai, point). Review lengkap: GET /daily-quizzes/:id/review.
type HistoryItemResponse struct {
	ID              string       `json:"id"`
	Subject         RefResponse  `json:"subject"`
	SubjectField    RefResponse  `json:"subject_field"`
	Chapter         *RefResponse `json:"chapter"`
	Subchapter      *RefResponse `json:"subchapter"`
	Topic           *RefResponse `json:"topic"`
	Grade           RefResponse  `json:"grade"`
	Difficulty      string       `json:"difficulty"`       // easy, medium, hard
	DifficultyLabel string       `json:"difficulty_label"` // Mudah, Sedang, Sulit
	TotalQuestions  int          `json:"total_questions"`
	CorrectCount    int          `json:"correct_count"`
	Score           float64      `json:"score"`
	Points          int          `json:"points"`
	DurationSeconds int          `json:"duration_seconds"`
	CoinCharged     int          `json:"coin_charged"` // 0 = jatah gratis
	SubmittedAt     time.Time    `json:"submitted_at"` // WIB
}

func historyRef(r *dailyquizmodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// ToHistoryResponses mengubah riwayat menjadi response. Kosong = [].
func ToHistoryResponses(items []dailyquizmodel.HistoryItem) []HistoryItemResponse {
	result := make([]HistoryItemResponse, 0, len(items))
	for _, d := range items {
		result = append(result, HistoryItemResponse{
			ID: d.ID, Subject: RefResponse{ID: d.Subject.ID, Name: d.Subject.Name},
			SubjectField: RefResponse{ID: d.SubjectField.ID, Name: d.SubjectField.Name},
			Chapter:      historyRef(d.Chapter), Subchapter: historyRef(d.Subchapter), Topic: historyRef(d.Topic),
			Grade:      RefResponse{ID: d.Grade.ID, Name: d.Grade.Name},
			Difficulty: d.Difficulty, DifficultyLabel: DifficultyLabels[d.Difficulty],
			TotalQuestions: d.TotalQuestions, CorrectCount: d.CorrectCount, Score: d.Score, Points: d.Points,
			DurationSeconds: d.DurationSeconds, CoinCharged: d.CoinCharged, SubmittedAt: d.SubmittedAt.In(clock.Jakarta),
		})
	}
	return result
}

// SummaryResponse = jumlah, rata-rata nilai, point Daily Quiz.
type SummaryResponse struct {
	Quizzes      int      `json:"quizzes"`
	AverageScore *float64 `json:"average_score"` // null = belum ada
	Points       int      `json:"points"`
}

// HistoryStatsResponse = statistik halaman Daily Quiz siswa.
type HistoryStatsResponse struct {
	Month     string          `json:"month"`
	ThisMonth SummaryResponse `json:"this_month"` // bulan "month"
	AllTime   SummaryResponse `json:"all_time"`
	// Peringkat Daily Quiz bulan itu di antara siswa sekelas (leaderboard); null = belum ikut.
	Rank *RankResponse `json:"rank"`
}

// RankResponse = peringkat, contoh "#12 dari 47 siswa".
type RankResponse struct {
	Rank  int `json:"rank"`
	Total int `json:"total"`
}

// ToHistoryStatsResponse mengubah statistik menjadi response.
func ToHistoryStatsResponse(s dailyquizmodel.HistoryStats) HistoryStatsResponse {
	r := HistoryStatsResponse{
		Month:     s.Month,
		ThisMonth: SummaryResponse{Quizzes: s.ThisMonth.Quizzes, AverageScore: s.ThisMonth.AverageScore, Points: s.ThisMonth.Points},
		AllTime:   SummaryResponse{Quizzes: s.AllTime.Quizzes, AverageScore: s.AllTime.AverageScore, Points: s.AllTime.Points},
	}
	if s.Rank != nil {
		r.Rank = &RankResponse{Rank: *s.Rank, Total: s.RankTotal}
	}
	return r
}
