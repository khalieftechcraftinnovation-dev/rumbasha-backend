// Package testutil berisi helper yang hanya dipakai oleh file _test.go.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
)

// SetupTestDB menyiapkan database KHUSUS TEST (default: rumbasha_test, diatur
// lewat DB_NAME_TEST di .env), supaya data development tidak ikut terhapus.
//
// Yang dilakukan: konek → jalankan migration → kosongkan semua tabel.
// Kalau database test belum dibuat, test di-skip (bukan gagal).
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Saat test berjalan, folder aktif adalah folder package (misalnya internal/router),
	// jadi file .env di root project perlu dicari dan dibaca secara manual.
	loadEnvFromProjectRoot()

	cfg := config.Load()
	cfg.DBName = os.Getenv("DB_NAME_TEST")
	if cfg.DBName == "" {
		cfg.DBName = "rumbasha_test"
	}
	cfg.DBDebug = false

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("test di-skip, database test tidak tersedia: %v", err)
	}
	t.Cleanup(func() { database.Close(db) })

	if err := database.MigrateUp(cfg); err != nil {
		t.Fatalf("migration database test gagal: %v", err)
	}

	// Kosongkan semua tabel supaya setiap test mulai dari data kosong.
	// Tambahkan tabel baru di sini setiap ada migration baru.
	// JANGAN masukkan master_data_items: isinya data awal dari migration
	// (jenjang & kelas) yang tidak dibuat ulang.
	err = db.Exec(`
		TRUNCATE
			subjects,
			branches,
			users,
			refresh_tokens,
			students,
			subscription_plans,
			student_subscriptions,
			student_feature_overrides,
			coin_packages,
			coin_transactions,
			daily_quiz_questions,
			daily_quizzes,
			quiz_attempts,
			quiz_questions,
			quizzes,
			questions,
			media_assets
		RESTART IDENTITY`).Error
	if err != nil {
		t.Fatalf("gagal mengosongkan tabel: %v", err)
	}

	return db
}

// loadEnvFromProjectRoot mencari folder yang berisi go.mod (root project),
// lalu membaca file .env di sana.
func loadEnvFromProjectRoot() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return // sudah sampai root drive, go.mod tidak ditemukan
		}
		dir = parent
	}
}
