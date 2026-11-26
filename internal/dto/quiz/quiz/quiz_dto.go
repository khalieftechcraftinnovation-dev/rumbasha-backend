// Package quiz (dto) berisi bentuk request dan response JSON untuk fitur quiz.
//
// Di package lain, import dengan alias quizdto:
//
//	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/quiz"
package quiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

// CreateQuizRequest adalah body JSON POST /api/v1/quizzes.
//
//	{
//	  "name": "Uji Kompetensi Matematika - Pecahan", "type": "uji_kom",
//	  "placement": {"education_level_id": "...", "grade_id": "...", "subject_id": "..."},
//	  "start_at": "2026-12-01T08:00", "end_at": "2026-12-03T17:00",
//	  "duration_minutes": 60, "coin_cost": 0, "total_questions": 12,
//	  "bundle_quiz_ids": ["..."],
//	  "question_ids": ["...", "..."],
//	  "points": {"easy": 10, "medium": 15, "hard": 20},
//	  "prize_coins": {"rank_1": 500, "rank_2": 300, "rank_3": 100}
//	}
//
// Soal quiz = semua soal dari bundle_quiz_ids (urut bundle, lalu nomor soal di bundle), lalu
// question_ids. Soal yang muncul lebih dari sekali hanya diambil sekali (yang pertama).
// Minimal salah satu dari bundle_quiz_ids / question_ids diisi.
//
// Aturan yang bergantung pada field lain (total_questions = jumlah soal, kecocokan letak,
// syarat soal, kelas Uji Kom, jadwal & bentrok jadwal) dicek di service.
//
// Body PUT /api/v1/quizzes/:id (ubah quiz) memakai struct yang sama (ganti penuh).
type CreateQuizRequest struct {
	Name string `json:"name" binding:"required,max=200"`
	// Tombol simpan: "draft" (Simpan draf) atau "published" (Publish: tampil di pengumuman siswa).
	Status    string               `json:"status" binding:"required,oneof=draft published"`
	Type      string               `json:"type" binding:"required,oneof=uji_kom tryout"`
	Placement QuizPlacementRequest `json:"placement" binding:"required"`
	// Jendela event dalam WIB, boleh lebih dari 1 hari. Format "2026-12-01T08:00" /
	// "2026-12-01 08:00" (dianggap WIB), atau RFC3339 dengan zona ("2026-12-01T08:00:00+07:00").
	// start_at harus setelah sekarang; end_at - start_at minimal sepanjang duration_minutes.
	StartAt         string `json:"start_at" binding:"required"`
	EndAt           string `json:"end_at" binding:"required"`
	DurationMinutes int    `json:"duration_minutes" binding:"required,min=1,max=600"` // lama mengerjakan 1 kali (menit)
	CoinCost        *int   `json:"coin_cost" binding:"omitempty,min=0,max=1000000"`   // opsional, default 0 (gratis)
	TotalQuestions  int    `json:"total_questions" binding:"required,min=1,max=200"`
	// Quiz lain yang soalnya diambil semua (bundle). Maksimal 20 bundle.
	BundleQuizIDs []string `json:"bundle_quiz_ids" binding:"omitempty,max=20,unique,dive,uuid"`
	// Soal satuan. Urutan di array = urutan soal (setelah soal dari bundle).
	// Total soal maksimal 200 per quiz, termasuk soal dari bundle (contoh desain Tryout: 40).
	QuestionIDs []string `json:"question_ids" binding:"omitempty,max=200,unique,dive,uuid"`
	// Point leaderboard 1 soal BENAR per level soal. Opsional; level yang tidak diisi memakai
	// default (easy 10, medium 15, hard 20).
	Points *QuizPointsRequest `json:"points"`
	// Hadiah coin juara 1, 2, 3 leaderboard quiz. Opsional; tidak diisi = 0 (tanpa hadiah).
	// Dibagikan otomatis jam 23:59 WIB setelah event selesai.
	PrizeCoins *QuizPrizesRequest `json:"prize_coins"`
}

