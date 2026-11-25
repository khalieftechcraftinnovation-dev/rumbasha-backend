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
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"

	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	subjectrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subject"
	subjectservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subject"

	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/logreport"

	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dailyquiz"
	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dropdown"
	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/image"
	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dailyquiz"
	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dropdown"
	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/image"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dailyquiz"
	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dropdown"
	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/image"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
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

	// Fitur: subject
	subjectService := subjectservice.NewSubjectService(subjectrepo.NewSubjectRepository(db))
	subjectHandler := subjecthandler.NewSubjectHandler(subjectService)

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
	imageHandler := imagehandler.NewImageHandler(
		imageservice.NewImageService(imagerepo.NewImageRepository(db), storage.NewLocal(cfg.StorageDir, cfg.AppURL)),
	)

	// Fitur: Quiz (Uji Kom & Tryout). Service juga dipakai scheduler (ganti kode akses tiap jam).
	quizService := quizservice.NewQuizService(quizrepo.NewQuizRepository(db), questionrepo.NewQuestionRepository(db))

	// Scheduler (cron di dalam proses API): ganti kode akses quiz yang belum selesai.
	// Jadwal dari ACCESS_CODE_REFRESH_CRON (default setiap jam tepat, WIB).
	jobs, err := scheduler.Start(cfg.AccessCodeRefreshCron, quizService)
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
		Login:     loginHandler,
		Subject:   subjectHandler,
		LogReport: logReportHandler,
		Question:  questionHandler,
		Image:     imageHandler,
		Dropdown:  dropdownhandler.NewDropdownHandler(dropdownservice.NewDropdownService(dropdownrepo.NewDropdownRepository(db))),
		Quiz:      quizhandler.NewQuizHandler(quizService),
		DailyQuiz: dailyquizhandler.NewDailyQuizHandler(dailyquizservice.NewDailyQuizService(dailyquizrepo.NewDailyQuizRepository(db), questionrepo.NewQuestionRepository(db), cfg.DailyQuizFreeQuota, cfg.DailyQuizCoinCost)),
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
