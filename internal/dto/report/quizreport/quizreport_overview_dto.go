package quizreport

import (
	"fmt"
	"sort"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/quizreport"
)

// OverviewQuery adalah query string GET /api/v1/quiz-reports/overview (kartu ringkasan & grafik).
//
//	?type=uji_kom&date_from=2026-09-10&date_to=2026-11-09&education_level_id=...&wrong_threshold=15
type OverviewQuery struct {
	Type             string `form:"type" binding:"required,oneof=daily_quiz uji_kom tryout"` // tab report
	DateFrom         string `form:"date_from" binding:"omitempty,max=10"`                    // tanggal submit >= (WIB)
	DateTo           string `form:"date_to" binding:"omitempty,max=10"`                      // tanggal submit <= (WIB)
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	// Ambang "soal sering salah" (%) untuk request ini saja; kosong = pengaturan report.
	WrongThreshold int `form:"wrong_threshold" binding:"omitempty,min=1,max=100"`
}

// SettingsRequest adalah body PUT /api/v1/quiz-report-settings.
type SettingsRequest struct {
	WrongThresholdPercent int `json:"wrong_threshold_percent" binding:"required,min=1,max=100"`
	DefaultRangeDays      int `json:"default_range_days" binding:"required,min=7,max=366"`
}

// SettingsResponse adalah pengaturan report quiz.
type SettingsResponse struct {
	WrongThresholdPercent int          `json:"wrong_threshold_percent"`
	DefaultRangeDays      int          `json:"default_range_days"`
	UpdatedBy             *RefResponse `json:"updated_by"` // null = belum pernah diubah
	UpdatedAt             time.Time    `json:"updated_at"`
}

// ToSettingsResponse mengubah pengaturan menjadi response.
func ToSettingsResponse(s quizreportmodel.Settings) SettingsResponse {
	r := SettingsResponse{WrongThresholdPercent: s.WrongThresholdPercent, DefaultRangeDays: s.DefaultRangeDays, UpdatedAt: s.UpdatedAt.In(clock.Jakarta)}
	if s.UpdatedBy != nil {
		r.UpdatedBy = &RefResponse{ID: s.UpdatedBy.ID, Name: s.UpdatedBy.Name}
	}
	return r
}

// OverviewResponse adalah kartu ringkasan + grafik 1 tab report.
type OverviewResponse struct {
	Type         string                `json:"type"`
	Period       PeriodResponse        `json:"period"`
	Cards        CardsResponse         `json:"cards"`
	Trend        TrendResponse         `json:"trend"`
	Distribution []ScoreBucketResponse `json:"distribution"`
}

// PeriodResponse = periode laporan (tanggal submit, WIB, inklusif).
type PeriodResponse struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
}

// CardsResponse = 4 kartu ringkasan (FR-RPT-002).
type CardsResponse struct {
	Sessions        SessionsCard `json:"sessions"`
	AverageScore    AverageCard  `json:"average_score"`
	OnTime          *OnTimeCard  `json:"on_time"` // null untuk Daily Quiz (tanpa timer)
	FrequentlyWrong WrongCard    `json:"frequently_wrong"`
}

// SessionsCard = "Sesi dikerjakan".
type SessionsCard struct {
	Total           int  `json:"total"`            // pengerjaan yang sudah dikumpulkan
	DistinctQuizzes *int `json:"distinct_quizzes"` // quiz berbeda; null untuk Daily Quiz
	Students        int  `json:"students"`         // siswa berbeda
}

// AverageCard = "Rata-rata nilai" keseluruhan + per jenjang.
type AverageCard struct {
	Overall *float64               `json:"overall"` // null kalau belum ada pengerjaan
	ByLevel []LevelAverageResponse `json:"by_level"`
}

// LevelAverageResponse = rata-rata 1 jenjang.
type LevelAverageResponse struct {
	EducationLevel RefResponse `json:"education_level"`
	Sessions       int         `json:"sessions"`
	Average        float64     `json:"average"`
}

// OnTimeCard = "Selesai tepat waktu".
type OnTimeCard struct {
	Percent  *float64 `json:"percent"`   // null kalau belum ada pengerjaan
	OnTime   int      `json:"on_time"`   // dikumpulkan sebelum waktu habis
	TimedOut int      `json:"timed_out"` // kehabisan waktu
}

// WrongCard = "Soal sering salah": soal yang salah oleh LEBIH DARI threshold_percent yang mengerjakan.
type WrongCard struct {
	ThresholdPercent int                     `json:"threshold_percent"`
	Count            int                     `json:"count"`           // jumlah soal sering salah
	TotalQuestions   int                     `json:"total_questions"` // soal berbeda yang dikerjakan
	TopChapter       *TopChapterResponse     `json:"top_chapter"`     // bab dengan soal sering salah terbanyak
	Questions        []WrongQuestionResponse `json:"questions"`       // maks 10, salah terbanyak dulu
}

