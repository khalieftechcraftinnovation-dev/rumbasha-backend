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

	dashboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/admin/dashboard"
	taskhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/admin/task"
	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/login"
	passwordresethandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/passwordreset"
	masterdatahandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/masterdata"
	announcementhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/notification/announcement"
	profilehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user/profile"
	dashboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/admin/dashboard"
	taskrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/admin/task"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/login"
	passwordresetrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/passwordreset"
	masterdatarepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/masterdata"
	announcementrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/notification/announcement"
	profilerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user/profile"
	dashboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/admin/dashboard"
	taskservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/admin/task"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/login"
	passwordresetservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/passwordreset"
	masterdataservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/masterdata"
	announcementservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/notification/announcement"
	profileservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user/profile"

	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/logreport"

	coinhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/coin"
	subscriptionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/subscription"
	learningmodulehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/learning/learningmodule"
	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/dropdown"
	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/image"
	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/dailyquiz"
	leaderboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/leaderboard"
	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/quiz"
	rumathhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/rumath"
	dailyquizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/dailyquizreport"
	quizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/quizreport"
	userhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user/user"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/coin"
	learningmodulerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/learning/learningmodule"
	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/dropdown"
	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/image"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/dailyquiz"
	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/leaderboard"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/quiz"
	rumathrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/rumath"
	dailyquizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/dailyquizreport"
	quizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/quizreport"
	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user/user"
	coinservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/coin"
	learningmoduleservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/learning/learningmodule"
	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/dropdown"
	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/image"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/dailyquiz"
	leaderboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/leaderboard"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/quiz"
	rumathservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/rumath"
	dailyquizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/dailyquizreport"
	quizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/quizreport"
	userservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"

	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/question"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/question"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/question"

	subscriptionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/subscription"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"

	trialquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/trialquiz"
	landingpagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/site/landingpage"
	trialquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/trialquiz"
	landingpagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/site/landingpage"
	trialquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/trialquiz"
	landingpageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/site/landingpage"
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

	// Fitur: bank soal. Service-nya juga dipakai Quiz Trial (validasi isi soal yang sama).
	questionService := questionservice.NewQuestionService(questionrepo.NewQuestionRepository(db))
	questionHandler := questionhandler.NewQuestionHandler(questionService)

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

	// Fitur: RUMATH (sesi hitung cepat dan hitung bersusun).
	rumathService := rumathservice.NewRumathService(rumathrepo.NewRumathRepository(db), subscriptionService)

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
		Dashboard:       dashboardhandler.NewDashboardHandler(dashboardservice.NewDashboardService(dashboardrepo.NewDashboardRepository(db))),
		Announcement:    announcementhandler.NewAnnouncementHandler(announcementservice.NewAnnouncementService(announcementrepo.NewAnnouncementRepository(db))),
		Task:            taskhandler.NewTaskHandler(taskservice.NewTaskService(taskrepo.NewTaskRepository(db))),
		Login:           loginHandler,
		LogReport:       logReportHandler,
		Question:        questionHandler,
		Image:           imageHandler,
		Dropdown:        dropdownhandler.NewDropdownHandler(dropdownservice.NewDropdownService(dropdownrepo.NewDropdownRepository(db))),
		Quiz:            quizhandler.NewQuizHandler(quizService),
		DailyQuiz:       dailyquizhandler.NewDailyQuizHandler(dailyQuizService),
		Rumath:          rumathhandler.NewRumathHandler(rumathService),
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
		LandingPage:     landingpagehandler.NewLandingPageHandler(landingpageservice.NewLandingPageService(landingpagerepo.NewLandingPageRepository(db))),
		TrialQuiz:       trialquizhandler.NewTrialQuizHandler(trialquizservice.NewTrialQuizService(trialquizrepo.NewTrialQuizRepository(db), questionService)),
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
