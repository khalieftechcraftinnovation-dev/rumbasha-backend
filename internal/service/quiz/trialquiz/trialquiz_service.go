// Package trialquiz (service) berisi aturan bisnis Quiz Trial publik.
//
// Alur peserta (tanpa login):
//  1. GET  /public/trial-quiz/grades           → pilih jenjang & kelas (wajib).
//  2. GET  /public/trial-quiz/questions?grade_id → soal golongan kelas itu (maks 10) + kunci + pembahasan.
//  3. FE menilai benar/salah, lalu POST /public/trial-quiz/submit dengan nama, no WhatsApp, kelas,
//     dan jawaban. BE menghitung nilai (total 100) dan menyimpan hasilnya untuk report admin.
//
// Supaya API publik ringan: daftar kelas & soal per golongan disimpan di memori (cache) selama
// publicCacheTTL. Setiap admin mengubah soal trial, cache soal langsung dihapus.
//
// Di package lain, import dengan alias trialquizservice:
//
//	trialquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/trialquiz"
package trialquiz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	trialquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/trialquiz"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	trialquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/trialquiz"
	trialquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/trialquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/question"
)

// publicCacheTTL = lama daftar kelas & soal publik disimpan di memori. Perubahan soal oleh admin
// langsung menghapus cache soal; TTL ini berpengaruh untuk perubahan master data kelas, atau kalau
// API dijalankan lebih dari 1 server.
const publicCacheTTL = 5 * time.Minute

// TrialQuizService berisi logika bisnis Quiz Trial.
type TrialQuizService struct {
	repo            *trialquizrepo.TrialQuizRepository
	questionService *questionservice.QuestionService // validasi isi soal (aturan sama dengan bank soal)

	mu        sync.RWMutex
	grades    []trialquizmodel.GradeOption // kelas yang punya golongan
	gradesExp time.Time
	groups    map[string]cachedQuestions // golongan → soal publik
}

type cachedQuestions struct {
	questions []trialquizmodel.PublicQuestion
	expiresAt time.Time
}

// NewTrialQuizService membuat TrialQuizService. Dipanggil sekali di main.go.
func NewTrialQuizService(repo *trialquizrepo.TrialQuizRepository, questionService *questionservice.QuestionService) *TrialQuizService {
	return &TrialQuizService{repo: repo, questionService: questionService, groups: map[string]cachedQuestions{}}
}

// ============================================================================
// Admin: kelola soal
// ============================================================================

// Create membuat 1 soal trial. Isi soal dicek dengan aturan bank soal; mapel harus aktif;
// 1 golongan maksimal 10 soal.
func (s *TrialQuizService) Create(ctx context.Context, req trialquizdto.QuestionRequest, userID string) (trialquizmodel.Question, error) {
	q, err := s.build(ctx, req)
	if err != nil {
		return q, err
	}
	q.CreatedBy = userID

	id, err := s.repo.Create(ctx, q)
	if err != nil {
		return q, s.groupFullAsValidation(err)
	}
	s.clearQuestionCache()
	return s.repo.FindByID(ctx, id)
}

// Update mengganti seluruh isi 1 soal trial (boleh pindah golongan kalau golongan tujuan belum penuh).
func (s *TrialQuizService) Update(ctx context.Context, id string, req trialquizdto.QuestionRequest, userID string) (trialquizmodel.Question, error) {
	if uuid.Validate(id) != nil {
		return trialquizmodel.Question{}, trialquizmodel.ErrQuestionNotFound
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return trialquizmodel.Question{}, err
	}

	q, err := s.build(ctx, req)
	if err != nil {
		return q, err
	}
	q.ID = id
	q.UpdatedBy = &userID

	if err := s.repo.Update(ctx, q); err != nil {
		return q, s.groupFullAsValidation(err)
	}
	s.clearQuestionCache()
	return s.repo.FindByID(ctx, id)
}

