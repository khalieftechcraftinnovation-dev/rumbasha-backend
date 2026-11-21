// Package config membaca konfigurasi aplikasi dari environment variable.
package config

import (
	"net"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config berisi semua pengaturan aplikasi.
// Tambahkan field di sini saat butuh pengaturan baru.
type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBDebug    bool // true = semua query SQL dicetak di terminal
}

// Load membaca file .env (kalau ada), lalu mengisi Config dari environment variable.
func Load() Config {
	// Error diabaikan karena file .env memang opsional (di server production
	// biasanya environment variable di-set langsung, tanpa file .env).
	_ = godotenv.Load()

	return Config{
		Port: getEnv("APP_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "rumbasha"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBDebug:    getEnv("DB_DEBUG", "false") == "true",
	}
}

// DatabaseURL menghasilkan alamat koneksi PostgreSQL, contoh:
//
//	postgres://postgres:rahasia@localhost:5432/rumbasha?sslmode=disable
//
// url.URL dipakai supaya password yang berisi karakter khusus (@, #, /)
// otomatis di-encode dengan benar.
func (c Config) DatabaseURL() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     net.JoinHostPort(c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: "sslmode=" + c.DBSSLMode,
	}
	return u.String()
}

// getEnv mengambil nilai environment variable, atau nilai default kalau kosong.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
