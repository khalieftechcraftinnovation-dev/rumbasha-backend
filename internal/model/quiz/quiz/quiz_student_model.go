package quiz

import "time"

// Halaman Uji Kom & Tryout siswa (BRD FR-TRY-013, 016, 017, 020, 021).

// Status pengerjaan siswa di bagian "Akan datang" (status jadwal memakai ScheduleUpcoming / ScheduleOngoing).
const (
	MyStatusNotStarted = "not_started" // belum pernah dibuka
	MyStatusInProgress = "in_progress" // sudah dibuka, belum dikumpulkan
)

// StudentQuizFilter adalah filter halaman Uji Kom & Tryout siswa.
type StudentQuizFilter struct {
	Types         []string // tab: Semua = uji_kom + tryout
	Search        string   // nama quiz
	SubjectID     string
	Page, PerPage int
}

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// UpcomingQuiz = 1 quiz di bagian "Akan datang" (publish, kelas siswa, event belum selesai, belum
// dikumpulkan siswa).
type UpcomingQuiz struct {
	ID, Name, Type   string
	Subject          Ref
	SubjectField     *Ref
	Chapter          *Ref
	TotalQuestions   int
	DurationMinutes  int
	CoinCost         int
	StartAt, EndAt   time.Time
	AttemptStartedAt *time.Time // sudah dibuka (belum dikumpulkan); nil = belum pernah dibuka
}

// DoneQuiz = 1 quiz di bagian "Sudah dikerjakan" (pengerjaan siswa yang sudah dikumpulkan).
type DoneQuiz struct {
	QuizID, Name, Type string
	Subject            Ref
	StartAt, EndAt     time.Time
	TotalQuestions     int
	SubmittedAt        time.Time
	Score              float64
	CorrectCount       int
	Points             int
	DurationSeconds    int
	Rank, RankTotal    int // peringkat di quiz itu (aturan leaderboard), contoh #1 dari 30
}

// StudentQuizStats = statistik halaman Uji Kom & Tryout siswa.
type StudentQuizStats struct {
	Sessions     int      // quiz yang sudah dikumpulkan
	AverageScore *float64 // nil = belum ada
	TotalPoints  int      // point Uji Kom + Tryout
	LastTryout   *DoneQuiz
}
