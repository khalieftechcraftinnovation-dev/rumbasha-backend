package quizreport

import "time"

// Tab report (jenis quiz) untuk kartu ringkasan & grafik.
const (
	TypeDailyQuiz = "daily_quiz"
	TypeUjiKom    = "uji_kom"
	TypeTryout    = "tryout"
)

// TrendWeeks = jumlah minggu grafik tren nilai (12 minggu terakhir, FR-RPT-003).
const TrendWeeks = 12

// ScoreBuckets = rentang nilai grafik sebaran (inklusif): 0-20, 21-40, 41-60, 61-80, 81-100.
// Nilai berkoma masuk ke rentang berikutnya kalau melewati batas atas (contoh 20,5 → 21-40).
var ScoreBuckets = []struct{ Min, Max int }{{0, 20}, {21, 40}, {41, 60}, {61, 80}, {81, 100}}

// Settings adalah pengaturan report quiz (tabel quiz_report_settings, 1 baris).
type Settings struct {
	WrongThresholdPercent int
	DefaultRangeDays      int
	UpdatedBy             *Ref
	UpdatedAt             time.Time
}

// OverviewFilter adalah filter kartu ringkasan & grafik. Field kosong = tidak difilter.
type OverviewFilter struct {
	Type                                 string    // daily_quiz / uji_kom / tryout (wajib)
	EducationLevelID, GradeID, SubjectID string    // letak quiz (Daily Quiz: pilihan siswa)
	From, To                             time.Time // tanggal submit di [From, To), WIB
	WrongThresholdPercent                int
}

// Totals = angka kartu yang dihitung dari semua pengerjaan di periode (1 query).
type Totals struct {
	Sessions        int      // pengerjaan yang sudah submit
	DistinctQuizzes int      // quiz berbeda (Uji Kom / Tryout); Daily Quiz = 0
	Students        int      // siswa berbeda
	Average         *float64 // nil kalau belum ada pengerjaan
	TimedOut        int      // dikumpulkan setelah waktu habis (Uji Kom / Tryout)
	Buckets         [5]int   // jumlah pengerjaan per rentang ScoreBuckets
}

// LevelAverage = rata-rata nilai 1 jenjang.
type LevelAverage struct {
	Level    Ref
	Sessions int
	Average  float64
}

// WrongQuestion = 1 soal yang sering salah di periode.
type WrongQuestion struct {
	QuestionID   string
	Code         string
	QuestionText string
	Chapter      *Ref
	Answered     int // berapa kali soal ini dikerjakan
	Wrong        int // salah + tidak dijawab
}

// WrongSummary = kartu "Soal sering salah".
type WrongSummary struct {
	TotalQuestions int             // soal berbeda yang dikerjakan di periode
	Questions      []WrongQuestion // soal yang salah > ambang, urut persentase salah tertinggi
}

// WeekLevel = jumlah & total nilai pengerjaan 1 jenjang di 1 minggu (bahan grafik tren).
type WeekLevel struct {
	WeekStart time.Time // Senin (WIB)
	Level     Ref
	LevelSort int
	Sessions  int
	ScoreSum  float64
}

// Overview adalah seluruh data kartu ringkasan & grafik 1 tab report.
type Overview struct {
	Filter     OverviewFilter
	Totals     Totals
	ByLevel    []LevelAverage
	Wrong      WrongSummary
	TrendFrom  time.Time // Senin minggu pertama grafik tren
	TrendWeeks []WeekLevel
}
