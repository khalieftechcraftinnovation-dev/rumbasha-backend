// Package learningmodule (service) berisi aturan modul pembelajaran (materi PDF):
//
//   - Buat / ubah (semua atau tidak sama sekali): cek isian, file harus benar-benar PDF (dibaca dari
//     isi file, maks 20 MB), letak cocok dengan master data, paket akses aktif. File disimpan ke
//     disk dulu, lalu data; kalau data gagal, file baru dihapus lagi. Ganti PDF → file lama dihapus
//     setelah data berhasil.
//   - Hapus: soft delete banyak sekaligus (file tetap disimpan).
//   - File PDF tidak publik: diunduh lewat API yang wajib login.
//
// Di package lain, import dengan alias learningmoduleservice:
//
//	learningmoduleservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/learning/learningmodule"
package learningmodule

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"

	learningmoduledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/learning/learningmodule"
	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learning/learningmodule"
	learningmodulerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/learning/learningmodule"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
)

// MaxFileSize adalah batas ukuran 1 file PDF modul.
const MaxFileSize = 20 << 20 // 20 MB

// UploadFile adalah file dari form. Data nil kalau file terlalu besar (tidak dibaca handler).
type UploadFile struct {
	FileName string
	Size     int64
	Data     []byte
}

// LearningModuleService berisi logika modul pembelajaran.
type LearningModuleService struct {
	repo          *learningmodulerepo.LearningModuleRepository
	storage       *storage.Local
	subscriptions *subscriptionservice.SubscriptionService // paket aktif & fitur siswa (akses modul)
}

// NewLearningModuleService membuat LearningModuleService. Dipanggil sekali di main.go.
func NewLearningModuleService(repo *learningmodulerepo.LearningModuleRepository, store *storage.Local, subscriptions *subscriptionservice.SubscriptionService) *LearningModuleService {
	return &LearningModuleService{repo: repo, storage: store, subscriptions: subscriptions}
}

// FileURL = link unduh PDF modul (API wajib login), contoh http://localhost:8080/api/v1/learning-modules/<id>/file.
func (s *LearningModuleService) FileURL(id string) string {
	return s.storage.BaseURL + "/api/v1/learning-modules/" + id + "/file"
}

// Create menyimpan 1 modul baru beserta file PDF-nya, lalu mengembalikan detailnya.
func (s *LearningModuleService) Create(ctx context.Context, req learningmoduledto.ModuleRequest, file *UploadFile, adminID string) (learningmodulemodel.Module, error) {
	errs := &fieldErrors{}
	if file == nil {
		errs.add("file", "wajib diisi (file PDF, maksimal 20 MB)")
	}
	m, err := s.buildModule(ctx, req, file, errs)
	if err != nil {
		return learningmodulemodel.Module{}, err
	}
	if errs.any() {
		return learningmodulemodel.Module{}, errs.toError()
	}

	if err := s.storage.Save(m.File.StoragePath, file.Data); err != nil {
		return learningmodulemodel.Module{}, fmt.Errorf("gagal menyimpan file %s: %w", file.FileName, err)
	}
	id, err := s.repo.Create(ctx, m, adminID)
	if err != nil {
		s.removeFile(m.File.StoragePath)
		return learningmodulemodel.Module{}, fmt.Errorf("gagal menyimpan modul: %w", err)
	}
	return s.repo.FindByID(ctx, id)
}

// Update mengganti isi 1 modul (ganti penuh). file nil = PDF tidak diganti.
func (s *LearningModuleService) Update(ctx context.Context, id string, req learningmoduledto.ModuleRequest, file *UploadFile, adminID string) (learningmodulemodel.Module, error) {
	if uuid.Validate(id) != nil {
		return learningmodulemodel.Module{}, learningmodulemodel.ErrModuleNotFound
	}
	errs := &fieldErrors{}
	m, err := s.buildModule(ctx, req, file, errs)
	if err != nil {
		return learningmodulemodel.Module{}, err
	}
	if errs.any() {
		return learningmodulemodel.Module{}, errs.toError()
	}
	m.ID = id
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return learningmodulemodel.Module{}, err
	}

	if m.File != nil {
		if err := s.storage.Save(m.File.StoragePath, file.Data); err != nil {
			return learningmodulemodel.Module{}, fmt.Errorf("gagal menyimpan file %s: %w", file.FileName, err)
		}
	} else if desired := learningmodulemodel.FilePath(m.PathNames, fileID(current.StoragePath)); desired != current.StoragePath {
		// Letak berubah tanpa PDF baru: file dipindah ke folder hierarki yang baru.
		if err := s.storage.Move(current.StoragePath, desired); err != nil {
			return learningmodulemodel.Module{}, err
		}
		m.RelocatedPath = desired
	}
	oldPath, err := s.repo.Update(ctx, m, adminID)
	if err != nil {
		if m.File != nil {
			s.removeFile(m.File.StoragePath)
		}
		if m.RelocatedPath != "" { // kembalikan file ke lokasi lama
			if moveErr := s.storage.Move(m.RelocatedPath, current.StoragePath); moveErr != nil {
				log.Printf("peringatan: %v", moveErr)
			}
		}
		return learningmodulemodel.Module{}, err
	}
	if oldPath != "" {
		s.removeFile(oldPath) // PDF lama diganti
	}
	return s.repo.FindByID(ctx, id)
}

