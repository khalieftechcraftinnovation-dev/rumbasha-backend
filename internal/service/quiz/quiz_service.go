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

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/accesscode"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// QuizService berisi logika bisnis quiz.
type QuizService struct {
	repo         *quizrepo.QuizRepository
	questionRepo *questionrepo.QuestionRepository // isi soal + kunci jawaban saat quiz dibuka siswa
}

// NewQuizService membuat QuizService. Dipanggil sekali di main.go.
func NewQuizService(repo *quizrepo.QuizRepository, questionRepo *questionrepo.QuestionRepository) *QuizService {
	return &QuizService{repo: repo, questionRepo: questionRepo}
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
// Status mengikuti tombol simpan (draft / published); quiz langsung mendapat kode akses acak.
func (s *QuizService) Create(ctx context.Context, req quizdto.CreateQuizRequest, createdBy string) (quizmodel.QuizDetail, error) {
	quiz, err := s.buildQuiz(ctx, req, "")
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	quiz.CreatedBy = createdBy

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
		if errors.Is(err, quizmodel.ErrScheduleConflict) {
			return quizmodel.QuizDetail{}, scheduleConflictError()
		}
		if err != nil {
			return quizmodel.QuizDetail{}, fmt.Errorf("gagal menyimpan quiz: %w", err)
		}
		break
	}
	return s.repo.FindDetailByID(ctx, quiz.ID)
}

// Update mengganti seluruh isi 1 quiz (ganti penuh, aturan sama dengan create), lalu
// mengembalikan detail terbarunya. Kode akses & status tidak berubah.
// Quiz yang event-nya sudah dimulai tidak bisa diubah (ErrQuizStarted).
func (s *QuizService) Update(ctx context.Context, id string, req quizdto.CreateQuizRequest, updatedBy string) (quizmodel.QuizDetail, error) {
	if uuid.Validate(id) != nil {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	startAt, err := s.repo.FindStartAt(ctx, id)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	if !time.Now().Before(startAt) {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizStarted
	}

	quiz, err := s.buildQuiz(ctx, req, id)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	quiz.ID = id
	quiz.UpdatedBy = &updatedBy

	err = s.repo.Update(ctx, quiz)
	if errors.Is(err, quizmodel.ErrScheduleConflict) {
		return quizmodel.QuizDetail{}, scheduleConflictError()
	}
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	return s.repo.FindDetailByID(ctx, id)
}

// scheduleConflictError = pesan bentrok jadwal saat database menolak (2 request bersamaan).
func scheduleConflictError() error {
	return &quizmodel.ValidationError{Fields: []quizmodel.FieldError{{
		Field:   "start_at",
		Message: "jadwal beririsan dengan quiz lain untuk mapel & kelas yang sama",
	}}}
}

// buildQuiz mengecek isi request create/update lalu menyusun quiz yang siap disimpan.
// excludeQuizID = quiz yang sedang diubah (supaya tidak dianggap bentrok dengan dirinya sendiri).
//
// Urutan: cek isi tanpa database → kumpulkan soal dari bundle (1 query) + soal satuan → cek letak
// (1 query master data) → cek bentrok jadwal (1 query) → cek soal (1 query questions).
// Semua kesalahan dikumpulkan dan dikembalikan sekaligus (ValidationError).
func (s *QuizService) buildQuiz(ctx context.Context, req quizdto.CreateQuizRequest, excludeQuizID string) (quizmodel.Quiz, error) {
	errs := &fieldErrors{}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		errs.add("name", "wajib diisi")
	}
	startAt, endAt, scheduleOK := checkSchedule(req, errs)

	selected, err := s.collectQuestions(ctx, req, errs)
	if err != nil {
		return quizmodel.Quiz{}, err
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
		return quizmodel.Quiz{}, err
	}
	// Bentrok jadwal hanya dicek kalau jadwal & letak sudah benar (supaya pesannya masuk akal).
	if scheduleOK && placementOK {
		if err := s.checkScheduleConflict(ctx, p, startAt, endAt, excludeQuizID, errs); err != nil {
			return quizmodel.Quiz{}, err
		}
	}
	if err := s.checkQuestions(ctx, req, selected, placementOK, errs); err != nil {
		return quizmodel.Quiz{}, err
	}
	if errs.any() {
		return quizmodel.Quiz{}, errs.toError()
	}

	coinCost := 0
	if req.CoinCost != nil {
		coinCost = *req.CoinCost
	}
	questionIDs := make([]string, 0, len(selected))
	for _, q := range selected {
		questionIDs = append(questionIDs, q.ID)
	}
	return quizmodel.Quiz{
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
		EndAt:            endAt,
		DurationMinutes:  req.DurationMinutes,
		Status:           req.Status,
		CoinCost:         coinCost,
		TotalQuestions:   len(questionIDs),
		QuestionIDs:      questionIDs,
	}, nil
}

