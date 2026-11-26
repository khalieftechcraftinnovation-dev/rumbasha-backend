// Package question (dto) berisi bentuk request dan response JSON untuk fitur bank soal.
//
// Di package lain, import dengan alias questiondto:
//
//	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
package question

import (
	"encoding/json"
	"math"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// CreateQuestionsRequest adalah body JSON untuk POST /api/v1/questions.
// Bisa membuat 1 sampai 100 soal sekaligus (batas di tag max=100), tapi SEMUANYA harus 1 tipe yang sama:
// tipe ditulis sekali di level atas, bukan per soal.
//
//	{"type": "single_choice", "questions": [{...}, {...}]}
//
// Disimpan semua atau tidak sama sekali: kalau ada 1 soal yang salah, tidak ada yang tersimpan.
type CreateQuestionsRequest struct {
	Type      string                  `json:"type" binding:"required,oneof=single_choice multiple_choice true_false free_text"`
	Questions []CreateQuestionRequest `json:"questions" binding:"required,min=1,max=100,dive"`
}

// CreateQuestionRequest adalah satu soal di dalam CreateQuestionsRequest.
//
// Aturan di tag binding hanya mengecek bentuk dasar. Aturan yang bergantung pada
// field lain (isi answer per type, display_order, kecocokan placement) dicek di service.
//
// Field "answer" dibiarkan mentah (json.RawMessage) karena bentuknya berbeda per type;
// service yang membacanya sesuai type di level atas.
//
// Type per soal TIDAK perlu diisi. Kalau diisi, harus sama dengan type di level atas
// (supaya soal bertipe lain yang terselip langsung ditolak, bukan diam-diam dianggap sama).
type CreateQuestionRequest struct {
	Type         string           `json:"type"`
	Question     string           `json:"question" binding:"required,max=5000"`
	Formula      *string          `json:"formula" binding:"omitempty,max=2000"`
	ImageID      *string          `json:"image_id" binding:"omitempty,uuid"`      // gambar dari Image Library
	ImageURL     *string          `json:"image_url" binding:"omitempty,max=2000"` // ATAU link gambar luar (http/https)
	Table        *TableRequest    `json:"table"`
	Explanation  string           `json:"explanation" binding:"required,max=5000"` // pembahasan, wajib untuk semua tipe
	DisplayOrder []string         `json:"display_order" binding:"omitempty,unique,dive,oneof=question formula image table"`
	Placement    PlacementRequest `json:"placement" binding:"required"`
	Difficulty   string           `json:"difficulty" binding:"required,oneof=easy medium hard"`
	UsedFor      []string         `json:"used_for" binding:"required,min=1,unique,dive,oneof=daily_quiz uji_kom tryout rumath"`
	Status       string           `json:"status" binding:"omitempty,oneof=active inactive"`
	Answer       json.RawMessage  `json:"answer" binding:"required"`
}

// TableRequest adalah tabel di dalam soal: maksimal 10 kolom dan 50 baris.
// Jumlah kolom tiap baris harus sama (dicek di service).
type TableRequest struct {
	Headers []string   `json:"headers" binding:"omitempty,max=10,dive,max=200"`
	Rows    [][]string `json:"rows" binding:"required,min=1,max=50,dive,min=1,max=10,dive,max=500"`
}

// PlacementRequest adalah letak soal. Wajib sampai bab, sub bab & topik opsional.
type PlacementRequest struct {
	EducationLevelID string  `json:"education_level_id" binding:"required,uuid"`
	GradeID          string  `json:"grade_id" binding:"required,uuid"`
	SubjectID        string  `json:"subject_id" binding:"required,uuid"`
	SubjectFieldID   string  `json:"subject_field_id" binding:"required,uuid"`
	ChapterID        string  `json:"chapter_id" binding:"required,uuid"`
	SubchapterID     *string `json:"subchapter_id" binding:"omitempty,uuid"`
	TopicID          *string `json:"topic_id" binding:"omitempty,uuid"`
}

// QuestionResponse adalah bentuk data soal yang dikirim ke client.
type QuestionResponse struct {
	ID           string               `json:"id"`
	Code         string               `json:"code"`
	Type         string               `json:"type"`
	Question     string               `json:"question"`
	Formula      *string              `json:"formula"`
	ImageID      *string              `json:"image_id"`
	ImageURL     *string              `json:"image_url"`
	Table        *questionmodel.Table `json:"table"`
	Explanation  string               `json:"explanation"`
	DisplayOrder []string             `json:"display_order"`
	Placement    PlacementResponse    `json:"placement"`
	Difficulty   string               `json:"difficulty"`
	UsedFor      []string             `json:"used_for"`
	Status       string               `json:"status"`
	Answer       json.RawMessage      `json:"answer"`
	CreatedBy    string               `json:"created_by"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// PlacementResponse adalah letak soal di response.
type PlacementResponse struct {
	EducationLevelID string  `json:"education_level_id"`
	GradeID          string  `json:"grade_id"`
	SubjectID        string  `json:"subject_id"`
	SubjectFieldID   string  `json:"subject_field_id"`
	ChapterID        string  `json:"chapter_id"`
	SubchapterID     *string `json:"subchapter_id"`
	TopicID          *string `json:"topic_id"`
}

// ToQuestionResponse mengubah questionmodel.Question menjadi QuestionResponse.
func ToQuestionResponse(q questionmodel.Question) QuestionResponse {
	return QuestionResponse{
		ID:           q.ID,
		Code:         q.Code,
		Type:         q.Type,
		Question:     q.QuestionText,
		Formula:      q.Formula,
		ImageID:      q.ImageID,
		ImageURL:     q.ImageURL,
		Table:        q.Table,
		Explanation:  q.Explanation,
		DisplayOrder: q.DisplayOrder,
		Placement: PlacementResponse{
			EducationLevelID: q.EducationLevelID,
			GradeID:          q.GradeID,
			SubjectID:        q.SubjectID,
			SubjectFieldID:   q.SubjectFieldID,
			ChapterID:        q.ChapterID,
			SubchapterID:     q.SubchapterID,
			TopicID:          q.TopicID,
		},
		Difficulty: q.Difficulty,
		UsedFor:    q.UsedFor,
		Status:     q.Status,
		Answer:     q.Answer,
		CreatedBy:  q.CreatedBy,
		CreatedAt:  q.CreatedAt,
		UpdatedAt:  q.UpdatedAt,
	}
}

// QuestionDetailResponse adalah data GET /api/v1/questions/:id.
// Sama seperti QuestionResponse, tapi placement & pembuat dilengkapi nama, supaya
// halaman detail/edit admin tidak perlu request tambahan.
// Berisi kunci jawaban (answer) & pembahasan, jadi HANYA untuk admin.
type QuestionDetailResponse struct {
	ID           string                  `json:"id"`
	Code         string                  `json:"code"`
	Type         string                  `json:"type"`
	Question     string                  `json:"question"`
	Formula      *string                 `json:"formula"`
	ImageID      *string                 `json:"image_id"`
	ImageURL     *string                 `json:"image_url"`
	Table        *questionmodel.Table    `json:"table"`
	Explanation  string                  `json:"explanation"`
	DisplayOrder []string                `json:"display_order"`
	Placement    PlacementDetailResponse `json:"placement"`
	Difficulty   string                  `json:"difficulty"`
	UsedFor      []string                `json:"used_for"`
	Status       string                  `json:"status"`
	Answer       json.RawMessage         `json:"answer"`
	CreatedBy    RefResponse             `json:"created_by"`
	UpdatedBy    *RefResponse            `json:"updated_by"` // null kalau belum pernah diubah
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// PlacementDetailResponse adalah letak soal lengkap dengan nama. Sub bab & topik null kalau kosong.
type PlacementDetailResponse struct {
	EducationLevel RefResponse  `json:"education_level"`
	Grade          RefResponse  `json:"grade"`
	Subject        RefResponse  `json:"subject"`
	SubjectField   RefResponse  `json:"subject_field"`
	Chapter        RefResponse  `json:"chapter"`
	Subchapter     *RefResponse `json:"subchapter"`
	Topic          *RefResponse `json:"topic"`
}

// RefResponse adalah pasangan id + nama, contoh {"id": "...", "name": "Kelas 4"}.
// name bisa null (contoh: user pembuat sudah dihapus permanen).
type RefResponse struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

// ToQuestionDetailResponse mengubah questionmodel.QuestionDetail menjadi QuestionDetailResponse.
func ToQuestionDetailResponse(d questionmodel.QuestionDetail) QuestionDetailResponse {
	return QuestionDetailResponse{
		ID:           d.ID,
		Code:         d.Code,
		Type:         d.Type,
		Question:     d.QuestionText,
		Formula:      d.Formula,
		ImageID:      d.ImageID,
		ImageURL:     d.ImageURL,
		Table:        d.Table,
		Explanation:  d.Explanation,
		DisplayOrder: d.DisplayOrder,
		Placement: PlacementDetailResponse{
			EducationLevel: ref(d.EducationLevelID, &d.EducationLevelName),
			Grade:          ref(d.GradeID, &d.GradeName),
			Subject:        ref(d.SubjectID, &d.SubjectName),
			SubjectField:   ref(d.SubjectFieldID, &d.SubjectFieldName),
			Chapter:        ref(d.ChapterID, &d.ChapterName),
			Subchapter:     optionalRef(d.SubchapterID, d.SubchapterName),
			Topic:          optionalRef(d.TopicID, d.TopicName),
		},
		Difficulty: d.Difficulty,
		UsedFor:    d.UsedFor,
		Status:     d.Status,
		Answer:     d.Answer,
		CreatedBy:  ref(d.CreatedBy, d.CreatedByName),
		UpdatedBy:  optionalRef(d.UpdatedBy, d.UpdatedByName),
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func ref(id string, name *string) RefResponse {
	return RefResponse{ID: id, Name: name}
}

// optionalRef mengembalikan nil (JSON null) kalau id kosong.
func optionalRef(id, name *string) *RefResponse {
	if id == nil {
		return nil
	}
	r := ref(*id, name)
	return &r
}

// UpdateQuestionRequest: body PUT /api/v1/questions/:id memakai CreateQuestionRequest
// (bentuk 1 soal yang sama dengan create), dengan "type" WAJIB diisi.
// Update bersifat ganti penuh: field opsional yang tidak dikirim (formula, image_id, table,
// subchapter_id, topic_id) menjadi kosong.

// DeleteQuestionsRequest adalah body DELETE /api/v1/questions: hapus 1 sampai 100 soal (soft delete).
type DeleteQuestionsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteQuestionsResponse berisi ID soal yang berhasil dihapus.
type DeleteQuestionsResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// ImportCheckResponse adalah data POST /api/v1/questions/import/check (cek file Excel).
// Soal BELUM disimpan. Bentuk "questions" sama dengan body create, jadi FE bisa langsung
// mengirim {"type": type, "questions": questions} ke POST /api/v1/questions.
// Field tambahan (row, *_name di placement) otomatis diabaikan oleh create.
type ImportCheckResponse struct {
	Type      string                     `json:"type"`
	Total     int                        `json:"total"`
	Questions []ImportedQuestionResponse `json:"questions"`
}

// ImportedQuestionResponse adalah 1 soal hasil cek Excel. row = nomor baris di Excel.
type ImportedQuestionResponse struct {
	Row          int                       `json:"row"`
	Question     string                    `json:"question"`
	Formula      *string                   `json:"formula"`
	ImageID      *string                   `json:"image_id"`
	ImageURL     *string                   `json:"image_url"`
	Table        *questionmodel.Table      `json:"table"`
	Explanation  string                    `json:"explanation"`
	DisplayOrder []string                  `json:"display_order"`
	Placement    ImportedPlacementResponse `json:"placement"`
	Difficulty   string                    `json:"difficulty"`
	UsedFor      []string                  `json:"used_for"`
	Status       string                    `json:"status"`
	Answer       json.RawMessage           `json:"answer"`
}

// ImportedPlacementResponse berisi ID (untuk create) + nama dari database (untuk ditampilkan).
type ImportedPlacementResponse struct {
	EducationLevelID   string  `json:"education_level_id"`
	EducationLevelName string  `json:"education_level_name"`
	GradeID            string  `json:"grade_id"`
	GradeName          string  `json:"grade_name"`
	SubjectID          string  `json:"subject_id"`
	SubjectName        string  `json:"subject_name"`
	SubjectFieldID     string  `json:"subject_field_id"`
	SubjectFieldName   string  `json:"subject_field_name"`
	ChapterID          string  `json:"chapter_id"`
	ChapterName        string  `json:"chapter_name"`
	SubchapterID       *string `json:"subchapter_id"`
	SubchapterName     *string `json:"subchapter_name"`
	TopicID            *string `json:"topic_id"`
	TopicName          *string `json:"topic_name"`
}

// ToImportCheckResponse mengubah hasil cek Excel menjadi response.
func ToImportCheckResponse(questionType string, items []questionmodel.ImportedQuestion) ImportCheckResponse {
	questions := make([]ImportedQuestionResponse, 0, len(items))
	for _, it := range items {
		questions = append(questions, ImportedQuestionResponse{
			Row:          it.Row,
			Question:     it.QuestionText,
			Formula:      it.Formula,
			ImageID:      it.ImageID,
			ImageURL:     it.ImageURL,
			Table:        it.Table,
			Explanation:  it.Explanation,
			DisplayOrder: it.DisplayOrder,
			Placement: ImportedPlacementResponse{
				EducationLevelID: it.EducationLevelID, EducationLevelName: it.EducationLevelName,
				GradeID: it.GradeID, GradeName: it.GradeName,
				SubjectID: it.SubjectID, SubjectName: it.SubjectName,
				SubjectFieldID: it.SubjectFieldID, SubjectFieldName: it.SubjectFieldName,
				ChapterID: it.ChapterID, ChapterName: it.ChapterName,
				SubchapterID: it.SubchapterID, SubchapterName: it.SubchapterName,
				TopicID: it.TopicID, TopicName: it.TopicName,
			},
			Difficulty: it.Difficulty,
			UsedFor:    it.UsedFor,
			Status:     it.Status,
			Answer:     it.Answer,
		})
	}
	return ImportCheckResponse{Type: questionType, Total: len(questions), Questions: questions}
}

// ListQuestionsQuery adalah query string GET /api/v1/questions (list Bank Soal).
// Semua filter opsional dan bisa digabung (AND). Contoh:
//
//	/api/v1/questions?page=1&per_page=20&type=free_text&difficulty=easy&subject_id=...&search=garis
//
// page & per_page sama dengan dto.PaginationQuery (ditulis ulang di sini supaya nama field
// di error validasi tetap "per_page", bukan "PaginationQuery.per_page").
type ListQuestionsQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search     string `form:"search" binding:"omitempty,max=100"` // kode soal / teks soal
	Type       string `form:"type" binding:"omitempty,oneof=single_choice multiple_choice true_false free_text"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=easy medium hard"` // kolom "Level"
	// Untuk memilih soal quiz: soal yang peruntukannya berisi nilai ini, dan status soal.
	UsedFor string `form:"used_for" binding:"omitempty,oneof=daily_quiz uji_kom tryout rumath"`
	Status  string `form:"status" binding:"omitempty,oneof=active inactive"`
	// Isi 1 bundle (quiz): hanya soal yang ada di quiz ini. Daftar bundle: GET /api/v1/quiz-bundles.
	QuizID string `form:"quiz_id" binding:"omitempty,uuid"`

	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID   string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID        string `form:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID     string `form:"subchapter_id" binding:"omitempty,uuid"`
	TopicID          string `form:"topic_id" binding:"omitempty,uuid"`
}

// QuestionListItemResponse adalah 1 baris tabel Bank Soal. Tidak berisi kunci jawaban &
// pembahasan; ambil lewat GET /api/v1/questions/:id kalau perlu.
type QuestionListItemResponse struct {
	ID         string                  `json:"id"`
	Code       string                  `json:"code"`
	Type       string                  `json:"type"`
	Question   string                  `json:"question"`
	Media      QuestionMediaResponse   `json:"media"`
	Placement  PlacementDetailResponse `json:"placement"`
	Difficulty string                  `json:"difficulty"`
	UsedFor    []string                `json:"used_for"`
	Status     string                  `json:"status"`
	CreatedBy  RefResponse             `json:"created_by"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedBy  *RefResponse            `json:"updated_by"` // null kalau belum pernah diubah
	UpdatedAt  time.Time               `json:"updated_at"`
	// Quiz yang sudah memakai soal ini ("bundle"), terbaru dulu. [] = belum pernah dipakai.
	Quizzes []QuizRefResponse `json:"quizzes"`
}

// QuizRefResponse adalah quiz yang memakai soal, contoh {"id": "...", "name": "Uji Kom Pecahan", "type": "uji_kom"}.
type QuizRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// QuestionMediaResponse menandai isi tambahan di soal (untuk label "+N media" di tabel).
type QuestionMediaResponse struct {
	Formula bool `json:"formula"`
	Image   bool `json:"image"` // gambar dari Image Library atau link luar
	Table   bool `json:"table"`
}

// ToQuestionListResponses mengubah hasil list menjadi response. Selalu slice (bukan nil),
// supaya hasil kosong menjadi [] di JSON, bukan null.
func ToQuestionListResponses(items []questionmodel.QuestionListItem) []QuestionListItemResponse {
	result := make([]QuestionListItemResponse, 0, len(items))
	for _, it := range items {
		quizzes := make([]QuizRefResponse, 0, len(it.Quizzes))
		for _, z := range it.Quizzes {
			quizzes = append(quizzes, QuizRefResponse{ID: z.ID, Name: z.Name, Type: z.Type})
		}

		result = append(result, QuestionListItemResponse{
			ID:       it.ID,
			Code:     it.Code,
			Type:     it.Type,
			Question: it.QuestionText,
			Media:    QuestionMediaResponse{Formula: it.HasFormula, Image: it.HasImage, Table: it.HasTable},
			Placement: PlacementDetailResponse{
				EducationLevel: ref(it.EducationLevelID, &it.EducationLevelName),
				Grade:          ref(it.GradeID, &it.GradeName),
				Subject:        ref(it.SubjectID, &it.SubjectName),
				SubjectField:   ref(it.SubjectFieldID, &it.SubjectFieldName),
				Chapter:        ref(it.ChapterID, &it.ChapterName),
				Subchapter:     optionalRef(it.SubchapterID, it.SubchapterName),
				Topic:          optionalRef(it.TopicID, it.TopicName),
			},
			Difficulty: it.Difficulty,
			UsedFor:    it.UsedFor,
			Status:     it.Status,
			CreatedBy:  ref(it.CreatedBy, it.CreatedByName),
			CreatedAt:  it.CreatedAt,
			UpdatedBy:  optionalRef(it.UpdatedBy, it.UpdatedByName),
			UpdatedAt:  it.UpdatedAt,
			Quizzes:    quizzes,
		})
	}
	return result
}

// TypeCountsResponse adalah data GET /api/v1/questions/type-counts: angka di tab Bank Soal.
// Semua tipe selalu ada (0 kalau tidak ada soalnya). all = jumlah semua tipe.
type TypeCountsResponse struct {
	All            int `json:"all"`             // tab "Semua"
	SingleChoice   int `json:"single_choice"`   // tab "Pilihan ganda"
	MultipleChoice int `json:"multiple_choice"` // tab "PG kompleks"
	TrueFalse      int `json:"true_false"`      // tab "Benar / Salah"
	FreeText       int `json:"free_text"`       // tab "Isian"
}

// ToTypeCountsResponse mengubah map type → jumlah menjadi response.
func ToTypeCountsResponse(counts map[string]int) TypeCountsResponse {
	res := TypeCountsResponse{
		SingleChoice:   counts[questionmodel.TypeSingleChoice],
		MultipleChoice: counts[questionmodel.TypeMultipleChoice],
		TrueFalse:      counts[questionmodel.TypeTrueFalse],
		FreeText:       counts[questionmodel.TypeFreeText],
	}
	res.All = res.SingleChoice + res.MultipleChoice + res.TrueFalse + res.FreeText
	return res
}

// AttemptQuestionResponse adalah 1 soal saat quiz dibuka siswa (Daily Quiz / Uji Kom / Tryout):
// isi soal + answer (opsi & KUNCI JAWABAN, bentuk sama dengan create soal), TANPA pembahasan.
// Gambar Image Library sudah berupa link (image_url soal & image_url tiap opsi).
type AttemptQuestionResponse struct {
	No           int                  `json:"no"`
	ID           string               `json:"id"`
	Code         string               `json:"code"`
	Type         string               `json:"type"`
	Question     string               `json:"question"`
	Formula      *string              `json:"formula"`
	ImageURL     *string              `json:"image_url"`
	Table        *questionmodel.Table `json:"table"`
	DisplayOrder []string             `json:"display_order"`
	Difficulty   string               `json:"difficulty"`
	Answer       json.RawMessage      `json:"answer"`
}

// ToAttemptQuestionResponses mengubah soal quiz yang dibuka menjadi response (no = urutan, mulai 1).
func ToAttemptQuestionResponses(questions []questionmodel.AttemptQuestion) []AttemptQuestionResponse {
	result := make([]AttemptQuestionResponse, 0, len(questions))
	for i, q := range questions {
		result = append(result, AttemptQuestionResponse{
			No: i + 1, ID: q.ID, Code: q.Code, Type: q.Type, Question: q.QuestionText,
			Formula: q.Formula, ImageURL: q.ImageURL, Table: q.Table, DisplayOrder: q.DisplayOrder,
			Difficulty: q.Difficulty, Answer: q.Answer,
		})
	}
	return result
}

// SubmitAnswersRequest adalah body submit jawaban quiz (Daily Quiz / Uji Kom / Tryout):
//
//	{"answers": [{"question_id": "...", "answer": "B", "is_correct": true}, ...]}
//
// Benar/salah dinilai di FE. answer = jawaban siswa apa adanya (contoh "B", ["A","C"],
// [{"key":"A","correct":true}], "0.75"); null / tidak dikirim = tidak dijawab (dianggap salah).
type SubmitAnswersRequest struct {
	Answers []SubmitAnswerItem `json:"answers" binding:"max=200,dive"`
}

// SubmitAnswerItem adalah jawaban 1 soal.
type SubmitAnswerItem struct {
	QuestionID string          `json:"question_id" binding:"required,uuid"`
	Answer     json.RawMessage `json:"answer"`
	IsCorrect  *bool           `json:"is_correct" binding:"required"`
	// Waktu per soal dari FE (opsional): jam soal terakhir dijawab (RFC3339, contoh
	// "2026-12-01T08:05:30+07:00") dan lama mengerjakan soal itu dalam detik.
	AnsweredAt       *time.Time `json:"answered_at"`
	TimeSpentSeconds *int       `json:"time_spent_seconds" binding:"omitempty,min=0,max=86400"`
}

// ReviewQuestionResponse adalah 1 soal di review setelah submit (urutan sama dengan saat
// dikerjakan): isi soal, kunci jawaban (answer), pembahasan, jawaban siswa, dan hasilnya.
type ReviewQuestionResponse struct {
	AttemptQuestionResponse
	Explanation      string          `json:"explanation"`
	StudentAnswer    json.RawMessage `json:"student_answer"` // null = tidak dijawab
	IsCorrect        bool            `json:"is_correct"`
	Score            float64         `json:"score"`              // NILAI yang didapat dari soal ini (100 / total soal kalau benar)
	Points           int             `json:"points"`             // POINT leaderboard dari soal ini (benar: easy 10, medium 15, hard 20)
	AnsweredAt       *time.Time      `json:"answered_at"`        // jam soal terakhir dijawab (WIB); null = tidak dikirim FE
	TimeSpentSeconds *int            `json:"time_spent_seconds"` // lama mengerjakan soal ini (detik); null = tidak dikirim FE
}

// ResultResponse adalah ringkasan hasil 1 pengerjaan quiz: nilai (total 100) dan point leaderboard.
type ResultResponse struct {
	Score            float64   `json:"score"`              // NILAI 0-100
	CorrectCount     int       `json:"correct_count"`      // jumlah benar
	TotalQuestions   int       `json:"total_questions"`    // jumlah soal
	ScorePerQuestion float64   `json:"score_per_question"` // nilai per soal = 100 / total soal (2 angka di belakang koma)
	Points           int       `json:"points"`             // total POINT leaderboard (bukan nilai)
	SubmittedAt      time.Time `json:"submitted_at"`
	DurationSeconds  int       `json:"duration_seconds"` // lama mengerjakan total = submit - mulai
}

// ToSubmittedAnswers mengubah body submit menjadi input penilaian.
func ToSubmittedAnswers(req SubmitAnswersRequest) []scoring.SubmittedAnswer {
	result := make([]scoring.SubmittedAnswer, 0, len(req.Answers))
	for _, a := range req.Answers {
		result = append(result, scoring.SubmittedAnswer{
			QuestionID: a.QuestionID, Answer: a.Answer, IsCorrect: *a.IsCorrect,
			AnsweredAt: a.AnsweredAt, TimeSpentSeconds: a.TimeSpentSeconds,
		})
	}
	return result
}

// ToReviewResponses mengubah review menjadi response (no = urutan soal, mulai 1).
func ToReviewResponses(items []scoring.ReviewItem) []ReviewQuestionResponse {
	content := make([]questionmodel.AttemptQuestion, 0, len(items))
	for _, it := range items {
		content = append(content, it.AttemptQuestion)
	}
	base := ToAttemptQuestionResponses(content)

	result := make([]ReviewQuestionResponse, 0, len(items))
	for i, it := range items {
		result = append(result, ReviewQuestionResponse{
			AttemptQuestionResponse: base[i],
			Explanation:             it.Explanation,
			StudentAnswer:           it.StudentAnswer,
			IsCorrect:               it.IsCorrect,
			Score:                   it.Score,
			Points:                  it.Points,
			AnsweredAt:              inJakarta(it.AnsweredAt),
			TimeSpentSeconds:        it.TimeSpentSeconds,
		})
	}
	return result
}

// ToResultResponse menyusun ringkasan nilai. submittedAt ditampilkan dalam WIB.
func ToResultResponse(score float64, points, correct, total int, submittedAt time.Time, durationSeconds int) ResultResponse {
	return ResultResponse{
		Score:            score,
		CorrectCount:     correct,
		TotalQuestions:   total,
		ScorePerQuestion: math.Round(scoring.ScorePerQuestion(total)*100) / 100,
		Points:           points,
		SubmittedAt:      submittedAt.In(clock.Jakarta),
		DurationSeconds:  durationSeconds,
	}
}

// inJakarta mengubah waktu ke WIB untuk ditampilkan (nil tetap nil).
func inJakarta(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	local := t.In(clock.Jakarta)
	return &local
}
