// Package masterdata (model) berisi struct data menu Master Data di panel admin: mapel, bidang
// materi, bab, sub bab, topik, kelas, sekolah (tabel master_data_items) dan cabang (tabel branches).
//
// Di package lain, import dengan alias masterdatamodel:
//
//	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/masterdata"
package masterdata

import (
	"errors"
	"strings"
	"time"
)

// Type master data. Nilainya sama dengan kolom master_data_items.type.
const (
	TypeEducationLevel = "education_level"
	TypeGrade          = "grade"
	TypeSubject        = "subject"
	TypeSubjectField   = "subject_field"
	TypeChapter        = "chapter"
	TypeSubchapter     = "subchapter"
	TypeTopic          = "topic"
	TypeQuestionType   = "question_type"
	TypeQuizType       = "quiz_type"
	TypeDifficulty     = "difficulty"
	TypeSchool         = "school"
)

// TypeRule = aturan 1 type: label, induknya, dan apakah admin boleh menambah / menghapus.
type TypeRule struct {
	Label      string
	ParentType string // "" = tanpa induk
	Manageable bool   // true = admin boleh menambah & menghapus; false = hanya ubah nama & urutan
	Curriculum bool   // true = punya mapel, jenjang, kelas (bidang materi s/d topik)
	Rank       int    // urutan tampil di tab "Semua"
}

// Types = semua type master data.
var Types = map[string]TypeRule{
	TypeSubject:        {Label: "Mapel", Manageable: true, Rank: 1},
	TypeSubjectField:   {Label: "Bidang materi", ParentType: TypeSubject, Manageable: true, Curriculum: true, Rank: 2},
	TypeChapter:        {Label: "Bab", ParentType: TypeSubjectField, Manageable: true, Curriculum: true, Rank: 3},
	TypeSubchapter:     {Label: "Sub bab", ParentType: TypeChapter, Manageable: true, Curriculum: true, Rank: 4},
	TypeTopic:          {Label: "Topik", ParentType: TypeSubchapter, Manageable: true, Curriculum: true, Rank: 5},
	TypeGrade:          {Label: "Kelas", ParentType: TypeEducationLevel, Manageable: true, Rank: 6},
	TypeSchool:         {Label: "Sekolah", Manageable: true, Rank: 7},
	TypeEducationLevel: {Label: "Jenjang", Rank: 8},
	TypeQuestionType:   {Label: "Tipe soal", Rank: 9},
	TypeQuizType:       {Label: "Tipe quiz", Rank: 10},
	TypeDifficulty:     {Label: "Level", Rank: 11},
}

// ManageableTypes = type yang tampil di tab "Semua" (yang dikelola admin), urut Rank.
var ManageableTypes = []string{TypeSubject, TypeSubjectField, TypeChapter, TypeSubchapter, TypeTopic, TypeGrade, TypeSchool}

// Status data.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Usage = berapa data lain yang memakai 1 item (kolom "Isi" & syarat hapus).
type Usage struct {
	Children  int // item anak (contoh bab di bidang materi; bidang materi yang memakai kelas)
	Questions int // soal
	Modules   int // modul pembelajaran (PDF)
	Quizzes   int // quiz Uji Kom / Tryout
	Images    int // gambar Image Library
	Users     int // akun (kelas siswa)
}

// Total = jumlah semua pemakaian.
func (u Usage) Total() int {
	return u.Children + u.Questions + u.Modules + u.Quizzes + u.Images + u.Users
}

// Item adalah 1 data master (tabel master_data_items) + nama induk, letak, dan pemakaiannya.
type Item struct {
	ID             string
	Type           string
	Code           *string
	Name           string
	Parent         *Ref
	Subject        *Ref // bidang materi s/d topik
	EducationLevel *Ref // bidang materi s/d topik; kelas = jenjang induknya
	Grade          *Ref // bidang materi s/d topik
	SortOrder      int
	Status         string
	IsSystem       bool
	Usage          Usage
	CreatedBy      *Ref
	UpdatedBy      *Ref
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ListFilter adalah filter list Master Data. Field kosong = tidak difilter.
type ListFilter struct {
	Types            []string // kosong = ManageableTypes
	Search           string   // nama / kode
	Status           string
	ParentID         string
	SubjectID        string
	EducationLevelID string
	GradeID          string
	Page, PerPage    int
}

// NewItem adalah data baru yang siap disimpan (letak sudah disalin dari induk).
type NewItem struct {
	Type             string
	Code             *string
	Name             string
	ParentID         *string
	SubjectID        *string
	EducationLevelID *string
	GradeID          *string
	SortOrder        *int // nil = urutan berikutnya di antara saudaranya
	Status           string
}

// ItemUpdate adalah isi baru 1 data (ganti penuh untuk field yang boleh diubah).
type ItemUpdate struct {
	ID        string
	Code      *string
	Name      string
	SortOrder int
	Status    string
}

// Parent adalah item induk yang dicek saat menambah data.
type Parent struct {
	ID, Type, Name   string
	Status           string
	ParentID         *string // induk dari induk (jenjang untuk kelas)
	SubjectID        *string
	EducationLevelID *string
	GradeID          *string
}

// Counts = angka tab menu Master Data.
type Counts map[string]int

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
	return "validasi master data gagal: " + strings.Join(parts, "; ")
}

var (
	ErrNotFound = errors.New("data master tidak ditemukan")
	// ErrNameTaken: nama kembar dalam type, induk, dan kelas yang sama (index unik
	// master_data_items_type_parent_name_unique / branches_name_unique).
	ErrNameTaken = errors.New("nama sudah dipakai")
	// ErrCodeTaken: kode kembar dalam type yang sama (index unik master_data_items_type_code_unique).
	ErrCodeTaken = errors.New("kode sudah dipakai")
	// ErrBranchNotFound: cabang tidak ada / sudah dihapus.
	ErrBranchNotFound = errors.New("cabang tidak ditemukan")
)
