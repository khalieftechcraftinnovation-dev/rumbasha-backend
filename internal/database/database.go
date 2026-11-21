// Package database mengatur koneksi ke PostgreSQL dan menjalankan migration.
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
)

// Connect membuka koneksi ke PostgreSQL memakai GORM.
// Dipanggil sekali di main.go, lalu *gorm.DB yang sama dipakai semua repository.
func Connect(cfg config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn // hanya tampilkan query lambat & error
	if cfg.DBDebug {
		logLevel = logger.Info // tampilkan SEMUA query SQL (berguna saat debug)
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),

		// Ubah error khusus PostgreSQL menjadi error GORM yang mudah dicek,
		// misalnya pelanggaran UNIQUE menjadi gorm.ErrDuplicatedKey.
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal konek ke database %s di %s:%s: %w", cfg.DBName, cfg.DBHost, cfg.DBPort, err)
	}

	// Pengaturan connection pool: berapa banyak koneksi yang boleh dibuka sekaligus.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

// Close menutup semua koneksi database.
func Close(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
