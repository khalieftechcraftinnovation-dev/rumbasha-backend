// Package leaderboard (dto) berisi bentuk request dan response JSON untuk fitur leaderboard.
//
// Di package lain, import dengan alias leaderboarddto:
//
//	leaderboarddto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/leaderboard"
package leaderboard

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/leaderboard"
)

// DailyQuery adalah query string GET /api/v1/leaderboards/daily-quiz.
//
//	?month=2026-11&education_level_id=...&grade_id=...&page=1&per_page=20
type DailyQuery struct {
	Month            string `form:"month" binding:"omitempty,max=7"` // YYYY-MM (WIB), default bulan ini
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	Page             int    `form:"page" binding:"omitempty,min=1"`
	PerPage          int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// QuizQuery adalah query string GET /api/v1/leaderboards/quizzes/:id (pagination).
type QuizQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// EntryResponse adalah 1 baris leaderboard.
type EntryResponse struct {
	Rank            int          `json:"rank"`
	Student         StudentRef   `json:"student"`
	EducationLevel  *RefResponse `json:"education_level"`
	Grade           *RefResponse `json:"grade"`
	Points          int          `json:"points"`           // total POINT (bukan nilai)
	DurationSeconds int          `json:"duration_seconds"` // total lama mengerjakan (penentu kalau point sama)
	QuizCount       *int         `json:"quiz_count,omitempty"`
	Score           *float64     `json:"score,omitempty"` // Uji Kom / Tryout: nilai 0-100 (info)
	// Uji Kom / Tryout: hadiah coin untuk peringkat ini (juara 1-3, 0 = tidak ada). Sebelum event
	// selesai masih sementara; dibagikan otomatis 23:59 WIB setelah event selesai.
	PrizeCoins *int `json:"prize_coins,omitempty"`
}

// StudentRef adalah data siswa di leaderboard.
type StudentRef struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	StudentCode  *string `json:"student_code"`
	SchoolOrigin *string `json:"school_origin"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DailyBoardResponse adalah data leaderboard Daily Quiz 1 bulan.
type DailyBoardResponse struct {
	Month   string          `json:"month"` // YYYY-MM
	From    time.Time       `json:"from"`  // awal bulan, WIB
	To      time.Time       `json:"to"`    // awal bulan berikutnya, WIB (tidak termasuk)
	Entries []EntryResponse `json:"entries"`
	Me      *EntryResponse  `json:"me"` // posisi siswa yang melihat; null kalau bukan siswa / belum ikut
	// Hadiah coin juara 1-3 bulan ini PER KELAS (pengaturan Daily Quiz; snapshot kalau sudah dibagikan).
	// prize_coins per entry hanya diisi kalau difilter grade_id (peringkat = peringkat di kelas itu).
	PrizeCoins          PrizeCoinsRef `json:"prize_coins"`
	PrizesDistributedAt *time.Time    `json:"prizes_distributed_at"` // WIB; null = belum (hari terakhir bulan 23:59 WIB)
}

// QuizBoardResponse adalah data leaderboard 1 quiz Uji Kom / Tryout.
type QuizBoardResponse struct {
	Quiz    QuizRef         `json:"quiz"`
	Entries []EntryResponse `json:"entries"`
	Me      *EntryResponse  `json:"me"`
}

// QuizRef adalah info quiz di leaderboard.
type QuizRef struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	StartAt             time.Time     `json:"start_at"`
	EndAt               time.Time     `json:"end_at"`
	PrizeCoins          PrizeCoinsRef `json:"prize_coins"`           // hadiah coin juara 1-3
	PrizesDistributedAt *time.Time    `json:"prizes_distributed_at"` // WIB; null = belum dibagikan
}

// PrizeCoinsRef = hadiah coin juara 1-3 (0 = tanpa hadiah).
type PrizeCoinsRef struct {
	Rank1 int `json:"rank_1"`
	Rank2 int `json:"rank_2"`
	Rank3 int `json:"rank_3"`
}

// ToDailyBoardResponse mengubah leaderboard Daily Quiz menjadi response.
func ToDailyBoardResponse(month string, f leaderboardmodel.DailyFilter, b leaderboardmodel.Board) DailyBoardResponse {
	r := DailyBoardResponse{
		Month: month, From: f.From.In(clock.Jakarta), To: f.To.In(clock.Jakarta),
		Entries: toEntries(b.Entries, true), Me: toMe(b.Me, true),
	}
	if b.Prize == nil {
		return r
	}
	p := *b.Prize
	r.PrizeCoins = PrizeCoinsRef{Rank1: p.PrizeRank1, Rank2: p.PrizeRank2, Rank3: p.PrizeRank3}
	if p.DistributedAt != nil {
		t := p.DistributedAt.In(clock.Jakarta)
		r.PrizesDistributedAt = &t
	}
	// Juara dihitung per kelas, jadi hadiah per baris hanya bermakna kalau difilter 1 kelas.
	if f.GradeID != "" {
		for i := range r.Entries {
			coins := p.ForRank(r.Entries[i].Rank)
			r.Entries[i].PrizeCoins = &coins
		}
		if r.Me != nil {
			coins := p.ForRank(r.Me.Rank)
			r.Me.PrizeCoins = &coins
		}
	}
	return r
}

// ToQuizBoardResponse mengubah leaderboard 1 quiz menjadi response.
func ToQuizBoardResponse(q leaderboardmodel.Quiz, b leaderboardmodel.Board) QuizBoardResponse {
	var distributedAt *time.Time
	if q.PrizesDistributedAt != nil {
		t := q.PrizesDistributedAt.In(clock.Jakarta)
		distributedAt = &t
	}
	entries := toEntries(b.Entries, false)
	for i := range entries {
		withPrize(&entries[i], q)
	}
	me := toMe(b.Me, false)
	if me != nil {
		withPrize(me, q)
	}
	return QuizBoardResponse{
		Quiz: QuizRef{
			ID: q.ID, Name: q.Name, Type: q.Type, StartAt: q.StartAt.In(clock.Jakarta), EndAt: q.EndAt.In(clock.Jakarta),
			PrizeCoins:          PrizeCoinsRef{Rank1: q.PrizeRank1, Rank2: q.PrizeRank2, Rank3: q.PrizeRank3},
			PrizesDistributedAt: distributedAt,
		},
		Entries: entries, Me: me,
	}
}

// withPrize mengisi hadiah coin sesuai peringkat entry (0 untuk peringkat di luar 1-3).
func withPrize(e *EntryResponse, q leaderboardmodel.Quiz) {
	coins := q.PrizeForRank(e.Rank)
	e.PrizeCoins = &coins
}

func toEntries(entries []leaderboardmodel.Entry, daily bool) []EntryResponse {
	result := make([]EntryResponse, 0, len(entries))
	for _, e := range entries {
		result = append(result, toEntry(e, daily))
	}
	return result
}

func toMe(e *leaderboardmodel.Entry, daily bool) *EntryResponse {
	if e == nil {
		return nil
	}
	r := toEntry(*e, daily)
	return &r
}

func toEntry(e leaderboardmodel.Entry, daily bool) EntryResponse {
	r := EntryResponse{
		Rank:            e.Rank,
		Student:         StudentRef{ID: e.StudentID, Name: e.Name, StudentCode: e.StudentCode, SchoolOrigin: e.SchoolOrigin},
		EducationLevel:  ref(e.EducationLevel),
		Grade:           ref(e.Grade),
		Points:          e.Points,
		DurationSeconds: e.DurationSeconds,
	}
	if daily {
		count := e.QuizCount
		r.QuizCount = &count
	} else {
		r.Score = e.Score
	}
	return r
}

func ref(r *leaderboardmodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}
