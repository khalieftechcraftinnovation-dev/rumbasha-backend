// Package dailyquizreport (dto) berisi bentuk request dan response JSON report Daily Quiz per siswa.
//
// Di package lain, import dengan alias dailyquizreportdto:
//
//	dailyquizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/report/dailyquizreport"
package dailyquizreport

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/dailyquizreport"
)

// PeriodQuery = periode report: tanggal submit Daily Quiz (WIB, inklusif). Kosong = bulan ini.
type PeriodQuery struct {
	DateFrom string `form:"date_from" binding:"omitempty,max=10"` // contoh 2026-11-01
	DateTo   string `form:"date_to" binding:"omitempty,max=10"`   // contoh 2026-11-30
}

// ListQuery adalah query string GET /api/v1/daily-quiz-reports. Semua opsional.
//
//	?date_from=2026-11-01&date_to=2026-11-30&subject_id=...&grade_id=...&search=budi&page=1&per_page=20
type ListQuery struct {
	PeriodQuery
	SubjectID        string `form:"subject_id" binding:"omitempty,uuid"`         // mapel; kosong = semua mapel digabung
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"` // jenjang akun siswa
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`           // kelas akun siswa
	Search           string `form:"search" binding:"omitempty,max=100"`          // nama / kode siswa
	Page             int    `form:"page" binding:"omitempty,min=1"`
	PerPage          int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// PeriodResponse = periode yang dipakai (tanggal WIB, inklusif).
type PeriodResponse struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
}

func toPeriod(p dailyquizreportmodel.Period) PeriodResponse {
	return PeriodResponse{
		DateFrom: p.From.In(clock.Jakarta).Format(time.DateOnly),
		DateTo:   p.To.In(clock.Jakarta).AddDate(0, 0, -1).Format(time.DateOnly),
	}
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *dailyquizreportmodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// StatsResponse adalah rekap Daily Quiz yang sudah di-submit.
type StatsResponse struct {
	TotalQuizzes           int                `json:"total_quizzes"`
	QuizzesByDifficulty    DifficultyResponse `json:"quizzes_by_difficulty"` // jumlah Daily Quiz per level
	AverageScore           float64            `json:"average_score"`         // rata-rata nilai 0-100
	AverageDurationSeconds int                `json:"average_duration_seconds"`
	TotalQuestions         int                `json:"total_questions"` // soal yang dikerjakan
	CorrectCount           int                `json:"correct_count"`
	WrongCount             int                `json:"wrong_count"`     // salah + tidak dijawab
	CorrectPercent         float64            `json:"correct_percent"` // 0-100
	TotalPoints            int                `json:"total_points"`
	TotalDurationSeconds   int                `json:"total_duration_seconds"` // penentu peringkat kalau point sama
}

// DifficultyResponse = jumlah Daily Quiz per level.
type DifficultyResponse struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

func toStats(s dailyquizreportmodel.Stats) StatsResponse {
	return StatsResponse{
		TotalQuizzes:           s.TotalQuizzes,
		QuizzesByDifficulty:    DifficultyResponse{Easy: s.Easy, Medium: s.Medium, Hard: s.Hard},
		AverageScore:           s.AverageScore,
		AverageDurationSeconds: s.AverageDurationSeconds,
		TotalQuestions:         s.TotalQuestions,
		CorrectCount:           s.Correct,
		WrongCount:             s.Wrong(),
		CorrectPercent:         s.CorrectPercent(),
		TotalPoints:            s.TotalPoints,
		TotalDurationSeconds:   s.TotalDurationSeconds,
	}
}

// StudentResponse adalah data siswa.
type StudentResponse struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	StudentCode    *string      `json:"student_code"`
	EducationLevel *RefResponse `json:"education_level"`
	Grade          *RefResponse `json:"grade"`
}

// ListResponse adalah data GET /api/v1/daily-quiz-reports.
type ListResponse struct {
	Period   PeriodResponse `json:"period"`
	Students []StudentRow   `json:"students"`
}

// StudentRow adalah 1 baris report: peringkat + siswa + rekap.
type StudentRow struct {
	Rank    int             `json:"rank"`
	Student StudentResponse `json:"student"`
	StatsResponse
}

// ToListResponse mengubah list report menjadi response.
func ToListResponse(p dailyquizreportmodel.Period, rows []dailyquizreportmodel.StudentRow) ListResponse {
	r := ListResponse{Period: toPeriod(p), Students: make([]StudentRow, 0, len(rows))}
	for _, row := range rows {
		r.Students = append(r.Students, StudentRow{
			Rank: row.Rank,
			Student: StudentResponse{
				ID: row.StudentID, Name: row.Name, StudentCode: row.StudentCode,
				EducationLevel: toRef(row.EducationLevel), Grade: toRef(row.Grade),
			},
			StatsResponse: toStats(row.Stats),
		})
	}
	return r
}

// StudentDetailResponse adalah data GET /api/v1/daily-quiz-reports/students/:id.
type StudentDetailResponse struct {
	Period   PeriodResponse    `json:"period"`
	Student  StudentResponse   `json:"student"`
	Total    StatsResponse     `json:"total"`    // gabungan semua mapel
	Subjects []SubjectResponse `json:"subjects"` // per mapel, quiz terbanyak dulu
}

// SubjectResponse = rekap 1 mapel.
type SubjectResponse struct {
	Subject RefResponse `json:"subject"`
	StatsResponse
}

// ToStudentDetailResponse mengubah detail report 1 siswa menjadi response.
func ToStudentDetailResponse(st dailyquizreportmodel.Student, p dailyquizreportmodel.Period, total dailyquizreportmodel.Stats, subjects []dailyquizreportmodel.SubjectRow) StudentDetailResponse {
	r := StudentDetailResponse{
		Period: toPeriod(p),
		Student: StudentResponse{
			ID: st.ID, Name: st.Name, StudentCode: st.StudentCode,
			EducationLevel: toRef(st.EducationLevel), Grade: toRef(st.Grade),
		},
		Total:    toStats(total),
		Subjects: make([]SubjectResponse, 0, len(subjects)),
	}
	for _, s := range subjects {
		r.Subjects = append(r.Subjects, SubjectResponse{Subject: RefResponse(s.Subject), StatsResponse: toStats(s.Stats)})
	}
	return r
}