// TopChapterResponse = bab terbanyak, contoh {"name": "Bangun Datar", "count": 4}.
type TopChapterResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// WrongQuestionResponse = 1 soal sering salah.
type WrongQuestionResponse struct {
	ID           string       `json:"id"`
	Code         string       `json:"code"`
	QuestionText string       `json:"question_text"`
	Chapter      *RefResponse `json:"chapter"`
	Answered     int          `json:"answered"`      // berapa kali dikerjakan
	Wrong        int          `json:"wrong"`         // salah + tidak dijawab
	WrongPercent float64      `json:"wrong_percent"` // 0-100
}

// TrendResponse = grafik tren nilai rata-rata per minggu, 12 minggu terakhir s/d date_to (FR-RPT-003).
type TrendResponse struct {
	Weeks  []TrendWeekResponse   `json:"weeks"`  // semua jenjang
	Series []TrendSeriesResponse `json:"series"` // 1 garis per jenjang
}

// TrendWeekResponse = 1 minggu (Senin-Minggu WIB) untuk semua jenjang.
type TrendWeekResponse struct {
	WeekStart string   `json:"week_start"` // Senin "2026-11-02"
	WeekEnd   string   `json:"week_end"`   // Minggu "2026-11-08"
	Sessions  int      `json:"sessions"`
	Average   *float64 `json:"average"` // null = tidak ada pengerjaan minggu itu
}

// TrendSeriesResponse = garis 1 jenjang; points urut sama dengan weeks.
type TrendSeriesResponse struct {
	EducationLevel RefResponse          `json:"education_level"`
	Points         []TrendPointResponse `json:"points"`
}

// TrendPointResponse = 1 titik garis jenjang.
type TrendPointResponse struct {
	WeekStart string   `json:"week_start"`
	Sessions  int      `json:"sessions"`
	Average   *float64 `json:"average"`
}

// ScoreBucketResponse = 1 batang grafik sebaran nilai.
type ScoreBucketResponse struct {
	Label       string `json:"label"` // "0-20"
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Count       int    `json:"count"`       // jumlah pengerjaan di rentang ini
	Highlighted bool   `json:"highlighted"` // rentang terbanyak (yang pertama kalau seri)
}

// maxWrongQuestions = jumlah soal sering salah yang ikut ditampilkan di kartu.
const maxWrongQuestions = 10

// round2 membulatkan ke 2 angka di belakang koma.
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// ToOverviewResponse menyusun kartu ringkasan & grafik dari hasil hitung.
func ToOverviewResponse(o quizreportmodel.Overview) OverviewResponse {
	f, t := o.Filter, o.Totals
	r := OverviewResponse{
		Type: f.Type,
		Period: PeriodResponse{
			DateFrom: f.From.In(clock.Jakarta).Format(time.DateOnly),
			DateTo:   f.To.AddDate(0, 0, -1).In(clock.Jakarta).Format(time.DateOnly),
		},
	}

	// Kartu 1: sesi dikerjakan.
	r.Cards.Sessions = SessionsCard{Total: t.Sessions, Students: t.Students}
	if f.Type != quizreportmodel.TypeDailyQuiz {
		n := t.DistinctQuizzes
		r.Cards.Sessions.DistinctQuizzes = &n
	}

	// Kartu 2: rata-rata nilai per jenjang.
	r.Cards.AverageScore = AverageCard{Overall: t.Average, ByLevel: make([]LevelAverageResponse, 0, len(o.ByLevel))}
	for _, l := range o.ByLevel {
		r.Cards.AverageScore.ByLevel = append(r.Cards.AverageScore.ByLevel, LevelAverageResponse{
			EducationLevel: RefResponse{ID: l.Level.ID, Name: l.Level.Name}, Sessions: l.Sessions, Average: l.Average,
		})
	}

	// Kartu 3: selesai tepat waktu (Daily Quiz tanpa timer → null).
	if f.Type != quizreportmodel.TypeDailyQuiz {
		card := OnTimeCard{OnTime: t.Sessions - t.TimedOut, TimedOut: t.TimedOut}
		if t.Sessions > 0 {
			p := round2(float64(card.OnTime) * 100 / float64(t.Sessions))
			card.Percent = &p
		}
		r.Cards.OnTime = &card
	}

	// Kartu 4: soal sering salah + bab terbanyak.
	r.Cards.FrequentlyWrong = toWrongCard(f.WrongThresholdPercent, o.Wrong)

	r.Trend = toTrend(o.TrendFrom, o.TrendWeeks)

	// Grafik sebaran nilai: rentang terbanyak disorot (yang pertama kalau seri, tidak ada kalau kosong).
	best := -1
	for i, n := range t.Buckets {
		if n > 0 && (best < 0 || n > t.Buckets[best]) {
			best = i
		}
	}
	for i, b := range quizreportmodel.ScoreBuckets {
		r.Distribution = append(r.Distribution, ScoreBucketResponse{
			Label: fmt.Sprintf("%d-%d", b.Min, b.Max), Min: b.Min, Max: b.Max, Count: t.Buckets[i], Highlighted: i == best,
		})
	}
	return r
}

