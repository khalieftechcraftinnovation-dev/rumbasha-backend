package image

import "errors"

// Urutan galeri.
const (
	SortNewest  = "newest"  // terbaru dulu (default)
	SortOldest  = "oldest"  // terlama dulu
	SortName    = "name"    // nama A-Z
	SortLargest = "largest" // ukuran file terbesar dulu
)

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Hierarchy = nama hierarki mapel gambar soal (nil = tidak diisi).
type Hierarchy struct {
	Subject, SubjectField, Chapter, Subchapter, Topic *Ref
}

// ListItem adalah 1 gambar di galeri: data gambar + nama hierarki, pengunggah, dan soal yang memakainya.
type ListItem struct {
	Image
	Hierarchy Hierarchy
	Uploader  *Ref     // nil kalau akun pengunggah sudah tidak ada
	UsedBy    []string // kode soal aktif yang memakai gambar (gambar soal / opsi jawaban)
}

// ListFilter adalah filter galeri. Field kosong = tidak difilter.
type ListFilter struct {
	Category                                                    string
	Search                                                      string // nama gambar / nama file
	SubjectID, SubjectFieldID, ChapterID, SubchapterID, TopicID string
	Sort                                                        string
	Page, PerPage                                               int
}

// CategoryCounts = jumlah gambar per kategori (angka tab / filter Kategori).
type CategoryCounts struct {
	Question, Banner, Landing int
}

// ErrImageNotFound: gambar tidak ada.
var ErrImageNotFound = errors.New("gambar tidak ditemukan")
