// File ini adalah titik awal aplikasi (seperti index.php di Laravel).
// Tugasnya: konek database, menyusun semua layer, lalu menjalankan server.
package main

import (
	"log"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router"

	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"

	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	subjectrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subject"
	subjectservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subject"

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

	r := router.New(router.Handlers{
		Login:   loginHandler,
		Subject: subjectHandler,
	}, cfg.JWTSecret)

	log.Printf("server berjalan di http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server gagal berjalan: %v", err)
	}
}
