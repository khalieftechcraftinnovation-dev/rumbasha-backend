// Test kartu ringkasan & grafik report quiz (GET /quiz-reports/overview) + pengaturan report.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// attemptAt menyimpan 1 pengerjaan Uji Kom / Tryout yang sudah submit pada waktu tertentu.
// correct: "T" benar, "F" salah, "-" tidak dijawab (urut questionIDs). submittedAt & minutes =
// ekspresi SQL waktu submit dan lama mengerjakan (menit), contoh "NOW()" & 10.
func attemptAt(t *testing.T, db *gorm.DB, quizID, studentID string, questionIDs []string, correct string, score float64, submittedAt string, minutes int) {
	t.Helper()
	answers := make([]string, 0, len(questionIDs))
	count := 0
	for i, id := range questionIDs {
		answer, ok := `"B"`, correct[i] == 'T'
		if correct[i] == '-' {
			answer = "null"
		}
		if ok {
			count++
		}
		answers = append(answers, fmt.Sprintf(`{"question_id":%q,"answer":%s,"is_correct":%v,"score":0,"points":0}`, id, answer, ok))
	}
	err := db.Exec(fmt.Sprintf(`
		INSERT INTO quiz_attempts (quiz_id, student_id, started_at, submitted_at, score, correct_count, answers, duration_seconds, points)
		VALUES (?, ?, %[1]s - INTERVAL '%[2]d minutes', %[1]s, ?, ?, ?::jsonb, %[3]d, 0)`, submittedAt, minutes, minutes*60),
		quizID, studentID, score, count, "["+strings.Join(answers, ",")+"]").Error
	if err != nil {
		t.Fatal(err)
	}
}

type overviewData struct {
	Type   string `json:"type"`
	Period struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	} `json:"period"`
	Cards struct {
		Sessions struct {
			Total           int  `json:"total"`
			DistinctQuizzes *int `json:"distinct_quizzes"`
			Students        int  `json:"students"`
		} `json:"sessions"`
		AverageScore struct {
			Overall *float64 `json:"overall"`
			ByLevel []struct {
				EducationLevel listRef `json:"education_level"`
				Sessions       int     `json:"sessions"`
				Average        float64 `json:"average"`
			} `json:"by_level"`
		} `json:"average_score"`
		OnTime *struct {
			Percent  *float64 `json:"percent"`
			OnTime   int      `json:"on_time"`
			TimedOut int      `json:"timed_out"`
		} `json:"on_time"`
		FrequentlyWrong struct {
			ThresholdPercent int `json:"threshold_percent"`
			Count            int `json:"count"`
			TotalQuestions   int `json:"total_questions"`
			TopChapter       *struct {
				ID    string `json:"id"`
				Count int    `json:"count"`
			} `json:"top_chapter"`
			Questions []struct {
				ID           string  `json:"id"`
				WrongPercent float64 `json:"wrong_percent"`
			} `json:"questions"`
		} `json:"frequently_wrong"`
	} `json:"cards"`
	Trend struct {
		Weeks []struct {
			WeekStart string   `json:"week_start"`
			WeekEnd   string   `json:"week_end"`
			Sessions  int      `json:"sessions"`
			Average   *float64 `json:"average"`
		} `json:"weeks"`
		Series []struct {
			EducationLevel listRef `json:"education_level"`
			Points         []struct {
				Sessions int      `json:"sessions"`
				Average  *float64 `json:"average"`
			} `json:"points"`
		} `json:"series"`
	} `json:"trend"`
	Distribution []struct {
		Label       string `json:"label"`
		Count       int    `json:"count"`
		Highlighted bool   `json:"highlighted"`
	} `json:"distribution"`
}

func getOverview(t *testing.T, r *gin.Engine, token, query string) overviewData {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/overview"+query, "", token)
	if res.Code != http.StatusOK {
		t.Fatalf("overview %s: status %d. body: %s", query, res.Code, res.Body)
	}
	var o overviewData
	decodeData(t, res, &o)
	return o
}

