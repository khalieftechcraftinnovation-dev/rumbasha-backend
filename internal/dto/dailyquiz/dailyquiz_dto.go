// Package dailyquiz (dto) berisi bentuk request dan response JSON untuk fitur Daily Quiz.
//
// Di package lain, import dengan alias dailyquizdto:
//
//	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/dailyquiz"
package dailyquiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// GenerateRequest adalah body POST /api/v1/daily-quizzes (generate Daily Quiz oleh siswa).
//
//	{
//	  "placement": {"subject_id": "...", "subject_field_id": "...", "chapter_id": "..."},
//	  "difficulty": "easy", "total_questions": 10
//	}
type GenerateRequest struct {
	Placement      PlacementRequest `json:"placement" binding:"required"`
	Difficulty     string           `json:"difficulty" binding:"required,oneof=easy medium hard"` // level
	TotalQuestions int              `json:"total_questions" binding:"required,oneof=5 10 15"`
}

// PlacementRequest adalah pilihan letak soal Daily Quiz. Mapel & bidang materi wajib.
// grade_id opsional: kosong = kelas siswa; kalau diisi harus salah satu dari GET /daily-quizzes/grade-options.
type PlacementRequest struct {
	GradeID        *string `json:"grade_id" binding:"omitempty,uuid"`
	SubjectID      string  `json:"subject_id" binding:"required,uuid"`
	SubjectFieldID string  `json:"subject_field_id" binding:"required,uuid"`
	ChapterID      *string `json:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID   *string `json:"subchapter_id" binding:"omitempty,uuid"`
	TopicID        *string `json:"topic_id" binding:"omitempty,uuid"`
}

// GradeOptionResponse adalah 1 pilihan kelas Daily Quiz (format sama dengan dropdown + is_default).
type GradeOptionResponse struct {
	Value     string  `json:"value"` // = id, dikirim sebagai placement.grade_id
	ID        string  `json:"id"`
	Code      *string `json:"code"`
	Name      string  `json:"name"`
	ParentID  string  `json:"parent_id"`  // jenjang
	IsDefault bool    `json:"is_default"` // kelas siswa sendiri (dipilih otomatis)
}

// ToGradeOptionResponses mengubah pilihan kelas menjadi response.
func ToGradeOptionResponses(options []dailyquizmodel.GradeOption) []GradeOptionResponse {
	result := make([]GradeOptionResponse, 0, len(options))
	for _, o := range options {
		result = append(result, GradeOptionResponse{
			Value: o.ID, ID: o.ID, Code: o.Code, Name: o.Name, ParentID: o.ParentID, IsDefault: o.IsDefault,
		})
	}
	return result
}

// DailyQuizResponse adalah Daily Quiz hasil generate: pilihan siswa + soal untuk dikerjakan.
// Tidak ada kode akses & timer. Soal TANPA kunci jawaban & pembahasan.
type DailyQuizResponse struct {
	ID             string                    `json:"id"`
	Placement      PlacementResponse         `json:"placement"`
	Difficulty     string                    `json:"difficulty"`
	TotalQuestions int                       `json:"total_questions"`
	CreatedAt      time.Time                 `json:"created_at"`
	Questions      []StudentQuestionResponse `json:"questions"`
}

// PlacementResponse adalah letak Daily Quiz dengan nama. Tingkat yang tidak dipilih = null.
type PlacementResponse struct {
	EducationLevel RefResponse  `json:"education_level"`
	Grade          RefResponse  `json:"grade"`
	Subject        RefResponse  `json:"subject"`
	SubjectField   RefResponse  `json:"subject_field"`
	Chapter        *RefResponse `json:"chapter"`
	Subchapter     *RefResponse `json:"subchapter"`
	Topic          *RefResponse `json:"topic"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StudentQuestionResponse adalah 1 soal untuk siswa. options hanya untuk pilihan ganda /
// PG kompleks / benar-salah; input_type hanya untuk isian ("number" / "text").
type StudentQuestionResponse struct {
	No           int                  `json:"no"`
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Question     string               `json:"question"`
	Formula      *string              `json:"formula"`
	ImageURL     *string              `json:"image_url"`
	Table        *questionmodel.Table `json:"table"`
	DisplayOrder []string             `json:"display_order"`
	Options      []OptionResponse     `json:"options,omitempty"`
	InputType    *string              `json:"input_type,omitempty"`
}

// OptionResponse adalah 1 pilihan jawaban (tanpa tanda benar/salah).
type OptionResponse struct {
	Key      string  `json:"key"`
	Content  string  `json:"content"`
	ImageURL *string `json:"image_url"`
}

// ToDailyQuizResponse mengubah DailyQuizDetail menjadi response.
func ToDailyQuizResponse(d dailyquizmodel.DailyQuizDetail) DailyQuizResponse {
	questions := make([]StudentQuestionResponse, 0, len(d.Questions))
	for i, q := range d.Questions {
		var options []OptionResponse
		for _, o := range q.Options {
			options = append(options, OptionResponse{Key: o.Key, Content: o.Content, ImageURL: o.ImageURL})
		}
		questions = append(questions, StudentQuestionResponse{
			No: i + 1, ID: q.ID, Type: q.Type, Question: q.QuestionText, Formula: q.Formula,
			ImageURL: q.ImageURL, Table: q.Table, DisplayOrder: q.DisplayOrder,
			Options: options, InputType: q.InputType,
		})
	}

	return DailyQuizResponse{
		ID: d.ID,
		Placement: PlacementResponse{
			EducationLevel: RefResponse{ID: d.EducationLevelID, Name: d.EducationLevelName},
			Grade:          RefResponse{ID: d.GradeID, Name: d.GradeName},
			Subject:        RefResponse{ID: d.SubjectID, Name: d.SubjectName},
			SubjectField:   RefResponse{ID: d.SubjectFieldID, Name: d.SubjectFieldName},
			Chapter:        optionalRef(d.ChapterID, d.ChapterName),
			Subchapter:     optionalRef(d.SubchapterID, d.SubchapterName),
			Topic:          optionalRef(d.TopicID, d.TopicName),
		},
		Difficulty:     d.Difficulty,
		TotalQuestions: d.TotalQuestions,
		CreatedAt:      d.CreatedAt,
		Questions:      questions,
	}
}

func optionalRef(id, name *string) *RefResponse {
	if id == nil {
		return nil
	}
	r := RefResponse{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return &r
}

// OpenedDailyQuizResponse adalah Daily Quiz yang dibuka siswa: info, pemakaian jatah/coin,
// dan soal lengkap dengan kunci jawaban (tanpa pembahasan). Tidak ada kode akses & timer.
type OpenedDailyQuizResponse struct {
	State          string                                `json:"state"` // "in_progress" (lihat juga review: "submitted")
	ID             string                                `json:"id"`
	Placement      PlacementResponse                     `json:"placement"`
	Difficulty     string                                `json:"difficulty"`
	TotalQuestions int                                   `json:"total_questions"`
	CreatedAt      time.Time                             `json:"created_at"`
	OpenedAt       time.Time                             `json:"opened_at"` // pertama kali dibuka, WIB
	Usage          UsageResponse                         `json:"usage"`
	Questions      []questiondto.AttemptQuestionResponse `json:"questions"`
}

// UsageResponse adalah pemakaian jatah Daily Quiz hari ini (WIB) & coin.
type UsageResponse struct {
	FreePerDay    int  `json:"free_per_day"`   // jatah gratis per hari
	OpenedToday   int  `json:"opened_today"`   // Daily Quiz yang sudah dibuka hari ini (termasuk ini)
	CoinCharged   int  `json:"coin_charged"`   // coin yang dipotong untuk Daily Quiz ini (0 = gratis)
	CoinBalance   int  `json:"coin_balance"`   // saldo coin sekarang
	AlreadyOpened bool `json:"already_opened"` // true = dibuka ulang (tidak dihitung & tidak dipotong lagi)
}

// ToOpenedDailyQuizResponse mengubah Daily Quiz yang dibuka menjadi response.
func ToOpenedDailyQuizResponse(o dailyquizmodel.OpenedDailyQuiz) OpenedDailyQuizResponse {
	base := ToDailyQuizResponse(o.DailyQuizDetail)
	return OpenedDailyQuizResponse{
		State: quizdto.StateInProgress,
		ID:    base.ID, Placement: base.Placement, Difficulty: base.Difficulty,
		TotalQuestions: base.TotalQuestions, CreatedAt: base.CreatedAt,
		OpenedAt: o.OpenedAt.In(clock.Jakarta),
		Usage: UsageResponse{
			FreePerDay: o.FreePerDay, OpenedToday: o.OpenedToday, CoinCharged: o.CoinCharged,
			CoinBalance: o.CoinBalance, AlreadyOpened: o.AlreadyOpened,
		},
		Questions: questiondto.ToAttemptQuestionResponses(o.Content),
	}
}

// DailyQuizReviewResponse adalah hasil & review Daily Quiz setelah submit (juga GET review):
// nilai (total 100) dan setiap soal dengan urutan yang sama seperti saat dikerjakan, lengkap dengan
// kunci jawaban, pembahasan, jawaban siswa, dan benar/salahnya.
type DailyQuizReviewResponse struct {
	State      string                               `json:"state"` // selalu "submitted"
	ID         string                               `json:"id"`
	Placement  PlacementResponse                    `json:"placement"`
	Difficulty string                               `json:"difficulty"`
	OpenedAt   time.Time                            `json:"opened_at"` // WIB
	Result     questiondto.ResultResponse           `json:"result"`
	Questions  []questiondto.ReviewQuestionResponse `json:"questions"`
}

// ToDailyQuizReviewResponse mengubah review Daily Quiz menjadi response.
func ToDailyQuizReviewResponse(r dailyquizmodel.DailyQuizReview) DailyQuizReviewResponse {
	base := ToDailyQuizResponse(r.DailyQuizDetail)
	var openedAt time.Time
	if r.OpenedAt != nil {
		openedAt = r.OpenedAt.In(clock.Jakarta)
	}
	return DailyQuizReviewResponse{
		State: quizdto.StateSubmitted,
		ID:    base.ID, Placement: base.Placement, Difficulty: base.Difficulty, OpenedAt: openedAt,
		Result:    questiondto.ToResultResponse(r.Score, r.CorrectCount, len(r.Items), *r.SubmittedAt, r.DurationSeconds),
		Questions: questiondto.ToReviewResponses(r.Items),
	}
}
