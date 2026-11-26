// Package dailyquizreport (service) berisi logika report Daily Quiz PER SISWA (Daily Quiz dibuat
// siswa sendiri, jadi tidak ada event):
//   - list: peringkat siswa dalam periode (default bulan ini, WIB), bisa difilter mapel / jenjang / kelas;
//   - detail 1 siswa: rekap total + rincian per mapel.
//
// Di package lain, import dengan alias dailyquizreportservice:
//
//	dailyquizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/dailyquizreport"
package dailyquizreport

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/report/dailyquizreport"
	dailyquizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/dailyquizreport"
	dailyquizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/dailyquizreport"
)

// DailyQuizReportService berisi logika report Daily Quiz.
type DailyQuizReportService struct {
	repo *dailyquizreportrepo.DailyQuizReportRepository
}

// NewDailyQuizReportService membuat DailyQuizReportService. Dipanggil sekali di main.go.
func NewDailyQuizReportService(repo *dailyquizreportrepo.DailyQuizReportRepository) *DailyQuizReportService {
	return &DailyQuizReportService{repo: repo}
}

// maxPeriodDays = batas panjang periode report (supaya query tidak membaca data bertahun-tahun).
const maxPeriodDays = 366

// periodFrom membaca date_from & date_to (tanggal submit, WIB, keduanya inklusif).
// Kosong = bulan berjalan (tanggal 1 s/d akhir bulan ini). Hanya salah satu diisi:
// date_from saja → sampai hari ini; date_to saja → mulai tanggal 1 bulan date_to.
func periodFrom(dateFrom, dateTo string, now time.Time) (dailyquizreportmodel.Period, error) {
	var errs []dailyquizreportmodel.FieldError
	parse := func(field, v string) (time.Time, bool) {
		t, err := time.ParseInLocation(time.DateOnly, v, clock.Jakarta)
		if err != nil {
			errs = append(errs, dailyquizreportmodel.FieldError{Field: field, Message: "format tanggal 2026-11-01"})
			return time.Time{}, false
		}
		return t, true
	}

	today := now.In(clock.Jakarta)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, clock.Jakarta)
	from, to := monthStart, monthStart.AddDate(0, 1, 0)
	switch {
	case dateFrom != "" && dateTo != "":
		f, okF := parse("date_from", dateFrom)
		t, okT := parse("date_to", dateTo)
		if okF && okT {
			from, to = f, t.AddDate(0, 0, 1)
		}
	case dateFrom != "":
		if f, ok := parse("date_from", dateFrom); ok {
			from = f
			to = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, clock.Jakarta).AddDate(0, 0, 1)
		}
	case dateTo != "":
		if t, ok := parse("date_to", dateTo); ok {
			from, to = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, clock.Jakarta), t.AddDate(0, 0, 1)
		}
	}
	if len(errs) == 0 {
		switch {
		case !from.Before(to):
			errs = append(errs, dailyquizreportmodel.FieldError{Field: "date_to", Message: "tidak boleh sebelum date_from"})
		case to.Sub(from) > maxPeriodDays*24*time.Hour:
			errs = append(errs, dailyquizreportmodel.FieldError{Field: "date_to", Message: "periode maksimal 366 hari"})
		}
	}
	if len(errs) > 0 {
		return dailyquizreportmodel.Period{}, &dailyquizreportmodel.ValidationError{Fields: errs}
	}
	return dailyquizreportmodel.Period{From: from, To: to}, nil
}

// List mengambil 1 halaman siswa urut peringkat dalam periode & filter.
func (s *DailyQuizReportService) List(ctx context.Context, q dailyquizreportdto.ListQuery) ([]dailyquizreportmodel.StudentRow, dailyquizreportmodel.ListFilter, int, error) {
	f := dailyquizreportmodel.ListFilter{
		SubjectID: q.SubjectID, EducationLevelID: q.EducationLevelID, GradeID: q.GradeID,
		Search: strings.TrimSpace(q.Search), Page: q.Page, PerPage: q.PerPage,
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	p, err := periodFrom(q.DateFrom, q.DateTo, time.Now())
	if err != nil {
		return nil, f, 0, err
	}
	f.Period = p
	items, total, err := s.repo.List(ctx, f)
	return items, f, total, err
}

// StudentDetail mengambil data 1 siswa + rekap total + rincian per mapel dalam periode.
func (s *DailyQuizReportService) StudentDetail(ctx context.Context, studentID string, q dailyquizreportdto.PeriodQuery) (dailyquizreportmodel.Student, dailyquizreportmodel.Period, dailyquizreportmodel.Stats, []dailyquizreportmodel.SubjectRow, error) {
	var (
		student dailyquizreportmodel.Student
		total   dailyquizreportmodel.Stats
	)
	p, err := periodFrom(q.DateFrom, q.DateTo, time.Now())
	if err != nil {
		return student, p, total, nil, err
	}
	if uuid.Validate(studentID) != nil {
		return student, p, total, nil, dailyquizreportmodel.ErrStudentNotFound
	}
	if student, err = s.repo.FindStudent(ctx, studentID); err != nil {
		return student, p, total, nil, err
	}
	if total, err = s.repo.StudentTotal(ctx, studentID, p); err != nil {
		return student, p, total, nil, err
	}
	subjects, err := s.repo.StudentSubjects(ctx, studentID, p)
	return student, p, total, subjects, err
}