// buildModule mengecek isian form + file + letak + paket akses, lalu menyusun data yang disimpan.
// Kesalahan isian dikumpulkan di errs; error kembalian hanya untuk masalah server (database).
func (s *LearningModuleService) buildModule(ctx context.Context, req learningmoduledto.ModuleRequest, file *UploadFile, errs *fieldErrors) (learningmodulemodel.Save, error) {
	m := learningmodulemodel.Save{Name: strings.TrimSpace(req.Name), Status: req.Status, PlanIDs: req.PlanIDs, PageCount: req.PageCount}
	if m.Name == "" {
		errs.add("name", "wajib diisi")
	}
	if req.Description != nil {
		if d := strings.TrimSpace(*req.Description); d != "" {
			m.Description = &d
		}
	}
	if m.Status == "" {
		m.Status = learningmodulemodel.StatusPublished
	}

	placement, names, err := s.checkPlacement(ctx, req, errs)
	if err != nil {
		return m, err
	}
	m.Placement = placement
	m.PathNames = names

	if file != nil {
		mime, problem := inspectPDF(*file)
		if problem != "" {
			errs.add("file", file.FileName+": "+problem)
		} else {
			m.File = &learningmodulemodel.File{
				FileName: file.FileName,
				// Folder mengikuti hierarki mapel, contoh modules/matematika/sd-kelas-5/bilangan/pecahan/<id>.pdf.
				StoragePath: learningmodulemodel.FilePath(names, uuid.NewString()),
				MimeType:    mime,
				SizeBytes:   file.Size,
			}
			if m.PageCount == nil {
				m.PageCount = countPDFPages(file.Data) // tidak diisi admin: hitung dari isi PDF
			}
		}
	}

	plans, err := s.repo.ActivePlans(ctx, req.PlanIDs)
	if err != nil {
		return m, err
	}
	for i, id := range req.PlanIDs {
		if !plans[id] {
			errs.add(fmt.Sprintf("plan_ids[%d]", i), "paket akses tidak ditemukan atau tidak aktif")
		}
	}
	return m, nil
}

// inspectPDF mengecek ukuran & isi file: harus PDF (dibaca dari isi, bukan nama file).
func inspectPDF(f UploadFile) (mime string, problem string) {
	switch {
	case f.Size > MaxFileSize || f.Data == nil:
		return "", fmt.Sprintf("ukuran maksimal %d MB", MaxFileSize>>20)
	case len(f.Data) == 0:
		return "", "file kosong"
	}
	mime = http.DetectContentType(f.Data) // "%PDF-" di awal file → application/pdf
	if mime != "application/pdf" {
		return "", "harus file PDF"
	}
	return mime, ""
}

