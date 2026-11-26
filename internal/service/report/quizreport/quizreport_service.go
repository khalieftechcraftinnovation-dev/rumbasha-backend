// Package quizreport (service) berisi logika report quiz Uji Kom & Tryout:
//   - list per event: tanggal, nama quiz, total peserta, nilai rata-rata / tertinggi / terendah;
//   - detail per event: peringkat, nama siswa, benar x/total, nilai, point, waktu penyelesaian;
//   - analisis soal: urut dari yang paling sering salah (persentase benar terendah).
//
// Di package lain, import dengan alias quizreportservice:
//
//	quizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/quizreport"
package quizreport

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/report/quizreport"
	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/quizreport"
	quizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/quizreport"
)

// QuizReportService berisi logika report quiz.
type QuizReportService struct {
	repo *quizreportrepo.QuizReportRepository
}

// NewQuizReportService membuat QuizReportService. Dipanggil sekali di main.go.
func NewQuizReportService(repo *quizreportrepo.QuizReportRepository) *QuizReportService {
	return &QuizReportService{repo: repo}
}

// List mengambil 1 halaman ringkasan report per event (event terbaru dulu).
// date_from / date_to (WIB, "2026-11-01") memfilter tanggal mulai event, keduanya inklusif.
func (s *QuizReportService) List(ctx context.Context, q quizreportdto.ListQuery) ([]quizreportmodel.Summary, quizreportmodel.ListFilter, int, error) {
	f := quizreportmodel.ListFilter{
		Type: q.Type, Search: strings.TrimSpace(q.Search),
		EducationLevelID: q.EducationLevelID, GradeID: q.GradeID, SubjectID: q.SubjectID,
		Page: q.Page, PerPage: q.PerPage,
	}
	setPage(&f.Page, &f.PerPage)

	var errs []quizreportmodel.FieldError
	if q.DateFrom != "" {
		if t, err := time.ParseInLocation(time.DateOnly, q.DateFrom, clock.Jakarta); err == nil {
			f.From = &t
		} else {
			errs = append(errs, quizreportmodel.FieldError{Field: "date_from", Message: "format tanggal 2026-11-01"})
		}
	}
	if q.DateTo != "" {
		if t, err := time.ParseInLocation(time.DateOnly, q.DateTo, clock.Jakarta); err == nil {
			end := t.AddDate(0, 0, 1) // inklusif: sampai akhir hari date_to
			f.To = &end
		} else {
			errs = append(errs, quizreportmodel.FieldError{Field: "date_to", Message: "format tanggal 2026-11-30"})
		}
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		errs = append(errs, quizreportmodel.FieldError{Field: "date_to", Message: "tidak boleh sebelum date_from"})
	}
	if len(errs) > 0 {
		return nil, f, 0, &quizreportmodel.ValidationError{Fields: errs}
	}

	items, total, err := s.repo.List(ctx, f)
	return items, f, total, err
}

// Detail mengambil ringkasan 1 event + 1 halaman peserta urut peringkat.
func (s *QuizReportService) Detail(ctx context.Context, quizID string, q quizreportdto.DetailQuery) (quizreportmodel.Summary, []quizreportmodel.Participant, quizreportmodel.ParticipantFilter, int, error) {
	f := quizreportmodel.ParticipantFilter{Search: strings.TrimSpace(q.Search), Page: q.Page, PerPage: q.PerPage}
	setPage(&f.Page, &f.PerPage)
	summary, err := s.findSummary(ctx, quizID)
	if err != nil {
		return summary, nil, f, 0, err
	}
	participants, total, err := s.repo.Participants(ctx, quizID, f)
	return summary, participants, f, total, err
}

// Questions mengambil statistik setiap soal quiz, urut dari yang PALING SERING SALAH (persentase
// benar terendah) ke yang paling jarang salah. Persentase sama → yang salahnya lebih banyak dulu,
// lalu nomor soal.
func (s *QuizReportService) Questions(ctx context.Context, quizID string) (quizreportmodel.Summary, []quizreportmodel.QuestionStat, error) {
	summary, err := s.findSummary(ctx, quizID)
	if err != nil {
		return summary, nil, err
	}
	stats, err := s.repo.QuestionStats(ctx, quizID)
	if err != nil {
		return summary, nil, err
	}
	sort.SliceStable(stats, func(i, j int) bool {
		a, b := stats[i], stats[j]
		if a.Correct != b.Correct { // peserta sama untuk semua soal: benar sedikit = persentase rendah
			return a.Correct < b.Correct
		}
		if a.Wrong+a.Unanswered != b.Wrong+b.Unanswered {
			return a.Wrong+a.Unanswered > b.Wrong+b.Unanswered
		}
		return a.No < b.No
	})
	return summary, stats, nil
}

func (s *QuizReportService) findSummary(ctx context.Context, quizID string) (quizreportmodel.Summary, error) {
	if uuid.Validate(quizID) != nil {
		return quizreportmodel.Summary{}, quizreportmodel.ErrQuizNotFound
	}
	return s.repo.FindSummary(ctx, quizID)
}

func setPage(page, perPage *int) {
	if *page == 0 {
		*page = 1
	}
	if *perPage == 0 {
		*perPage = 20
	}
}