// checkSchedule mengecek jendela event: format start_at & end_at, start_at setelah sekarang,
// dan panjang event (end_at - start_at) minimal sepanjang duration_minutes. ok = false kalau ada salah.
func checkSchedule(req quizdto.CreateQuizRequest, errs *fieldErrors) (startAt, endAt time.Time, ok bool) {
	before := len(errs.list)
	const formatMsg = "format tidak valid, contoh 2026-12-01T08:00 (WIB)"

	startAt, startOK := parseWIB(req.StartAt)
	endAt, endOK := parseWIB(req.EndAt)
	if !startOK {
		errs.add("start_at", formatMsg)
	} else if !startAt.After(time.Now()) {
		errs.add("start_at", "harus setelah waktu sekarang")
	}
	if !endOK {
		errs.add("end_at", formatMsg)
	}
	if startOK && endOK {
		minEnd := startAt.Add(time.Duration(req.DurationMinutes) * time.Minute)
		switch {
		case !endAt.After(startAt):
			errs.add("end_at", "harus setelah start_at")
		case endAt.Before(minEnd):
			errs.add("end_at", fmt.Sprintf("minimal %s (start_at + durasi %d menit)", formatWIB(minEnd), req.DurationMinutes))
		}
	}
	return startAt, endAt, len(errs.list) == before
}

// checkScheduleConflict mengecek tidak ada quiz lain dengan mapel & kelas sama yang jadwalnya
// beririsan (1 query, index GiST). Database juga menjaga aturan ini (constraint quizzes_no_overlap).
func (s *QuizService) checkScheduleConflict(ctx context.Context, p quizdto.QuizPlacementRequest, startAt, endAt time.Time, excludeQuizID string, errs *fieldErrors) error {
	conflicts, err := s.repo.FindScheduleConflicts(ctx, p.SubjectID, p.GradeID, startAt, endAt, excludeQuizID)
	if err != nil {
		return fmt.Errorf("gagal mengecek bentrok jadwal: %w", err)
	}
	for _, c := range conflicts {
		errs.add("start_at", fmt.Sprintf("jadwal beririsan dengan quiz %q (%s - %s) untuk mapel & kelas yang sama",
			c.Name, formatWIB(c.StartAt), formatWIB(c.EndAt)))
	}
	return nil
}

// wibLayouts adalah format waktu tanpa zona waktu; semuanya dianggap WIB.
// Format pertama sama dengan nilai input HTML datetime-local.
var wibLayouts = []string{
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
}

