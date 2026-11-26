// Package trialquiz (dto) berisi bentuk request dan response JSON fitur Quiz Trial publik.
//
// Di package lain, import dengan alias trialquizdto:
//
//	trialquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/trialquiz"
package trialquiz

import (
	"encoding/json"
	"math"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	trialquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/trialquiz"
)

// ============================================================================
// Admin: kelola soal (tab "Quiz Trial" di menu Landing Page)
// ============================================================================

// QuestionRequest adalah body POST /trial-quiz/questions dan PUT /trial-quiz/questions/:id.
// Isi soal (type s/d answer) SAMA PERSIS dengan 1 soal bank soal (POST /questions), tapi tanpa
// placement & used_for: soal trial cukup punya golongan dan mapel.
type QuestionRequest struct {
	GradeGroup string `json:"grade_group" binding:"required,oneof=pra_sd sd_1_3 sd_4_6 smp sma"`
	SubjectID  string `json:"subject_id" binding:"required,uuid"`            // mapel (master data): Matematika, IPA, IPS, B. Inggris, ...
	SortOrder  *int   `json:"sort_order" binding:"omitempty,min=0,max=1000"` // urutan soal (kecil dulu); kosong = 0

	Type         string                    `json:"type" binding:"required,oneof=single_choice multiple_choice true_false free_text"`
	Question     string                    `json:"question" binding:"required,max=5000"`
	Formula      *string                   `json:"formula" binding:"omitempty,max=2000"`
	ImageID      *string                   `json:"image_id" binding:"omitempty,uuid"`
	ImageURL     *string                   `json:"image_url" binding:"omitempty,max=2000"`
	Table        *questiondto.TableRequest `json:"table"`
	Explanation  string                    `json:"explanation" binding:"required,max=5000"`
	DisplayOrder []string                  `json:"display_order" binding:"omitempty,unique,dive,oneof=question formula image table"`
	Difficulty   string                    `json:"difficulty" binding:"required,oneof=easy medium hard"`
	Answer       json.RawMessage           `json:"answer" binding:"required"`
}

// ToContentRequest mengubah request menjadi bentuk soal bank soal, supaya isinya dicek dengan
// aturan yang sama (questionservice.ValidateContent).
func (r QuestionRequest) ToContentRequest() questiondto.CreateQuestionRequest {
	return questiondto.CreateQuestionRequest{
		Type: r.Type, Question: r.Question, Formula: r.Formula, ImageID: r.ImageID, ImageURL: r.ImageURL,
		Table: r.Table, Explanation: r.Explanation, DisplayOrder: r.DisplayOrder,
		Difficulty: r.Difficulty, Answer: r.Answer,
	}
}

// ListQuestionsQuery adalah query string GET /trial-quiz/questions?grade_group=sd_1_3 (kosong = semua).
type ListQuestionsQuery struct {
	GradeGroup string `form:"grade_group" binding:"omitempty,oneof=pra_sd sd_1_3 sd_4_6 smp sma"`
}

// Ref = id + nama.
type Ref struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

func ptrRef(id, name *string) *Ref {
	if id == nil {
		return nil
	}
	return &Ref{ID: *id, Name: name}
}

