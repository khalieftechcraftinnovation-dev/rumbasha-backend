// Package quiz (service) berisi logika bisnis quiz (Uji Kom & Tryout).
//
// Di package lain, import dengan alias quizservice:
//
//	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
package quiz

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/accesscode"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
)

// QuizService berisi logika bisnis quiz.
type QuizService struct {
	repo *quizrepo.QuizRepository
}

// NewQuizService membuat QuizService. Dipanggil sekali di main.go.
func NewQuizService(repo *quizrepo.QuizRepository) *QuizService {
	return &QuizService{repo: repo}
}

// selectedQuestion adalah 1 soal yang akan masuk quiz + field asalnya (untuk pesan error):
// "bundle_quiz_ids[0]" kalau dari bundle, "question_ids[2]" kalau soal satuan.
type selectedQuestion struct {
	ID    string
	Field string
}

// maxCodeAttempts = berapa kali membuat ulang kode akses kalau ternyata sudah dipakai quiz lain.
const maxCodeAttempts = 5

// Create membuat 1 quiz beserta soal-soalnya, lalu mengembalikan detailnya.
//
// Urutan: cek isi tanpa database → kumpulkan soal dari bundle (1 query) + soal satuan → cek letak
// (1 query master data) → cek soal (1 query questions) → simpan dengan kode akses acak
// (1 transaksi) → ambil detail (2 query).
// Semua kesalahan dikumpulkan dan dikembalikan sekaligus (ValidationError).
func (s *QuizService) Create(ctx context.Context, req quizdto.CreateQuizRequest, createdBy string) (quizmodel.QuizDetail, error) {
	errs := &fieldErrors{}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		errs.add("name", "wajib diisi")
	}

	startAt, ok := parseStartAt(req.StartAt)
	switch {
	case !ok:
		errs.add("start_at", "format tidak valid, contoh 2026-12-01T08:00 (WIB)")
	case !startAt.After(time.Now()):
		errs.add("start_at", "harus setelah waktu sekarang")
	}

	selected, err := s.collectQuestions(ctx, req, errs)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	fromBundle := ""
	if len(req.BundleQuizIDs) > 0 {
		fromBundle = ", termasuk soal dari bundle"
	}
	switch {
	case len(req.BundleQuizIDs) == 0 && len(req.QuestionIDs) == 0:
		errs.add("question_ids", "pilih minimal 1 soal atau 1 bundle")
	case len(selected) > quizmodel.MaxQuestions:
		errs.add("question_ids", fmt.Sprintf("total soal maksimal %d%s (sekarang %d)", quizmodel.MaxQuestions, fromBundle, len(selected)))
	}
	if req.TotalQuestions != len(selected) {
		errs.add("total_questions", fmt.Sprintf("harus sama dengan jumlah soal yang dipilih (%d%s)", len(selected), fromBundle))
	}

	p := req.Placement
	placementOK, err := s.checkPlacement(ctx, req.Type, p, errs)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	if err := s.checkQuestions(ctx, req, selected, placementOK, errs); err != nil {
		return quizmodel.QuizDetail{}, err
	}
	if errs.any() {
		return quizmodel.QuizDetail{}, errs.toError()
	}

	coinCost := 0
	if req.CoinCost != nil {
		coinCost = *req.CoinCost
	}
	questionIDs := make([]string, 0, len(selected))
	for _, q := range selected {
		questionIDs = append(questionIDs, q.ID)
	}
	quiz := quizmodel.Quiz{
		Name:             name,
		Type:             req.Type,
		EducationLevelID: p.EducationLevelID,
		GradeID:          p.GradeID,
		SubjectID:        p.SubjectID,
		SubjectFieldID:   p.SubjectFieldID,
		ChapterID:        p.ChapterID,
		SubchapterID:     p.SubchapterID,
		TopicID:          p.TopicID,
		StartAt:          startAt,
		DurationMinutes:  req.DurationMinutes,
		CoinCost:         coinCost,
		TotalQuestions:   len(questionIDs),
		Status:           quizmodel.StatusDraft,
		QuestionIDs:      questionIDs,
		CreatedBy:        createdBy,
	}

	// Kode akses acak (Uji Kom & Tryout). Kalau kebetulan sudah dipakai quiz lain
	// (index unik menolak), buat kode baru lalu simpan ulang.
	for attempt := 1; ; attempt++ {
		code, err := accesscode.New()
		if err != nil {
			return quizmodel.QuizDetail{}, err
		}
		quiz.AccessCode = &code

		err = s.repo.Create(ctx, &quiz)
		if errors.Is(err, quizmodel.ErrAccessCodeTaken) && attempt < maxCodeAttempts {
			continue
		}
		if err != nil {
			return quizmodel.QuizDetail{}, fmt.Errorf("gagal menyimpan quiz: %w", err)
		}
		break
	}
	return s.repo.FindDetailByID(ctx, quiz.ID)
}

