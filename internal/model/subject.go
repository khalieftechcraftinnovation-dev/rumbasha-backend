// Package model berisi struct data utama (entity) aplikasi.
// Satu struct model mewakili satu tabel di database.
// Untuk bentuk JSON yang dikirim ke client, lihat package dto.
package model

import (
	"errors"
	"time"
)

// Subject mewakili satu baris di tabel "subjects" (mata pelajaran).
//
// GORM otomatis mencocokkan nama field ke nama kolom:
// ID → id, Code → code, CreatedAt → created_at, dan seterusnya.
//
// Catatan Go: nama yang diawali huruf BESAR (Subject, Code) bersifat public
// dan bisa dipakai package lain. Huruf kecil berarti private.
type Subject struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName memberi tahu GORM nama tabel untuk model ini.
func (Subject) TableName() string {
	return "subjects"
}

// Daftar error untuk Subject. Repository/service mengembalikan error ini,
// lalu handler mengubahnya menjadi HTTP status (404, 409, ...).
var (
	ErrSubjectNotFound  = errors.New("subject tidak ditemukan")
	ErrSubjectCodeTaken = errors.New("kode subject sudah digunakan")
)
