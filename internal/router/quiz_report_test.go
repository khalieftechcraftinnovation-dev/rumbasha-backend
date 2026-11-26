// Test report quiz Uji Kom & Tryout: list per event, detail peserta, analisis soal.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// submittedAttempt menyimpan hasil submit 1 siswa langsung ke quiz_attempts (bentuk sama dengan submit).
// correct = benar/salah per soal (urut questionIDs); nil = tidak dijawab.
func submittedAttempt(t *testing.T, db *gorm.DB, quizID, studentID string, questionIDs []string, correct []*bool, score float64, points, duration int) {
	t.Helper()
	answers := []string{}
	count := 0
	for i, id := range questionIDs {
		answer, isCorrect := `"B"`, false
		if correct[i] == nil {
			answer = "null"
		} else {
			isCorrect = *correct[i]
		}
		if isCorrect {
			count++
		}
		answers = append(answers, fmt.Sprintf(`{"question_id":%q,"answer":%s,"is_correct":%v,"score":0,"points":0}`, id, answer, isCorrect))
	}
	err := db.Exec(`
		INSERT INTO quiz_attempts (quiz_id, student_id, started_at, submitted_at, score, correct_count, answers, duration_seconds, points)
		VALUES (?, ?, NOW() - INTERVAL '1 hour', NOW(), ?, ?, ?::jsonb, ?, ?)`,
		quizID, studentID, score, count, "["+strings.Join(answers, ",")+"]", duration, points).Error
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuizReport_ListDetailDanSoal(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	opts := map[string]string{"used_for": `["uji_kom","tryout"]`}
	q1 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, opts))
	q2 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, opts))
	q3 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, opts))
	ids := []string{q1, q2, q3}

	tryout := createQuiz(t, r, admin, quizBody(basePlacement(p), ids, map[string]string{"status": `"published"`, "type": `"tryout"`, "name": `"Tryout Matematika Kelas 6"`}))
	ujiKom := createQuiz(t, r, admin, quizBody(basePlacement(p), ids, map[string]string{"status": `"published"`, "name": `"Uji Kom Pecahan"`}))
	draft := createQuiz(t, r, admin, quizBody(basePlacement(p), ids, nil))
	deleted := createQuiz(t, r, admin, quizBody(basePlacement(p), ids, map[string]string{"status": `"published"`}))
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, deleted.ID)
	db.Exec(`UPDATE quizzes SET start_at = '2026-10-05 08:00+07', end_at = '2026-10-05 10:00+07' WHERE id = ?`, tryout.ID)

	yes, no := true, false
	ani, _ := namedStudent(t, db, "Ani", "GRADE_6")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_6")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_6")
	dodi, _ := namedStudent(t, db, "Dodi", "GRADE_6")
	submittedAttempt(t, db, tryout.ID, ani, ids, []*bool{&yes, &yes, &yes}, 100, 30, 300)
	submittedAttempt(t, db, tryout.ID, budi, ids, []*bool{&yes, &yes, &no}, 66.67, 20, 200)
	submittedAttempt(t, db, tryout.ID, cici, ids, []*bool{&yes, nil, &no}, 33.33, 10, 100)
	submittedAttempt(t, db, tryout.ID, dodi, ids, []*bool{&no, &no, &no}, 0, 0, 50)
	db.Exec(`UPDATE users SET deleted_at = NOW() WHERE id = ?`, dodi) // akun terhapus tidak dihitung
	// Sudah dibuka tapi belum submit: bukan peserta.
	other, _ := namedStudent(t, db, "Eko", "GRADE_6")
	db.Exec(`INSERT INTO quiz_attempts (quiz_id, student_id) VALUES (?, ?)`, tryout.ID, other)

	// ---- List per event ----------------------------------------------------------
	type summary struct {
		QuizID         string   `json:"quiz_id"`
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		StartAt        string   `json:"start_at"`
		TotalQuestions int      `json:"total_questions"`
		Participants   int      `json:"participants"`
		AverageScore   *float64 `json:"average_score"`
		HighestScore   *float64 `json:"highest_score"`
		LowestScore    *float64 `json:"lowest_score"`
	}
	var list []summary
	res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports", "", admin)
	decodeData(t, res, &list)
	// Event terbaru dulu (Uji Kom dijadwalkan di masa depan); draf & terhapus tidak ikut.
	if len(list) != 2 || list[0].QuizID != ujiKom.ID || list[1].QuizID != tryout.ID || decodeMeta(t, res).Total != 2 {
		t.Fatalf("list report: %+v", list)
	}
	if u := list[0]; u.Participants != 0 || u.AverageScore != nil || u.HighestScore != nil {
		t.Errorf("event tanpa peserta: %+v", u)
	}
	if s := list[1]; s.Participants != 3 || s.AverageScore == nil || *s.AverageScore != 66.67 ||
		*s.HighestScore != 100 || *s.LowestScore != 33.33 || s.TotalQuestions != 3 || s.StartAt != "2026-10-05T08:00:00+07:00" {
		t.Errorf("ringkasan tryout: %+v", s)
	}
	for query, want := range map[string]int{
		"?type=tryout":    1,
		"?search=pecahan": 1,
		"?date_from=2026-10-05&date_to=2026-10-05": 1,
		"?date_from=2026-10-06&date_to=2026-10-31": 0,
		"?grade_id=" + p.GradeID + "&type=uji_kom": 1,
	} {
		var items []summary
		decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/quiz-reports"+query, "", admin), &items)
		if len(items) != want {
			t.Errorf("filter %s: %d event, seharusnya %d", query, len(items), want)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports?date_from=05-10-2026", "", admin); errorFields(t, res.Body.String())["date_from"] == "" {
		t.Errorf("tanggal salah format: %s", res.Body)
	}

	// ---- Detail peserta ------------------------------------------------------------
	var detail struct {
		Quiz         summary `json:"quiz"`
		Participants []struct {
			Rank    int `json:"rank"`
			Student struct {
				Name string `json:"name"`
			} `json:"student"`
			CorrectLabel    string  `json:"correct_label"`
			Score           float64 `json:"score"`
			Points          int     `json:"points"`
			DurationSeconds int     `json:"duration_seconds"`
		} `json:"participants"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+tryout.ID, "", admin), &detail)
	got := []string{}
	for _, p := range detail.Participants {
		got = append(got, fmt.Sprintf("%d:%s:%s:%v:%d:%d", p.Rank, p.Student.Name, p.CorrectLabel, p.Score, p.Points, p.DurationSeconds))
	}
	if detail.Quiz.Participants != 3 || strings.Join(got, ",") != "1:Ani:3/3:100:30:300,2:Budi:2/3:66.67:20:200,3:Cici:1/3:33.33:10:100" {
		t.Errorf("detail: %v", got)
	}
	// Cari: peringkat tetap peringkat asli.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+tryout.ID+"?search=cici", "", admin), &detail)
	if len(detail.Participants) != 1 || detail.Participants[0].Rank != 3 {
		t.Errorf("cari peserta: %+v", detail.Participants)
	}

	// ---- Analisis soal: paling sering salah dulu -------------------------------------
	var qs struct {
		Questions []struct {
			No              int     `json:"no"`
			ID              string  `json:"id"`
			CorrectCount    int     `json:"correct_count"`
			WrongCount      int     `json:"wrong_count"`
			UnansweredCount int     `json:"unanswered_count"`
			CorrectPercent  float64 `json:"correct_percent"`
			CorrectLabel    string  `json:"correct_label"`
		} `json:"questions"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+tryout.ID+"/questions", "", admin), &qs)
	got = got[:0]
	for _, q := range qs.Questions {
		got = append(got, fmt.Sprintf("%d:%d/%d/%d:%v:%s", q.No, q.CorrectCount, q.WrongCount, q.UnansweredCount, q.CorrectPercent, q.CorrectLabel))
	}
	if strings.Join(got, ",") != "3:1/2/0:33.33:33%,2:2/0/1:66.67:67%,1:3/0/0:100:100%" {
		t.Errorf("analisis soal: %v", got)
	}
	// Event tanpa peserta: semua soal tetap tampil, 0%.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+ujiKom.ID+"/questions", "", admin), &qs)
	if len(qs.Questions) != 3 || qs.Questions[0].No != 1 || qs.Questions[0].CorrectPercent != 0 {
		t.Errorf("tanpa peserta: %+v", qs.Questions)
	}

	// ---- Tidak ditemukan & hak akses ---------------------------------------------------
	for _, id := range []string{draft.ID, deleted.ID, uuid.NewString(), "bukan-uuid"} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+id, "", admin); res.Code != http.StatusNotFound {
			t.Errorf("report %s: status %d, seharusnya 404", id, res.Code)
		}
	}
	_, student := namedStudent(t, db, "Fani", "GRADE_6")
	if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports", "", student); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status %d, seharusnya 403", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-reports/"+tryout.ID+"/questions", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusOK {
		t.Errorf("tutor: status %d, seharusnya 200", res.Code)
	}
}
