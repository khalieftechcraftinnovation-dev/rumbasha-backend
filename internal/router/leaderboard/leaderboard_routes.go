// Package leaderboard (router) berisi daftar URL untuk fitur leaderboard.
//
// Di package lain, import dengan alias leaderboardrouter:
//
//	leaderboardrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/leaderboard"
package leaderboard

import (
	"github.com/gin-gonic/gin"

	leaderboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/leaderboard"
)

// Register mendaftarkan route /leaderboards. Wajib login, semua role (siswa ikut mendapat posisinya
// sendiri di field "me").
func Register(private *gin.RouterGroup, h *leaderboardhandler.LeaderboardHandler) {
	private.GET("/leaderboards/daily-quiz", h.DailyQuiz) // total point Daily Quiz 1 bulan, filter jenjang/kelas
	private.GET("/leaderboards/quizzes/:id", h.Quiz)     // 1 quiz Uji Kom / Tryout
}
