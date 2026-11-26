package quiz

import (
	"context"
	"strings"
	"time"

	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
)

// Halaman Uji Kom & Tryout siswa: "Akan datang", "Sudah dikerjakan", dan statistik
// (BRD FR-TRY-013, 016, 017, 020, 021). Buka / kumpulkan / review memakai endpoint quiz yang sudah ada.

// studentFilter menyusun filter dari query (tab kosong = Uji Kom + Tryout; default halaman 1, 20 baris).
func studentFilter(q quizdto.StudentQuizQuery) quizmodel.StudentQuizFilter {
	f := quizmodel.StudentQuizFilter{
		Types: []string{quizmodel.TypeUjiKom, quizmodel.TypeTryout}, Search: strings.TrimSpace(q.Search),
		SubjectID: q.SubjectID, Page: q.Page, PerPage: q.PerPage,
	}
	if q.Type != "" {
		f.Types = []string{q.Type}
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	return f
}

// StudentUpcoming mengambil bagian "Akan datang" untuk kelas siswa: quiz publish yang event-nya belum
// selesai dan belum dikumpulkan siswa, urut paling dekat dimulai. Siswa tanpa kelas → kosong.
func (s *QuizService) StudentUpcoming(ctx context.Context, studentID string, q quizdto.StudentQuizQuery) ([]quizmodel.UpcomingQuiz, quizmodel.StudentQuizFilter, int, time.Time, error) {
	f := studentFilter(q)
	now := time.Now()
	gradeID, err := s.repo.FindUserGrade(ctx, studentID)
	if err != nil || gradeID == nil {
		return []quizmodel.UpcomingQuiz{}, f, 0, now, err
	}
	items, total, err := s.repo.StudentUpcoming(ctx, studentID, *gradeID, now, f)
	return items, f, total, now, err
}

// StudentHistory mengambil bagian "Sudah dikerjakan": quiz yang dikumpulkan siswa (terbaru dulu) +
// peringkatnya di setiap quiz.
func (s *QuizService) StudentHistory(ctx context.Context, studentID string, q quizdto.StudentQuizQuery) ([]quizmodel.DoneQuiz, quizmodel.StudentQuizFilter, int, error) {
	f := studentFilter(q)
	items, total, err := s.repo.StudentHistory(ctx, studentID, f)
	return items, f, total, err
}

// StudentStats menghitung statistik siswa (tab ikut query type) + Tryout terakhir.
func (s *QuizService) StudentStats(ctx context.Context, studentID string, q quizdto.StudentQuizQuery) (quizmodel.StudentQuizStats, error) {
	return s.repo.StudentStats(ctx, studentID, studentFilter(q))
}