// checkPlacement mengecek letak modul dengan 1 query master data: mapel & bidang materi wajib,
// bab/sub bab/topik opsional berurutan, setiap tingkat milik tingkat di atasnya & aktif.
// Jenjang & kelas diambil dari bidang materi.
func (s *LearningModuleService) checkPlacement(ctx context.Context, req learningmoduledto.ModuleRequest, errs *fieldErrors) (learningmodulemodel.Placement, learningmodulemodel.PathNames, error) {
	var names learningmodulemodel.PathNames
	p := learningmodulemodel.Placement{
		SubjectID: req.SubjectID, SubjectFieldID: req.SubjectFieldID,
		ChapterID: req.ChapterID, SubchapterID: req.SubchapterID, TopicID: req.TopicID,
	}
	if p.SubchapterID != nil && p.ChapterID == nil {
		errs.add("chapter_id", "wajib diisi jika subchapter_id diisi")
	}
	if p.TopicID != nil && p.SubchapterID == nil {
		errs.add("subchapter_id", "wajib diisi jika topic_id diisi")
	}

	ids := []string{p.SubjectID, p.SubjectFieldID}
	for _, id := range []*string{p.ChapterID, p.SubchapterID, p.TopicID} {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	items, err := s.repo.FindHierarchy(ctx, ids)
	if err != nil {
		return p, names, fmt.Errorf("gagal membaca master data: %w", err)
	}

	// check: ada, tipenya benar, aktif, dan (kalau parent diisi) milik induknya.
	check := func(field, label, itemType string, id *string, parent *string, parentLabel string) (learningmodulemodel.HierarchyItem, bool) {
		item, ok := items[*id]
		switch {
		case !ok || item.Type != itemType:
			errs.add(field, label+" tidak ditemukan")
		case item.Status != "active":
			errs.add(field, label+" tidak aktif")
		case parent != nil && (item.ParentID == nil || *item.ParentID != *parent):
			errs.add(field, label+" bukan bagian dari "+parentLabel+" yang dipilih")
		default:
			return item, true
		}
		return item, false
	}

	subject, ok := check("subject_id", "mapel", "subject", &p.SubjectID, nil, "")
	if !ok {
		return p, names, nil
	}
	field, ok := check("subject_field_id", "bidang materi", "subject_field", &p.SubjectFieldID, &p.SubjectID, "mapel")
	if !ok {
		return p, names, nil
	}
	if field.EducationLevelID != nil && field.GradeID != nil {
		p.EducationLevelID, p.GradeID = *field.EducationLevelID, *field.GradeID
	}
	names.Subject, names.SubjectField = subject.Name, field.Name
	if field.LevelName != nil && field.GradeName != nil {
		names.Level, names.Grade = *field.LevelName, *field.GradeName
	}
	if p.ChapterID != nil {
		chapter, ok := check("chapter_id", "bab", "chapter", p.ChapterID, &p.SubjectFieldID, "bidang materi")
		names.Chapter = chapter.Name
		if ok && p.SubchapterID != nil {
			sub, ok := check("subchapter_id", "sub bab", "subchapter", p.SubchapterID, p.ChapterID, "bab")
			names.Subchapter = sub.Name
			if ok && p.TopicID != nil {
				topic, _ := check("topic_id", "topik", "topic", p.TopicID, p.SubchapterID, "sub bab")
				names.Topic = topic.Name
			}
		}
	}
	return p, names, nil
}

// Get mengambil 1 modul. Bukan UUID / tidak ada / dihapus → ErrModuleNotFound.
func (s *LearningModuleService) Get(ctx context.Context, id string) (learningmodulemodel.Module, error) {
	if uuid.Validate(id) != nil {
		return learningmodulemodel.Module{}, learningmodulemodel.ErrModuleNotFound
	}
	return s.repo.FindByID(ctx, id)
}

// List mengambil 1 halaman modul (terbaru dulu).
func (s *LearningModuleService) List(ctx context.Context, q learningmoduledto.ListQuery) ([]learningmodulemodel.Module, learningmodulemodel.ListFilter, int, error) {
	f := learningmodulemodel.ListFilter{
		Search: strings.TrimSpace(q.Search), EducationLevelID: q.EducationLevelID, GradeID: q.GradeID,
		SubjectID: q.SubjectID, SubjectFieldID: q.SubjectFieldID, ChapterID: q.ChapterID,
		PlanID: q.PlanID, Status: q.Status, Page: q.Page, PerPage: q.PerPage,
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	items, total, err := s.repo.List(ctx, f)
	return items, f, total, err
}

// FilePath mengembalikan lokasi file PDF di disk + nama file asli (untuk diunduh lewat API).
func (s *LearningModuleService) FilePath(ctx context.Context, id string) (string, string, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return "", "", err
	}
	path, err := s.storage.FilePath(m.StoragePath)
	return path, m.FileName, err
}

// DeleteMany menghapus 1-100 modul (soft delete). Semua ID harus ada; kalau tidak, tidak ada yang
// terhapus dan kesalahannya per field "ids[i]".
func (s *LearningModuleService) DeleteMany(ctx context.Context, ids []string, adminID string) ([]string, error) {
	missing, err := s.repo.SoftDeleteMany(ctx, ids, adminID)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		isMissing := make(map[string]bool, len(missing))
		for _, id := range missing {
			isMissing[id] = true
		}
		errs := &fieldErrors{}
		for i, id := range ids {
			if isMissing[id] {
				errs.add(fmt.Sprintf("ids[%d]", i), "modul tidak ditemukan atau sudah dihapus")
			}
		}
		return nil, errs.toError()
	}
	return ids, nil
}

// removeFile menghapus file dari disk; kegagalan hanya dicatat di log server.
func (s *LearningModuleService) removeFile(path string) {
	if err := s.storage.Remove(path); err != nil {
		log.Printf("peringatan: %v", err)
	}
}

// fieldErrors mengumpulkan kesalahan per field.
type fieldErrors struct {
	list []learningmodulemodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, learningmodulemodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool { return len(e.list) > 0 }

func (e *fieldErrors) toError() error {
	return &learningmodulemodel.ValidationError{Fields: e.list}
}

// pdfPageObject = objek halaman di PDF ("/Type /Page", bukan "/Type /Pages").
var pdfPageObject = regexp.MustCompile(`/Type\s*/Page[^s]`)

// countPDFPages menghitung jumlah halaman dari isi PDF (jumlah objek "/Type /Page").
// PDF yang objeknya dikompres (object stream) bisa tidak terbaca → nil (admin bisa isi page_count).
func countPDFPages(data []byte) *int {
	n := len(pdfPageObject.FindAll(data, -1))
	if n == 0 {
		return nil
	}
	return &n
}

// fileID = nama file tanpa folder & ".pdf" (contoh modules/.../<id>.pdf → <id>). Dipakai supaya
// nama file tetap sama saat file dipindah ke folder lain.
func fileID(storagePath string) string {
	return strings.TrimSuffix(path.Base(storagePath), ".pdf")
}
