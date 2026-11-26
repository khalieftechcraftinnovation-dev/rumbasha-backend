package quiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

// StudentQuizQuery adalah query string halaman Uji Kom & Tryout siswa (upcoming, history, stats).
//
//	?type=tryout&search=matematika&subject_id=...&page=1&per_page=20
type StudentQuizQuery struct {
	Type      string `form:"type" binding:"omitempty,oneof=uji_kom tryout"` // tab; kosong = Semua
	Search    string `form:"search" binding:"omitempty,max=100"`            // nama quiz
	SubjectID string `form:"subject_id" binding:"omitempty,uuid"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PerPage   int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// NameRef adalah pasangan id + nama (nama selalu ada).
type NameRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toNameRef(r *quizmodel.Ref) *NameRef {
	if r == nil {
		return nil
	}
	return &NameRef{ID: r.ID, Name: r.Name}
}

// UpcomingResponse = 1 quiz di bagian "Akan datang".
type UpcomingResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // uji_kom / tryout
	Subject         NameRef   `json:"subject"`
	SubjectField    *NameRef  `json:"subject_field"`
	Chapter         *NameRef  `json:"chapter"`
	TotalQuestions  int       `json:"total_questions"`
	DurationMinutes int       `json:"duration_minutes"`
	CoinCost        int       `json:"coin_cost"`
	StartAt         time.Time `json:"start_at"` // waktu dibuka (WIB)
	EndAt           time.Time `json:"end_at"`
	ScheduleStatus  string    `json:"schedule_status"` // upcoming (terkunci) / ongoing
	Locked          bool      `json:"locked"`          // true sampai start_at
	// Hitung mundur ke start_at (detik, dihitung server saat request); 0 kalau sudah dimulai.
	SecondsUntilStart int64 `json:"seconds_until_start"`
	// not_started = belum dibuka; in_progress = sudah dibuka, belum dikumpulkan (lanjutkan tanpa kode akses).
	MyStatus  string     `json:"my_status"`
	StartedAt *time.Time `json:"started_at"` // kapan siswa membuka (in_progress)
}

// ToUpcomingResponses menyusun bagian "Akan datang" pada waktu now. Kosong = [].
func ToUpcomingResponses(items []quizmodel.UpcomingQuiz, now time.Time) []UpcomingResponse {
	result := make([]UpcomingResponse, 0, len(items))
	for _, q := range items {
		r := UpcomingResponse{
			ID: q.ID, Name: q.Name, Type: q.Type, Subject: NameRef{ID: q.Subject.ID, Name: q.Subject.Name},
			SubjectField: toNameRef(q.SubjectField), Chapter: toNameRef(q.Chapter),
			TotalQuestions: q.TotalQuestions, DurationMinutes: q.DurationMinutes, CoinCost: q.CoinCost,
			StartAt: q.StartAt.In(clock.Jakarta), EndAt: q.EndAt.In(clock.Jakarta),
			ScheduleStatus: quizmodel.ScheduleStatus(q.StartAt, q.EndAt, now),
			MyStatus:       quizmodel.MyStatusNotStarted,
		}
		r.Locked = now.Before(q.StartAt)
		if r.Locked {
			r.SecondsUntilStart = int64(q.StartAt.Sub(now).Seconds())
		}
		if q.AttemptStartedAt != nil {
			t := q.AttemptStartedAt.In(clock.Jakarta)
			r.MyStatus, r.StartedAt = quizmodel.MyStatusInProgress, &t
		}
		result = append(result, r)
	}
	return result
}

// DoneResponse = 1 quiz di bagian "Sudah dikerjakan". Pembahasan: GET /quizzes/:id/review.
type DoneResponse struct {
	QuizID          string    `json:"quiz_id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Subject         NameRef   `json:"subject"`
	StartAt         time.Time `json:"start_at"`     // tanggal event (WIB)
	SubmittedAt     time.Time `json:"submitted_at"` // dikumpulkan (WIB)
	Score           float64   `json:"score"`
	CorrectCount    int       `json:"correct_count"`
	TotalQuestions  int       `json:"total_questions"`
	Points          int       `json:"points"`
	DurationSeconds int       `json:"duration_seconds"`
	Rank            RankInfo  `json:"rank"` // contoh {"rank": 1, "total": 30} = "peringkat #1 dari 30"
}

// RankInfo = peringkat di 1 papan.
type RankInfo struct {
	Rank  int `json:"rank"`
	Total int `json:"total"`
}

// ToDoneResponse mengubah 1 quiz yang sudah dikerjakan menjadi response.
func ToDoneResponse(d quizmodel.DoneQuiz) DoneResponse {
	return DoneResponse{
		QuizID: d.QuizID, Name: d.Name, Type: d.Type, Subject: NameRef{ID: d.Subject.ID, Name: d.Subject.Name},
		StartAt: d.StartAt.In(clock.Jakarta), SubmittedAt: d.SubmittedAt.In(clock.Jakarta),
		Score: d.Score, CorrectCount: d.CorrectCount, TotalQuestions: d.TotalQuestions, Points: d.Points,
		DurationSeconds: d.DurationSeconds, Rank: RankInfo{Rank: d.Rank, Total: d.RankTotal},
	}
}

// ToDoneResponses mengubah daftar quiz yang sudah dikerjakan menjadi response. Kosong = [].
func ToDoneResponses(items []quizmodel.DoneQuiz) []DoneResponse {
	result := make([]DoneResponse, 0, len(items))
	for _, d := range items {
		result = append(result, ToDoneResponse(d))
	}
	return result
}

// StudentStatsResponse = statistik halaman Uji Kom & Tryout siswa (tab ikut query type).
type StudentStatsResponse struct {
	Sessions     int           `json:"sessions"`      // quiz yang sudah dikumpulkan
	AverageScore *float64      `json:"average_score"` // null = belum ada
	TotalPoints  int           `json:"total_points"`  // point Uji Kom + Tryout
	LastTryout   *DoneResponse `json:"last_tryout"`   // Tryout terakhir + peringkatnya; null = belum pernah
}

// ToStudentStatsResponse mengubah statistik menjadi response.
func ToStudentStatsResponse(s quizmodel.StudentQuizStats) StudentStatsResponse {
	r := StudentStatsResponse{Sessions: s.Sessions, AverageScore: s.AverageScore, TotalPoints: s.TotalPoints}
	if s.LastTryout != nil {
		d := ToDoneResponse(*s.LastTryout)
		r.LastTryout = &d
	}
	return r
}
