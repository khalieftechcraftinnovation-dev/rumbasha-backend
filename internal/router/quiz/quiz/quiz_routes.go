// Package quiz (router) berisi daftar URL untuk fitur quiz.
//
// Di package lain, import dengan alias quizrouter:
//
//	quizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/quiz"
package quiz

import (
	"github.com/gin-gonic/gin"

	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route /quizzes. Wajib login, hanya admin & super admin.
func Register(private *gin.RouterGroup, h *quizhandler.QuizHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/quizzes", adminOnly, h.Create)                        // buat quiz Uji Kom / Tryout + soal yang dipilih
	private.GET("/quizzes", adminOnly, h.List)                           // list quiz: filter, search, status jadwal, pagination
	private.GET("/quizzes/schedule-counts", adminOnly, h.ScheduleCounts) // jumlah quiz per status jadwal (angka tab)
	private.GET("/quizzes/:id", adminOnly, h.GetDetail)                  // detail 1 quiz + daftar soal
	private.PUT("/quizzes/:id", adminOnly, h.Update)                     // ubah 1 quiz (ganti penuh), hanya yang belum dimulai
	private.DELETE("/quizzes", adminOnly, h.Delete)                      // hapus banyak quiz (soft delete), body {"ids": [...]}
	private.GET("/quiz-bundles", adminOnly, h.ListBundles)               // pilihan bundle (soal quiz lama) untuk quiz baru

	// Dashboard siswa: pengumuman quiz publish untuk kelasnya (tanpa kode akses).
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)
	private.GET("/student/quiz-announcements", studentOnly, h.Announcements)
	private.GET("/student/quizzes/upcoming", studentOnly, h.StudentUpcoming) // "Akan datang"
	private.GET("/student/quizzes/history", studentOnly, h.StudentHistory)   // "Sudah dikerjakan" + peringkat
	private.GET("/student/quizzes/stats", studentOnly, h.StudentStats)       // statistik + Tryout terakhir
	private.POST("/quizzes/:id/open", studentOnly, h.Open)                   // buka Uji Kom / Tryout (kode akses), soal + kunci jawaban
	private.POST("/quizzes/:id/submit", studentOnly, h.Submit)               // kumpulkan jawaban (dinilai FE) → nilai + review
	private.GET("/quizzes/:id/review", studentOnly, h.Review)                // hasil & review setelah submit

	// Kode akses quiz untuk dibagikan ke siswa: tutor, admin, super admin.
	staff := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleTutor)
	private.GET("/quizzes/:id/access-code", staff, h.AccessCode)
}
