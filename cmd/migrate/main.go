// Program kecil untuk menjalankan migration database.
//
// Pemakaian (dari folder project):
//
//	go run ./cmd/migrate up           → jalankan semua migration yang belum
//	go run ./cmd/migrate down         → batalkan 1 migration terakhir
//	go run ./cmd/migrate version      → lihat versi migration saat ini
//	go run ./cmd/migrate force 1      → paksa versi (hanya kalau status "dirty")
//
// Atau lewat Makefile: make migrate-up, make migrate-down, make migrate-version.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("pemakaian: go run ./cmd/migrate [up|down|version|force VERSION]")
		os.Exit(1)
	}

	cfg := config.Load()
	command := os.Args[1]

	switch command {
	case "up":
		if err := database.MigrateUp(cfg); err != nil {
			log.Fatalf("migrate up gagal: %v", err)
		}
		log.Println("migrate up selesai")

	case "down":
		if err := database.MigrateDown(cfg); err != nil {
			log.Fatalf("migrate down gagal: %v", err)
		}
		log.Println("migrate down selesai (1 migration dibatalkan)")

	case "version":
		version, dirty, err := database.MigrateVersion(cfg)
		if err != nil {
			log.Fatalf("gagal membaca versi: %v", err)
		}
		log.Printf("versi migration: %d (dirty: %v)", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("pemakaian: go run ./cmd/migrate force VERSION")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("VERSION harus angka: %v", err)
		}
		if err := database.MigrateForce(cfg, version); err != nil {
			log.Fatalf("migrate force gagal: %v", err)
		}
		log.Printf("versi migration dipaksa ke %d", version)

	default:
		log.Fatalf("perintah tidak dikenal: %q (pilih up, down, version, atau force)", command)
	}
}
