package quizreport

import (
	"context"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quizreport"
	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quizreport"
)

// maxOverviewDays = periode laporan terpanjang (sama dengan report Daily Quiz).
const maxOverviewDays = 366

// Overview menghitung kartu ringkasan & grafik 1 tab report (Daily Quiz, Uji Kom, atau Tryout):
//   - kartu: sesi dikerjakan, rata-rata nilai per jenjang, selesai tepat waktu, soal sering salah;
//   - grafik tren rata-rata per minggu (12 minggu terakhir s/d date_to) per jenjang;
//   - grafik sebaran nilai (0-20, 21-40, 41-60, 61-80, 81-100).
//
// Periode = tanggal submit (WIB, inklusif). Kosong = rentang default di pengaturan report, berakhir
// hari ini. Total 5 query (pengaturan + 4 query hitung), masing-masing memakai index tanggal submit.
func (s *QuizReportService) Overview(ctx context.Context, q quizreportdto.OverviewQuery) (quizreportmodel.Overview, error) {
	settings, err := s.repo.FindSettings(ctx)
	if err != nil {
		return quizreportmodel.Overview{}, err
	}
	from, to, err := overviewPeriod(q.DateFrom, q.DateTo, settings.DefaultRangeDays, time.Now())
	if err != nil {
		return quizreportmodel.Overview{}, err
	}
	f := quizreportmodel.OverviewFilter{
		Type: q.Type, EducationLevelID: q.EducationLevelID, GradeID: q.GradeID, SubjectID: q.SubjectID,
		From: from, To: to, WrongThresholdPercent: settings.WrongThresholdPercent,
	}
	if q.WrongThreshold > 0 {
		f.WrongThresholdPercent = q.WrongThreshold
	}

	o := quizreportmodel.Overview{Filter: f}
	if o.Totals, err = s.repo.OverviewTotals(ctx, f); err != nil {
		return o, err
	}
	if o.ByLevel, err = s.repo.OverviewByLevel(ctx, f); err != nil {
		return o, err
	}
	if o.Wrong, err = s.repo.OverviewWrongQuestions(ctx, f); err != nil {
		return o, err
	}

	// Tren: 12 minggu (Senin-Minggu) sampai minggu yang berisi date_to, tidak dibatasi date_from.
	lastDay := to.AddDate(0, 0, -1)
	o.TrendFrom = mondayOf(lastDay).AddDate(0, 0, -7*(quizreportmodel.TrendWeeks-1))
	if o.TrendWeeks, err = s.repo.OverviewTrend(ctx, f, o.TrendFrom, to); err != nil {
		return o, err
	}
	return o, nil
}

// overviewPeriod membaca date_from & date_to (WIB, inklusif) menjadi rentang [from, to).
// Kosong semua = defaultDays hari terakhir s/d hari ini; date_from saja = s/d hari ini;
// date_to saja = defaultDays hari s/d date_to. Maksimal 366 hari.
func overviewPeriod(dateFrom, dateTo string, defaultDays int, now time.Time) (time.Time, time.Time, error) {
	var errs []quizreportmodel.FieldError
	parse := func(field, value string) (time.Time, bool) {
		t, err := time.ParseInLocation(time.DateOnly, value, clock.Jakarta)
		if err != nil {
			errs = append(errs, quizreportmodel.FieldError{Field: field, Message: "format tanggal 2026-11-01"})
			return time.Time{}, false
		}
		return t, true
	}

	today, _ := time.ParseInLocation(time.DateOnly, clock.TodayInJakarta(now), clock.Jakarta)
	last := today
	if dateTo != "" {
		if t, ok := parse("date_to", dateTo); ok {
			last = t
		}
	}
	first := last.AddDate(0, 0, -(defaultDays - 1))
	if dateFrom != "" {
		if t, ok := parse("date_from", dateFrom); ok {
			first = t
		}
	}
	if len(errs) == 0 {
		switch {
		case last.Before(first):
			errs = append(errs, quizreportmodel.FieldError{Field: "date_to", Message: "tidak boleh sebelum date_from"})
		case last.Sub(first).Hours()/24 >= maxOverviewDays:
			errs = append(errs, quizreportmodel.FieldError{Field: "date_to", Message: "periode maksimal 366 hari"})
		}
	}
	if len(errs) > 0 {
		return time.Time{}, time.Time{}, &quizreportmodel.ValidationError{Fields: errs}
	}
	return first, last.AddDate(0, 0, 1), nil
}

// mondayOf mengembalikan hari Senin (00:00 WIB) di minggu tanggal t.
func mondayOf(t time.Time) time.Time {
	t = t.In(clock.Jakarta)
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, clock.Jakarta)
	offset := (int(day.Weekday()) + 6) % 7 // Senin = 0, Minggu = 6
	return day.AddDate(0, 0, -offset)
}

// GetSettings mengambil pengaturan report quiz.
func (s *QuizReportService) GetSettings(ctx context.Context) (quizreportmodel.Settings, error) {
	return s.repo.FindSettings(ctx)
}

// UpdateSettings mengganti pengaturan report quiz (ambang soal sering salah & rentang default).
func (s *QuizReportService) UpdateSettings(ctx context.Context, req quizreportdto.SettingsRequest, adminID string) (quizreportmodel.Settings, error) {
	err := s.repo.UpdateSettings(ctx, quizreportmodel.Settings{
		WrongThresholdPercent: req.WrongThresholdPercent, DefaultRangeDays: req.DefaultRangeDays,
	}, adminID)
	if err != nil {
		return quizreportmodel.Settings{}, err
	}
	return s.repo.FindSettings(ctx)
}
