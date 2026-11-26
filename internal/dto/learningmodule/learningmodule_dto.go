// Package learningmodule (dto) berisi bentuk request dan response modul pembelajaran (PDF).
//
// Di package lain, import dengan alias learningmoduledto:
//
//	learningmoduledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/learningmodule"
package learningmodule

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learningmodule"
)

// ModuleRequest adalah field teks form POST /api/v1/learning-modules dan PUT /api/v1/learning-modules/:id
// (multipart/form-data, ganti penuh). File PDF dikirim di field "file" (wajib saat buat, opsional
// saat ubah = ganti PDF). plan_ids dikirim berulang: plan_ids=<id1>&plan_ids=<id2>.
type ModuleRequest struct {
	Name        string  `form:"name" binding:"required,max=200"`
	Description *string `form:"description" binding:"omitempty,max=5000"`
	// Jumlah halaman PDF. Opsional: tidak diisi = dihitung otomatis dari file PDF.
	PageCount *int `form:"page_count" binding:"omitempty,min=1,max=10000"`
	// Letak: mapel & bidang materi wajib (jenjang & kelas diambil dari bidang materi);
	// bab, sub bab, topik opsional dan berurutan.
	SubjectID      string  `form:"subject_id" binding:"required,uuid"`
	SubjectFieldID string  `form:"subject_field_id" binding:"required,uuid"`
	ChapterID      *string `form:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID   *string `form:"subchapter_id" binding:"omitempty,uuid"`
	TopicID        *string `form:"topic_id" binding:"omitempty,uuid"`
	// Paket akses yang membuka modul ini (minimal 1).
	PlanIDs []string `form:"plan_ids" binding:"required,min=1,max=20,unique,dive,uuid"`
	// Tombol simpan: "published" (tampil di Modul siswa) atau "draft" (Simpan sebagai draf). Default published.
	Status string `form:"status" binding:"omitempty,oneof=draft published"`
}

// ListQuery adalah query string GET /api/v1/learning-modules. Semua opsional.
type ListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search           string `form:"search" binding:"omitempty,max=100"` // nama modul
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID   string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID        string `form:"chapter_id" binding:"omitempty,uuid"`
	PlanID           string `form:"plan_id" binding:"omitempty,uuid"` // modul yang tersedia di paket ini
	Status           string `form:"status" binding:"omitempty,oneof=draft published"`
}

// DeleteRequest adalah body DELETE /api/v1/learning-modules: hapus 1-100 modul (soft delete).
type DeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteResponse berisi ID modul yang dihapus.
type DeleteResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func optRef(r *learningmodulemodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// PlacementResponse adalah letak modul dengan nama. Tingkat yang tidak diisi = null.
type PlacementResponse struct {
	EducationLevel RefResponse  `json:"education_level"`
	Grade          RefResponse  `json:"grade"`
	Subject        RefResponse  `json:"subject"`
	SubjectField   RefResponse  `json:"subject_field"`
	Chapter        *RefResponse `json:"chapter"`
	Subchapter     *RefResponse `json:"subchapter"`
	Topic          *RefResponse `json:"topic"`
}

// FileResponse adalah info file PDF. url = API unduh (wajib login, kirim header Authorization).
type FileResponse struct {
	Name      string `json:"name"` // nama file asli
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	PageCount *int   `json:"page_count"` // null = tidak diketahui
	URL       string `json:"url"`
}

// ModuleResponse adalah data 1 modul.
type ModuleResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	Placement   PlacementResponse `json:"placement"`
	File        FileResponse      `json:"file"`
	Plans       []RefResponse     `json:"plans"` // paket akses
	Status      string            `json:"status"`
	CreatedBy   *RefResponse      `json:"created_by"`
	UpdatedBy   *RefResponse      `json:"updated_by"` // null = belum pernah diubah
	CreatedAt   time.Time         `json:"created_at"` // WIB
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ToModuleResponse mengubah 1 modul menjadi response. fileURL = link unduh PDF.
func ToModuleResponse(m learningmodulemodel.Module, fileURL string) ModuleResponse {
	plans := make([]RefResponse, 0, len(m.Plans))
	for _, p := range m.Plans {
		plans = append(plans, RefResponse(p))
	}
	return ModuleResponse{
		ID: m.ID, Name: m.Name, Description: m.Description,
		Placement: PlacementResponse{
			EducationLevel: RefResponse(m.EducationLevel), Grade: RefResponse(m.Grade),
			Subject: RefResponse(m.Subject), SubjectField: RefResponse(m.SubjectField),
			Chapter: optRef(m.Chapter), Subchapter: optRef(m.Subchapter), Topic: optRef(m.Topic),
		},
		File:      FileResponse{Name: m.FileName, MimeType: m.MimeType, SizeBytes: m.SizeBytes, PageCount: m.PageCount, URL: fileURL},
		Plans:     plans,
		Status:    m.Status,
		CreatedBy: optRef(m.CreatedBy), UpdatedBy: optRef(m.UpdatedBy),
		CreatedAt: m.CreatedAt.In(clock.Jakarta), UpdatedAt: m.UpdatedAt.In(clock.Jakarta),
	}
}

// ---------------------------------------------------------------------------
// Siswa
// ---------------------------------------------------------------------------

// StudentListQuery adalah query string GET /api/v1/student/learning-modules. Semua opsional.
// Jenjang selalu jenjang akun siswa (tidak bisa dipilih).
type StudentListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search         string `form:"search" binding:"omitempty,max=100"` // judul modul
	GradeID        string `form:"grade_id" binding:"omitempty,uuid"`  // kelas (di jenjang siswa)
	SubjectID      string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID      string `form:"chapter_id" binding:"omitempty,uuid"`
	// unlocked = hanya yang bisa dibuka, locked = hanya yang terkunci, kosong = semua.
	Access string `form:"access" binding:"omitempty,oneof=unlocked locked"`
}

// StudentAccessResponse = akses modul siswa hari ini (ditampilkan di atas list).
type StudentAccessResponse struct {
	EducationLevel *RefResponse `json:"education_level"` // jenjang akun; null = belum diisi (tidak ada modul)
	Plan           *RefResponse `json:"plan"`            // paket aktif hari ini; null = tidak ada
	ModuleFeature  bool         `json:"module_feature"`  // fitur modul terbuka
	AllModules     bool         `json:"all_modules"`     // akses khusus admin: semua modul di jenjang terbuka
}

// StudentModuleResponse adalah 1 modul untuk siswa. file.url hanya diisi kalau tidak terkunci.
type StudentModuleResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	Placement   PlacementResponse `json:"placement"`
	File        StudentFile       `json:"file"`
	Locked      bool              `json:"locked"` // true = di luar paket siswa
	Plans       []RefResponse     `json:"plans"`  // paket yang membuka modul (untuk ajakan upgrade)
	Opened      *OpenedResponse   `json:"opened"` // null = belum pernah dibuka
}

// StudentFile = info file PDF untuk siswa. url null kalau modul terkunci.
type StudentFile struct {
	SizeBytes int64   `json:"size_bytes"`
	PageCount *int    `json:"page_count"`
	URL       *string `json:"url"`
}

// OpenedResponse = riwayat siswa membuka modul.
type OpenedResponse struct {
	FirstOpenedAt time.Time `json:"first_opened_at"` // WIB
	LastOpenedAt  time.Time `json:"last_opened_at"`
	OpenCount     int       `json:"open_count"`
}

// StudentListResponse adalah data GET /api/v1/student/learning-modules.
type StudentListResponse struct {
	Access  StudentAccessResponse   `json:"access"`
	Modules []StudentModuleResponse `json:"modules"`
}

// ToStudentAccessResponse mengubah akses siswa menjadi response.
func ToStudentAccessResponse(a learningmodulemodel.StudentAccess) StudentAccessResponse {
	return StudentAccessResponse{
		EducationLevel: optRef(a.EducationLevel), Plan: optRef(a.Plan),
		ModuleFeature: a.ModuleFeature, AllModules: a.AllModules,
	}
}

// ToStudentModuleResponse mengubah 1 modul siswa menjadi response. fileURL dipakai kalau tidak terkunci.
func ToStudentModuleResponse(m learningmodulemodel.StudentModule, fileURL string) StudentModuleResponse {
	base := ToModuleResponse(m.Module, "")
	r := StudentModuleResponse{
		ID: base.ID, Name: base.Name, Description: base.Description, Placement: base.Placement,
		File:   StudentFile{SizeBytes: m.SizeBytes, PageCount: m.PageCount},
		Locked: m.Locked, Plans: base.Plans,
	}
	if !m.Locked {
		r.File.URL = &fileURL
	}
	if m.LastOpenedAt != nil && m.FirstOpenedAt != nil {
		r.Opened = &OpenedResponse{
			FirstOpenedAt: m.FirstOpenedAt.In(clock.Jakarta), LastOpenedAt: m.LastOpenedAt.In(clock.Jakarta), OpenCount: m.OpenCount,
		}
	}
	return r
}
