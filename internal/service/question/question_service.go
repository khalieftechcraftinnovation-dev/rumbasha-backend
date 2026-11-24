// Package question (service) berisi aturan bisnis bank soal.
//
// Alur Create (urutannya disengaja supaya hemat query):
//  1. Cek isi soal & jawaban (tanpa database). Kalau ada yang salah → berhenti.
//  2. Cek placement & gambar ke database (2 query kecil, 1 tabel masing-masing).
//  3. Simpan (1 query INSERT).
//
// Semua kesalahan isi dikumpulkan dulu, lalu dikembalikan sekaligus sebagai
// *questionmodel.ValidationError, supaya admin bisa memperbaiki semuanya dalam sekali kirim.
//
// Di package lain, import dengan alias questionservice:
//
//	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/question"
package question

import (
	"context"
	"fmt"
	"strings"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
)

// QuestionService berisi logika bisnis bank soal.
type QuestionService struct {
	repo *questionrepo.QuestionRepository
}

// NewQuestionService membuat QuestionService. Dipanggil sekali di main.go.
func NewQuestionService(repo *questionrepo.QuestionRepository) *QuestionService {
	return &QuestionService{repo: repo}
}

// Create membuat soal baru. createdBy = ID user (admin) yang membuat.
func (s *QuestionService) Create(ctx context.Context, req questiondto.CreateQuestionRequest, createdBy string) (questionmodel.Question, error) {
	errs := &fieldErrors{}

	// ---- 1. Cek isi tanpa database ----------------------------------------
	q := questionmodel.Question{
		Type:         req.Type,
		QuestionText: strings.TrimSpace(req.Question),
		Formula:      trimOptional(req.Formula),
		ImageID:      req.ImageID,
		Explanation:  trimOptional(req.Explanation),

		EducationLevelID: req.Placement.EducationLevelID,
		GradeID:          req.Placement.GradeID,
		SubjectID:        req.Placement.SubjectID,
		SubjectFieldID:   req.Placement.SubjectFieldID,
		ChapterID:        req.Placement.ChapterID,
		SubchapterID:     req.Placement.SubchapterID,
		TopicID:          req.Placement.TopicID,

		Difficulty: req.Difficulty,
		UsedFor:    req.UsedFor,
		Status:     req.Status,
		CreatedBy:  createdBy,
	}
	if q.Status == "" {
		q.Status = "active"
	}
	if q.QuestionText == "" {
		errs.add("question", "wajib diisi")
	}
	if req.Table != nil {
		q.Table = &questionmodel.Table{Headers: req.Table.Headers, Rows: req.Table.Rows}
		validateTable(q.Table, errs)
	}
	q.DisplayOrder = validateDisplayOrder(req.DisplayOrder, q, errs)

	// imageFields: field request → ID gambar, dicek keberadaannya di langkah 2.
	imageFields := map[string]string{}
	if q.ImageID != nil {
		imageFields["image_id"] = *q.ImageID
	}

	answer, err := validateAnswer(q.Type, req.Answer, errs, imageFields)
	if err != nil {
		return questionmodel.Question{}, err
	}
	q.Answer = answer

	if errs.any() {
		return questionmodel.Question{}, errs.toError()
	}

	// ---- 2. Cek ke database --------------------------------------------------
	if err := s.validatePlacement(ctx, req.Placement, errs); err != nil {
		return questionmodel.Question{}, err
	}
	if err := s.validateImages(ctx, imageFields, errs); err != nil {
		return questionmodel.Question{}, err
	}
	if errs.any() {
		return questionmodel.Question{}, errs.toError()
	}

	// ---- 3. Simpan -----------------------------------------------------------
	if err := s.repo.Create(ctx, &q); err != nil {
		return questionmodel.Question{}, fmt.Errorf("gagal menyimpan soal: %w", err)
	}
	return q, nil
}