// QuestionResponse adalah 1 soal trial lengkap (admin): termasuk kunci & pembahasan.
type QuestionResponse struct {
	ID             string               `json:"id"`
	Code           string               `json:"code"` // contoh TQ-3
	GradeGroup     string               `json:"grade_group"`
	GradeGroupName string               `json:"grade_group_name"`
	Subject        Ref                  `json:"subject"`
	SortOrder      int                  `json:"sort_order"`
	Type           string               `json:"type"`
	Question       string               `json:"question"`
	Formula        *string              `json:"formula"`
	ImageID        *string              `json:"image_id"`
	ImageURL       *string              `json:"image_url"`
	Table          *questionmodel.Table `json:"table"`
	Explanation    string               `json:"explanation"`
	DisplayOrder   []string             `json:"display_order"`
	Difficulty     string               `json:"difficulty"`
	Answer         json.RawMessage      `json:"answer"`
	CreatedBy      Ref                  `json:"created_by"`
	UpdatedBy      *Ref                 `json:"updated_by"` // null = belum pernah diubah
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// ToQuestionResponse mengubah model soal trial menjadi response admin.
func ToQuestionResponse(q trialquizmodel.Question) QuestionResponse {
	subjectName := q.SubjectName
	return QuestionResponse{
		ID: q.ID, Code: q.Code, GradeGroup: q.GradeGroup, GradeGroupName: trialquizmodel.GroupNames[q.GradeGroup],
		Subject: Ref{ID: q.SubjectID, Name: &subjectName}, SortOrder: q.SortOrder,
		Type: q.Type, Question: q.QuestionText, Formula: q.Formula, ImageID: q.ImageID, ImageURL: q.ImageURL,
		Table: q.Table, Explanation: q.Explanation, DisplayOrder: q.DisplayOrder, Difficulty: q.Difficulty,
		Answer:    q.Answer,
		CreatedBy: Ref{ID: q.CreatedBy, Name: q.CreatedByName}, UpdatedBy: ptrRef(q.UpdatedBy, q.UpdatedByName),
		CreatedAt: q.CreatedAt.In(clock.Jakarta), UpdatedAt: q.UpdatedAt.In(clock.Jakarta),
	}
}

// GroupCountResponse = jumlah soal 1 golongan (untuk tab/badge "7/10").
type GroupCountResponse struct {
	GradeGroup string `json:"grade_group"`
	Name       string `json:"name"`
	Total      int    `json:"total"`
}

// ListQuestionsResponse adalah data GET /trial-quiz/questions.
type ListQuestionsResponse struct {
	MaxPerGroup int                  `json:"max_per_group"` // 10
	GroupCounts []GroupCountResponse `json:"group_counts"`  // semua golongan, urut tampil
	Questions   []QuestionResponse   `json:"questions"`     // urut golongan lalu nomor soal
}

// ToListQuestionsResponse menyusun data list soal admin.
func ToListQuestionsResponse(questions []trialquizmodel.Question, counts map[string]int) ListQuestionsResponse {
	res := ListQuestionsResponse{
		MaxPerGroup: trialquizmodel.MaxQuestionsPerGroup,
		GroupCounts: make([]GroupCountResponse, 0, len(trialquizmodel.Groups)),
		Questions:   make([]QuestionResponse, 0, len(questions)),
	}
	for _, g := range trialquizmodel.Groups {
		res.GroupCounts = append(res.GroupCounts, GroupCountResponse{GradeGroup: g, Name: trialquizmodel.GroupNames[g], Total: counts[g]})
	}
	for _, q := range questions {
		res.Questions = append(res.Questions, ToQuestionResponse(q))
	}
	return res
}

// ============================================================================
// Publik (tanpa login)
// ============================================================================

// GradeResponse = 1 kelas di form trial.
type GradeResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	GradeGroup     string `json:"grade_group"`
	GradeGroupName string `json:"grade_group_name"`
}

// LevelResponse = 1 jenjang beserta kelasnya.
type LevelResponse struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Grades []GradeResponse `json:"grades"`
}

// ToLevelResponses mengelompokkan kelas per jenjang (urutan dari repository dipertahankan).
func ToLevelResponses(grades []trialquizmodel.GradeOption) []LevelResponse {
	result := []LevelResponse{}
	index := map[string]int{}
	for _, g := range grades {
		i, ok := index[g.LevelID]
		if !ok {
			i = len(result)
			index[g.LevelID] = i
			result = append(result, LevelResponse{ID: g.LevelID, Name: g.LevelName, Grades: []GradeResponse{}})
		}
		result[i].Grades = append(result[i].Grades, GradeResponse{
			ID: g.GradeID, Name: g.GradeName, GradeGroup: g.Group, GradeGroupName: trialquizmodel.GroupNames[g.Group],
		})
	}
	return result
}

// PublicQuestionsQuery adalah query string GET /public/trial-quiz/questions?grade_id=...
type PublicQuestionsQuery struct {
	GradeID string `form:"grade_id" binding:"required,uuid"`
}

// PublicQuestionResponse = 1 soal trial publik: isi soal + kunci (answer) + pembahasan.
// FE menilai sendiri dan menampilkan pembahasan setelah selesai.
type PublicQuestionResponse struct {
	No           int                  `json:"no"`
	ID           string               `json:"id"`
	Code         string               `json:"code"`
	Type         string               `json:"type"`
	Subject      Ref                  `json:"subject"`
	Question     string               `json:"question"`
	Formula      *string              `json:"formula"`
	ImageURL     *string              `json:"image_url"`
	Table        *questionmodel.Table `json:"table"`
	DisplayOrder []string             `json:"display_order"`
	Difficulty   string               `json:"difficulty"`
	Answer       json.RawMessage      `json:"answer"`
	Explanation  string               `json:"explanation"`
}

