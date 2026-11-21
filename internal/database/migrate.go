package database

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // driver PostgreSQL untuk migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/migrations"
)

// Catatan: migrate mencatat versi migration yang sudah dijalankan di tabel
// schema_migrations (dibuat otomatis). Jangan ubah tabel itu secara manual.

// MigrateUp menjalankan semua migration yang belum pernah dijalankan.
func MigrateUp(cfg config.Config) error {
	m, err := newMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	// ErrNoChange artinya semua migration sudah dijalankan, jadi bukan error.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// MigrateDown membatalkan SATU migration terakhir.
func MigrateDown(cfg config.Config) error {
	m, err := newMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Steps(-1)
}

// MigrateForce menandai database berada di versi tertentu tanpa menjalankan SQL.
// Dipakai hanya kalau migration gagal di tengah jalan dan statusnya "dirty":
// perbaiki dulu tabelnya secara manual, lalu force ke versi terakhir yang benar.
func MigrateForce(cfg config.Config, version int) error {
	m, err := newMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Force(version)
}

// MigrateVersion mengembalikan versi migration saat ini dan status dirty-nya.
func MigrateVersion(cfg config.Config) (version uint, dirty bool, err error) {
	m, err := newMigrate(cfg)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil // belum ada migration yang dijalankan
	}
	return version, dirty, err
}

func newMigrate(cfg config.Config) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", source, cfg.DatabaseURL())
}
