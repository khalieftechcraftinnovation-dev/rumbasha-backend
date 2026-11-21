// Package config membaca konfigurasi aplikasi dari environment variable.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config berisi semua pengaturan aplikasi.
// Tambahkan field di sini saat butuh pengaturan baru (misalnya koneksi database).
type Config struct {
	Port string
}

// Load membaca file .env (kalau ada), lalu mengisi Config dari environment variable.
func Load() Config {
	// Error diabaikan karena file .env memang opsional (di server production
	// biasanya environment variable di-set langsung, tanpa file .env).
	_ = godotenv.Load()

	return Config{
		Port: getEnv("APP_PORT", "8080"),
	}
}

// getEnv mengambil nilai environment variable, atau nilai default kalau kosong.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