// PublicQuizResponse adalah data GET /public/trial-quiz/questions.
type PublicQuizResponse struct {
	EducationLevel   Ref                      `json:"education_level"`
	Grade            Ref                      `json:"grade"`
	GradeGroup       string                   `json:"grade_group"`
	GradeGroupName   string                   `json:"grade_group_name"`
	TotalQuestions   int                      `json:"total_questions"`
	ScorePerQuestion float64                  `json:"score_per_question"` // 100 / total soal
	Questions        []PublicQuestionResponse `json:"questions"`
}

// ToPublicQuizResponse menyusun data quiz trial publik.
func ToPublicQuizResponse(grade trialquizmodel.GradeOption, questions []trialquizmodel.PublicQuestion, scorePerQuestion float64) PublicQuizResponse {
	res := PublicQuizResponse{
		EducationLevel: Ref{ID: grade.LevelID, Name: &grade.LevelName},
		Grade:          Ref{ID: grade.GradeID, Name: &grade.GradeName},
		GradeGroup:     grade.Group, GradeGroupName: trialquizmodel.GroupNames[grade.Group],
		TotalQuestions: len(questions), ScorePerQuestion: scorePerQuestion,
		Questions: make([]PublicQuestionResponse, 0, len(questions)),
	}
	for i, q := range questions {
		subjectName := q.SubjectName
		res.Questions = append(res.Questions, PublicQuestionResponse{
			No: i + 1, ID: q.ID, Code: q.Code, Type: q.Type, Subject: Ref{ID: q.SubjectID, Name: &subjectName},
			Question: q.QuestionText, Formula: q.Formula, ImageURL: q.ImageURL, Table: q.Table,
			DisplayOrder: q.DisplayOrder, Difficulty: q.Difficulty, Answer: q.Answer, Explanation: q.Explanation,
		})
	}
	return res
}

// SubmitRequest adalah body POST /public/trial-quiz/submit: data peserta (wajib) + jawaban.
//
//	{"name": "Budi", "whatsapp": "081234567890", "grade_id": "...", "duration_seconds": 420,
//	 "answers": [{"question_id": "...", "answer": "B", "is_correct": true}, ...]}
//
// Benar/salah dinilai FE (is_correct); nilai dihitung BE dari jumlah benar (total 100).
// Soal yang tidak dikirim dianggap tidak dijawab (salah).
type SubmitRequest struct {
	Name            string                         `json:"name" binding:"required,max=100"`
	WhatsApp        string                         `json:"whatsapp" binding:"required,max=20"`
	GradeID         string                         `json:"grade_id" binding:"required,uuid"` // kelas (jenjang ikut kelasnya)
	DurationSeconds *int                           `json:"duration_seconds" binding:"omitempty,min=0,max=86400"`
	Answers         []questiondto.SubmitAnswerItem `json:"answers" binding:"max=50,dive"`
}

// SubmitResponse adalah data POST /public/trial-quiz/submit.
type SubmitResponse struct {
	ID               string    `json:"id"`
	Score            float64   `json:"score"` // 0-100
	CorrectCount     int       `json:"correct_count"`
	TotalQuestions   int       `json:"total_questions"`
	ScorePerQuestion float64   `json:"score_per_question"`
	SubmittedAt      time.Time `json:"submitted_at"` // WIB
}

// ToSubmitResponse mengubah hasil trial yang disimpan menjadi response.
func ToSubmitResponse(a trialquizmodel.Attempt) SubmitResponse {
	return SubmitResponse{
		ID: a.ID, Score: a.Result.Score, CorrectCount: a.Result.CorrectCount, TotalQuestions: len(a.Result.Answers),
		ScorePerQuestion: round2(a.Result.ScorePerQuestion), SubmittedAt: a.SubmittedAt.In(clock.Jakarta),
	}
}

// ============================================================================
// Admin: report peserta trial
// ============================================================================