// toWrongCard menyusun kartu soal sering salah: jumlah, bab terbanyak, dan maks 10 soal teratas.
func toWrongCard(threshold int, w quizreportmodel.WrongSummary) WrongCard {
	card := WrongCard{ThresholdPercent: threshold, Count: len(w.Questions), TotalQuestions: w.TotalQuestions, Questions: []WrongQuestionResponse{}}

	// Bab terbanyak: hitung per bab; seri → bab yang soalnya muncul lebih dulu (salah lebih tinggi).
	counts := map[string]int{}
	var order []quizreportmodel.Ref
	for _, q := range w.Questions {
		if q.Chapter == nil {
			continue
		}
		if counts[q.Chapter.ID] == 0 {
			order = append(order, *q.Chapter)
		}
		counts[q.Chapter.ID]++
	}
	for _, c := range order {
		if card.TopChapter == nil || counts[c.ID] > card.TopChapter.Count {
			card.TopChapter = &TopChapterResponse{ID: c.ID, Name: c.Name, Count: counts[c.ID]}
		}
	}

	for i, q := range w.Questions {
		if i == maxWrongQuestions {
			break
		}
		item := WrongQuestionResponse{
			ID: q.QuestionID, Code: q.Code, QuestionText: q.QuestionText, Answered: q.Answered, Wrong: q.Wrong,
			WrongPercent: round2(float64(q.Wrong) * 100 / float64(q.Answered)),
		}
		if q.Chapter != nil {
			item.Chapter = &RefResponse{ID: q.Chapter.ID, Name: q.Chapter.Name}
		}
		card.Questions = append(card.Questions, item)
	}
	return card
}

// toTrend mengisi 12 minggu berurutan (minggu tanpa pengerjaan tetap ada, average null) untuk semua
// jenjang dan per jenjang. Rata-rata = total nilai / jumlah pengerjaan (bukan rata-rata dari rata-rata).
func toTrend(from time.Time, rows []quizreportmodel.WeekLevel) TrendResponse {
	type acc struct {
		sessions int
		sum      float64
	}
	weekKey := func(t time.Time) string { return t.Format(time.DateOnly) }
	avg := func(a acc) *float64 {
		if a.sessions == 0 {
			return nil
		}
		v := round2(a.sum / float64(a.sessions))
		return &v
	}

	all := map[string]acc{}
	perLevel := map[string]map[string]acc{}
	var levels []quizreportmodel.Ref // urut jenjang (baris sudah urut sort_order per minggu)
	seen := map[string]bool{}
	sortOf := map[string]int{}
	for _, row := range rows {
		k := weekKey(row.WeekStart)
		a := all[k]
		a.sessions += row.Sessions
		a.sum += row.ScoreSum
		all[k] = a
		if perLevel[row.Level.ID] == nil {
			perLevel[row.Level.ID] = map[string]acc{}
		}
		perLevel[row.Level.ID][k] = acc{row.Sessions, row.ScoreSum}
		if !seen[row.Level.ID] {
			seen[row.Level.ID] = true
			sortOf[row.Level.ID] = row.LevelSort
			levels = append(levels, row.Level)
		}
	}
	sort.SliceStable(levels, func(i, j int) bool { return sortOf[levels[i].ID] < sortOf[levels[j].ID] })

	r := TrendResponse{Weeks: make([]TrendWeekResponse, 0, quizreportmodel.TrendWeeks), Series: make([]TrendSeriesResponse, 0, len(levels))}
	keys := make([]string, 0, quizreportmodel.TrendWeeks)
	for i := 0; i < quizreportmodel.TrendWeeks; i++ {
		start := from.AddDate(0, 0, 7*i)
		k := weekKey(start)
		keys = append(keys, k)
		r.Weeks = append(r.Weeks, TrendWeekResponse{
			WeekStart: k, WeekEnd: weekKey(start.AddDate(0, 0, 6)), Sessions: all[k].sessions, Average: avg(all[k]),
		})
	}
	for _, l := range levels {
		s := TrendSeriesResponse{EducationLevel: RefResponse{ID: l.ID, Name: l.Name}, Points: make([]TrendPointResponse, 0, len(keys))}
		for _, k := range keys {
			a := perLevel[l.ID][k]
			s.Points = append(s.Points, TrendPointResponse{WeekStart: k, Sessions: a.sessions, Average: avg(a)})
		}
		r.Series = append(r.Series, s)
	}
	return r
}