// Delete menghapus 1 soal trial (soft delete).
func (s *TrialQuizService) Delete(ctx context.Context, id, userID string) error {
	if uuid.Validate(id) != nil {
		return trialquizmodel.ErrQuestionNotFound
	}
	if err := s.repo.SoftDelete(ctx, id, userID); err != nil {
		return err
	}
	s.clearQuestionCache()
	return nil
}

// GetDetail mengambil 1 soal trial lengkap (admin).
func (s *TrialQuizService) GetDetail(ctx context.Context, id string) (trialquizmodel.Question, error) {
	if uuid.Validate(id) != nil {
		return trialquizmodel.Question{}, trialquizmodel.ErrQuestionNotFound
	}
	return s.repo.FindByID(ctx, id)
}

// List mengambil soal trial 1 golongan ("" = semua) beserta jumlah soal setiap golongan.
func (s *TrialQuizService) List(ctx context.Context, group string) ([]trialquizmodel.Question, map[string]int, error) {
	questions, err := s.repo.ListByGroup(ctx, group)
	if err != nil {
		return nil, nil, err
	}
	counts, err := s.repo.CountByGroup(ctx)
	if err != nil {
		return nil, nil, err
	}
	return questions, counts, nil
}

// build mengecek request lalu mengubahnya menjadi model soal trial.
func (s *TrialQuizService) build(ctx context.Context, req trialquizdto.QuestionRequest) (trialquizmodel.Question, error) {
	content, err := s.questionService.ValidateContent(ctx, req.Type, req.ToContentRequest())
	if err != nil {
		return trialquizmodel.Question{}, err
	}

	activeSubject, err := s.repo.IsActiveSubject(ctx, req.SubjectID)
	if err != nil {
		return trialquizmodel.Question{}, fmt.Errorf("gagal mengecek mapel: %w", err)
	}
	if !activeSubject {
		return trialquizmodel.Question{}, validationError("subject_id", "mapel tidak ditemukan atau tidak aktif")
	}

	q := trialquizmodel.Question{
		GradeGroup: req.GradeGroup, SubjectID: req.SubjectID,
		Type: content.Type, QuestionText: content.QuestionText, Formula: content.Formula,
		ImageID: content.ImageID, ImageURL: content.ImageURL, Table: content.Table,
		Explanation: content.Explanation, DisplayOrder: content.DisplayOrder,
		Difficulty: req.Difficulty, Answer: content.Answer,
	}
	if req.SortOrder != nil {
		q.SortOrder = *req.SortOrder
	}
	return q, nil
}

// groupFullAsValidation mengubah ErrGroupFull menjadi error validasi field grade_group (422).
func (s *TrialQuizService) groupFullAsValidation(err error) error {
	if err == trialquizmodel.ErrGroupFull {
		return validationError("grade_group", fmt.Sprintf("golongan ini sudah berisi %d soal (maksimal), hapus soal lain dulu", trialquizmodel.MaxQuestionsPerGroup))
	}
	return err
}

// clearQuestionCache menghapus cache soal publik semua golongan (dipanggil setiap soal berubah).
func (s *TrialQuizService) clearQuestionCache() {
	s.mu.Lock()
	s.groups = map[string]cachedQuestions{}
	s.mu.Unlock()
}

// ============================================================================
// Publik
// ============================================================================

// Grades mengambil kelas yang bisa dipilih di form trial (hanya kelas yang punya golongan).
func (s *TrialQuizService) Grades(ctx context.Context) ([]trialquizmodel.GradeOption, error) {
	now := time.Now()
	s.mu.RLock()
	grades, exp := s.grades, s.gradesExp
	s.mu.RUnlock()
	if grades != nil && now.Before(exp) {
		return grades, nil
	}

	all, err := s.repo.ListGrades(ctx)
	if err != nil {
		return nil, err
	}
	grades = make([]trialquizmodel.GradeOption, 0, len(all))
	for _, g := range all {
		g.Group = trialquizmodel.GroupForGrade(g.LevelCode, g.GradeCode)
		if g.Group != "" {
			grades = append(grades, g)
		}
	}

	s.mu.Lock()
	s.grades, s.gradesExp = grades, now.Add(publicCacheTTL)
	s.mu.Unlock()
	return grades, nil
}