// startAtLayouts adalah format waktu mulai tanpa zona waktu; semuanya dianggap WIB.
// Format pertama sama dengan nilai input HTML datetime-local.
var startAtLayouts = []string{
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
}

// parseStartAt membaca waktu mulai. RFC3339 dengan zona waktu dipakai apa adanya
// ("2026-12-01T01:00:00Z" = 08:00 WIB); tanpa zona waktu dianggap WIB.
func parseStartAt(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	for _, layout := range startAtLayouts {
		if t, err := time.ParseInLocation(layout, s, clock.Jakarta); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// collectQuestions menyusun daftar soal quiz: semua soal dari bundle (urut bundle, lalu nomor
// soal di bundle), lalu soal satuan. Soal yang sudah ada di daftar dilewati (hanya diambil sekali).
// Bundle yang tidak ada / sudah dihapus dicatat sebagai error.
func (s *QuizService) collectQuestions(ctx context.Context, req quizdto.CreateQuizRequest, errs *fieldErrors) ([]selectedQuestion, error) {
	seen := map[string]bool{}
	selected := make([]selectedQuestion, 0, len(req.QuestionIDs))
	add := func(id, field string) {
		if !seen[id] {
			seen[id] = true
			selected = append(selected, selectedQuestion{ID: id, Field: field})
		}
	}

	if len(req.BundleQuizIDs) > 0 {
		bundles, err := s.repo.FindBundleQuestionIDs(ctx, req.BundleQuizIDs)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca soal bundle: %w", err)
		}
		for i, bundleID := range req.BundleQuizIDs {
			field := fmt.Sprintf("bundle_quiz_ids[%d]", i)
			questionIDs, found := bundles[bundleID]
			if !found {
				errs.add(field, "bundle tidak ditemukan")
				continue
			}
			for _, id := range questionIDs {
				add(id, field)
			}
		}
	}

	for i, id := range req.QuestionIDs {
		add(id, fmt.Sprintf("question_ids[%d]", i))
	}
	return selected, nil
}

// checkPlacement mengecek letak quiz terhadap master data (1 query untuk semua tingkat).
// ok = false kalau ada kesalahan letak; dipakai supaya soal tidak dibandingkan ke letak yang salah
// (contoh mapel tidak ditemukan → semua soal pasti "berbeda mapel", pesannya jadi tidak berguna).
// Aturan kelas Uji Kom (hanya Kelas 6, 9, 12) juga dicek di sini, tapi tidak membuat ok = false
// karena letaknya tetap valid untuk dibandingkan dengan soal.
func (s *QuizService) checkPlacement(ctx context.Context, quizType string, p quizdto.QuizPlacementRequest, errs *fieldErrors) (ok bool, err error) {
	before := len(errs.list)

	// Urutan hierarki: tingkat bawah butuh tingkat di atasnya.
	if p.ChapterID != nil && p.SubjectFieldID == nil {
		errs.add("placement.subject_field_id", "wajib diisi jika chapter_id diisi")
	}
	if p.SubchapterID != nil && p.ChapterID == nil {
		errs.add("placement.chapter_id", "wajib diisi jika subchapter_id diisi")
	}
	if p.TopicID != nil && p.SubchapterID == nil {
		errs.add("placement.subchapter_id", "wajib diisi jika topic_id diisi")
	}

	ids := []string{p.EducationLevelID, p.GradeID, p.SubjectID}
	for _, id := range []*string{p.SubjectFieldID, p.ChapterID, p.SubchapterID, p.TopicID} {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	items, err := s.repo.FindMasterDataByIDs(ctx, ids)
	if err != nil {
		return false, fmt.Errorf("gagal membaca master data letak quiz: %w", err)
	}

	// check mengecek 1 tingkat: ada, type-nya benar, aktif, dan (kalau parentID diisi) induknya cocok.
	check := func(field, label, itemType, id string, parentID *string, parentLabel string) (quizmodel.MasterDataItem, bool) {
		item, found := items[id]
		switch {
		case !found || item.Type != itemType:
			errs.add(field, label+" tidak ditemukan")
		case parentID != nil && !equalPtr(item.ParentID, *parentID):
			errs.add(field, label+" bukan bagian dari "+parentLabel+" yang dipilih")
		case item.Status != "active":
			errs.add(field, label+" tidak aktif")
		default:
			return item, true
		}
		return item, false
	}

	check("placement.education_level_id", "jenjang", "education_level", p.EducationLevelID, nil, "")
	grade, gradeOK := check("placement.grade_id", "kelas", "grade", p.GradeID, &p.EducationLevelID, "jenjang")
	check("placement.subject_id", "mapel", "subject", p.SubjectID, nil, "")
	if p.SubjectFieldID != nil {
		// Bidang materi: anak mapel, dan harus milik kelas yang dipilih.
		field, found := check("placement.subject_field_id", "bidang materi", "subject_field", *p.SubjectFieldID, &p.SubjectID, "mapel")
		if found && !equalPtr(field.GradeID, p.GradeID) {
			errs.add("placement.subject_field_id", "bidang materi bukan untuk kelas yang dipilih")
		}
	}
	if p.ChapterID != nil && p.SubjectFieldID != nil {
		check("placement.chapter_id", "bab", "chapter", *p.ChapterID, p.SubjectFieldID, "bidang materi")
	}
	if p.SubchapterID != nil && p.ChapterID != nil {
		check("placement.subchapter_id", "sub bab", "subchapter", *p.SubchapterID, p.ChapterID, "bab")
	}
	if p.TopicID != nil && p.SubchapterID != nil {
		check("placement.topic_id", "topik", "topic", *p.TopicID, p.SubchapterID, "sub bab")
	}
	ok = len(errs.list) == before

	// Uji Kom hanya untuk kelas akhir jenjang: Kelas 6 (SD), 9 (SMP), 12 (SMA).
	if quizType == quizmodel.TypeUjiKom && gradeOK && (grade.Code == nil || !quizmodel.UjiKomGradeCodes[*grade.Code]) {
		errs.add("placement.grade_id", "Uji Kom hanya untuk Kelas 6, 9, dan 12")
	}
	return ok, nil
}

// checkQuestions mengecek semua soal yang dipilih dengan 1 query. Syarat tiap soal:
// ada & belum dihapus, aktif, peruntukan (used_for) berisi tipe quiz, dan letaknya ada di
// dalam letak quiz (mapel & kelas sama; bidang materi s/d topik sama kalau quiz mengisinya).
// Error ditulis di field asal soal ("question_ids[i]" atau "bundle_quiz_ids[i]") dengan kode
// soal, contoh "Q-12: soal tidak aktif".
func (s *QuizService) checkQuestions(ctx context.Context, req quizdto.CreateQuizRequest, selected []selectedQuestion, comparePlacement bool, errs *fieldErrors) error {
	if len(selected) == 0 {
		return nil
	}
	ids := make([]string, 0, len(selected))
	for _, q := range selected {
		ids = append(ids, q.ID)
	}
	questions, err := s.repo.FindQuestionsByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("gagal membaca soal yang dipilih: %w", err)
	}

	p := req.Placement
	typeName := quizmodel.TypeNames[req.Type]
	for _, sel := range selected {
		field := sel.Field
		q, found := questions[sel.ID]
		if !found {
			errs.add(field, "soal tidak ditemukan")
			continue
		}
		if q.Deleted {
			errs.add(field, q.Code+": soal sudah dihapus")
			continue
		}

		if q.Status != "active" {
			errs.add(field, q.Code+": soal tidak aktif")
		}
		if !slices.Contains(q.UsedFor, req.Type) {
			errs.add(field, q.Code+": soal tidak ditandai untuk "+typeName)
		}
		if !comparePlacement {
			continue
		}

		// Letak soal harus di dalam letak quiz. Cukup tingkat yang diisi quiz yang dibandingkan.
		mismatch := []string{}
		if q.SubjectID != p.SubjectID {
			mismatch = append(mismatch, "mapel")
		}
		if q.GradeID != p.GradeID {
			mismatch = append(mismatch, "kelas")
		}
		if p.SubjectFieldID != nil && q.SubjectFieldID != *p.SubjectFieldID {
			mismatch = append(mismatch, "bidang materi")
		}
		if p.ChapterID != nil && q.ChapterID != *p.ChapterID {
			mismatch = append(mismatch, "bab")
		}
		if p.SubchapterID != nil && !equalPtr(q.SubchapterID, *p.SubchapterID) {
			mismatch = append(mismatch, "sub bab")
		}
		if p.TopicID != nil && !equalPtr(q.TopicID, *p.TopicID) {
			mismatch = append(mismatch, "topik")
		}
		if len(mismatch) > 0 {
			errs.add(field, q.Code+": "+strings.Join(mismatch, ", ")+" soal berbeda dengan quiz")
		}
	}
	return nil
}

// ListBundles mengambil 1 halaman bundle (quiz yang soalnya bisa dipakai ulang), terbaru dulu.
func (s *QuizService) ListBundles(ctx context.Context, q quizdto.ListBundlesQuery) ([]quizmodel.Bundle, quizmodel.BundleFilter, int, error) {
	filter := quizmodel.BundleFilter{
		Search:           strings.TrimSpace(q.Search),
		Type:             q.Type,
		EducationLevelID: q.EducationLevelID,
		GradeID:          q.GradeID,
		SubjectID:        q.SubjectID,
		Page:             q.Page,
		PerPage:          q.PerPage,
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PerPage == 0 {
		filter.PerPage = 20
	}
	bundles, total, err := s.repo.ListBundles(ctx, filter)
	return bundles, filter, total, err
}

// RefreshAccessCodes mengganti kode akses semua quiz yang belum selesai. Dipanggil scheduler
// setiap jam. skipped = true kalau server lain sedang mengerjakannya (advisory lock).
func (s *QuizService) RefreshAccessCodes(ctx context.Context) (refreshed int, skipped bool, err error) {
	return s.repo.RefreshAccessCodes(ctx, time.Now())
}

// equalPtr = true kalau p tidak nil dan isinya sama dengan v.
func equalPtr(p *string, v string) bool {
	return p != nil && *p == v
}

// fieldErrors mengumpulkan kesalahan supaya semuanya dikembalikan sekaligus.
type fieldErrors struct {
	list []quizmodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, quizmodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool {
	return len(e.list) > 0
}

func (e *fieldErrors) toError() error {
	return &quizmodel.ValidationError{Fields: e.list}
}
