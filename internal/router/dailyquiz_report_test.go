// Test report Daily Quiz per siswa: list peringkat (filter periode, mapel, kelas) dan detail per mapel.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// submittedDailyQuiz menyimpan 1 Daily Quiz yang sudah di-submit langsung ke daily_quizzes.
// submittedAt = ekspresi SQL waktu submit, contoh NOW().
func submittedDailyQuiz(t *testing.T, db *gorm.DB, studentID string, p testPlacement, difficulty string,
	total, correct int, score float64, points, duration int, submittedAt string) {
	t.Helper()
	err := db.Exec(`
		INSERT INTO daily_quizzes (student_id, education_level_id, grade_id, subject_id, subject_field_id,
			difficulty, total_questions, opened_at, submitted_at, score, correct_count, answers, duration_seconds, points)
		VALUES (?, ?, ?, ?, ?, ?, ?, `+submittedAt+` - INTERVAL '10 minutes', `+submittedAt+`, ?, ?, '[]'::jsonb, ?, ?)`,
		studentID, p.LevelID, p.GradeID, p.SubjectID, p.FieldID, difficulty, total, score, correct, duration, points).Error
	if err != nil {
		t.Fatal(err)
	}
}

type dailyReportRow struct {
	Rank    int `json:"rank"`
	Student struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"student"`
	TotalQuizzes        int `json:"total_quizzes"`
	QuizzesByDifficulty struct {
		Easy, Medium, Hard int
	} `json:"quizzes_by_difficulty"`
	AverageScore           float64 `json:"average_score"`
	AverageDurationSeconds int     `json:"average_duration_seconds"`
	TotalQuestions         int     `json:"total_questions"`
	CorrectCount           int     `json:"correct_count"`
	WrongCount             int     `json:"wrong_count"`
	CorrectPercent         float64 `json:"correct_percent"`
	TotalPoints            int     `json:"total_points"`
}

func dailyReport(t *testing.T, r *gin.Engine, accessToken, query string) []dailyReportRow {
	t.Helper()
	var data struct {
		Students []dailyReportRow `json:"students"`
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports"+query, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("report %s: status %d. body: %s", query, res.Code, res.Body)
	}
	decodeData(t, res, &data)
	return data.Students
}

func rankNames(rows []dailyReportRow) string {
	parts := []string{}
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%d:%s:%d", r.Rank, r.Student.Name, r.TotalPoints))
	}
	return strings.Join(parts, ",")
}

func TestDailyQuizReport_PerSiswa(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	matematika := seedPlacementInGrade(t, db, "GRADE_5")
	ipa := seedPlacementInGrade(t, db, "GRADE_5") // mapel lain
	ani, _ := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_5")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_7")
	dodi, _ := namedStudent(t, db, "Dodi", "GRADE_5")

	lastMonth := `(NOW() - INTERVAL '1 month')`
	// Ani: 3 Daily Quiz bulan ini (easy, medium, hard) + 1 bulan lalu.
	submittedDailyQuiz(t, db, ani, matematika, "easy", 5, 5, 100, 50, 100, "NOW()")
	submittedDailyQuiz(t, db, ani, matematika, "medium", 10, 6, 60, 90, 300, "NOW()")
	submittedDailyQuiz(t, db, ani, ipa, "hard", 5, 2, 40, 40, 200, "NOW()")
	submittedDailyQuiz(t, db, ani, matematika, "easy", 5, 5, 100, 50, 100, lastMonth)
	// Budi: 2 bulan ini + 1 dibuka tapi belum submit (tidak dihitung).
	submittedDailyQuiz(t, db, budi, matematika, "easy", 5, 4, 80, 40, 120, "NOW()")
	submittedDailyQuiz(t, db, budi, ipa, "easy", 5, 5, 100, 50, 60, "NOW()")
	db.Exec(`INSERT INTO daily_quizzes (student_id, education_level_id, grade_id, subject_id, subject_field_id, difficulty, total_questions, opened_at)
		VALUES (?, ?, ?, ?, ?, 'easy', 5, NOW())`, budi, matematika.LevelID, matematika.GradeID, matematika.SubjectID, matematika.FieldID)
	// Cici (Kelas 7) & Dodi (akun dihapus, tidak ikut).
	submittedDailyQuiz(t, db, cici, ipa, "medium", 10, 10, 100, 150, 500, "NOW()")
	submittedDailyQuiz(t, db, dodi, matematika, "hard", 15, 15, 100, 300, 100, "NOW()")
	db.Exec(`UPDATE users SET deleted_at = NOW() WHERE id = ?`, dodi)

	// ---- Default: bulan ini, semua mapel -----------------------------------------
	var data struct {
		Period struct {
			DateFrom string `json:"date_from"`
			DateTo   string `json:"date_to"`
		} `json:"period"`
		Students []dailyReportRow `json:"students"`
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports", "", admin)
	decodeData(t, res, &data)
	now := time.Now().In(clock.Jakarta)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, clock.Jakarta)
	if data.Period.DateFrom != monthStart.Format(time.DateOnly) || data.Period.DateTo != monthStart.AddDate(0, 1, -1).Format(time.DateOnly) {
		t.Errorf("periode default: %+v", data.Period)
	}
	if got := rankNames(data.Students); got != "1:Ani:180,2:Cici:150,3:Budi:90" || decodeMeta(t, res).Total != 3 {
		t.Fatalf("peringkat: %s", got)
	}
	a := data.Students[0]
	if a.TotalQuizzes != 3 || a.QuizzesByDifficulty.Easy != 1 || a.QuizzesByDifficulty.Medium != 1 || a.QuizzesByDifficulty.Hard != 1 ||
		a.AverageScore != 66.67 || a.AverageDurationSeconds != 200 || a.TotalQuestions != 20 || a.CorrectCount != 13 ||
		a.WrongCount != 7 || a.CorrectPercent != 65 {
		t.Errorf("rekap Ani: %+v", a)
	}

	// ---- Filter: kelas, mapel, cari, periode ---------------------------------------
	for query, want := range map[string]string{
		"?grade_id=" + matematika.GradeID: "1:Ani:180,2:Budi:90",           // peringkat dihitung di dalam filter
		"?subject_id=" + ipa.SubjectID:    "1:Cici:150,2:Budi:50,3:Ani:40", // hanya Daily Quiz mapel IPA
		"?search=budi":                    "3:Budi:90",                     // cari tidak mengubah peringkat
		"?date_from=" + monthStart.AddDate(0, -1, 0).Format(time.DateOnly) + "&date_to=" + monthStart.AddDate(0, 0, -1).Format(time.DateOnly): "1:Ani:50",
	} {
		if got := rankNames(dailyReport(t, r, admin, query)); got != want {
			t.Errorf("filter %s: %s, seharusnya %s", query, got, want)
		}
	}
	for _, query := range []string{"?date_from=01-11-2026", "?date_from=2026-11-30&date_to=2026-11-01", "?date_from=2025-01-01&date_to=2026-12-31"} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports"+query, "", admin); res.Code != http.StatusUnprocessableEntity {
			t.Errorf("periode salah %s: status %d", query, res.Code)
		}
	}

	// ---- Detail 1 siswa: total + per mapel -------------------------------------------
	var detail struct {
		Student struct {
			ID string `json:"id"`
		} `json:"student"`
		Total    dailyReportRow `json:"total"`
		Subjects []struct {
			Subject listRef `json:"subject"`
			dailyReportRow
		} `json:"subjects"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports/students/"+ani, "", admin), &detail)
	if detail.Student.ID != ani || detail.Total.TotalQuizzes != 3 || detail.Total.TotalPoints != 180 || detail.Total.AverageScore != 66.67 ||
		len(detail.Subjects) != 2 || detail.Subjects[0].Subject.ID != matematika.SubjectID || detail.Subjects[0].TotalQuizzes != 2 ||
		detail.Subjects[0].CorrectPercent != 73.33 || detail.Subjects[1].Subject.ID != ipa.SubjectID || detail.Subjects[1].QuizzesByDifficulty.Hard != 1 {
		t.Errorf("detail Ani: %+v", detail)
	}
	// Siswa tanpa Daily Quiz di periode: rekap 0, tanpa mapel.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports/students/"+cici+"?date_from=2020-01-01&date_to=2020-01-31", "", admin), &detail)
	if detail.Total.TotalQuizzes != 0 || detail.Total.AverageScore != 0 || len(detail.Subjects) != 0 {
		t.Errorf("periode kosong: %+v", detail)
	}
	for _, id := range []string{uuid.NewString(), "bukan-uuid", dodi} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports/students/"+id, "", admin); res.Code != http.StatusNotFound {
			t.Errorf("detail %s: status %d, seharusnya 404", id, res.Code)
		}
	}

	// ---- Hak akses ----------------------------------------------------------------------
	_, student := namedStudent(t, db, "Eko", "GRADE_5")
	if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports", "", student); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status %d, seharusnya 403", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-reports", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusOK {
		t.Errorf("tutor: status %d, seharusnya 200", res.Code)
	}
}
