// Package learningmodule (model) berisi struct data modul pembelajaran: materi belajar dalam
// file PDF, dengan letak (mapel → topik) dan paket akses yang membukanya.
//
// Di package lain, import dengan alias learningmodulemodel:
//
//	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learning/learningmodule"
package learningmodule

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Status modul: published = tampil di menu Modul siswa, draft = belum tampil.
const (
	StatusPublished = "published"
	StatusDraft     = "draft"
)

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Module adalah 1 modul pembelajaran (tabel learning_modules) + nama letak + paket aksesnya.
type Module struct {
	ID          string
	Name        string
	Description *string

	EducationLevel Ref
	Grade          Ref
	Subject        Ref
	SubjectField   Ref
	Chapter        *Ref
	Subchapter     *Ref
	Topic          *Ref

	FileName    string
	StoragePath string
	MimeType    string
	SizeBytes   int64
	PageCount   *int // jumlah halaman; nil = tidak diketahui

	Plans []Ref // paket akses yang membuka modul ini

	Status    string
	CreatedBy *Ref
	UpdatedBy *Ref
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Placement adalah letak modul yang sudah dicek (ID saja), siap disimpan.
type Placement struct {
	EducationLevelID, GradeID, SubjectID, SubjectFieldID string
	ChapterID, SubchapterID, TopicID                     *string
}

// Save adalah isi modul yang disimpan saat buat / ubah.
type Save struct {
	ID          string
	Name        string
	Description *string
	PageCount   *int // nil = dihitung dari PDF baru (kalau ada), atau tetap (ubah tanpa PDF)
	Placement
	Status  string
	PlanIDs []string
	// File baru (nil saat ubah tanpa mengganti PDF).
	File *File
	// Ubah tanpa PDF baru tapi letak berubah: file lama dipindah ke folder letak baru (lokasi baru).
	RelocatedPath string
	// Nama tiap tingkat letak (untuk folder file PDF).
	PathNames PathNames
}

// File adalah file PDF yang disimpan.
type File struct {
	FileName, StoragePath, MimeType string
	SizeBytes                       int64
}

// HierarchyItem adalah item master data letak modul (dicek saat buat / ubah).
type HierarchyItem struct {
	ID, Type, Status string
	Name             string
	LevelName        *string // nama jenjang (bidang materi)
	GradeName        *string // nama kelas (bidang materi)
	ParentID         *string
	EducationLevelID *string
	GradeID          *string
}

// ListFilter adalah filter list modul. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Search                                    string // nama modul
	EducationLevelID, GradeID, SubjectID      string
	SubjectFieldID, ChapterID, PlanID, Status string
	Page, PerPage                             int
}

// FieldError & ValidationError: kesalahan data request (dibalas 422).
type FieldError struct {
	Field, Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi modul gagal: " + strings.Join(parts, "; ")
}

var ErrModuleNotFound = errors.New("modul tidak ditemukan")

// ---------------------------------------------------------------------------
// Lokasi file PDF: folder mengikuti hierarki mapel
// ---------------------------------------------------------------------------

// PathNames adalah nama tiap tingkat letak modul, untuk menyusun folder file PDF.
type PathNames struct {
	Subject, Level, Grade, SubjectField string
	Chapter, Subchapter, Topic          string // kosong = tingkat tidak diisi
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug mengubah nama menjadi nama folder: huruf kecil, angka, dan "-" (contoh "Kelas 5" → "kelas-5").
func slug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "lainnya"
	}
	if len(s) > 60 {
		s = strings.TrimRight(s[:60], "-")
	}
	return s
}

// FilePath menyusun lokasi file PDF modul (relatif di STORAGE_DIR) mengikuti hierarki mapel:
//
//	modules/<mapel>/<jenjang>-<kelas>/<bidang materi>[/<bab>[/<sub bab>[/<topik>]]]/<id>.pdf
//	contoh: modules/matematika/sd-kelas-5/bilangan/pecahan/<id>.pdf
//
// Nama folder diambil saat upload / saat letak diubah (file ikut dipindah).
func FilePath(n PathNames, id string) string {
	parts := []string{"modules", slug(n.Subject), slug(n.Level + " " + n.Grade), slug(n.SubjectField)}
	for _, level := range []string{n.Chapter, n.Subchapter, n.Topic} {
		if level == "" {
			break
		}
		parts = append(parts, slug(level))
	}
	return strings.Join(parts, "/") + "/" + id + ".pdf"
}

// ---------------------------------------------------------------------------
// Modul untuk siswa
// ---------------------------------------------------------------------------

// StudentAccess adalah akses modul siswa hari ini.
type StudentAccess struct {
	EducationLevelID *string // jenjang akun siswa; nil = belum diisi (tidak ada modul)
	EducationLevel   *Ref
	Plan             *Ref // paket aktif hari ini; nil = tidak ada
	ModuleFeature    bool // fitur modul terbuka (dari paket atau akses khusus admin)
	// AllModules = true kalau fitur modul dibuka admin lewat akses khusus (bukan dari paket):
	// semua modul di jenjang siswa terbuka.
	AllModules bool
}

// CanOpen = true kalau siswa boleh membuka modul yang tersedia di paket-paket planIDs.
func (a StudentAccess) CanOpen(planIDs []string) bool {
	if !a.ModuleFeature {
		return false
	}
	if a.AllModules {
		return true
	}
	for _, id := range planIDs {
		if a.Plan != nil && id == a.Plan.ID {
			return true
		}
	}
	return false
}

// StudentFilter adalah filter list modul siswa. Field kosong ("") = tidak difilter.
type StudentFilter struct {
	EducationLevelID                   string // jenjang siswa (wajib, dari akun)
	GradeID, SubjectID, SubjectFieldID string
	ChapterID                          string
	Search                             string
	Access                             string // "" semua, "unlocked" bisa dibuka, "locked" terkunci
	PlanID                             string // paket aktif siswa ("" = tidak ada)
	AllModules                         bool
	ModuleFeature                      bool
	StudentID                          string
	Page, PerPage                      int
}

// StudentModule adalah 1 modul di list / buka modul siswa.
type StudentModule struct {
	Module
	Locked        bool       // true = di luar paket siswa (tidak bisa dibuka)
	LastOpenedAt  *time.Time // nil = belum pernah dibuka
	FirstOpenedAt *time.Time
	OpenCount     int
}

// LockedError dikembalikan saat siswa membuka modul di luar paketnya (dibalas 403).
type LockedError struct {
	Message string
	Plans   []Ref // paket yang membuka modul ini
}

func (e *LockedError) Error() string { return e.Message }
