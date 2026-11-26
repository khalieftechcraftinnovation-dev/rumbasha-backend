// Package image (model) berisi struct data untuk fitur Image Library (tabel media_assets).
//
// Di package lain, import dengan alias imagemodel:
//
//	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/image"
package image

import (
	"fmt"
	"strings"
	"time"
)

// Kategori gambar. Nilainya sama persis dengan isi kolom media_assets.category.
const (
	CategoryQuestion = "question" // gambar soal & opsi jawaban (boleh punya hierarki mapel)
	CategoryBanner   = "banner"
	CategoryLanding  = "landing" // landing page
)

// Image mewakili satu baris di tabel "media_assets".
type Image struct {
	ID          string
	Category    string
	Name        string
	FileName    string // nama file asli saat diunggah
	StoragePath string // lokasi relatif di STORAGE_DIR, contoh images/question/2026/11/<uuid>.png
	URL         string // link lengkap untuk FE
	MimeType    string
	SizeBytes   int64
	Width       int
	Height      int

	SubjectID      *string
	SubjectFieldID *string
	ChapterID      *string
	SubchapterID   *string
	TopicID        *string

	UploadedBy string
	CreatedAt  time.Time
}

// HierarchyItem adalah kolom master_data_items yang dibutuhkan untuk mengecek hierarki mapel gambar.
type HierarchyItem struct {
	ID        string
	Type      string
	ParentID  *string
	SubjectID *string
	Status    string
}

// DeleteResult adalah hasil percobaan hapus banyak gambar (lihat repository DeleteMany).
type DeleteResult struct {
	Deleted      bool              // true = semua terhapus
	FoundIDs     map[string]bool   // ID yang ada di database
	UsedBy       map[string]string // ID gambar → kode soal aktif yang memakainya, contoh "Q-1, Q-5"
	StoragePaths []string          // lokasi file yang perlu dihapus dari disk (kalau Deleted)
}

// FieldError menjelaskan kesalahan pada satu field request, contoh {"files[1]", "harus gambar ..."}.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError berisi semua kesalahan request. Handler mengubahnya menjadi response 422.
type ValidationError struct {
	Fields []FieldError
}

// Error membuat ValidationError memenuhi interface error (teksnya tercatat di log API).
func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return fmt.Sprintf("validasi gambar gagal: %s", strings.Join(parts, "; "))
}
