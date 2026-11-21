// Program untuk mengisi data dummy (development & demo).
//
// Pemakaian (dari folder project):
//
//	make seed     (atau: go run ./cmd/seed)
//
// Isinya ada di dummy_data.sql (folder yang sama), yang juga bisa dibuka di DBeaver.
package main

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
)

// dummySQL berisi seluruh isi file dummy_data.sql.
// Baris //go:embed membuat isi file itu ikut dibundel ke dalam program.
//
//go:embed dummy_data.sql
var dummySQL string

func main() {
	cfg := config.Load()
	cfg.DBDebug = false // isi SQL-nya panjang, tidak perlu dicetak

	// Akun demo tidak boleh ada di production (BRD 10.2).
	if cfg.IsProduction() {
		log.Fatal("data dummy tidak boleh dijalankan di production (APP_ENV=production)")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close(db)

	// Supaya aman dijalankan berkali-kali: kalau akun superadmin sudah ada, berhenti.
	var count int64
	if err := db.Raw(`SELECT COUNT(*) FROM users WHERE username = 'superadmin'`).Scan(&count).Error; err != nil {
		log.Fatalf("gagal mengecek data: %v (sudah menjalankan make migrate-up?)", err)
	}
	if count > 0 {
		log.Println("data dummy sudah ada, tidak ada yang diubah")
		return
	}

	if err := db.Exec(dummySQL).Error; err != nil {
		log.Fatalf("gagal mengisi data dummy: %v", err)
	}

	log.Printf("data dummy berhasil diisi ke database %s", cfg.DBName)
	fmt.Println(`
Akun dummy (password semua: password123)
  superadmin      super_admin
  admin.sbytimur  admin  (Surabaya Timur)
  tutor.rina      tutor  (Surabaya Timur)
  siswa.budi      siswa SMP Kelas 7  - paket OFFLINE Reguler SMP aktif bulan ini, 100 coin
  siswa.citra     siswa SD Kelas 5   - paket ONLINE Reguler SD aktif + akses khusus RUMATH
  siswa.dimas     siswa SMA Kelas 10 - paket ONLINE sudah expired kemarin
  siswa.eka       siswa PraSD        - akun NONAKTIF (login ditolak)`)
}