func TestQuizReportOverview_KartuDanGrafik(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	sd, smp := seedPlacementInGrade(t, db, "GRADE_6"), seedPlacementInGrade(t, db, "GRADE_9")
	opts := map[string]string{"used_for": `["uji_kom","tryout","daily_quiz"]`}
	q1 := createQuestionID(t, r, admin, questionBody(sd, "single_choice", singleChoiceAnswer, opts))
	q2 := createQuestionID(t, r, admin, questionBody(sd, "single_choice", singleChoiceAnswer, opts))
	q3 := createQuestionID(t, r, admin, questionBody(sd, "single_choice", singleChoiceAnswer, opts))
	q4 := createQuestionID(t, r, admin, questionBody(smp, "single_choice", singleChoiceAnswer, opts))
	q5 := createQuestionID(t, r, admin, questionBody(smp, "single_choice", singleChoiceAnswer, opts))
	sdIDs, smpIDs := []string{q1, q2, q3}, []string{q4, q5}
	quizSD := createQuiz(t, r, admin, quizBody(basePlacement(sd), sdIDs, map[string]string{"status": `"published"`}))
	quizSMP := createQuiz(t, r, admin, quizBody(basePlacement(smp), smpIDs, map[string]string{"status": `"published"`}))
	tryout := createQuiz(t, r, admin, quizBody(basePlacement(sd), sdIDs, map[string]string{"status": `"published"`, "type": `"tryout"`}))
	// Durasi 30 menit, event sudah berjalan lama dan belum berakhir.
	if err := db.Exec(`UPDATE quizzes SET duration_minutes = 30, start_at = NOW() - INTERVAL '200 days', end_at = NOW() + INTERVAL '1 day' WHERE type = 'uji_kom'`).Error; err != nil {
		t.Fatal(err)
	}
	db.Exec(`UPDATE quizzes SET start_at = NOW() + INTERVAL '2 days', end_at = NOW() + INTERVAL '3 days' WHERE type = 'tryout'`)

	student := func(name, grade string) string { id, _ := namedStudent(t, db, name, grade); return id }
	ani, budi, cici, dedi := student("Ani", "GRADE_6"), student("Budi", "GRADE_6"), student("Cici", "GRADE_6"), student("Dedi", "GRADE_9")
	eko, fani := student("Eko", "GRADE_6"), student("Fani", "GRADE_6")
	attemptAt(t, db, quizSD.ID, ani, sdIDs, "TTT", 100, "NOW()", 10)
	attemptAt(t, db, quizSD.ID, budi, sdIDs, "TTF", 66.67, "NOW()", 12)
	attemptAt(t, db, quizSD.ID, cici, sdIDs, "T-F", 20, "NOW()", 31)                       // kehabisan waktu
	attemptAt(t, db, quizSMP.ID, dedi, smpIDs, "TF", 50, "NOW()", 15)                      // SMP
	attemptAt(t, db, quizSD.ID, eko, sdIDs, "TTF", 80, "NOW() - INTERVAL '30 days'", 10)   // minggu lain
	attemptAt(t, db, quizSD.ID, fani, sdIDs, "FFF", 10, "NOW() - INTERVAL '100 days'", 10) // di luar periode & tren
	attemptAt(t, db, tryout.ID, ani, sdIDs, "FFF", 0, "NOW()", 10)                         // tab lain

	o := getOverview(t, r, admin, "?type=uji_kom")
	c := o.Cards
	if o.Type != "uji_kom" || c.Sessions.Total != 5 || c.Sessions.DistinctQuizzes == nil || *c.Sessions.DistinctQuizzes != 2 || c.Sessions.Students != 5 {
		t.Errorf("kartu sesi: %+v", c.Sessions)
	}
	if c.AverageScore.Overall == nil || *c.AverageScore.Overall != 63.33 || len(c.AverageScore.ByLevel) != 2 ||
		c.AverageScore.ByLevel[0].EducationLevel.Name != "SD" || c.AverageScore.ByLevel[0].Average != 66.67 || c.AverageScore.ByLevel[0].Sessions != 4 ||
		c.AverageScore.ByLevel[1].EducationLevel.Name != "SMP" || c.AverageScore.ByLevel[1].Average != 50 {
		t.Errorf("kartu rata-rata: %+v", c.AverageScore)
	}
	if c.OnTime == nil || c.OnTime.TimedOut != 1 || c.OnTime.OnTime != 4 || *c.OnTime.Percent != 80 {
		t.Errorf("kartu tepat waktu: %+v", c.OnTime)
	}
	// q2 salah 1/4 (25%), q3 3/4 (75%), q5 1/1 (100%) > 15%; q1 & q4 selalu benar.
	w := c.FrequentlyWrong
	if w.ThresholdPercent != 15 || w.Count != 3 || w.TotalQuestions != 5 || len(w.Questions) != 3 ||
		w.Questions[0].ID != q5 || w.Questions[1].ID != q3 || w.Questions[1].WrongPercent != 75 || w.Questions[2].ID != q2 ||
		w.TopChapter == nil || w.TopChapter.ID != sd.ChapterID || w.TopChapter.Count != 2 {
		t.Errorf("kartu soal sering salah: %+v", w)
	}

	// Sebaran: 20 → 0-20, 50 → 41-60, 66,67 & 80 → 61-80 (disorot), 100 → 81-100.
	got := []string{}
	for _, b := range o.Distribution {
		mark := ""
		if b.Highlighted {
			mark = "*"
		}
		got = append(got, fmt.Sprintf("%s:%d%s", b.Label, b.Count, mark))
	}
	if strings.Join(got, " ") != "0-20:1 21-40:0 41-60:1 61-80:2* 81-100:1" {
		t.Errorf("sebaran nilai: %v", got)
	}

	// Tren 12 minggu: minggu terakhir = minggu ini (4 pengerjaan), +1 pengerjaan 30 hari lalu.
	weeks := o.Trend.Weeks
	total := 0
	for _, wk := range weeks {
		total += wk.Sessions
	}
	last := weeks[len(weeks)-1]
	if len(weeks) != 12 || total != 5 || last.Sessions != 4 || last.Average == nil || *last.Average != 59.17 || last.WeekEnd < o.Period.DateTo {
		t.Errorf("tren minggu: %d minggu, total %d, terakhir %+v", len(weeks), total, last)
	}
	if len(o.Trend.Series) != 2 || o.Trend.Series[0].EducationLevel.Name != "SD" || *o.Trend.Series[0].Points[11].Average != 62.22 ||
		*o.Trend.Series[1].Points[11].Average != 50 || o.Trend.Series[1].Points[0].Average != nil {
		t.Errorf("tren per jenjang: %+v", o.Trend.Series)
	}

	// Filter jenjang, ambang per request, rentang tanggal.
	if o := getOverview(t, r, admin, "?type=uji_kom&education_level_id="+smp.LevelID); o.Cards.Sessions.Total != 1 || len(o.Trend.Series) != 1 {
		t.Errorf("filter SMP: %+v", o.Cards.Sessions)
	}
	if o := getOverview(t, r, admin, "?type=uji_kom&wrong_threshold=50"); o.Cards.FrequentlyWrong.Count != 2 || o.Cards.FrequentlyWrong.ThresholdPercent != 50 {
		t.Errorf("ambang 50%%: %+v", o.Cards.FrequentlyWrong)
	}
	if o := getOverview(t, r, admin, "?type=uji_kom&date_from=2020-01-01&date_to=2020-01-31"); o.Cards.Sessions.Total != 0 ||
		o.Cards.AverageScore.Overall != nil || o.Cards.OnTime.Percent != nil || o.Distribution[0].Highlighted || o.Cards.FrequentlyWrong.TotalQuestions != 0 {
		t.Errorf("periode kosong: %+v", o.Cards)
	}
	for query, field := range map[string]string{
		"":             "type",
		"?type=rumath": "type",
		"?type=tryout&date_from=2026-11-10&date_to=2026-11-01": "date_to",
		"?type=tryout&date_from=2025-01-01&date_to=2026-11-01": "date_to",
		"?type=tryout&date_from=10-11-2026":                    "date_from",
	} {
		res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/overview"+query, "", admin)
		if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())[field] == "" {
			t.Errorf("validasi %q: status %d, body %s", query, res.Code, res.Body)
		}
	}

	// Tab Tryout & Daily Quiz.
	if o := getOverview(t, r, admin, "?type=tryout"); o.Cards.Sessions.Total != 1 || *o.Cards.AverageScore.Overall != 0 {
		t.Errorf("tab tryout: %+v", o.Cards)
	}
	answers := fmt.Sprintf(`[{"question_id":%q,"answer":"B","is_correct":true},{"question_id":%q,"answer":null,"is_correct":false}]`, q1, q2)
	for _, score := range []float64{50, 90} {
		err := db.Exec(`
			INSERT INTO daily_quizzes (student_id, education_level_id, grade_id, subject_id, subject_field_id,
				difficulty, total_questions, opened_at, submitted_at, score, correct_count, answers, duration_seconds, points)
			VALUES (?, ?, ?, ?, ?, 'easy', 5, NOW() - INTERVAL '5 minutes', NOW(), ?, 1, ?::jsonb, 300, 10)`,
			ani, sd.LevelID, sd.GradeID, sd.SubjectID, sd.FieldID, score, answers).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	daily := getOverview(t, r, admin, "?type=daily_quiz")
	if daily.Cards.Sessions.Total != 2 || daily.Cards.Sessions.DistinctQuizzes != nil || daily.Cards.Sessions.Students != 1 ||
		*daily.Cards.AverageScore.Overall != 70 || daily.Cards.OnTime != nil || daily.Cards.FrequentlyWrong.Count != 1 {
		t.Errorf("tab Daily Quiz: %+v", daily.Cards)
	}

	// Pengaturan report: nilai awal, ubah (admin), tutor hanya boleh membaca.
	tutor := tokenFor(t, loginmodel.RoleTutor)
	res := sendRequest(r, http.MethodGet, "/api/v1/quiz-report-settings", "", tutor)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"wrong_threshold_percent":15`) || !strings.Contains(res.Body.String(), `"default_range_days":60`) {
		t.Errorf("pengaturan awal: %d %s", res.Code, res.Body)
	}
	body := `{"wrong_threshold_percent":40,"default_range_days":30}`
	if res := sendRequest(r, http.MethodPut, "/api/v1/quiz-report-settings", body, tutor); res.Code != http.StatusForbidden {
		t.Errorf("tutor ubah pengaturan: status %d", res.Code)
	}
	if res := sendRequest(r, http.MethodPut, "/api/v1/quiz-report-settings", `{"wrong_threshold_percent":0,"default_range_days":400}`, admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("pengaturan tidak valid: status %d", res.Code)
	}
	res = sendRequest(r, http.MethodPut, "/api/v1/quiz-report-settings", body, admin)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"updated_by":{`) {
		t.Errorf("ubah pengaturan: %d %s", res.Code, res.Body)
	}
	// Default baru: ambang 40% (q5 & q3), rentang 30 hari (pengerjaan 30 hari lalu tidak ikut).
	o = getOverview(t, r, tutor, "?type=uji_kom")
	if o.Cards.FrequentlyWrong.ThresholdPercent != 40 || o.Cards.FrequentlyWrong.Count != 2 || o.Cards.Sessions.Total != 4 {
		t.Errorf("overview dengan pengaturan baru: %+v %+v", o.Cards.FrequentlyWrong, o.Cards.Sessions)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/overview?type=uji_kom", "", tokenFor(t, loginmodel.RoleStudent)); res.Code != http.StatusForbidden {
		t.Errorf("siswa membuka report: status %d", res.Code)
	}
}