// validatePlacement mengecek letak soal dengan 1 query ke master_data_items.
//
// Cukup bab, sub bab, dan topik yang dicek isinya: bab sudah menyimpan mapel, jenjang,
// dan kelasnya sendiri (kolom bantu migration 000007), jadi tinggal dibandingkan.
// Mapel & bidang materi ikut diambil hanya untuk mengecek statusnya masih aktif.
func (s *QuestionService) validatePlacement(ctx context.Context, p questiondto.PlacementRequest, errs *fieldErrors) error {
	ids := []string{p.SubjectID, p.SubjectFieldID, p.ChapterID}
	if p.SubchapterID != nil {
		ids = append(ids, *p.SubchapterID)
	}
	if p.TopicID != nil {
		ids = append(ids, *p.TopicID)
	}

	items, err := s.repo.FindMasterDataByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("gagal membaca master data placement: %w", err)
	}

	// Mapel & bidang materi: cukup dicek masih aktif (kecocokannya dicek lewat bab di bawah).
	checkActive(items, p.SubjectID, "placement.subject_id", "mapel", errs)
	checkActive(items, p.SubjectFieldID, "placement.subject_field_id", "bidang materi", errs)

	// Bab: wajib ada, aktif, dan cocok dengan mapel, bidang materi, jenjang, kelas yang dikirim.
	chapter, ok := items[p.ChapterID]
	if !ok || chapter.Type != "chapter" {
		errs.add("placement.chapter_id", "bab tidak ditemukan")
		return nil // field lain dibandingkan ke bab, jadi tidak bisa dilanjutkan
	}
	if chapter.Status != "active" {
		errs.add("placement.chapter_id", "bab tidak aktif")
	}
	if !equalPtr(chapter.ParentID, p.SubjectFieldID) {
		errs.add("placement.subject_field_id", "tidak sesuai dengan bab yang dipilih")
	}
	if !equalPtr(chapter.SubjectID, p.SubjectID) {
		errs.add("placement.subject_id", "tidak sesuai dengan bab yang dipilih")
	}
	if !equalPtr(chapter.EducationLevelID, p.EducationLevelID) {
		errs.add("placement.education_level_id", "tidak sesuai dengan bab yang dipilih")
	}
	if !equalPtr(chapter.GradeID, p.GradeID) {
		errs.add("placement.grade_id", "tidak sesuai dengan bab yang dipilih")
	}

	// Sub bab (opsional): harus bagian dari bab.
	if p.SubchapterID != nil {
		sub, ok := items[*p.SubchapterID]
		switch {
		case !ok || sub.Type != "subchapter":
			errs.add("placement.subchapter_id", "sub bab tidak ditemukan")
		case !equalPtr(sub.ParentID, p.ChapterID):
			errs.add("placement.subchapter_id", "sub bab bukan bagian dari bab yang dipilih")
		case sub.Status != "active":
			errs.add("placement.subchapter_id", "sub bab tidak aktif")
		}
	}

	// Topik (opsional): wajib ada sub bab-nya, dan harus bagian dari sub bab tersebut.
	if p.TopicID != nil {
		topic, ok := items[*p.TopicID]
		switch {
		case p.SubchapterID == nil:
			errs.add("placement.subchapter_id", "wajib diisi jika topic_id diisi")
		case !ok || topic.Type != "topic":
			errs.add("placement.topic_id", "topik tidak ditemukan")
		case !equalPtr(topic.ParentID, *p.SubchapterID):
			errs.add("placement.topic_id", "topik bukan bagian dari sub bab yang dipilih")
		case topic.Status != "active":
			errs.add("placement.topic_id", "topik tidak aktif")
		}
	}
	return nil
}

// validateImages mengecek semua image_id (soal & opsi) ada di media_assets, dengan 1 query.
func (s *QuestionService) validateImages(ctx context.Context, imageFields map[string]string, errs *fieldErrors) error {
	if len(imageFields) == 0 {
		return nil
	}

	ids := make([]string, 0, len(imageFields))
	for _, id := range imageFields {
		ids = append(ids, id)
	}
	existing, err := s.repo.FindExistingImageIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("gagal membaca media: %w", err)
	}

	for field, id := range imageFields {
		if !existing[id] {
			errs.add(field, "gambar tidak ditemukan")
		}
	}
	return nil
}

// validateTable: setiap baris harus punya jumlah kolom yang sama
// (sama dengan headers kalau headers diisi, kalau tidak sama dengan baris pertama).
func validateTable(t *questionmodel.Table, errs *fieldErrors) {
	columns := len(t.Headers)
	if columns == 0 && len(t.Rows) > 0 {
		columns = len(t.Rows[0])
	}
	for i, row := range t.Rows {
		if len(row) != columns {
			errs.add(fmt.Sprintf("table.rows[%d]", i), fmt.Sprintf("jumlah kolom harus %d", columns))
		}
	}
}

// validateDisplayOrder mengecek urutan tampil bagian soal.
//   - Kalau tidak dikirim: dibuat otomatis question → formula → image → table (yang diisi saja).
//   - Kalau dikirim: harus berisi "question" dan PERSIS bagian yang diisi (tidak kurang, tidak lebih).
func validateDisplayOrder(order []string, q questionmodel.Question, errs *fieldErrors) []string {
	filled := map[string]bool{
		questionmodel.PartQuestion: true,
		questionmodel.PartFormula:  q.Formula != nil,
		questionmodel.PartImage:    q.ImageID != nil,
		questionmodel.PartTable:    q.Table != nil,
	}
	defaultOrder := []string{questionmodel.PartQuestion, questionmodel.PartFormula, questionmodel.PartImage, questionmodel.PartTable}

	if len(order) == 0 {
		result := make([]string, 0, len(defaultOrder))
		for _, part := range defaultOrder {
			if filled[part] {
				result = append(result, part)
			}
		}
		return result
	}

	listed := map[string]bool{}
	for i, part := range order {
		listed[part] = true
		if !filled[part] {
			errs.add(fmt.Sprintf("display_order[%d]", i), fmt.Sprintf("%s tidak diisi, hapus dari display_order", part))
		}
	}
	for _, part := range defaultOrder {
		if filled[part] && !listed[part] {
			errs.add("display_order", fmt.Sprintf("%s diisi, wajib dicantumkan di display_order", part))
		}
	}
	return order
}

// checkActive mengecek item master data ada dan aktif (dipakai untuk mapel & bidang materi).
// Kalau tidak ditemukan, tidak dicatat di sini: pesannya sudah muncul dari pengecekan bab.
func checkActive(items map[string]questionmodel.MasterDataItem, id, field, label string, errs *fieldErrors) {
	if item, ok := items[id]; ok && item.Status != "active" {
		errs.add(field, label+" tidak aktif")
	}
}

// equalPtr membandingkan kolom nullable (*string) dengan nilai biasa.
func equalPtr(p *string, v string) bool {
	return p != nil && *p == v
}

// trimOptional menghapus spasi di ujung. Teks kosong dianggap tidak diisi (nil).
func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// fieldErrors mengumpulkan kesalahan per field.
type fieldErrors struct {
	list []questionmodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, questionmodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool {
	return len(e.list) > 0
}

func (e *fieldErrors) toError() error {
	return &questionmodel.ValidationError{Fields: e.list}
}
