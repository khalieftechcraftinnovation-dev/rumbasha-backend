// Program untuk mengisi data dummy (development & demo).
//
// Pemakaian (dari folder project):
//
//	make seed     (atau: go run ./cmd/seed)
//
// File SQL yang dijalankan (urut), bisa juga dibuka di DBeaver:
//  1. dummy_data.sql  → cabang, akun utama tiap role, paket, coin (hanya sekali)
//  2. dummy_users.sql → akun tambahan sampai total 55 user (aman diulang)
//  3. dummy_subjects.sql → 5 mapel + hierarki Matematika SD Kelas 1-6: bidang materi → bab → sub bab → topik (aman diulang)
package main

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/config"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/database"
)

// Isi file SQL dibundel ke dalam program lewat //go:embed.
var (
	//go:embed dummy_data.sql
	dummyDataSQL string

	//go:embed dummy_users.sql
	dummyUsersSQL string

	//go:embed dummy_subjects.sql
	dummySubjectsSQL string
)

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

	// 1. dummy_data.sql: tidak aman diulang, jadi hanya dijalankan kalau akun superadmin belum ada.
	var count int64
	if err := db.Raw(`SELECT COUNT(*) FROM users WHERE username = 'superadmin'`).Scan(&count).Error; err != nil {
		log.Fatalf("gagal mengecek data: %v (sudah menjalankan make migrate-up?)", err)
	}
	if count == 0 {
		if err := db.Exec(dummyDataSQL).Error; err != nil {
			log.Fatalf("gagal mengisi dummy_data.sql: %v", err)
		}
		log.Println("dummy_data.sql berhasil diisi")
	} else {
		log.Println("dummy_data.sql sudah pernah diisi, dilewati")
	}

	// 2. dummy_users.sql: aman diulang (username yang sudah ada dilewati).
	if err := db.Exec(dummyUsersSQL).Error; err != nil {
		log.Fatalf("gagal mengisi dummy_users.sql: %v", err)
	}
	log.Println("dummy_users.sql berhasil diisi")

	// 3. dummy_subjects.sql: aman diulang (data yang sudah ada dilewati).
	if err := db.Exec(dummySubjectsSQL).Error; err != nil {
		log.Fatalf("gagal mengisi dummy_subjects.sql: %v (sudah menjalankan make migrate-up?)", err)
	}
	log.Println("dummy_subjects.sql berhasil diisi")

	// Ringkasan jumlah data mapel per tingkat.
	type typeCount struct {
		Type  string
		Total int
	}
	var subjectCounts []typeCount
	db.Raw(`
		SELECT type, COUNT(*) AS total
		FROM master_data_items
		WHERE type IN ('subject', 'subject_field', 'chapter', 'subchapter', 'topic')
		GROUP BY type
		ORDER BY MIN(CASE type WHEN 'subject' THEN 1 WHEN 'subject_field' THEN 2
		                       WHEN 'chapter' THEN 3 WHEN 'subchapter' THEN 4 ELSE 5 END)`).
		Scan(&subjectCounts)

	// Ringkasan jumlah user per role.
	type roleCount struct {
		Role  string
		Total int
	}
	var counts []roleCount
	db.Raw(`SELECT role, COUNT(*) AS total FROM users GROUP BY role ORDER BY role`).Scan(&counts)

	fmt.Printf("\nData dummy di database %s (password semua: password123)\n", cfg.DBName)
	for _, c := range counts {
		fmt.Printf("  %-12s %d user\n", c.Role, c.Total)
	}

	fmt.Println("\nMaster data mapel (contoh lengkap: Matematika SD Kelas 1-6)")
	for _, c := range subjectCounts {
		fmt.Printf("  %-14s %d data\n", c.Type, c.Total)
	}
	fmt.Println(`
Akun utama untuk dicoba:
  superadmin      super_admin
  admin.sbytimur  admin  (Surabaya Timur)
  tutor.rina      tutor  (Surabaya Timur)
  siswa.budi      siswa SMP Kelas 7  - paket OFFLINE Reguler SMP aktif bulan ini, 100 coin
  siswa.citra     siswa SD Kelas 5   - paket ONLINE Reguler SD aktif + akses khusus RUMATH
  siswa.dimas     siswa SMA Kelas 10 - paket ONLINE sudah expired kemarin
  siswa.eka       siswa PraSD        - akun NONAKTIF (login ditolak)`)
}
