// Package question (service) berisi aturan bisnis bank soal.
//
// Alur CreateMany (urutannya disengaja supaya hemat query, berapa pun jumlah soalnya):
//  1. Cek isi setiap soal & jawaban (tanpa database). Kalau ada yang salah → berhenti.
//  2. Cek placement & gambar SEMUA soal ke database: 1 query master data + 1 query gambar.
//  3. Simpan semua soal dengan 1 query INSERT (semua tersimpan atau tidak sama sekali).
//
// Semua kesalahan dikumpulkan dulu, lalu dikembalikan sekaligus sebagai
// *questionmodel.ValidationError. Nama field diawali nomor soalnya (mulai dari 0), contoh
// "questions[2].answer.correct_key", supaya admin tahu soal mana yang perlu diperbaiki.
//
// Di package lain, import dengan alias questionservice:
//
//	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/question"
package question

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

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

// CreateMany membuat 1 atau banyak soal dengan tipe yang sama. createdBy = ID admin pembuat.
// Hasilnya urut sesuai urutan di request.
func (s *QuestionService) CreateMany(ctx context.Context, req questiondto.CreateQuestionsRequest, createdBy string) ([]questionmodel.Question, error) {
	errs := &fieldErrors{}

	// imageFields: field request → ID gambar, dicek keberadaannya di langkah 2.
	imageFields := map[string]string{}

	// ---- 1. Cek isi setiap soal tanpa database ------------------------------
	questions := make([]questionmodel.Question, 0, len(req.Questions))
	for i, item := range req.Questions {
		errs.prefix = questionPrefix(i)
		q, err := buildQuestion(req.Type, item, createdBy, errs, imageFields)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	errs.prefix = ""
	if errs.any() {
		return nil, errs.toError()
	}

	// ---- 2. Cek ke database (sekali untuk semua soal) -----------------------
	if err := s.validatePlacements(ctx, req.Questions, questionPrefix, errs); err != nil {
		return nil, err
	}
	if err := s.validateImages(ctx, imageFields, errs); err != nil {
		return nil, err
	}
	if errs.any() {
		return nil, errs.toError()
	}

	// ---- 3. Simpan semua dalam 1 query ---------------------------------------
	if err := s.repo.CreateMany(ctx, questions); err != nil {
		return nil, fmt.Errorf("gagal menyimpan %d soal: %w", len(questions), err)
	}
	return questions, nil
}

// questionPrefix adalah awalan nama field error untuk soal ke-i pada create banyak soal.
func questionPrefix(i int) string {
	return fmt.Sprintf("questions[%d].", i)
}

// noPrefix dipakai update (hanya 1 soal, field error tanpa awalan).
func noPrefix(int) string {
	return ""
}

// buildQuestion mengubah 1 soal dari request menjadi model, sambil mengecek isinya.
// Kesalahan dicatat di errs (sudah diberi awalan "questions[i]." oleh pemanggil).
func buildQuestion(questionType string, req questiondto.CreateQuestionRequest, createdBy string, errs *fieldErrors, imageFields map[string]string) (questionmodel.Question, error) {
	// 1 request = 1 tipe. Type per soal boleh kosong, tapi kalau diisi harus sama.
	if req.Type != "" && req.Type != questionType {
		errs.add("type", fmt.Sprintf("harus %s (1 request hanya boleh 1 tipe soal)", questionType))
	}

	q := questionmodel.Question{
		Type:         questionType,
		QuestionText: strings.TrimSpace(req.Question),
		Formula:      trimOptional(req.Formula),
		ImageID:      req.ImageID,
		ImageURL:     trimOptional(req.ImageURL),
		Explanation:  strings.TrimSpace(req.Explanation),

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
	// Teks soal & pembahasan wajib untuk semua tipe. Isi spasi saja dianggap kosong
	// (binding "required" hanya menolak string yang benar-benar kosong).
	if q.QuestionText == "" {
		errs.add("question", "wajib diisi")
	}
	if q.Explanation == "" {
		errs.add("explanation", "wajib diisi")
	}
	if req.Table != nil {
		q.Table = &questionmodel.Table{Headers: req.Table.Headers, Rows: req.Table.Rows}
		validateTable(q.Table, errs)
	}
	q.DisplayOrder = validateDisplayOrder(req.DisplayOrder, q, errs)

	// Gambar soal: image_id (Image Library) ATAU image_url (link luar), tidak boleh dua-duanya.
	switch {
	case q.ImageID != nil && q.ImageURL != nil:
		errs.add("image_url", "pilih salah satu: image_id atau image_url")
	case q.ImageID != nil:
		imageFields[errs.prefix+"image_id"] = *q.ImageID // keberadaannya dicek di validateImages
	case q.ImageURL != nil && !isHTTPURL(*q.ImageURL):
		errs.add("image_url", "harus link http/https yang valid")
	}

	answer, err := validateAnswer(q.Type, req.Answer, errs, imageFields)
	if err != nil {
		return questionmodel.Question{}, err
	}
	q.Answer = answer
	return q, nil
}

// validatePlacements mengambil master data placement SEMUA soal dengan 1 query
// (ID yang sama hanya diambil sekali), lalu mengecek placement setiap soal.
// prefixFor menentukan awalan nama field error soal ke-i (create: "questions[i].", update: "").
func (s *QuestionService) validatePlacements(ctx context.Context, items []questiondto.CreateQuestionRequest, prefixFor func(i int) string, errs *fieldErrors) error {
	unique := map[string]bool{}
	for _, item := range items {
		p := item.Placement
		unique[p.SubjectID] = true
		unique[p.SubjectFieldID] = true
		unique[p.ChapterID] = true
		if p.SubchapterID != nil {
			unique[*p.SubchapterID] = true
		}
		if p.TopicID != nil {
			unique[*p.TopicID] = true
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}

	masterData, err := s.repo.FindMasterDataByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("gagal membaca master data placement: %w", err)
	}

	for i, item := range items {
		errs.prefix = prefixFor(i)
		checkPlacement(masterData, item.Placement, errs)
	}
	errs.prefix = ""
	return nil
}

// checkPlacement mengecek letak 1 soal terhadap master data yang sudah diambil.
//
// Cukup bab, sub bab, dan topik yang dicek isinya: bab sudah menyimpan mapel, jenjang,
// dan kelasnya sendiri (kolom bantu migration 000007), jadi tinggal dibandingkan.
// Mapel & bidang materi hanya dicek statusnya masih aktif.
func checkPlacement(items map[string]questionmodel.MasterDataItem, p questiondto.PlacementRequest, errs *fieldErrors) {
	// Mapel & bidang materi: cukup dicek masih aktif (kecocokannya dicek lewat bab di bawah).
	checkActive(items, p.SubjectID, "placement.subject_id", "mapel", errs)
	checkActive(items, p.SubjectFieldID, "placement.subject_field_id", "bidang materi", errs)

	// Bab: wajib ada, aktif, dan cocok dengan mapel, bidang materi, jenjang, kelas yang dikirim.
	chapter, ok := items[p.ChapterID]
	if !ok || chapter.Type != "chapter" {
		errs.add("placement.chapter_id", "bab tidak ditemukan")
		return // field lain dibandingkan ke bab, jadi tidak bisa dilanjutkan
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
}

// validateImages mengecek semua image_id (soal & opsi, semua soal) ada di media_assets, dengan 1 query.
// Key imageFields sudah berisi nama field lengkap (dengan awalan "questions[i]."),
// jadi errs.prefix harus kosong saat fungsi ini dipanggil.
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
		questionmodel.PartImage:    q.ImageID != nil || q.ImageURL != nil,
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
// prefix ditempel di depan setiap nama field, contoh "questions[2]." → "questions[2].answer.correct_key".
// Diatur oleh CreateMany sebelum mengecek tiap soal.
type fieldErrors struct {
	list   []questionmodel.FieldError
	prefix string
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, questionmodel.FieldError{Field: e.prefix + field, Message: message})
}

func (e *fieldErrors) any() bool {
	return len(e.list) > 0
}

func (e *fieldErrors) toError() error {
	return &questionmodel.ValidationError{Fields: e.list}
}

// GetDetail mengambil 1 soal lengkap, termasuk kunci jawaban (answer) dan pembahasan.
// ID yang bukan UUID langsung dianggap tidak ditemukan (tanpa query ke database).
func (s *QuestionService) GetDetail(ctx context.Context, id string) (questionmodel.QuestionDetail, error) {
	if uuid.Validate(id) != nil {
		return questionmodel.QuestionDetail{}, questionmodel.ErrQuestionNotFound
	}
	return s.repo.FindDetailByID(ctx, id)
}

// List mengambil 1 halaman list Bank Soal (urut terbaru dulu) beserta total soal yang cocok.
// Query sudah divalidasi binding di handler; di sini hanya default pagination & rapikan search.
func (s *QuestionService) List(ctx context.Context, q questiondto.ListQuestionsQuery) ([]questionmodel.QuestionListItem, questionmodel.ListFilter, int, error) {
	filter := toListFilter(q)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PerPage == 0 {
		filter.PerPage = 20
	}

	items, total, err := s.repo.List(ctx, filter)
	return items, filter, total, err
}

// CountByType menghitung jumlah soal per tipe untuk tab Bank Soal (Semua, Pilihan ganda, ...).
// Memakai filter yang sama dengan List (search, level, hierarki), supaya angka di tab cocok
// dengan isi tabel. Filter type & pagination diabaikan: semua tipe selalu dihitung.
func (s *QuestionService) CountByType(ctx context.Context, q questiondto.ListQuestionsQuery) (map[string]int, error) {
	filter := toListFilter(q)
	filter.Type = ""
	return s.repo.CountByType(ctx, filter)
}

// toListFilter mengubah query string list menjadi filter repository.
func toListFilter(q questiondto.ListQuestionsQuery) questionmodel.ListFilter {
	return questionmodel.ListFilter{
		Type:             q.Type,
		Difficulty:       q.Difficulty,
		EducationLevelID: q.EducationLevelID,
		GradeID:          q.GradeID,
		SubjectID:        q.SubjectID,
		SubjectFieldID:   q.SubjectFieldID,
		ChapterID:        q.ChapterID,
		SubchapterID:     q.SubchapterID,
		TopicID:          q.TopicID,
		UsedFor:          q.UsedFor,
		Status:           q.Status,
		QuizID:           q.QuizID,
		Search:           strings.TrimSpace(q.Search),
		Page:             q.Page,
		PerPage:          q.PerPage,
	}
}

// validTypes adalah tipe soal yang dikenal (dipakai update, karena create mengecek lewat binding).
var validTypes = map[string]bool{
	questionmodel.TypeSingleChoice:   true,
	questionmodel.TypeMultipleChoice: true,
	questionmodel.TypeTrueFalse:      true,
	questionmodel.TypeFreeText:       true,
}

// Update mengganti seluruh isi 1 soal, lalu mengembalikan detail terbarunya.
// Aturan isi sama persis dengan create. Tipe soal boleh diganti (answer dicek sesuai tipe baru).
//
// Urutan: cek soal ada (1 query) → cek isi tanpa database → cek placement & gambar
// (2 query) → UPDATE (1 query) → ambil detail (1 query).
func (s *QuestionService) Update(ctx context.Context, id string, req questiondto.CreateQuestionRequest, updatedBy string) (questionmodel.QuestionDetail, error) {
	if uuid.Validate(id) != nil {
		return questionmodel.QuestionDetail{}, questionmodel.ErrQuestionNotFound
	}
	exists, err := s.repo.ExistsByID(ctx, id)
	if err != nil {
		return questionmodel.QuestionDetail{}, fmt.Errorf("gagal mengecek soal: %w", err)
	}
	if !exists {
		return questionmodel.QuestionDetail{}, questionmodel.ErrQuestionNotFound
	}

	errs := &fieldErrors{}

	// Tanpa tipe yang valid, answer tidak bisa dicek, jadi langsung berhenti.
	switch {
	case req.Type == "":
		errs.add("type", "wajib diisi")
		return questionmodel.QuestionDetail{}, errs.toError()
	case !validTypes[req.Type]:
		errs.add("type", "harus salah satu dari: single_choice, multiple_choice, true_false, free_text")
		return questionmodel.QuestionDetail{}, errs.toError()
	}

	imageFields := map[string]string{}
	q, err := buildQuestion(req.Type, req, "", errs, imageFields)
	if err != nil {
		return questionmodel.QuestionDetail{}, err
	}
	if errs.any() {
		return questionmodel.QuestionDetail{}, errs.toError()
	}

	if err := s.validatePlacements(ctx, []questiondto.CreateQuestionRequest{req}, noPrefix, errs); err != nil {
		return questionmodel.QuestionDetail{}, err
	}
	if err := s.validateImages(ctx, imageFields, errs); err != nil {
		return questionmodel.QuestionDetail{}, err
	}
	if errs.any() {
		return questionmodel.QuestionDetail{}, errs.toError()
	}

	q.ID = id
	q.UpdatedBy = &updatedBy
	if err := s.repo.Update(ctx, q); err != nil {
		return questionmodel.QuestionDetail{}, err
	}
	return s.repo.FindDetailByID(ctx, id)
}

// DeleteMany menghapus 1 sampai 100 soal dengan soft delete (data tetap ada di database).
// Semua atau tidak sama sekali: kalau ada ID yang tidak ada / sudah dihapus, tidak ada yang
// terhapus dan ID tersebut dilaporkan per posisi, contoh "ids[1]".
func (s *QuestionService) DeleteMany(ctx context.Context, ids []string, deletedBy string) ([]string, error) {
	// Catatan: binding "uuid" hanya menerima huruf kecil (sama dengan format UUID dari
	// PostgreSQL), jadi ids bisa langsung dibandingkan dengan hasil RETURNING di bawah.
	found, ok, err := s.repo.SoftDeleteMany(ctx, ids, deletedBy)
	if err != nil {
		return nil, fmt.Errorf("gagal menghapus soal: %w", err)
	}
	if ok {
		return ids, nil
	}

	foundSet := make(map[string]bool, len(found))
	for _, id := range found {
		foundSet[id] = true
	}
	errs := &fieldErrors{}
	for i, id := range ids {
		if !foundSet[id] {
			errs.add(fmt.Sprintf("ids[%d]", i), "soal tidak ditemukan atau sudah dihapus")
		}
	}
	return nil, errs.toError()
}