// findGrade mencari 1 kelas di daftar kelas trial. ok = false kalau kelas tidak ada / tidak punya golongan.
func (s *TrialQuizService) findGrade(ctx context.Context, gradeID string) (trialquizmodel.GradeOption, bool, error) {
	grades, err := s.Grades(ctx)
	if err != nil {
		return trialquizmodel.GradeOption{}, false, err
	}
	for _, g := range grades {
		if g.GradeID == gradeID {
			return g, true, nil
		}
	}
	return trialquizmodel.GradeOption{}, false, nil
}

// groupQuestions mengambil soal publik 1 golongan (dari cache kalau masih berlaku).
func (s *TrialQuizService) groupQuestions(ctx context.Context, group string) ([]trialquizmodel.PublicQuestion, error) {
	now := time.Now()
	s.mu.RLock()
	cached, ok := s.groups[group]
	s.mu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return cached.questions, nil
	}

	questions, err := s.repo.FindPublicQuestions(ctx, group)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.groups[group] = cachedQuestions{questions: questions, expiresAt: now.Add(publicCacheTTL)}
	s.mu.Unlock()
	return questions, nil
}

// PublicQuiz mengambil soal trial untuk kelas yang dipilih peserta.
// Kelas tidak dikenal → 422 grade_id; golongan belum punya soal → ErrNoQuestions.
func (s *TrialQuizService) PublicQuiz(ctx context.Context, gradeID string) (trialquizmodel.GradeOption, []trialquizmodel.PublicQuestion, error) {
	grade, ok, err := s.findGrade(ctx, gradeID)
	if err != nil {
		return grade, nil, err
	}
	if !ok {
		return grade, nil, validationError("grade_id", "kelas tidak ditemukan")
	}

	questions, err := s.groupQuestions(ctx, grade.Group)
	if err != nil {
		return grade, nil, err
	}
	if len(questions) == 0 {
		return grade, nil, trialquizmodel.ErrNoQuestions
	}
	return grade, questions, nil
}

// Submit menyimpan hasil trial: data peserta + jawaban. Nilai dihitung dari jumlah benar
// (benar/salah per soal dari FE), total 100.
func (s *TrialQuizService) Submit(ctx context.Context, req trialquizdto.SubmitRequest) (trialquizmodel.Attempt, error) {
	errs := []questionmodel.FieldError{}

	name := strings.Join(strings.Fields(req.Name), " ") // rapikan spasi berlebih
	if name == "" {
		errs = append(errs, questionmodel.FieldError{Field: "name", Message: "wajib diisi"})
	}
	whatsapp, ok := NormalizeWhatsApp(req.WhatsApp)
	if !ok {
		errs = append(errs, questionmodel.FieldError{Field: "whatsapp", Message: "nomor WhatsApp tidak valid, contoh 081234567890"})
	}
	grade, found, err := s.findGrade(ctx, req.GradeID)
	if err != nil {
		return trialquizmodel.Attempt{}, err
	}
	if !found {
		errs = append(errs, questionmodel.FieldError{Field: "grade_id", Message: "kelas tidak ditemukan"})
	}
	if len(errs) > 0 {
		return trialquizmodel.Attempt{}, &questionmodel.ValidationError{Fields: errs}
	}

	questions, err := s.groupQuestions(ctx, grade.Group)
	if err != nil {
		return trialquizmodel.Attempt{}, err
	}
	if len(questions) == 0 {
		return trialquizmodel.Attempt{}, trialquizmodel.ErrNoQuestions
	}

	ids := make([]string, 0, len(questions))
	for _, q := range questions {
		ids = append(ids, q.ID)
	}
	// Point leaderboard tidak berlaku untuk trial (pointTable nil = semua 0).
	result, gradeErrs := scoring.Grade(ids, nil, nil, questiondto.ToSubmittedAnswers(questiondto.SubmitAnswersRequest{Answers: req.Answers}))
	if len(gradeErrs) > 0 {
		fields := make([]questionmodel.FieldError, 0, len(gradeErrs))
		for _, e := range gradeErrs {
			fields = append(fields, questionmodel.FieldError{Field: e.Field, Message: e.Message})
		}
		// Biasanya karena soal trial baru saja diubah admin saat peserta mengerjakan.
		return trialquizmodel.Attempt{}, &questionmodel.ValidationError{
			Message: "Soal quiz trial sudah berubah, silakan muat ulang dan kerjakan lagi", Fields: fields,
		}
	}

	attempt := trialquizmodel.Attempt{
		Name: name, WhatsApp: whatsapp, EducationLevelID: grade.LevelID, GradeID: grade.GradeID,
		GradeGroup: grade.Group, Result: result, DurationSeconds: req.DurationSeconds,
	}
	attempt.ID, attempt.SubmittedAt, err = s.repo.InsertAttempt(ctx, attempt)
	if err != nil {
		return attempt, fmt.Errorf("gagal menyimpan hasil quiz trial: %w", err)
	}
	return attempt, nil
}

