// File ini adalah titik awal aplikasi (seperti index.php di Laravel).
// Tugasnya: konek database, menyusun semua layer, lalu menjalankan server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scheduler"

	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	masterdatahandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/masterdata"
	passwordresethandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/passwordreset"
	profilehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/profile"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
	masterdatarepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/masterdata"
	passwordresetrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/passwordreset"
	profilerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/profile"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
	masterdataservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/masterdata"
	passwordresetservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/passwordreset"
	profileservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/profile"

	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/logreport"

	coinhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/coin"
	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dailyquiz"
	dailyquizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dailyquizreport"
	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dropdown"
	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/image"
	leaderboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/leaderboard"
	learningmodulehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/learningmodule"
	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
	quizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quizreport"
	subscriptionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subscription"
	userhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/coin"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dailyquiz"
	dailyquizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dailyquizreport"
	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dropdown"
	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/image"
	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/leaderboard"
	learningmodulerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/learningmodule"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
	quizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quizreport"
	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user"
	coinservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/coin"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dailyquiz"
	dailyquizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dailyquizreport"
	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dropdown"
	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/image"
	leaderboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/leaderboard"
	learningmoduleservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/learningmodule"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
	quizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quizreport"
	userservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"

	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/question"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/question"

	subscriptionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subscription"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subscription"
)