// QuizPointsRequest = point 1 soal yang dijawab benar per level. Contoh {"easy": 10, "medium": 15, "hard": 20}.
type QuizPointsRequest struct {
	Easy   *int `json:"easy" binding:"omitempty,min=0,max=1000"`
	Medium *int `json:"medium" binding:"omitempty,min=0,max=1000"`
	Hard   *int `json:"hard" binding:"omitempty,min=0,max=1000"`
}

// QuizPrizesRequest = hadiah coin juara. Contoh {"rank_1": 500, "rank_2": 300, "rank_3": 100}.
type QuizPrizesRequest struct {
	Rank1 *int `json:"rank_1" binding:"omitempty,min=0,max=1000000"`
	Rank2 *int `json:"rank_2" binding:"omitempty,min=0,max=1000000"`
	Rank3 *int `json:"rank_3" binding:"omitempty,min=0,max=1000000"`
}

// QuizPointsResponse = point 1 soal benar per level.
type QuizPointsResponse struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

// QuizPrizesResponse = hadiah coin juara 1-3 (0 = tanpa hadiah).
type QuizPrizesResponse struct {
	Rank1 int `json:"rank_1"`
	Rank2 int `json:"rank_2"`
	Rank3 int `json:"rank_3"`
}

// ToPointsResponse & ToPrizesResponse mengubah model menjadi response.
func ToPointsResponse(p quizmodel.Points) QuizPointsResponse {
	return QuizPointsResponse{Easy: p.Easy, Medium: p.Medium, Hard: p.Hard}
}

func ToPrizesResponse(p quizmodel.Prizes) QuizPrizesResponse {
	return QuizPrizesResponse{Rank1: p.Rank1, Rank2: p.Rank2, Rank3: p.Rank3}
}

// PrizeAwardResponse = 1 penerima hadiah juara (detail quiz admin).
type PrizeAwardResponse struct {
	Rank            int         `json:"rank"`
	Student         RefResponse `json:"student"`
	Points          int         `json:"points"`
	DurationSeconds int         `json:"duration_seconds"`
	CoinAmount      int         `json:"coin_amount"`
	AwardedAt       time.Time   `json:"awarded_at"` // WIB
}

// QuizPlacementRequest adalah letak quiz. Jenjang, kelas, mapel wajib; sisanya opsional
// tapi harus berurutan (bab butuh bidang materi, sub bab butuh bab, topik butuh sub bab).
type QuizPlacementRequest struct {
	EducationLevelID string  `json:"education_level_id" binding:"required,uuid"`
	GradeID          string  `json:"grade_id" binding:"required,uuid"`
	SubjectID        string  `json:"subject_id" binding:"required,uuid"`
	SubjectFieldID   *string `json:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID        *string `json:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID     *string `json:"subchapter_id" binding:"omitempty,uuid"`
	TopicID          *string `json:"topic_id" binding:"omitempty,uuid"`
}

// QuizResponse adalah data 1 quiz (baris list quiz), tanpa daftar soal.
type QuizResponse struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Type            string                `json:"type"`
	Placement       QuizPlacementResponse `json:"placement"`
	StartAt         time.Time             `json:"start_at"` // mulai event, WIB, contoh "2026-12-01T08:00:00+07:00"
	EndAt           time.Time             `json:"end_at"`   // selesai event, WIB (boleh beda hari)
	ScheduleStatus  string                `json:"schedule_status"`
	DurationMinutes int                   `json:"duration_minutes"` // lama mengerjakan 1 kali
	AccessCode      *string               `json:"access_code"`      // berganti setiap jam selama event belum selesai
	CoinCost        int                   `json:"coin_cost"`
	TotalQuestions  int                   `json:"total_questions"`
	Points          QuizPointsResponse    `json:"points"`      // point 1 soal benar per level
	PrizeCoins      QuizPrizesResponse    `json:"prize_coins"` // hadiah coin juara 1-3
	// Kapan hadiah dibagikan (WIB); null = belum (dibagikan 23:59 WIB setelah event selesai).
	PrizesDistributedAt *time.Time   `json:"prizes_distributed_at"`
	Status              string       `json:"status"`
	CreatedBy           RefResponse  `json:"created_by"`
	UpdatedBy           *RefResponse `json:"updated_by"` // null kalau belum pernah diubah
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// QuizDetailResponse adalah data quiz lengkap + daftar soal (response create & update).
// Field QuizResponse ikut di level atas JSON (embedded), "questions" di akhir.
type QuizDetailResponse struct {
	QuizResponse
	Questions []QuizQuestionResponse `json:"questions"`
	// Penerima hadiah juara (urut peringkat). [] sebelum dibagikan / tidak ada pemenang.
	PrizeAwards []PrizeAwardResponse `json:"prize_awards"`
}