// NormalizeWhatsApp merapikan nomor WhatsApp Indonesia menjadi format 62xxxxxxxxxx.
// Diterima: "0812-3456-7890", "+62 812 3456 7890", "6281234567890", "81234567890".
// ok = false kalau bukan nomor HP yang wajar (setelah dirapikan: 62 + 8xx, total 10-15 digit).
func NormalizeWhatsApp(raw string) (string, bool) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(raw) {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '+' && i == 0, r == ' ', r == '-', r == '.', r == '(', r == ')':
			// pemisah yang biasa diketik, diabaikan
		default:
			return "", false
		}
	}
	digits := b.String()
	switch {
	case strings.HasPrefix(digits, "62"):
	case strings.HasPrefix(digits, "0"):
		digits = "62" + digits[1:]
	case strings.HasPrefix(digits, "8"):
		digits = "62" + digits
	default:
		return "", false
	}
	if !strings.HasPrefix(digits, "628") || len(digits) < 10 || len(digits) > 15 {
		return "", false
	}
	return digits, true
}

// ============================================================================
// Admin: report
// ============================================================================

// ListAttempts mengambil 1 halaman peserta trial (terbaru dulu).
func (s *TrialQuizService) ListAttempts(ctx context.Context, q trialquizdto.AttemptsQuery) ([]trialquizmodel.AttemptListItem, trialquizmodel.AttemptFilter, int, error) {
	f, err := toFilter(q.DateFrom, q.DateTo, q.GradeGroup)
	if err != nil {
		return nil, f, 0, err
	}
	f.Search = strings.TrimSpace(q.Search)
	f.Page, f.PerPage = q.Page, q.PerPage
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	items, total, err := s.repo.ListAttempts(ctx, f)
	return items, f, total, err
}

// Summary menghitung ringkasan report trial (peserta, rata-rata nilai, soal paling sering salah).
func (s *TrialQuizService) Summary(ctx context.Context, q trialquizdto.SummaryQuery) (trialquizmodel.Summary, error) {
	f, err := toFilter(q.DateFrom, q.DateTo, q.GradeGroup)
	if err != nil {
		return trialquizmodel.Summary{}, err
	}
	return s.repo.Summary(ctx, f)
}

// toFilter membaca date_from & date_to (WIB, inklusif; format sudah dicek binding) menjadi [From, To).
func toFilter(dateFrom, dateTo, group string) (trialquizmodel.AttemptFilter, error) {
	f := trialquizmodel.AttemptFilter{GradeGroup: group}
	if dateFrom != "" {
		t, _ := time.ParseInLocation(time.DateOnly, dateFrom, clock.Jakarta)
		f.From = &t
	}
	if dateTo != "" {
		t, _ := time.ParseInLocation(time.DateOnly, dateTo, clock.Jakarta)
		t = t.AddDate(0, 0, 1) // inklusif: sampai akhir hari date_to
		f.To = &t
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		return f, validationError("date_to", "tidak boleh sebelum date_from")
	}
	return f, nil
}

func validationError(field, message string) error {
	return &questionmodel.ValidationError{Fields: []questionmodel.FieldError{{Field: field, Message: message}}}
}
