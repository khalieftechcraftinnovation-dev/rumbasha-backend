// File ini adalah titik awal aplikasi (seperti index.php di Laravel).
// Tugasnya: konek database, menyusun semua layer, lalu menjalankan server.
package main

import (
	"log"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service"
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

	// Repositories
	userRepository := repository.NewUserRepository(db)
	refreshTokenRepository := repository.NewRefreshTokenRepository(db)
	subscriptionRepository := repository.NewSubscriptionRepository(db)
	subjectRepository := repository.NewSubjectRepository(db)

	// Services
	authService := service.NewAuthService(userRepository, refreshTokenRepository, cfg.JWTSecret, cfg.JWTAccessTTL)
	subscriptionService := service.NewSubscriptionService(subscriptionRepository)
	subjectService := service.NewSubjectService(subjectRepository)

	// Handlers
	handlers := router.Handlers{
		Auth:    handler.NewAuthHandler(authService, subscriptionService),
		Subject: handler.NewSubjectHandler(subjectService),
	}

	r := router.New(handlers, cfg.JWTSecret)

	log.Printf("server berjalan di http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server gagal berjalan: %v", err)
	}
}