// parseWIB membaca waktu dari request. RFC3339 dengan zona waktu dipakai apa adanya
// ("2026-12-01T01:00:00Z" = 08:00 WIB); tanpa zona waktu dianggap WIB.
func parseWIB(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	for _, layout := range wibLayouts {
		if t, err := time.ParseInLocation(layout, s, clock.Jakarta); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatWIB menampilkan waktu untuk pesan error, contoh "01-12-2026 08:00 WIB".
func formatWIB(t time.Time) string {
	return t.In(clock.Jakarta).Format("02-01-2006 15:04") + " WIB"
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

// List mengambil 1 halaman list quiz (terbaru dibuat dulu) beserta total yang cocok dengan filter.
// now = waktu acuan status jadwal (dikembalikan supaya response memakai waktu yang sama).
func (s *QuizService) List(ctx context.Context, q quizdto.ListQuizzesQuery) ([]quizmodel.QuizDetail, quizmodel.ListFilter, int, time.Time, error) {
	filter := toListFilter(q)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PerPage == 0 {
		filter.PerPage = 20
	}
	now := time.Now()
	items, total, err := s.repo.List(ctx, filter, now)
	return items, filter, total, now, err
}

// CountBySchedule menghitung quiz per status jadwal (angka tab Akan dimulai / Berjalan / Sudah lewat),
// dengan filter yang sama seperti List. schedule_status & pagination diabaikan.
func (s *QuizService) CountBySchedule(ctx context.Context, q quizdto.ListQuizzesQuery) (quizmodel.ScheduleCounts, error) {
	return s.repo.CountBySchedule(ctx, toListFilter(q), time.Now())
}

// toListFilter mengubah query string list quiz menjadi filter repository.
func toListFilter(q quizdto.ListQuizzesQuery) quizmodel.ListFilter {
	return quizmodel.ListFilter{
		Search:           strings.TrimSpace(q.Search),
		Type:             q.Type,
		Status:           q.Status,
		Schedule:         q.ScheduleStatus,
		EducationLevelID: q.EducationLevelID,
		GradeID:          q.GradeID,
		SubjectID:        q.SubjectID,
		Page:             q.Page,
		PerPage:          q.PerPage,
	}
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

// GetDetail mengambil 1 quiz lengkap dengan letak (nama), jadwal, kode akses, dan daftar soal.
// ID yang bukan UUID langsung dianggap tidak ditemukan (tanpa query ke database).
func (s *QuizService) GetDetail(ctx context.Context, id string) (quizmodel.QuizDetail, error) {
	if uuid.Validate(id) != nil {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	return s.repo.FindDetailByID(ctx, id)
}

// DeleteMany menghapus 1-100 quiz sekaligus (soft delete). Semua atau tidak sama sekali:
// kalau ada ID yang tidak ada / sudah dihapus, atau quiz yang event-nya sedang berjalan,
// tidak ada yang terhapus dan semua kesalahan dikembalikan (field "ids[i]").
// Quiz yang belum dimulai maupun yang sudah selesai boleh dihapus.
func (s *QuizService) DeleteMany(ctx context.Context, ids []string, deletedBy string) ([]string, error) {
	now := time.Now()
	err := s.repo.SoftDeleteMany(ctx, ids, deletedBy, func(found []quizrepo.DeleteCandidate) error {
		byID := make(map[string]quizrepo.DeleteCandidate, len(found))
		for _, q := range found {
			byID[q.ID] = q
		}

		errs := &fieldErrors{}
		for i, id := range ids {
			field := fmt.Sprintf("ids[%d]", i)
			q, ok := byID[id]
			switch {
			case !ok:
				errs.add(field, "quiz tidak ditemukan atau sudah dihapus")
			case quizmodel.ScheduleStatus(q.StartAt, q.EndAt, now) == quizmodel.ScheduleOngoing:
				errs.add(field, fmt.Sprintf("quiz %q sedang berjalan (sampai %s), tidak bisa dihapus", q.Name, formatWIB(q.EndAt)))
			}
		}
		if errs.any() {
			return errs.toError()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ListAnnouncements mengambil pengumuman quiz untuk siswa (dashboard): Uji Kom & Tryout yang
// sudah publish, belum selesai, dan untuk kelas siswa, urut yang paling dekat dimulai.
// Siswa yang belum punya kelas mendapat daftar kosong.
func (s *QuizService) ListAnnouncements(ctx context.Context, studentID string, q quizdto.AnnouncementsQuery) ([]quizmodel.QuizDetail, int, int, int, time.Time, error) {
	page, perPage := q.Page, q.PerPage
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 20
	}
	now := time.Now()

	gradeID, err := s.repo.FindUserGrade(ctx, studentID)
	if err != nil {
		return nil, 0, 0, 0, now, err
	}
	if gradeID == nil {
		return []quizmodel.QuizDetail{}, 0, page, perPage, now, nil
	}
	items, total, err := s.repo.ListAnnouncements(ctx, *gradeID, now, page, perPage)
	return items, total, page, perPage, now, err
}

// Open membuka quiz Uji Kom / Tryout untuk dikerjakan siswa, lalu mengembalikan quiz lengkap
// dengan soal + kunci jawaban (tanpa pembahasan).
//
// Syarat: quiz sudah publish, untuk kelas siswa (selain itu dianggap tidak ditemukan), dan
// event sedang berjalan. Pertama kali dibuka wajib kode akses yang benar; setelah itu (contoh
// refresh halaman saat kode sudah berganti) tidak perlu kode lagi.
//
// Kalau siswa SUDAH submit quiz ini, review yang dikembalikan (review != nil), tanpa kode akses
// dan walaupun event sudah selesai: FE langsung mengarahkan siswa ke halaman review.
func (s *QuizService) Open(ctx context.Context, quizID, studentID, accessCode string) (opened quizmodel.OpenedQuiz, review *quizmodel.QuizReview, err error) {
	detail, err := s.loadForStudent(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.OpenedQuiz{}, nil, err
	}
	attempt, err := s.repo.FindAttempt(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.OpenedQuiz{}, nil, err
	}
	if attempt != nil && attempt.SubmittedAt != nil {
		r, err := s.buildReview(ctx, detail, *attempt)
		if err != nil {
			return quizmodel.OpenedQuiz{}, nil, err
		}
		return quizmodel.OpenedQuiz{}, &r, nil
	}

	now := time.Now()
	switch quizmodel.ScheduleStatus(detail.StartAt, detail.EndAt, now) {
	case quizmodel.ScheduleUpcoming:
		return quizmodel.OpenedQuiz{}, nil, &quizmodel.ClosedError{Message: "Quiz belum dimulai. Mulai " + formatWIB(detail.StartAt)}
	case quizmodel.ScheduleFinished:
		return quizmodel.OpenedQuiz{}, nil, &quizmodel.ClosedError{Message: "Quiz sudah selesai pada " + formatWIB(detail.EndAt)}
	}

	if attempt == nil {
		// Pertama kali dibuka: wajib kode akses yang berlaku sekarang (tidak peka huruf besar/kecil).
		code := strings.ToUpper(strings.TrimSpace(accessCode))
		switch {
		case code == "":
			return quizmodel.OpenedQuiz{}, nil, accessCodeError("wajib diisi")
		case code != *detail.AccessCode:
			return quizmodel.OpenedQuiz{}, nil, accessCodeError("kode akses salah atau sudah berganti, minta kode terbaru ke tutor")
		}
		created, err := s.repo.CreateAttempt(ctx, quizID, studentID)
		if err != nil {
			return quizmodel.OpenedQuiz{}, nil, err
		}
		attempt = &created
	}

	ids := make([]string, 0, len(detail.Questions))
	for _, q := range detail.Questions {
		ids = append(ids, q.ID)
	}
	content, err := s.questionRepo.FindForAttempt(ctx, ids, false)
	if err != nil {
		return quizmodel.OpenedQuiz{}, nil, err
	}

	return quizmodel.OpenedQuiz{QuizDetail: detail, StartedAt: attempt.StartedAt, DeadlineAt: deadlineOf(detail, *attempt), Content: content}, nil, nil
}

// loadForStudent mengambil quiz yang boleh diakses siswa: Uji Kom / Tryout yang sudah publish dan
// untuk kelas siswa. Selain itu (draf, kelas lain, dihapus, bukan UUID) → ErrQuizNotFound, supaya
// siswa tidak tahu quiz tersebut ada.
func (s *QuizService) loadForStudent(ctx context.Context, quizID, studentID string) (quizmodel.QuizDetail, error) {
	if uuid.Validate(quizID) != nil {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	detail, err := s.repo.FindDetailByID(ctx, quizID)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	gradeID, err := s.repo.FindUserGrade(ctx, studentID)
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}
	if detail.Status != quizmodel.StatusPublished || detail.AccessCode == nil ||
		gradeID == nil || *gradeID != detail.GradeID {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	return detail, nil
}

// deadlineOf = batas mengerjakan: mulai mengerjakan + durasi, tapi tidak melewati akhir event.
func deadlineOf(detail quizmodel.QuizDetail, attempt quizmodel.Attempt) time.Time {
	deadline := attempt.StartedAt.Add(time.Duration(detail.DurationMinutes) * time.Minute)
	if detail.EndAt.Before(deadline) {
		return detail.EndAt
	}
	return deadline
}

// submitGrace = toleransi submit setelah batas waktu (jeda jaringan saat FE submit otomatis
// ketika timer habis).
const submitGrace = 2 * time.Minute

// Submit menyimpan jawaban siswa untuk Uji Kom / Tryout yang sudah dibuka, menghitung nilai
// (total 100, per soal = 100 / total soal; benar/salah dari FE), lalu mengembalikan review.
// Hanya sekali; setelah batas waktu (+ toleransi 2 menit) ditolak.
func (s *QuizService) Submit(ctx context.Context, quizID, studentID string, submitted []scoring.SubmittedAnswer) (quizmodel.QuizReview, error) {
	detail, err := s.loadForStudent(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	attempt, err := s.repo.FindAttempt(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	if attempt == nil {
		return quizmodel.QuizReview{}, &quizmodel.ClosedError{Message: "Quiz belum dibuka. Buka quiz dulu dengan kode akses"}
	}
	if attempt.SubmittedAt != nil {
		return quizmodel.QuizReview{}, alreadySubmitted(*attempt.SubmittedAt)
	}
	now := time.Now()
	if deadline := deadlineOf(detail, *attempt); now.After(deadline.Add(submitGrace)) {
		return quizmodel.QuizReview{}, &quizmodel.ClosedError{Message: "Waktu mengerjakan sudah habis pada " + formatWIB(deadline)}
	}

	questionIDs := make([]string, 0, len(detail.Questions))
	for _, q := range detail.Questions {
		questionIDs = append(questionIDs, q.ID)
	}
	result, fieldErrs := scoring.Grade(questionIDs, submitted)
	if len(fieldErrs) > 0 {
		errs := &fieldErrors{}
		for _, f := range fieldErrs {
			errs.add(f.Field, f.Message)
		}
		return quizmodel.QuizReview{}, errs.toError()
	}

	result.DurationSeconds = scoring.DurationSeconds(attempt.StartedAt, now)
	ok, err := s.repo.SubmitAttempt(ctx, attempt.ID, result, now)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	if !ok { // 2 request submit bersamaan: yang kedua kalah
		return quizmodel.QuizReview{}, alreadySubmitted(now)
	}
	attempt.SubmittedAt, attempt.Score, attempt.CorrectCount, attempt.Answers = &now, result.Score, result.CorrectCount, result.Answers
	attempt.DurationSeconds = result.DurationSeconds
	return s.buildReview(ctx, detail, *attempt)
}

// Review mengambil hasil & review quiz yang sudah di-submit siswa (contoh dibuka ulang dari riwayat).
func (s *QuizService) Review(ctx context.Context, quizID, studentID string) (quizmodel.QuizReview, error) {
	detail, err := s.loadForStudent(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	attempt, err := s.repo.FindAttempt(ctx, quizID, studentID)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	if attempt == nil || attempt.SubmittedAt == nil {
		return quizmodel.QuizReview{}, &quizmodel.ClosedError{Message: "Belum ada jawaban yang dikumpulkan untuk quiz ini"}
	}
	return s.buildReview(ctx, detail, *attempt)
}

// buildReview menyusun review: soal urut nomor (dengan pembahasan) + jawaban siswa yang disimpan.
func (s *QuizService) buildReview(ctx context.Context, detail quizmodel.QuizDetail, attempt quizmodel.Attempt) (quizmodel.QuizReview, error) {
	ids := make([]string, 0, len(detail.Questions))
	for _, q := range detail.Questions {
		ids = append(ids, q.ID)
	}
	content, err := s.questionRepo.FindForAttempt(ctx, ids, true)
	if err != nil {
		return quizmodel.QuizReview{}, err
	}
	return quizmodel.QuizReview{QuizDetail: detail, Attempt: attempt, Items: scoring.Review(content, attempt.Answers)}, nil
}

func alreadySubmitted(at time.Time) error {
	return &quizmodel.ClosedError{Message: "Jawaban sudah dikumpulkan pada " + formatWIB(at) + ". Lihat hasilnya di review"}
}

func accessCodeError(message string) error {
	return &quizmodel.ValidationError{Fields: []quizmodel.FieldError{{Field: "access_code", Message: message}}}
}

// GetAccessCode mengambil kode akses 1 quiz yang berlaku sekarang (untuk tutor/admin/super admin
// yang membagikannya ke siswa). Quiz tidak ada / sudah dihapus → ErrQuizNotFound.
func (s *QuizService) GetAccessCode(ctx context.Context, quizID string) (quizmodel.QuizDetail, error) {
	if uuid.Validate(quizID) != nil {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	return s.repo.FindDetailByID(ctx, quizID)
}