// QuizPlacementResponse adalah letak quiz dengan nama. Tingkat yang tidak diisi = null.
type QuizPlacementResponse struct {
	EducationLevel RefResponse  `json:"education_level"`
	Grade          RefResponse  `json:"grade"`
	Subject        RefResponse  `json:"subject"`
	SubjectField   *RefResponse `json:"subject_field"`
	Chapter        *RefResponse `json:"chapter"`
	Subchapter     *RefResponse `json:"subchapter"`
	Topic          *RefResponse `json:"topic"`
}

// QuizQuestionResponse adalah 1 soal di quiz (ringkas, tanpa kunci jawaban).
type QuizQuestionResponse struct {
	No         int    `json:"no"` // nomor soal di quiz, mulai 1
	ID         string `json:"id"`
	Code       string `json:"code"`
	Type       string `json:"type"`
	Question   string `json:"question"`
	Difficulty string `json:"difficulty"`
}

// RefResponse adalah pasangan id + nama, contoh {"id": "...", "name": "Kelas 4"}.
type RefResponse struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

// ToQuizDetailResponse mengubah quizmodel.QuizDetail (dengan soal) menjadi response.
// now dipakai untuk menghitung schedule_status.
func ToQuizDetailResponse(d quizmodel.QuizDetail, now time.Time) QuizDetailResponse {
	questions := make([]QuizQuestionResponse, 0, len(d.Questions))
	for _, q := range d.Questions {
		questions = append(questions, QuizQuestionResponse{
			No: q.SortOrder, ID: q.ID, Code: q.Code, Type: q.Type, Question: q.QuestionText, Difficulty: q.Difficulty,
		})
	}
	awards := make([]PrizeAwardResponse, 0, len(d.PrizeAwards))
	for _, a := range d.PrizeAwards {
		awards = append(awards, PrizeAwardResponse{
			Rank: a.Rank, Student: RefResponse{ID: a.StudentID, Name: a.StudentName},
			Points: a.Points, DurationSeconds: a.DurationSeconds, CoinAmount: a.CoinAmount,
			AwardedAt: a.CreatedAt.In(clock.Jakarta),
		})
	}
	return QuizDetailResponse{QuizResponse: ToQuizResponse(d, now), Questions: questions, PrizeAwards: awards}
}

// ToQuizResponses mengubah hasil list quiz menjadi response. Hasil kosong menjadi [] (bukan null).
func ToQuizResponses(items []quizmodel.QuizDetail, now time.Time) []QuizResponse {
	result := make([]QuizResponse, 0, len(items))
	for _, d := range items {
		result = append(result, ToQuizResponse(d, now))
	}
	return result
}

