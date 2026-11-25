// Package dailyquiz (service) berisi logika Daily Quiz: pilihan kelas siswa dan generate soal acak.
//
// Di package lain, import dengan alias dailyquizservice:
//
//	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dailyquiz"
package dailyquiz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"

	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/dailyquiz"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dailyquiz"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
)

// DailyQuizService berisi logika Daily Quiz.
type DailyQuizService struct {
	repo         *dailyquizrepo.DailyQuizRepository
	questionRepo *questionrepo.QuestionRepository // isi soal + kunci jawaban saat dibuka
	freePerDay   int                              // jatah gratis per hari (DAILY_QUIZ_FREE_QUOTA)
	coinCost     int                              // coin per Daily Quiz setelah jatah habis (DAILY_QUIZ_COIN_COST)
}

// NewDailyQuizService membuat DailyQuizService. Dipanggil sekali di main.go.
// freePerDay & coinCost sementara dari .env; nanti dipindah ke master parameter quiz.
func NewDailyQuizService(repo *dailyquizrepo.DailyQuizRepository, questionRepo *questionrepo.QuestionRepository, freePerDay, coinCost int) *DailyQuizService {
	return &DailyQuizService{repo: repo, questionRepo: questionRepo, freePerDay: freePerDay, coinCost: coinCost}
}

// Open membuka Daily Quiz milik siswa untuk dikerjakan, lalu mengembalikan soal lengkap dengan
// kunci jawaban (tanpa pembahasan). Tidak ada kode akses & timer.
//
// Jatah: setiap hari (WIB) siswa bisa membuka freePerDay Daily Quiz gratis; berikutnya memotong
// coinCost coin per Daily Quiz. Coin kurang → InsufficientCoinError (tidak ada yang berubah).
// Membuka ulang Daily Quiz yang sama tidak dihitung lagi.
//
// Kalau Daily Quiz ini SUDAH di-submit, review yang dikembalikan (review != nil): FE langsung
// mengarahkan siswa ke halaman review.
func (s *DailyQuizService) Open(ctx context.Context, quizID, studentID string) (dailyquizmodel.OpenedDailyQuiz, *dailyquizmodel.DailyQuizReview, error) {
	if uuid.Validate(quizID) != nil {
		return dailyquizmodel.OpenedDailyQuiz{}, nil, dailyquizmodel.ErrDailyQuizNotFound
	}
	state, err := s.repo.FindSubmission(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.OpenedDailyQuiz{}, nil, err
	}
	if state.SubmittedAt != nil {
		detail, err := s.repo.FindDetail(ctx, quizID, studentID)
		if err != nil {
			return dailyquizmodel.OpenedDailyQuiz{}, nil, err
		}
		review, err := s.buildReview(ctx, detail, state)
		if err != nil {
			return dailyquizmodel.OpenedDailyQuiz{}, nil, err
		}
		return dailyquizmodel.OpenedDailyQuiz{}, &review, nil
	}

	opened, err := s.open(ctx, quizID, studentID)
	return opened, nil, err
}

// open menjalankan buka Daily Quiz yang belum di-submit (jatah gratis / potong coin) lalu
// mengambil soal + kunci jawaban.
func (s *DailyQuizService) open(ctx context.Context, quizID, studentID string) (dailyquizmodel.OpenedDailyQuiz, error) {
	now := time.Now()
	result, err := s.repo.Open(ctx, quizID, studentID, now, clock.StartOfDayInJakarta(now),
		func(openedToday, balance int) (int, error) {
			if openedToday < s.freePerDay {
				return 0, nil // masih jatah gratis
			}
			if balance < s.coinCost {
				return 0, &dailyquizmodel.InsufficientCoinError{Need: s.coinCost, Balance: balance}
			}
			return s.coinCost, nil
		})
	if err != nil {
		return dailyquizmodel.OpenedDailyQuiz{}, err
	}

	detail, err := s.repo.FindDetail(ctx, quizID, studentID)
	if err != nil {
		return dailyquizmodel.OpenedDailyQuiz{}, err
	}
	content, err := s.questionRepo.FindForAttempt(ctx, detail.QuestionIDs, false)
	if err != nil {
		return dailyquizmodel.OpenedDailyQuiz{}, err
	}
	return dailyquizmodel.OpenedDailyQuiz{
		DailyQuizDetail: detail, OpenResult: result, FreePerDay: s.freePerDay, Content: content,
	}, nil
}