// AttemptsQuery adalah query string GET /trial-quiz/attempts.
//
//	?date_from=2026-11-01&date_to=2026-11-30&grade_group=smp&search=budi&page=1&per_page=20
//
// date_from / date_to = tanggal submit (WIB, inklusif). search = nama atau no WhatsApp.
type AttemptsQuery struct {
	DateFrom   string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo     string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
	GradeGroup string `form:"grade_group" binding:"omitempty,oneof=pra_sd sd_1_3 sd_4_6 smp sma"`
	Search     string `form:"search" binding:"omitempty,max=100"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PerPage    int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// AttemptResponse = 1 baris report peserta trial.
type AttemptResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	WhatsApp        string    `json:"whatsapp"` // 62xxxxxxxxxx
	EducationLevel  Ref       `json:"education_level"`
	Grade           Ref       `json:"grade"`
	GradeGroup      string    `json:"grade_group"`
	GradeGroupName  string    `json:"grade_group_name"`
	Score           float64   `json:"score"`
	CorrectCount    int       `json:"correct_count"`
	TotalQuestions  int       `json:"total_questions"`
	DurationSeconds *int      `json:"duration_seconds"`
	SubmittedAt     time.Time `json:"submitted_at"` // WIB
}

// ToAttemptResponses mengubah baris report menjadi response.
func ToAttemptResponses(items []trialquizmodel.AttemptListItem) []AttemptResponse {
	result := make([]AttemptResponse, 0, len(items))
	for _, it := range items {
		levelName, gradeName := it.EducationLevelName, it.GradeName
		result = append(result, AttemptResponse{
			ID: it.ID, Name: it.Name, WhatsApp: it.WhatsApp,
			EducationLevel: Ref{ID: it.EducationLevelID, Name: &levelName}, Grade: Ref{ID: it.GradeID, Name: &gradeName},
			GradeGroup: it.GradeGroup, GradeGroupName: trialquizmodel.GroupNames[it.GradeGroup],
			Score: it.Score, CorrectCount: it.CorrectCount, TotalQuestions: it.TotalQuestions,
			DurationSeconds: it.DurationSeconds, SubmittedAt: it.SubmittedAt.In(clock.Jakarta),
		})
	}
	return result
}

// SummaryQuery adalah query string GET /trial-quiz/attempts/summary (filter sama dengan list,
// tanpa search & pagination).
type SummaryQuery struct {
	DateFrom   string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo     string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
	GradeGroup string `form:"grade_group" binding:"omitempty,oneof=pra_sd sd_1_3 sd_4_6 smp sma"`
}

// GroupStatResponse = statistik 1 golongan.
type GroupStatResponse struct {
	GradeGroup   string  `json:"grade_group"`
	Name         string  `json:"name"`
	Participants int     `json:"participants"`
	AverageScore float64 `json:"average_score"`
}

// WrongQuestionResponse = 1 soal yang paling sering salah.
type WrongQuestionResponse struct {
	QuestionID string  `json:"question_id"`
	Code       string  `json:"code"`
	Question   string  `json:"question"`
	Subject    string  `json:"subject"`
	GradeGroup string  `json:"grade_group"`
	Answered   int     `json:"answered"`   // jumlah peserta yang mendapat soal ini
	Wrong      int     `json:"wrong"`      // jumlah yang salah / tidak menjawab
	WrongRate  float64 `json:"wrong_rate"` // persen salah, contoh 62.5
}

// SummaryResponse adalah data GET /trial-quiz/attempts/summary.
type SummaryResponse struct {
	Participants int                     `json:"participants"`
	AverageScore float64                 `json:"average_score"`
	ByGroup      []GroupStatResponse     `json:"by_group"`   // semua golongan (0 kalau tidak ada peserta)
	MostWrong    []WrongQuestionResponse `json:"most_wrong"` // maks 10
}

// ToSummaryResponse mengubah ringkasan report menjadi response.
func ToSummaryResponse(s trialquizmodel.Summary) SummaryResponse {
	res := SummaryResponse{
		Participants: s.Participants, AverageScore: s.AverageScore,
		ByGroup:   make([]GroupStatResponse, 0, len(trialquizmodel.Groups)),
		MostWrong: make([]WrongQuestionResponse, 0, len(s.MostWrong)),
	}
	byGroup := map[string]trialquizmodel.GroupStat{}
	for _, g := range s.ByGroup {
		byGroup[g.GradeGroup] = g
	}
	for _, g := range trialquizmodel.Groups {
		stat := byGroup[g]
		res.ByGroup = append(res.ByGroup, GroupStatResponse{
			GradeGroup: g, Name: trialquizmodel.GroupNames[g], Participants: stat.Participants, AverageScore: stat.AverageScore,
		})
	}
	for _, w := range s.MostWrong {
		rate := 0.0
		if w.Answered > 0 {
			rate = round2(float64(w.Wrong) * 100 / float64(w.Answered))
		}
		res.MostWrong = append(res.MostWrong, WrongQuestionResponse{
			QuestionID: w.QuestionID, Code: w.Code, Question: w.QuestionText, Subject: w.SubjectName,
			GradeGroup: w.GradeGroup, Answered: w.Answered, Wrong: w.Wrong, WrongRate: rate,
		})
	}
	return res
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
