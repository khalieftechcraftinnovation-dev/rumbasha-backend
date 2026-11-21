// Package migrations berisi file SQL untuk membuat dan mengubah tabel database.
//
// Aturan penamaan file (wajib berpasangan):
//
//	000001_create_subjects_table.up.sql    → dijalankan saat migrate up
//	000001_create_subjects_table.down.sql  → dijalankan saat migrate down (rollback)
//
// Untuk perubahan baru, buat pasangan file dengan nomor berikutnya (000002_...).
// Jangan mengubah file migration yang sudah pernah dijalankan; buat file baru.
package migrations

import "embed"

// FS berisi semua file .sql di folder ini.
// Baris //go:embed di bawah membuat file-file tersebut ikut dibundel ke dalam
// program Go, jadi migration bisa dijalankan dari folder mana pun.
//
//go:embed *.sql
var FS embed.FS
