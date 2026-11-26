// Package landingpage (model) berisi struct data untuk fitur Landing Page (BRD 3.19).
//
// Di package lain, import dengan alias landingpagemodel:
//
//	landingpagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/site/landingpage"
package landingpage

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Page adalah 1 halaman (tab) Landing Page beserta info publish terakhir.
type Page struct {
	Code            string
	Name            string
	PublishedAt     *time.Time // NULL = belum pernah dipublish
	PublishedBy     *string
	PublishedByName *string
	PendingCount    int // jumlah perubahan yang belum dipublish
}

// Section adalah 1 section versi DRAFT (yang sedang diubah admin).
type Section struct {
	Key           string
	Name          string
	Content       json.RawMessage // object JSON, bentuknya ditentukan FE per section
	IsVisible     bool
	SortOrder     int
	UpdatedBy     *string
	UpdatedByName *string
	UpdatedAt     time.Time
}

// Change adalah 1 perubahan yang belum dipublish (panel "Belum dipublish").
type Change struct {
	SectionKey    *string // NULL = perubahan halaman (contoh urutan section)
	Description   string
	ChangedBy     *string
	ChangedByName *string
	ChangedAt     time.Time
}

// PublishedSection adalah 1 section di snapshot halaman publik.
// Tag json = isi kolom landing_pages.published_content.
type PublishedSection struct {
	Key     string          `json:"key"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

// PublishedPage adalah isi halaman publik (hasil publish terakhir).
type PublishedPage struct {
	Code        string
	Name        string
	Sections    []PublishedSection // kosong kalau belum pernah dipublish
	PublishedAt *time.Time
}

// PublishResult adalah hasil tombol Publish.
type PublishResult struct {
	PublishedAt time.Time
	Changes     []string // perubahan yang ikut dipublish
}

var (
	ErrPageNotFound    = errors.New("halaman landing page tidak ditemukan")
	ErrSectionNotFound = errors.New("section landing page tidak ditemukan")
	// ErrNothingToPublish: halaman sudah pernah dipublish dan tidak ada perubahan baru.
	ErrNothingToPublish = errors.New("tidak ada perubahan yang perlu dipublish")
)

// FieldError menjelaskan kesalahan pada 1 field request.
type FieldError struct {
	Field, Message string
}

// ValidationError berisi kesalahan isi request. Handler mengubahnya menjadi response 422.
type ValidationError struct {
	Fields []FieldError
}

// Error membuat ValidationError memenuhi interface error (teksnya tercatat di log API).
func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi landing page gagal: " + strings.Join(parts, "; ")
}
