// Package config membaca konfigurasi aplikasi dari environment variable.
package config

import (
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config berisi semua pengaturan aplikasi.
// Tambahkan field di sini saat butuh pengaturan baru.
type Config struct {
	AppEnv string // development, staging, atau production
	Port   string
	AppURL string // alamat publik server, contoh https://api.rumbasha.com (dipakai untuk link gambar)

	// Folder penyimpanan file upload (gambar disimpan di StorageDir/images/...).
	StorageDir string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBDebug    bool // true = semua query SQL dicetak di terminal

	JWTSecret    string        // kunci rahasia untuk menandatangani access token
	JWTAccessTTL time.Duration // masa berlaku access token, contoh 15m

	// Aturan Daily Quiz, berlaku sama untuk semua siswa (BR-019).
	DailyQuizFreeQuota int // jatah gratis per hari
	DailyQuizCoinCost  int // biaya coin per quiz setelah jatah gratis habis

	// Log API: satu file .txt per hari di folder LogDir.
	LogDir           string
	LogRetentionDays int // file log yang lebih lama dari ini dihapus otomatis

	// Email (SMTP), dipakai untuk mengirim log ke developer saat admin melaporkan bug.
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string // untuk Gmail: App Password, bukan password akun
	SMTPFrom       string // alamat pengirim
	LogReportEmail string // alamat tujuan laporan bug + log
}

// IsProduction bernilai true kalau aplikasi berjalan di production.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load membaca file .env (kalau ada), lalu mengisi Config dari environment variable.
func Load() Config {
	// Error diabaikan karena file .env memang opsional (di server production
	// biasanya environment variable di-set langsung, tanpa file .env).
	_ = godotenv.Load()

	return Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("APP_PORT", "8080"),
		AppURL: strings.TrimRight(getEnv("APP_URL", "http://localhost:"+getEnv("APP_PORT", "8080")), "/"),

		StorageDir: getEnv("STORAGE_DIR", "storage"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "rumbasha"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBDebug:    getEnv("DB_DEBUG", "false") == "true",

		JWTSecret:    getEnv("JWT_SECRET", ""),
		JWTAccessTTL: getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),

		DailyQuizFreeQuota: getEnvInt("DAILY_QUIZ_FREE_QUOTA", 2),
		DailyQuizCoinCost:  getEnvInt("DAILY_QUIZ_COIN_COST", 10),

		LogDir:           getEnv("LOG_DIR", "logs"),
		LogRetentionDays: getEnvInt("LOG_RETENTION_DAYS", 15),

		SMTPHost:       getEnv("SMTP_HOST", ""),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUsername:   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:       getEnv("SMTP_FROM", ""),
		LogReportEmail: getEnv("LOG_REPORT_EMAIL", ""),
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

// getEnvInt sama seperti getEnv, tapi untuk angka.
// Aplikasi langsung berhenti kalau nilainya bukan angka, supaya salah
// konfigurasi ketahuan saat start, bukan diam-diam memakai nilai default.
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("konfigurasi %s harus berupa angka, tapi isinya %q", key, value)
	}
	return number
}

// getEnvDuration sama seperti getEnv, tapi untuk durasi, contoh "15m", "1h".
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("konfigurasi %s harus berupa durasi (contoh 15m), tapi isinya %q", key, value)
	}
	return duration
}