// GradeOptions mengembalikan kelas yang boleh dipilih siswa untuk Daily Quiz (2 query):
// kelas-kelas di jenjang siswa, dan untuk SD hanya kelompoknya (Kelas 1-3 atau Kelas 4-6).
// Kelas siswa sendiri ditandai IsDefault. Kelas/jenjang siswa belum diisi → ErrStudentGradeMissing.
func (s *DailyQuizService) GradeOptions(ctx context.Context, studentID string) ([]dailyquizmodel.GradeOption, dailyquizmodel.Student, error) {
	student, err := s.repo.FindStudent(ctx, studentID)
	if err != nil {
		return nil, student, err
	}
	if student.GradeID == nil || student.EducationLevelID == nil {
		return nil, student, dailyquizmodel.ErrStudentGradeMissing
	}

	grades, err := s.repo.FindGradesInLevel(ctx, *student.EducationLevelID)
	if err != nil {
		return nil, student, err
	}

	// Kelompok kelas siswa ditentukan dari code kelasnya (contoh GRADE_5 → SD_4_6).
	var studentGroup string
	for _, g := range grades {
		if g.ID == *student.GradeID {
			studentGroup = dailyquizmodel.GradeGroup(g.Code, g.ParentID)
		}
	}
	if studentGroup == "" { // kelas siswa tidak ada / tidak aktif di jenjangnya: data siswa salah
		return nil, student, dailyquizmodel.ErrStudentGradeMissing
	}

	options := make([]dailyquizmodel.GradeOption, 0, len(grades))
	for _, g := range grades {
		if dailyquizmodel.GradeGroup(g.Code, g.ParentID) == studentGroup {
			g.IsDefault = g.ID == *student.GradeID
			options = append(options, g)
		}
	}
	return options, student, nil
}

// Generate membuat Daily Quiz dari soal acak sesuai pilihan siswa, lalu mengembalikan soalnya
// (tanpa kunci jawaban). Tidak ada kode akses & timer.
//
// Urutan: pilihan kelas (2 query) → cek letak (1 query master data) → ambil soal acak (1 query)
// → simpan (1 transaksi) → isi soal untuk siswa (2 query).
func (s *DailyQuizService) Generate(ctx context.Context, studentID string, req dailyquizdto.GenerateRequest) (dailyquizmodel.DailyQuizDetail, error) {
	options, student, err := s.GradeOptions(ctx, studentID)
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, err
	}

	errs := &fieldErrors{}
	p := req.Placement

	// Kelas: default kelas siswa; kalau dipilih harus termasuk pilihan yang diizinkan.
	gradeID := *student.GradeID
	if p.GradeID != nil {
		gradeID = *p.GradeID
	}
	var grade *dailyquizmodel.GradeOption
	names := make([]string, 0, len(options))
	for i := range options {
		names = append(names, options[i].Name)
		if options[i].ID == gradeID {
			grade = &options[i]
		}
	}
	if grade == nil {
		errs.add("placement.grade_id", "kelas hanya boleh: "+strings.Join(names, ", "))
		return dailyquizmodel.DailyQuizDetail{}, errs.toError()
	}

	items, err := s.checkPlacement(ctx, p, gradeID, grade.ParentID, errs)
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, err
	}
	if errs.any() {
		return dailyquizmodel.DailyQuizDetail{}, errs.toError()
	}

	questionIDs, err := s.repo.PickRandomQuestionIDs(ctx, dailyquizmodel.QuestionFilter{
		GradeID: gradeID, SubjectFieldID: p.SubjectFieldID,
		ChapterID: p.ChapterID, SubchapterID: p.SubchapterID, TopicID: p.TopicID,
		Difficulty: req.Difficulty, Limit: req.TotalQuestions,
	})
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, fmt.Errorf("gagal mengambil soal acak: %w", err)
	}
	switch {
	case len(questionIDs) == 0:
		errs.add("total_questions", "belum ada soal untuk pilihan ini, ubah pilihan mapel/bab/level")
	case len(questionIDs) < req.TotalQuestions:
		errs.add("total_questions", fmt.Sprintf("soal yang tersedia hanya %d, pilih total soal yang lebih kecil atau ubah pilihan", len(questionIDs)))
	}
	if errs.any() {
		return dailyquizmodel.DailyQuizDetail{}, errs.toError()
	}

	quiz := dailyquizmodel.DailyQuiz{
		StudentID:        studentID,
		EducationLevelID: grade.ParentID,
		GradeID:          gradeID,
		SubjectID:        p.SubjectID,
		SubjectFieldID:   p.SubjectFieldID,
		ChapterID:        p.ChapterID,
		SubchapterID:     p.SubchapterID,
		TopicID:          p.TopicID,
		Difficulty:       req.Difficulty,
		TotalQuestions:   len(questionIDs),
		QuestionIDs:      questionIDs,
	}
	if err := s.repo.Create(ctx, &quiz); err != nil {
		return dailyquizmodel.DailyQuizDetail{}, fmt.Errorf("gagal menyimpan daily quiz: %w", err)
	}

	questions, err := s.repo.FindStudentQuestions(ctx, questionIDs)
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, err
	}

	// Nama letak diambil dari master data yang sudah dibaca saat validasi (tanpa query tambahan).
	nameOf := func(id *string) *string {
		if id == nil {
			return nil
		}
		name := items[*id].Name
		return &name
	}
	return dailyquizmodel.DailyQuizDetail{
		DailyQuiz:          quiz,
		EducationLevelName: items[grade.ParentID].Name,
		GradeName:          grade.Name,
		SubjectName:        items[p.SubjectID].Name,
		SubjectFieldName:   items[p.SubjectFieldID].Name,
		ChapterName:        nameOf(p.ChapterID),
		SubchapterName:     nameOf(p.SubchapterID),
		TopicName:          nameOf(p.TopicID),
		Questions:          questions,
	}, nil
}