// ToQuizResponse mengubah 1 quiz (tanpa soal) menjadi response. Waktu event ditampilkan dalam WIB.
func ToQuizResponse(d quizmodel.QuizDetail, now time.Time) QuizResponse {
	return QuizResponse{
		ID:   d.ID,
		Name: d.Name,
		Type: d.Type,
		Placement: QuizPlacementResponse{
			EducationLevel: RefResponse{ID: d.EducationLevelID, Name: &d.EducationLevelName},
			Grade:          RefResponse{ID: d.GradeID, Name: &d.GradeName},
			Subject:        RefResponse{ID: d.SubjectID, Name: &d.SubjectName},
			SubjectField:   optionalRef(d.SubjectFieldID, d.SubjectFieldName),
			Chapter:        optionalRef(d.ChapterID, d.ChapterName),
			Subchapter:     optionalRef(d.SubchapterID, d.SubchapterName),
			Topic:          optionalRef(d.TopicID, d.TopicName),
		},
		StartAt:             d.StartAt.In(clock.Jakarta),
		EndAt:               d.EndAt.In(clock.Jakarta),
		ScheduleStatus:      quizmodel.ScheduleStatus(d.StartAt, d.EndAt, now),
		DurationMinutes:     d.DurationMinutes,
		AccessCode:          d.AccessCode,
		CoinCost:            d.CoinCost,
		TotalQuestions:      d.TotalQuestions,
		Points:              ToPointsResponse(d.Points),
		PrizeCoins:          ToPrizesResponse(d.Prizes),
		PrizesDistributedAt: inWIB(d.PrizesDistributedAt),
		Status:              d.Status,
		CreatedBy:           RefResponse{ID: d.CreatedBy, Name: d.CreatedByName},
		UpdatedBy:           optionalRef(d.UpdatedBy, d.UpdatedByName),
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

// inWIB mengubah waktu opsional ke WIB (nil tetap nil).
func inWIB(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	w := t.In(clock.Jakarta)
	return &w
}

// optionalRef mengembalikan nil (JSON null) kalau id kosong.
func optionalRef(id, name *string) *RefResponse {
	if id == nil {
		return nil
	}
	return &RefResponse{ID: *id, Name: name}
}

// ListBundlesQuery adalah query string GET /api/v1/quiz-bundles (pilihan bundle saat membuat quiz).
// Semua opsional. Contoh: ?type=uji_kom&subject_id=...&grade_id=...&search=pecahan
type ListBundlesQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search           string `form:"search" binding:"omitempty,max=100"` // nama quiz
	Type             string `form:"type" binding:"omitempty,oneof=uji_kom tryout"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
}

// BundleResponse adalah 1 bundle (quiz yang soalnya bisa dipakai ulang).
// Isi soalnya: GET /api/v1/questions?quiz_id=<id>.
type BundleResponse struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	EducationLevel RefResponse `json:"education_level"`
	Grade          RefResponse `json:"grade"`
	Subject        RefResponse `json:"subject"`
	TotalQuestions int         `json:"total_questions"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ToBundleResponses mengubah daftar bundle menjadi response. Hasil kosong menjadi [] (bukan null).
func ToBundleResponses(bundles []quizmodel.Bundle) []BundleResponse {
	result := make([]BundleResponse, 0, len(bundles))
	for _, b := range bundles {
		result = append(result, BundleResponse{
			ID:             b.ID,
			Name:           b.Name,
			Type:           b.Type,
			EducationLevel: RefResponse{ID: b.EducationLevelID, Name: &b.EducationLevelName},
			Grade:          RefResponse{ID: b.GradeID, Name: &b.GradeName},
			Subject:        RefResponse{ID: b.SubjectID, Name: &b.SubjectName},
			TotalQuestions: b.TotalQuestions,
			CreatedAt:      b.CreatedAt,
		})
	}
	return result
}

// ListQuizzesQuery adalah query string GET /api/v1/quizzes (list quiz admin). Semua opsional.
// Contoh: ?schedule_status=upcoming&type=uji_kom&subject_id=...&search=pecahan
type ListQuizzesQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Search string `form:"search" binding:"omitempty,max=100"` // nama quiz
	Type   string `form:"type" binding:"omitempty,oneof=daily_quiz uji_kom tryout rumath"`
	Status string `form:"status" binding:"omitempty,oneof=draft published"`
	// Jadwal event: upcoming (akan dimulai), ongoing (sedang berjalan), finished (sudah lewat).
	ScheduleStatus   string `form:"schedule_status" binding:"omitempty,oneof=upcoming ongoing finished"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"`
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`
}

// ScheduleCountsResponse adalah data GET /api/v1/quizzes/schedule-counts: angka di tab jadwal.
// Semua status selalu ada (0 kalau kosong). all = jumlah semuanya.
type ScheduleCountsResponse struct {
	All      int `json:"all"`
	Upcoming int `json:"upcoming"` // akan dimulai
	Ongoing  int `json:"ongoing"`  // sedang berjalan
	Finished int `json:"finished"` // sudah lewat
}

// ToScheduleCountsResponse mengubah jumlah per status jadwal menjadi response.
func ToScheduleCountsResponse(c quizmodel.ScheduleCounts) ScheduleCountsResponse {
	return ScheduleCountsResponse{
		All:      c.Upcoming + c.Ongoing + c.Finished,
		Upcoming: c.Upcoming,
		Ongoing:  c.Ongoing,
		Finished: c.Finished,
	}
}

// DeleteQuizzesRequest adalah body DELETE /api/v1/quizzes: hapus 1 sampai 100 quiz (soft delete).
type DeleteQuizzesRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteQuizzesResponse berisi ID quiz yang berhasil dihapus.
type DeleteQuizzesResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// AnnouncementsQuery adalah query string GET /api/v1/student/quiz-announcements.
type AnnouncementsQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// AnnouncementResponse adalah 1 pengumuman quiz untuk siswa. TANPA kode akses (dibagikan
// tutor saat quiz dimulai) dan tanpa daftar soal.
type AnnouncementResponse struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Type            string                `json:"type"`
	Placement       QuizPlacementResponse `json:"placement"`
	StartAt         time.Time             `json:"start_at"` // WIB
	EndAt           time.Time             `json:"end_at"`   // WIB
	ScheduleStatus  string                `json:"schedule_status"`
	DurationMinutes int                   `json:"duration_minutes"`
	TotalQuestions  int                   `json:"total_questions"`
	CoinCost        int                   `json:"coin_cost"`
	Points          QuizPointsResponse    `json:"points"`      // point 1 soal benar per level
	PrizeCoins      QuizPrizesResponse    `json:"prize_coins"` // hadiah coin juara 1-3
}

// ToAnnouncementResponses mengubah daftar quiz menjadi pengumuman siswa. Hasil kosong = [].
func ToAnnouncementResponses(items []quizmodel.QuizDetail, now time.Time) []AnnouncementResponse {
	result := make([]AnnouncementResponse, 0, len(items))
	for _, d := range items {
		q := ToQuizResponse(d, now)
		result = append(result, AnnouncementResponse{
			ID: q.ID, Name: q.Name, Type: q.Type, Placement: q.Placement,
			StartAt: q.StartAt, EndAt: q.EndAt, ScheduleStatus: q.ScheduleStatus,
			DurationMinutes: q.DurationMinutes, TotalQuestions: q.TotalQuestions, CoinCost: q.CoinCost,
			Points: q.Points, PrizeCoins: q.PrizeCoins,
		})
	}
	return result
}

// OpenQuizRequest adalah body POST /api/v1/quizzes/:id/open (siswa membuka Uji Kom / Tryout).
// access_code wajib saat pertama kali membuka; membuka ulang boleh tanpa body.
type OpenQuizRequest struct {
	AccessCode string `json:"access_code" binding:"omitempty,max=20"`
}

// OpenedQuizResponse adalah quiz yang dibuka siswa: info quiz (tanpa kode akses), waktu
// mengerjakan, dan soal lengkap dengan kunci jawaban (tanpa pembahasan).
// StateInProgress / StateSubmitted = nilai field "state" di response buka quiz & review:
// in_progress = soal untuk dikerjakan; submitted = siswa sudah submit, data berisi review
// (FE langsung menampilkan halaman review).
const (
	StateInProgress = "in_progress"
	StateSubmitted  = "submitted"
)

type OpenedQuizResponse struct {
	State           string                                `json:"state"` // "in_progress"
	ID              string                                `json:"id"`
	Name            string                                `json:"name"`
	Type            string                                `json:"type"`
	Placement       QuizPlacementResponse                 `json:"placement"`
	StartAt         time.Time                             `json:"start_at"` // event, WIB
	EndAt           time.Time                             `json:"end_at"`
	DurationMinutes int                                   `json:"duration_minutes"`
	StartedAt       time.Time                             `json:"started_at"`  // siswa mulai mengerjakan, WIB
	DeadlineAt      time.Time                             `json:"deadline_at"` // = min(started_at + durasi, end_at), WIB
	TotalQuestions  int                                   `json:"total_questions"`
	CoinCost        int                                   `json:"coin_cost"`
	Points          QuizPointsResponse                    `json:"points"` // point 1 soal benar per level
	Questions       []questiondto.AttemptQuestionResponse `json:"questions"`
}

// ToOpenedQuizResponse mengubah quiz yang dibuka menjadi response.
func ToOpenedQuizResponse(o quizmodel.OpenedQuiz, now time.Time) OpenedQuizResponse {
	q := ToQuizResponse(o.QuizDetail, now)
	return OpenedQuizResponse{
		State: StateInProgress,
		ID:    q.ID, Name: q.Name, Type: q.Type, Placement: q.Placement,
		StartAt: q.StartAt, EndAt: q.EndAt, DurationMinutes: q.DurationMinutes,
		StartedAt:      o.StartedAt.In(clock.Jakarta),
		DeadlineAt:     o.DeadlineAt.In(clock.Jakarta),
		TotalQuestions: q.TotalQuestions, CoinCost: q.CoinCost,
		Points:    q.Points,
		Questions: questiondto.ToAttemptQuestionResponses(o.Content),
	}
}

// AccessCodeResponse adalah data GET /api/v1/quizzes/:id/access-code (tutor/admin/super admin).
type AccessCodeResponse struct {
	QuizID              string     `json:"quiz_id"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	Status              string     `json:"status"`
	StartAt             time.Time  `json:"start_at"`
	EndAt               time.Time  `json:"end_at"`
	ScheduleStatus      string     `json:"schedule_status"`
	AccessCode          *string    `json:"access_code"`
	AccessCodeUpdatedAt *time.Time `json:"access_code_updated_at"` // diganti tiap jam selama event belum selesai
}

// ToAccessCodeResponse mengubah quiz menjadi response kode akses.
func ToAccessCodeResponse(d quizmodel.QuizDetail, now time.Time) AccessCodeResponse {
	var updatedAt *time.Time
	if d.AccessCodeUpdatedAt != nil {
		t := d.AccessCodeUpdatedAt.In(clock.Jakarta)
		updatedAt = &t
	}
	return AccessCodeResponse{
		QuizID: d.ID, Name: d.Name, Type: d.Type, Status: d.Status,
		StartAt: d.StartAt.In(clock.Jakarta), EndAt: d.EndAt.In(clock.Jakarta),
		ScheduleStatus:      quizmodel.ScheduleStatus(d.StartAt, d.EndAt, now),
		AccessCode:          d.AccessCode,
		AccessCodeUpdatedAt: updatedAt,
	}
}

// QuizReviewResponse adalah hasil & review Uji Kom / Tryout setelah submit (juga GET review):
// nilai (total 100) dan setiap soal dengan urutan yang sama seperti saat dikerjakan, lengkap dengan
// kunci jawaban, pembahasan, jawaban siswa, dan benar/salahnya.
type QuizReviewResponse struct {
	State     string                               `json:"state"` // selalu "submitted"
	ID        string                               `json:"id"`
	Name      string                               `json:"name"`
	Type      string                               `json:"type"`
	Placement QuizPlacementResponse                `json:"placement"`
	StartedAt time.Time                            `json:"started_at"` // WIB
	Result    questiondto.ResultResponse           `json:"result"`
	Questions []questiondto.ReviewQuestionResponse `json:"questions"`
}

// ToQuizReviewResponse mengubah review quiz menjadi response.
func ToQuizReviewResponse(r quizmodel.QuizReview, now time.Time) QuizReviewResponse {
	q := ToQuizResponse(r.QuizDetail, now)
	return QuizReviewResponse{
		State: StateSubmitted,
		ID:    q.ID, Name: q.Name, Type: q.Type, Placement: q.Placement,
		StartedAt: r.StartedAt.In(clock.Jakarta),
		Result:    questiondto.ToResultResponse(r.Score, r.Points, r.CorrectCount, len(r.Items), *r.SubmittedAt, r.DurationSeconds),
		Questions: questiondto.ToReviewResponses(r.Items),
	}
}
