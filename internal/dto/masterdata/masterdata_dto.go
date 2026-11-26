// Package masterdata (dto) berisi bentuk request dan response JSON menu Master Data & Cabang.
//
// Di package lain, import dengan alias masterdatadto:
//
//	masterdatadto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/masterdata"
package masterdata

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/masterdata"
)

// ListQuery adalah query string GET /api/v1/master-data. Semua opsional.
//
//	?type=chapter&subject_id=...&grade_id=...&search=pecahan&status=active&page=1&per_page=20
type ListQuery struct {
	// Tab: kosong = semua yang dikelola admin (mapel, bidang materi, bab, sub bab, topik, kelas, sekolah).
	Type             string `form:"type" binding:"omitempty,oneof=subject subject_field chapter subchapter topic grade school education_level question_type quiz_type difficulty"`
	Search           string `form:"search" binding:"omitempty,max=100"` // nama / kode
	Status           string `form:"status" binding:"omitempty,oneof=active inactive"`
	ParentID         string `form:"parent_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	Page             int    `form:"page" binding:"omitempty,min=1"`
	PerPage          int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// CreateRequest adalah body POST /api/v1/master-data. Contoh:
//
//	{"type": "subject", "name": "IPA", "code": "IPA"}
//	{"type": "subject_field", "name": "Bilangan", "parent_id": "<mapel>", "grade_id": "<kelas>"}
//	{"type": "chapter", "name": "Pecahan", "parent_id": "<bidang materi>"}
//	{"type": "grade", "name": "Kelas 13", "parent_id": "<jenjang>"}
//	{"type": "school", "name": "SMPN 1 Surabaya"}
type CreateRequest struct {
	Type string  `json:"type" binding:"required,oneof=subject subject_field chapter subchapter topic grade school"`
	Name string  `json:"name" binding:"required,max=150"`
	Code *string `json:"code" binding:"omitempty,max=30"` // opsional, unik per type
	// Induk: wajib untuk bidang materi (mapel), bab (bidang materi), sub bab (bab), topik (sub bab),
	// kelas (jenjang); harus kosong untuk mapel & sekolah.
	ParentID *string `json:"parent_id" binding:"omitempty,uuid"`
	// Kelas bidang materi (wajib untuk bidang materi saja; jenjang = jenjang kelas itu). Bab s/d topik
	// mengikuti induknya.
	GradeID   *string `json:"grade_id" binding:"omitempty,uuid"`
	SortOrder *int    `json:"sort_order" binding:"omitempty,min=0,max=100000"`  // kosong = urutan terakhir
	Status    string  `json:"status" binding:"omitempty,oneof=active inactive"` // default active
}

// UpdateRequest adalah body PUT /api/v1/master-data/:id (ganti penuh). Type, induk, dan letak tidak
// bisa diubah. Data sistem: hanya name & sort_order (code & status harus sama dengan sebelumnya / dikosongkan).
type UpdateRequest struct {
	Name      string  `json:"name" binding:"required,max=150"`
	Code      *string `json:"code" binding:"omitempty,max=30"`
	SortOrder *int    `json:"sort_order" binding:"required,min=0,max=100000"`
	Status    string  `json:"status" binding:"required,oneof=active inactive"`
}

// DeleteRequest adalah body DELETE /api/v1/master-data dan /api/v1/branches (soft delete).
type DeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteResponse berisi ID yang dihapus.
type DeleteResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *masterdatamodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// UsageResponse = kolom "Isi": berapa data lain yang memakai item ini.
type UsageResponse struct {
	Children  int `json:"children"`  // item anak (bab di bidang materi, bidang materi di kelas, ...)
	Questions int `json:"questions"` // soal
	Modules   int `json:"modules"`   // modul PDF
	Quizzes   int `json:"quizzes"`   // quiz Uji Kom / Tryout
	Images    int `json:"images"`    // gambar
	Users     int `json:"users"`     // akun siswa (kelas)
}

// ItemResponse adalah 1 data master.
type ItemResponse struct {
	ID             string        `json:"id"`
	Type           string        `json:"type"`
	TypeLabel      string        `json:"type_label"` // "Mapel", "Bidang materi", ...
	Code           *string       `json:"code"`
	Name           string        `json:"name"`
	Parent         *RefResponse  `json:"parent"`          // kolom "Induk"
	Subject        *RefResponse  `json:"subject"`         // bidang materi s/d topik
	EducationLevel *RefResponse  `json:"education_level"` // bidang materi s/d topik, kelas
	Grade          *RefResponse  `json:"grade"`           // bidang materi s/d topik
	SortOrder      int           `json:"sort_order"`
	Status         string        `json:"status"`
	IsSystem       bool          `json:"is_system"` // true = data bawaan: tidak bisa dihapus, hanya nama & urutan
	Usage          UsageResponse `json:"usage"`
	CanDelete      bool          `json:"can_delete"` // bukan data sistem dan tidak dipakai data lain
	CreatedBy      *RefResponse  `json:"created_by"` // null = data awal
	UpdatedBy      *RefResponse  `json:"updated_by"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// ToItemResponse mengubah 1 data master menjadi response.
func ToItemResponse(i masterdatamodel.Item) ItemResponse {
	u := i.Usage
	return ItemResponse{
		ID: i.ID, Type: i.Type, TypeLabel: masterdatamodel.Types[i.Type].Label, Code: i.Code, Name: i.Name,
		Parent: toRef(i.Parent), Subject: toRef(i.Subject), EducationLevel: toRef(i.EducationLevel), Grade: toRef(i.Grade),
		SortOrder: i.SortOrder, Status: i.Status, IsSystem: i.IsSystem,
		Usage:     UsageResponse{Children: u.Children, Questions: u.Questions, Modules: u.Modules, Quizzes: u.Quizzes, Images: u.Images, Users: u.Users},
		CanDelete: !i.IsSystem && u.Total() == 0,
		CreatedBy: toRef(i.CreatedBy), UpdatedBy: toRef(i.UpdatedBy),
		CreatedAt: i.CreatedAt.In(clock.Jakarta), UpdatedAt: i.UpdatedAt.In(clock.Jakarta),
	}
}

// ToItemResponses mengubah daftar data master menjadi response. Kosong = [].
func ToItemResponses(items []masterdatamodel.Item) []ItemResponse {
	result := make([]ItemResponse, 0, len(items))
	for _, i := range items {
		result = append(result, ToItemResponse(i))
	}
	return result
}

// CountsResponse = angka tab menu Master Data. all = jumlah semua tab (termasuk cabang).
type CountsResponse struct {
	All          int `json:"all"`
	Subject      int `json:"subject"`
	SubjectField int `json:"subject_field"`
	Chapter      int `json:"chapter"`
	Subchapter   int `json:"subchapter"`
	Topic        int `json:"topic"`
	Grade        int `json:"grade"`
	School       int `json:"school"`
	Branch       int `json:"branch"`
}

// ToCountsResponse mengubah jumlah per type + cabang menjadi response.
func ToCountsResponse(c masterdatamodel.Counts, branches int) CountsResponse {
	r := CountsResponse{
		Subject: c[masterdatamodel.TypeSubject], SubjectField: c[masterdatamodel.TypeSubjectField],
		Chapter: c[masterdatamodel.TypeChapter], Subchapter: c[masterdatamodel.TypeSubchapter], Topic: c[masterdatamodel.TypeTopic],
		Grade: c[masterdatamodel.TypeGrade], School: c[masterdatamodel.TypeSchool], Branch: branches,
	}
	r.All = r.Subject + r.SubjectField + r.Chapter + r.Subchapter + r.Topic + r.Grade + r.School + r.Branch
	return r
}

// ---------------------------------------------------------------------------
// Cabang
// ---------------------------------------------------------------------------

// BranchListQuery adalah query string GET /api/v1/branches.
type BranchListQuery struct {
	Search string `form:"search" binding:"omitempty,max=100"`
	Status string `form:"status" binding:"omitempty,oneof=active inactive"`
}

// BranchRequest adalah body POST /api/v1/branches dan PUT /api/v1/branches/:id (ganti penuh).
type BranchRequest struct {
	Name   string `json:"name" binding:"required,max=100"`
	Status string `json:"status" binding:"omitempty,oneof=active inactive"` // default active
}

// BranchResponse adalah 1 cabang.
type BranchResponse struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
	Users     int          `json:"users"`      // akun di cabang ini
	CanDelete bool         `json:"can_delete"` // tidak ada akun yang memakai
	CreatedBy *RefResponse `json:"created_by"`
	UpdatedBy *RefResponse `json:"updated_by"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ToBranchResponse mengubah 1 cabang menjadi response.
func ToBranchResponse(b masterdatamodel.Branch) BranchResponse {
	return BranchResponse{
		ID: b.ID, Name: b.Name, Status: b.Status, Users: b.Users, CanDelete: b.Users == 0,
		CreatedBy: toRef(b.CreatedBy), UpdatedBy: toRef(b.UpdatedBy),
		CreatedAt: b.CreatedAt.In(clock.Jakarta), UpdatedAt: b.UpdatedAt.In(clock.Jakarta),
	}
}

// ToBranchResponses mengubah daftar cabang menjadi response. Kosong = [].
func ToBranchResponses(items []masterdatamodel.Branch) []BranchResponse {
	result := make([]BranchResponse, 0, len(items))
	for _, b := range items {
		result = append(result, ToBranchResponse(b))
	}
	return result
}