// checkPlacement mengecek pilihan letak terhadap master data (1 query untuk semua tingkat):
// mapel; bidang materi milik mapel & kelas yang dipilih; bab/sub bab/topik (opsional, berurutan)
// milik induknya. Semua harus aktif. Hasilnya map master data (dipakai untuk nama di response).
func (s *DailyQuizService) checkPlacement(ctx context.Context, p dailyquizdto.PlacementRequest, gradeID, levelID string, errs *fieldErrors) (map[string]dailyquizmodel.MasterDataItem, error) {
	if p.SubchapterID != nil && p.ChapterID == nil {
		errs.add("placement.chapter_id", "wajib diisi jika subchapter_id diisi")
	}
	if p.TopicID != nil && p.SubchapterID == nil {
		errs.add("placement.subchapter_id", "wajib diisi jika topic_id diisi")
	}

	ids := []string{p.SubjectID, p.SubjectFieldID}
	for _, id := range []*string{p.ChapterID, p.SubchapterID, p.TopicID} {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	ids = append(ids, levelID) // jenjang ikut diambil, hanya untuk nama di response
	items, err := s.repo.FindMasterDataByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca master data: %w", err)
	}

	check := func(field, label, itemType, id string, parentID *string, parentLabel string) bool {
		item, found := items[id]
		switch {
		case !found || item.Type != itemType:
			errs.add(field, label+" tidak ditemukan")
		case parentID != nil && (item.ParentID == nil || *item.ParentID != *parentID):
			errs.add(field, label+" bukan bagian dari "+parentLabel+" yang dipilih")
		case item.Status != "active":
			errs.add(field, label+" tidak aktif")
		default:
			return true
		}
		return false
	}

	check("placement.subject_id", "mapel", "subject", p.SubjectID, nil, "")
	if check("placement.subject_field_id", "bidang materi", "subject_field", p.SubjectFieldID, &p.SubjectID, "mapel") {
		if field := items[p.SubjectFieldID]; field.GradeID == nil || *field.GradeID != gradeID {
			errs.add("placement.subject_field_id", "bidang materi bukan untuk kelas yang dipilih")
		}
	}
	if p.ChapterID != nil {
		check("placement.chapter_id", "bab", "chapter", *p.ChapterID, &p.SubjectFieldID, "bidang materi")
	}
	if p.SubchapterID != nil && p.ChapterID != nil {
		check("placement.subchapter_id", "sub bab", "subchapter", *p.SubchapterID, p.ChapterID, "bab")
	}
	if p.TopicID != nil && p.SubchapterID != nil {
		check("placement.topic_id", "topik", "topic", *p.TopicID, p.SubchapterID, "sub bab")
	}
	return items, nil
}

// fieldErrors mengumpulkan kesalahan supaya semuanya dikembalikan sekaligus.
type fieldErrors struct {
	list []dailyquizmodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, dailyquizmodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool {
	return len(e.list) > 0
}

func (e *fieldErrors) toError() error {
	return &dailyquizmodel.ValidationError{Fields: e.list}
}
