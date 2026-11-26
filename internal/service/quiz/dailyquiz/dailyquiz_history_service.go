package dailyquiz

import (
	"context"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/dailyquiz"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/dailyquiz"
)

// Halaman Daily Quiz siswa: riwayat & statistik (BRD FR-QUIZ-019). Review 1 Daily Quiz memakai
// GET /daily-quizzes/:id/review yang sudah ada.

// monthRange mengubah "2026-11" (WIB) menjadi [tanggal 1, tanggal 1 bulan berikutnya). Format sudah
// dicek binding.
func monthRange(month string) (time.Time, time.Time) {
	from, _ := time.ParseInLocation("2006-01", month, clock.Jakarta)
	return from, from.AddDate(0, 1, 0)
}

// History mengambil 1 halaman riwayat Daily Quiz siswa (yang sudah dikumpulkan, terbaru dulu).
// month kosong = semua waktu.
func (s *DailyQuizService) History(ctx context.Context, studentID string, q dailyquizdto.HistoryQuery) ([]dailyquizmodel.HistoryItem, dailyquizmodel.HistoryFilter, int, error) {
	f := dailyquizmodel.HistoryFilter{SubjectID: q.SubjectID, Difficulty: q.Difficulty, Page: q.Page, PerPage: q.PerPage}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	if q.Month != "" {
		from, to := monthRange(q.Month)
		f.From, f.To = &from, &to
	}
	items, total, err := s.repo.History(ctx, studentID, f)
	return items, f, total, err
}

// HistoryStats menghitung statistik Daily Quiz siswa: bulan month (default bulan ini, WIB) +
// sepanjang waktu + peringkat bulan itu di antara siswa sekelas.
func (s *DailyQuizService) HistoryStats(ctx context.Context, studentID string, q dailyquizdto.HistoryStatsQuery) (dailyquizmodel.HistoryStats, error) {
	month := q.Month
	if month == "" {
		month = time.Now().In(clock.Jakarta).Format("2006-01")
	}
	student, err := s.repo.FindStudent(ctx, studentID)
	if err != nil {
		return dailyquizmodel.HistoryStats{}, err
	}
	from, to := monthRange(month)
	return s.repo.HistoryStats(ctx, studentID, student.GradeID, month, from, to)
}
