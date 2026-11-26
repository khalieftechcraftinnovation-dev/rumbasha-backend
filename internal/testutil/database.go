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

// SetupTestDB menyiapkan database untuk test: database development yang sama dengan aplikasi
// (DB_NAME di .env, contoh rumbasha).
//
// Yang dilakukan: konek → jalankan migration → KOSONGKAN semua tabel data (lihat TRUNCATE di
// bawah). Artinya data development (akun dummy, soal, quiz, dst.) TERHAPUS setiap kali test
// jalan; isi ulang dengan `make seed` setelah test.
// Kalau database tidak bisa dihubungi, test di-skip (bukan gagal). Di production (APP_ENV=production)
// test langsung dihentikan supaya data asli tidak terhapus.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Saat test berjalan, folder aktif adalah folder package (misalnya internal/router),
	// jadi file .env di root project perlu dicari dan dibaca secara manual.
	loadEnvFromProjectRoot()

	cfg := config.Load()
	if cfg.IsProduction() {
		t.Fatal("test tidak boleh dijalankan di production: test mengosongkan semua tabel (APP_ENV=production)")
	}
	cfg.DBDebug = false

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("test di-skip, database %s tidak tersedia: %v", cfg.DBName, err)
	}
	t.Cleanup(func() { database.Close(db) })

	if err := database.MigrateUp(cfg); err != nil {
		t.Fatalf("migration database gagal: %v", err)
	}

	// master_data_items tidak dikosongkan (lihat di bawah); kolom auditnya (id user, tanpa foreign key)
	// dikosongkan supaya tidak menunjuk ke user yang sebentar lagi dihapus.
	err = db.Exec(`
		UPDATE master_data_items SET created_by = NULL, updated_by = NULL, deleted_by = NULL
		WHERE created_by IS NOT NULL OR updated_by IS NOT NULL OR deleted_by IS NOT NULL`).Error
	if err != nil {
		t.Fatalf("gagal melepas audit master data: %v", err)
	}

	// Kosongkan semua tabel supaya setiap test mulai dari data kosong.
	// Tambahkan tabel baru di sini setiap ada migration baru.
	// JANGAN masukkan master_data_items: isinya data awal dari migration
	// (jenjang & kelas) yang tidak dibuat ulang.
	err = db.Exec(`
		TRUNCATE
			announcement_reads,
			announcement_grades,
			announcements,
			admin_tasks,
			branches,
			users,
			refresh_tokens,
			password_reset_requests,
			students,
			subscription_plans,
			student_subscriptions,
			student_feature_overrides,
			coin_packages,
			learning_module_opens,
			learning_module_plans,
			learning_modules,
			quiz_prize_awards,
			daily_quiz_prize_awards,
			daily_quiz_prize_periods,
			daily_quiz_settings,
			quiz_report_settings,
			coin_transactions,
			leaderboard_point_logs,
			leaderboard_entries,
			daily_quiz_questions,
			daily_quizzes,
			quiz_attempts,
			quiz_questions,
			quizzes,
			rumath_questions,
			rumath_sessions,
			questions,
			trial_quiz_attempts,
			trial_quiz_questions,
			landing_changes,
			landing_publish_logs,
			media_assets
		RESTART IDENTITY`).Error
	if err != nil {
		t.Fatalf("gagal mengosongkan tabel: %v", err)
	}

	// Pengaturan report quiz juga 1 baris (ambang 15%, rentang 60 hari).
	// ON CONFLICT: go test ./... menjalankan beberapa paket paralel di database yang sama.
	if err := db.Exec(`INSERT INTO quiz_report_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`).Error; err != nil {
		t.Fatalf("gagal mengisi pengaturan report quiz: %v", err)
	}
	// Pengaturan global Daily Quiz selalu 1 baris: isi ulang nilai awal (10/15/20, tanpa hadiah).
	if err := db.Exec(`INSERT INTO daily_quiz_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`).Error; err != nil {
		t.Fatalf("gagal mengisi pengaturan Daily Quiz: %v", err)
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