func main() {
	cfg := config.Load()

	// JWT_SECRET wajib ada dan cukup panjang. Tanpa ini, siapa pun bisa membuat token palsu.
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET wajib diisi di .env, minimal 32 karakter")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close(db) // defer: dijalankan otomatis saat main() selesai

	// Susun layer dari bawah ke atas: repository → service → handler.
	// Setiap layer menerima layer di bawahnya lewat parameter (dependency injection),
	// jadi tidak ada layer yang membuat dependency-nya sendiri.

	// Fitur: subscription (paket akses siswa; dipakai oleh fitur login di /auth/me)
	subscriptionService := subscriptionservice.NewSubscriptionService(
		subscriptionrepo.NewSubscriptionRepository(db),
	)

	// Fitur: login
	loginService := loginservice.NewLoginService(
		loginrepo.NewUserRepository(db),
		loginrepo.NewRefreshTokenRepository(db),
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
	)
	loginHandler := loginhandler.NewLoginHandler(loginService, subscriptionService)

	// Fitur: kirim log ke email (laporan bug dari admin).
	// Kalau SMTP belum diisi di .env, server tetap jalan; endpoint-nya saja yang membalas 503.
	emailSender := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	if !emailSender.IsConfigured() || cfg.LogReportEmail == "" {
		log.Println("info: SMTP / LOG_REPORT_EMAIL belum diisi di .env, fitur kirim log ke email nonaktif")
	}
	logReportService := logreportservice.NewLogReportService(emailSender, cfg.LogDir, cfg.LogReportEmail, cfg.AppEnv)
	logReportHandler := logreporthandler.NewLogReportHandler(logReportService, loginService)

	// Fitur: bank soal
	questionHandler := questionhandler.NewQuestionHandler(
		questionservice.NewQuestionService(questionrepo.NewQuestionRepository(db)),
	)

	// Fitur: Image Library (upload & hapus gambar). File disimpan di STORAGE_DIR/images/...
	// File upload (gambar publik di /storage/images; PDF modul hanya lewat API wajib login).
	store := storage.NewLocal(cfg.StorageDir, cfg.AppURL)
	imageHandler := imagehandler.NewImageHandler(
		imageservice.NewImageService(imagerepo.NewImageRepository(db), store),
	)

	// Fitur: Quiz (Uji Kom & Tryout). Service juga dipakai scheduler (kode akses tiap jam, hadiah juara 23:59).
	quizService := quizservice.NewQuizService(quizrepo.NewQuizRepository(db), questionrepo.NewQuestionRepository(db))

	// Fitur: Daily Quiz. Service juga dipakai scheduler (hadiah juara bulanan per kelas).
	dailyQuizService := dailyquizservice.NewDailyQuizService(dailyquizrepo.NewDailyQuizRepository(db), questionrepo.NewQuestionRepository(db))

	// Scheduler (cron di dalam proses API):
	//   - ganti kode akses quiz yang belum selesai: ACCESS_CODE_REFRESH_CRON (default setiap jam tepat, WIB)
	//   - bagi hadiah coin juara quiz yang sudah selesai + juara bulanan Daily Quiz (hari terakhir bulan):
	//     QUIZ_PRIZE_CRON (default setiap hari 23:59 WIB)
	jobs, err := scheduler.Start(scheduler.Specs{AccessCode: cfg.AccessCodeRefreshCron, QuizPrize: cfg.QuizPrizeCron}, quizService, dailyQuizService)
	if err != nil {
		log.Fatal(err)
	}
	// Saat berhenti: tunggu pekerjaan terjadwal yang sedang jalan selesai (sebelum database ditutup).
	defer func() { <-jobs.Stop().Done() }()

	// Log API harian: logs/api-YYYY-MM-DD.txt, disimpan LOG_RETENTION_DAYS hari terakhir.
	apiLog, err := apilog.NewWriter(cfg.LogDir, cfg.LogRetentionDays)
	if err != nil {
		log.Fatal(err)
	}
	defer apiLog.Close() // menulis sisa log di antrean sebelum aplikasi berhenti

	r := router.New(router.Handlers{
		Login:           loginHandler,
		LogReport:       logReportHandler,
		Question:        questionHandler,
		Image:           imageHandler,
		Dropdown:        dropdownhandler.NewDropdownHandler(dropdownservice.NewDropdownService(dropdownrepo.NewDropdownRepository(db))),
		Quiz:            quizhandler.NewQuizHandler(quizService),
		DailyQuiz:       dailyquizhandler.NewDailyQuizHandler(dailyQuizService),
		Leaderboard:     leaderboardhandler.NewLeaderboardHandler(leaderboardservice.NewLeaderboardService(leaderboardrepo.NewLeaderboardRepository(db))),
		Coin:            coinhandler.NewCoinHandler(coinservice.NewCoinService(coinrepo.NewCoinRepository(db))),
		User:            userhandler.NewUserHandler(userservice.NewUserService(userrepo.NewUserRepository(db), subscriptionService)),
		QuizReport:      quizreporthandler.NewQuizReportHandler(quizreportservice.NewQuizReportService(quizreportrepo.NewQuizReportRepository(db))),
		DailyQuizReport: dailyquizreporthandler.NewDailyQuizReportHandler(dailyquizreportservice.NewDailyQuizReportService(dailyquizreportrepo.NewDailyQuizReportRepository(db))),
		LearningModule:  learningmodulehandler.NewLearningModuleHandler(learningmoduleservice.NewLearningModuleService(learningmodulerepo.NewLearningModuleRepository(db), store, subscriptionService)),
		Subscription:    subscriptionhandler.NewSubscriptionHandler(subscriptionService),
		MasterData:      masterdatahandler.NewMasterDataHandler(masterdataservice.NewMasterDataService(masterdatarepo.NewMasterDataRepository(db))),
		PasswordReset:   passwordresethandler.NewPasswordResetHandler(passwordresetservice.NewPasswordResetService(passwordresetrepo.NewPasswordResetRepository(db))),
		Profile:         profilehandler.NewProfileHandler(profileservice.NewProfileService(profilerepo.NewProfileRepository(db), subscriptionService, store)),
	}, cfg.JWTSecret, apiLog, cfg.StorageDir)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	// Server dijalankan di goroutine (background), supaya main() bisa menunggu Ctrl+C.
	go func() {
		log.Printf("server berjalan di http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server gagal berjalan: %v", err)
		}
	}()

	// Tunggu sampai ada sinyal berhenti (Ctrl+C di terminal, atau SIGTERM dari server).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	// Graceful shutdown: berhenti menerima request baru, tunggu request yang sedang
	// berjalan selesai (maksimal 10 detik). Setelah itu semua defer di atas dijalankan:
	// sisa log ditulis ke file, lalu koneksi database ditutup.
	log.Println("server berhenti...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("gagal menghentikan server dengan rapi: %v", err)
	}
}
