package dailyquiz

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/leaderboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// Submit menyimpan jawaban Daily Quiz yang sudah dibuka, menghitung nilai (total 100, per soal =
// 100 / total soal; benar/salah dari FE), lalu mengembalikan review. Hanya sekali. Tanpa batas
// waktu (Daily Quiz tidak memakai timer).
func (s *DailyQuizService) Submit(ctx context.Context, quizID, studentID string, submitted []scoring.SubmittedAnswer) (dailyquizmodel.DailyQuizReview, error) {
	if uuid.Validate(quizID) != nil {
		return dailyquizmodel.DailyQuizReview{}, dailyquizmodel.ErrDailyQuizNotFound
	}
	detail, err := s.repo.FindDetail(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	state, err := s.repo.FindSubmission(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	switch {
	case state.OpenedAt == nil:
		return dailyquizmodel.DailyQuizReview{}, &dailyquizmodel.StateError{Message: "Daily Quiz belum dibuka. Buka dulu sebelum mengumpulkan jawaban"}
	case state.SubmittedAt != nil:
		return dailyquizmodel.DailyQuizReview{}, alreadySubmitted(*state.SubmittedAt)
	}

	difficulties, err := s.questionRepo.FindDifficulties(ctx, detail.QuestionIDs)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	// Point per level = pengaturan global Daily Quiz saat submit (diatur admin).
	settings, err := s.repo.FindSettings(ctx)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	result, fieldErrs := scoring.Grade(detail.QuestionIDs, difficulties, settings.PointTable(), submitted)
	if len(fieldErrs) > 0 {
		errs := &fieldErrors{}
		for _, f := range fieldErrs {
			errs.add(f.Field, f.Message)
		}
		return dailyquizmodel.DailyQuizReview{}, errs.toError()
	}

	now := time.Now()
	result.DurationSeconds = scoring.DurationSeconds(*state.OpenedAt, now)
	// Point ikut dicatat ke leaderboard Daily Quiz bulan ini (WIB), dalam transaksi yang sama.
	period := now.In(clock.Jakarta).Format(leaderboardmodel.PeriodLayout)
	ok, err := s.repo.Submit(ctx, quizID, studentID, result, now, leaderboardmodel.PointRecord{
		Board: leaderboardmodel.BoardDailyQuiz, Period: &period, StudentID: studentID,
		Points: result.Points, DurationSeconds: result.DurationSeconds, SubmittedAt: now,
		SourceType: leaderboardmodel.SourceDailyQuiz, SourceID: quizID,
	})
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	if !ok { // 2 request submit bersamaan: yang kedua kalah
		return dailyquizmodel.DailyQuizReview{}, alreadySubmitted(now)
	}
	state.SubmittedAt, state.Score, state.CorrectCount, state.Answers = &now, result.Score, result.CorrectCount, result.Answers
	state.DurationSeconds = result.DurationSeconds
	state.Points = result.Points
	return s.buildReview(ctx, detail, state)
}

// Review mengambil hasil & review Daily Quiz yang sudah di-submit siswa.
func (s *DailyQuizService) Review(ctx context.Context, quizID, studentID string) (dailyquizmodel.DailyQuizReview, error) {
	if uuid.Validate(quizID) != nil {
		return dailyquizmodel.DailyQuizReview{}, dailyquizmodel.ErrDailyQuizNotFound
	}
	detail, err := s.repo.FindDetail(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	state, err := s.repo.FindSubmission(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	if state.SubmittedAt == nil {
		return dailyquizmodel.DailyQuizReview{}, &dailyquizmodel.StateError{Message: "Belum ada jawaban yang dikumpulkan untuk Daily Quiz ini"}
	}
	return s.buildReview(ctx, detail, state)
}

// buildReview menyusun review: soal urut nomor (dengan pembahasan) + jawaban siswa yang disimpan.
func (s *DailyQuizService) buildReview(ctx context.Context, detail dailyquizmodel.DailyQuizDetail, state dailyquizmodel.Submission) (dailyquizmodel.DailyQuizReview, error) {
	content, err := s.questionRepo.FindForAttempt(ctx, detail.QuestionIDs, true)
	if err != nil {
		return dailyquizmodel.DailyQuizReview{}, err
	}
	return dailyquizmodel.DailyQuizReview{DailyQuizDetail: detail, Submission: state, Items: scoring.Review(content, state.Answers)}, nil
}

func alreadySubmitted(at time.Time) error {
	return &dailyquizmodel.StateError{
		Message: "Jawaban sudah dikumpulkan pada " + at.In(clock.Jakarta).Format("02-01-2006 15:04") + " WIB. Lihat hasilnya di review",
	}
}
