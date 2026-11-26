// Package leaderboard (service) berisi aturan leaderboard:
//   - Daily Quiz: total point semua Daily Quiz dalam 1 bulan (WIB), bisa difilter jenjang & kelas.
//   - Uji Kom & Tryout: 1 quiz = 1 leaderboard (tidak digabung).
//
// Peringkat: point tertinggi; point sama → total waktu pengerjaan paling singkat.
// Point (bukan nilai): soal benar easy 10, medium 15, hard 20; salah / tidak dijawab 0.
//
// Di package lain, import dengan alias leaderboardservice:
//
//	leaderboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/leaderboard"
package leaderboard

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	leaderboarddto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/leaderboard"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/leaderboard"
	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/leaderboard"
)

// LeaderboardService berisi logika leaderboard.
type LeaderboardService struct {
	repo *leaderboardrepo.LeaderboardRepository
}

// NewLeaderboardService membuat LeaderboardService. Dipanggil sekali di main.go.
func NewLeaderboardService(repo *leaderboardrepo.LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo: repo}
}

// monthLayout adalah format query month: "2026-11".
const monthLayout = "2006-01"

// DailyQuiz mengambil leaderboard Daily Quiz 1 bulan (default bulan ini, WIB).
// Kalau yang melihat siswa, posisinya sendiri ikut dikembalikan (Board.Me).
func (s *LeaderboardService) DailyQuiz(ctx context.Context, userID, role string, q leaderboarddto.DailyQuery) (string, leaderboardmodel.DailyFilter, leaderboardmodel.Board, error) {
	month := q.Month
	if month == "" {
		month = time.Now().In(clock.Jakarta).Format(monthLayout)
	}
	from, err := time.ParseInLocation(monthLayout, month, clock.Jakarta)
	if err != nil {
		return "", leaderboardmodel.DailyFilter{}, leaderboardmodel.Board{}, &leaderboardmodel.ValidationError{
			Fields: []leaderboardmodel.FieldError{{Field: "month", Message: "format tidak valid, contoh 2026-11"}},
		}
	}

	f := leaderboardmodel.DailyFilter{
		From: from, To: from.AddDate(0, 1, 0),
		EducationLevelID: q.EducationLevelID, GradeID: q.GradeID,
		Page: q.Page, PerPage: q.PerPage,
	}
	setPage(&f.Page, &f.PerPage)

	board, err := s.repo.DailyQuiz(ctx, month, f, meID(userID, role))
	if err != nil {
		return month, f, board, err
	}
	prize, err := s.repo.DailyPrize(ctx, month)
	board.Prize = &prize
	return month, f, board, err
}

// Quiz mengambil leaderboard 1 quiz Uji Kom / Tryout. Siswa hanya boleh melihat quiz yang sudah
// publish untuk kelasnya (selain itu ErrQuizNotFound); tutor/admin boleh semua quiz.
func (s *LeaderboardService) Quiz(ctx context.Context, quizID, userID, role string, q leaderboarddto.QuizQuery) (leaderboardmodel.Quiz, leaderboardmodel.Board, int, int, error) {
	page, perPage := q.Page, q.PerPage
	setPage(&page, &perPage)
	if uuid.Validate(quizID) != nil {
		return leaderboardmodel.Quiz{}, leaderboardmodel.Board{}, page, perPage, leaderboardmodel.ErrQuizNotFound
	}
	quiz, err := s.repo.FindQuiz(ctx, quizID)
	if err != nil {
		return leaderboardmodel.Quiz{}, leaderboardmodel.Board{}, page, perPage, err
	}
	if quiz == nil {
		return leaderboardmodel.Quiz{}, leaderboardmodel.Board{}, page, perPage, leaderboardmodel.ErrQuizNotFound
	}
	if role == loginmodel.RoleStudent {
		grade, err := s.repo.FindUserGrade(ctx, userID)
		if err != nil {
			return leaderboardmodel.Quiz{}, leaderboardmodel.Board{}, page, perPage, err
		}
		if quiz.Status != "published" || grade == nil || *grade != quiz.GradeID {
			return leaderboardmodel.Quiz{}, leaderboardmodel.Board{}, page, perPage, leaderboardmodel.ErrQuizNotFound
		}
	}

	board, err := s.repo.Quiz(ctx, quizID, page, perPage, meID(userID, role))
	return *quiz, board, page, perPage, err
}

// meID = id siswa yang melihat (untuk posisi sendiri); "" kalau bukan siswa.
func meID(userID, role string) string {
	if role == loginmodel.RoleStudent {
		return userID
	}
	return ""
}

func setPage(page, perPage *int) {
	if *page == 0 {
		*page = 1
	}
	if *perPage == 0 {
		*perPage = 20
	}
}
