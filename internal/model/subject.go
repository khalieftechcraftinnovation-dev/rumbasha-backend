// Package model berisi struct data utama (entity) aplikasi.
// Model dipakai oleh service dan repository. Untuk bentuk JSON yang dikirim
// ke client, lihat package dto.
package model

import (
	"errors"
	"time"
)

// Subject adalah data mata pelajaran, misalnya "Matematika".
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

// Daftar error untuk Subject. Repository/service mengembalikan error ini,
// lalu handler mengubahnya menjadi HTTP status (404, 409, ...).
var (
	ErrSubjectNotFound  = errors.New("subject tidak ditemukan")
	ErrSubjectCodeTaken = errors.New("kode subject sudah digunakan")
)
