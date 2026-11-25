// Test submit jawaban & review: Uji Kom / Tryout dan Daily Quiz.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type reviewData struct {
	State  string `json:"state"`
	Result struct {
		Score             float64 `json:"score"`
		CorrectCount      int     `json:"correct_count"`
		TotalQuestions    int     `json:"total_questions"`
		PointsPerQuestion float64 `json:"points_per_question"`
		SubmittedAt       string  `json:"submitted_at"`
		DurationSeconds   int     `json:"duration_seconds"`
	} `json:"result"`
	Questions []struct {
		No               int            `json:"no"`
		ID               string         `json:"id"`
		Explanation      string         `json:"explanation"`
		Answer           map[string]any `json:"answer"`
		StudentAnswer    any            `json:"student_answer"`
		IsCorrect        bool           `json:"is_correct"`
		Points           float64        `json:"points"`
		AnsweredAt       *string        `json:"answered_at"`
		TimeSpentSeconds *int           `json:"time_spent_seconds"`
	} `json:"questions"`
}

func TestSubmitQuiz_NilaiDanReview(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)

	q1 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	q2 := createQuestionID(t, r, admin, questionBody(p, "free_text", freeTextNumberAnswer, nil))
	q3 := createQuestionID(t, r, admin, questionBody(p, "true_false", trueFalseAnswer, nil))
	qOther := createQuestionID(t, r, admin, questionBody(other, "single_choice", singleChoiceAnswer, nil))

	published := map[string]string{"status": `"published"`}
	quiz := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q1, q2, q3}, published))
	late := createQuiz(t, r, admin, quizBody(basePlacement(other), []string{qOther}, published))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '3 hours', end_at = NOW() + INTERVAL '3 hours' WHERE id IN (?, ?)`, quiz.ID, late.ID)

	student := studentToken(t, db, "GRADE_6")
	submitURL := "/api/v1/quizzes/" + quiz.ID + "/submit"
	answers := fmt.Sprintf(`{"answers":[{"question_id":%q,"answer":"B","is_correct":true,"answered_at":"2026-12-01T01:05:30Z","time_spent_seconds":42},{"question_id":%q,"answer":"0.75","is_correct":true}]}`, q1, q2)

	// Belum dibuka → 409; review sebelum submit → 409.
	if res := sendRequest(r, http.MethodPost, submitURL, answers, student); res.Code != http.StatusConflict ||
		!strings.Contains(res.Body.String(), "Quiz belum dibuka") {
		t.Errorf("submit sebelum dibuka: status %d, body %s", res.Code, res.Body)
	}
	if code, body := openQuiz(r, student, quiz.ID, fmt.Sprintf(`{"access_code":%q}`, quiz.AccessCode)); code != http.StatusOK {
		t.Fatalf("buka quiz: status %d, body %s", code, body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+quiz.ID+"/review", "", student); res.Code != http.StatusConflict {
		t.Errorf("review sebelum submit: status %d, seharusnya 409", res.Code)
	}

	// Body tidak valid: soal bukan bagian quiz, is_correct tidak dikirim.
	res := sendRequest(r, http.MethodPost, submitURL, fmt.Sprintf(`{"answers":[{"question_id":%q,"answer":"A","is_correct":true}]}`, qOther), student)
	if errorFields(t, res.Body.String())["answers[0].question_id"] != "soal ini bukan bagian dari quiz" {
		t.Errorf("soal asing: body %s", res.Body)
	}
	res = sendRequest(r, http.MethodPost, submitURL, fmt.Sprintf(`{"answers":[{"question_id":%q,"answer":"A"}]}`, q1), student)
	if errorFields(t, res.Body.String())["answers[0].is_correct"] != "wajib diisi" {
		t.Errorf("is_correct kosong: body %s", res.Body)
	}

	// Submit: 2 benar dari 3 (q3 tidak dijawab) → 66.67; review urut sama, dengan pembahasan & kunci.
	// Mulai mengerjakan dibuat 10 menit lalu supaya lama mengerjakan bisa dicek.
	db.Exec(`UPDATE quiz_attempts SET started_at = NOW() - INTERVAL '10 minutes' WHERE quiz_id = ?`, quiz.ID)
	res = sendRequest(r, http.MethodPost, submitURL, answers, student)
	if res.Code != http.StatusOK {
		t.Fatalf("submit: status %d. body: %s", res.Code, res.Body)
	}
	var review reviewData
	decodeData(t, res, &review)
	if review.Result.Score != 66.67 || review.Result.CorrectCount != 2 || review.Result.TotalQuestions != 3 ||
		review.Result.PointsPerQuestion != 33.33 || !strings.HasSuffix(review.Result.SubmittedAt, "+07:00") {
		t.Errorf("hasil: %+v", review.Result)
	}
	wantIDs := []string{q1, q2, q3}
	for i, q := range review.Questions {
		if q.No != i+1 || q.ID != wantIDs[i] || q.Explanation == "" || q.Answer == nil {
			t.Errorf("review soal %d: %+v", i+1, q)
		}
	}
	if len(review.Questions) != 3 || review.Questions[0].StudentAnswer != "B" || !review.Questions[0].IsCorrect ||
		review.Questions[0].Points != 33.33 || review.Questions[0].Answer["correct_key"] != "B" ||
		review.Questions[2].StudentAnswer != nil || review.Questions[2].IsCorrect || review.Questions[2].Points != 0 {
		t.Errorf("review jawaban: %+v", review.Questions)
	}
	// Waktu: per soal dari FE (ditampilkan WIB), total = submit - mulai (± 10 menit).
	if review.State != "submitted" || review.Result.DurationSeconds < 595 || review.Result.DurationSeconds > 660 {
		t.Errorf("state %q, lama mengerjakan %d detik, seharusnya submitted & ±600", review.State, review.Result.DurationSeconds)
	}
	if q := review.Questions[0]; q.AnsweredAt == nil || *q.AnsweredAt != "2026-12-01T08:05:30+07:00" || q.TimeSpentSeconds == nil || *q.TimeSpentSeconds != 42 {
		t.Errorf("waktu soal 1: answered_at %v, time_spent %v", q.AnsweredAt, q.TimeSpentSeconds)
	}
	if q := review.Questions[1]; q.AnsweredAt != nil || q.TimeSpentSeconds != nil {
		t.Errorf("soal 2 tanpa data waktu seharusnya null: %v %v", q.AnsweredAt, q.TimeSpentSeconds)
	}
	var stored struct {
		Score           float64
		CorrectCount    int
		DurationSeconds int
		Answers         string
	}
	db.Raw(`SELECT score, correct_count, duration_seconds, answers::text AS answers FROM quiz_attempts WHERE quiz_id = ?`, quiz.ID).Scan(&stored)
	if stored.Score != 66.67 || stored.CorrectCount != 2 || stored.DurationSeconds != review.Result.DurationSeconds ||
		!strings.Contains(stored.Answers, `"is_correct": true`) || !strings.Contains(stored.Answers, `"time_spent_seconds": 42`) {
		t.Errorf("tersimpan: %+v", stored)
	}

	// Sudah submit: buka quiz lagi (tanpa kode, walaupun event sudah selesai) → langsung review.
	db.Exec(`UPDATE quizzes SET end_at = NOW() - INTERVAL '1 minute', start_at = NOW() - INTERVAL '4 hours' WHERE id = ?`, quiz.ID)
	code, body := openQuiz(r, student, quiz.ID, "")
	var redirected reviewData
	decodeJSONData(t, body, &redirected)
	if code != http.StatusOK || redirected.State != "submitted" || redirected.Result.Score != 66.67 || len(redirected.Questions) != 3 ||
		redirected.Questions[0].Explanation == "" {
		t.Errorf("buka setelah submit: status %d, body %s", code, body)
	}

	// Submit ke-2 → 409; review bisa dibuka ulang dengan isi yang sama.
	if res := sendRequest(r, http.MethodPost, submitURL, answers, student); res.Code != http.StatusConflict ||
		!strings.Contains(res.Body.String(), "Jawaban sudah dikumpulkan pada ") {
		t.Errorf("submit 2x: status %d, body %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+quiz.ID+"/review", "", student)
	var again reviewData
	decodeData(t, res, &again)
	if res.Code != http.StatusOK || again.Result.Score != 66.67 || len(again.Questions) != 3 || again.Questions[1].StudentAnswer != "0.75" {
		t.Errorf("review ulang: status %d, %+v", res.Code, again.Result)
	}

	// Lewat batas waktu (mulai 2 jam lalu, durasi 60 menit) → 409.
	if code, body := openQuiz(r, student, late.ID, fmt.Sprintf(`{"access_code":%q}`, late.AccessCode)); code != http.StatusOK {
		t.Fatalf("buka quiz kedua: status %d, body %s", code, body)
	}
	db.Exec(`UPDATE quiz_attempts SET started_at = NOW() - INTERVAL '2 hours' WHERE quiz_id = ?`, late.ID)
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes/"+late.ID+"/submit", `{"answers":[]}`, student)
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "Waktu mengerjakan sudah habis pada ") {
		t.Errorf("lewat batas waktu: status %d, body %s", res.Code, res.Body)
	}

	// Siswa kelas lain → 404.
	if res := sendRequest(r, http.MethodPost, submitURL, answers, studentToken(t, db, "GRADE_9")); res.Code != http.StatusNotFound {
		t.Errorf("kelas lain: status %d, seharusnya 404", res.Code)
	}
}

func TestSubmitDailyQuiz_NilaiDanReview(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	admin := adminToken(t, db)
	for i := 0; i < 5; i++ {
		createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	}
	student := studentToken(t, db, "GRADE_5")
	code, body := generateDaily(r, student, dailyBody(fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q}`, p.SubjectID, p.FieldID), "easy", 5))
	if code != http.StatusCreated {
		t.Fatalf("generate: status %d, body %s", code, body)
	}
	var generated struct {
		ID        string `json:"id"`
		Questions []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	decodeJSONData(t, body, &generated)
	base := "/api/v1/daily-quizzes/" + generated.ID

	// 3 benar dari 5 → 60.
	items := []string{}
	for i, q := range generated.Questions {
		items = append(items, fmt.Sprintf(`{"question_id":%q,"answer":"A","is_correct":%v}`, q.ID, i < 3))
	}
	answers := `{"answers":[` + strings.Join(items, ",") + `]}`

	if res := sendRequest(r, http.MethodPost, base+"/submit", answers, student); res.Code != http.StatusConflict ||
		!strings.Contains(res.Body.String(), "Daily Quiz belum dibuka") {
		t.Errorf("submit sebelum dibuka: status %d, body %s", res.Code, res.Body)
	}
	sendRequest(r, http.MethodPost, base+"/open", "", student)
	// Dibuka 5 menit lalu supaya lama mengerjakan bisa dicek.
	db.Exec(`UPDATE daily_quizzes SET opened_at = NOW() - INTERVAL '5 minutes' WHERE id = ?`, generated.ID)
	// Membuka lagi sebelum submit: masih soal (state in_progress).
	if res := sendRequest(r, http.MethodPost, base+"/open", "", student); !strings.Contains(res.Body.String(), `"state":"in_progress"`) {
		t.Errorf("buka ulang sebelum submit: body %s", res.Body)
	}

	res := sendRequest(r, http.MethodPost, base+"/submit", answers, student)
	if res.Code != http.StatusOK {
		t.Fatalf("submit: status %d. body: %s", res.Code, res.Body)
	}
	var review reviewData
	decodeData(t, res, &review)
	if review.State != "submitted" || review.Result.DurationSeconds < 295 || review.Result.DurationSeconds > 360 {
		t.Errorf("state %q, lama mengerjakan %d detik, seharusnya submitted & ±300", review.State, review.Result.DurationSeconds)
	}
	if review.Result.Score != 60 || review.Result.CorrectCount != 3 || review.Result.PointsPerQuestion != 20 || len(review.Questions) != 5 {
		t.Errorf("hasil: %+v", review.Result)
	}
	for i, q := range review.Questions {
		if q.ID != generated.Questions[i].ID || q.Explanation == "" || q.Answer["correct_key"] != "B" || q.IsCorrect != (i < 3) {
			t.Errorf("review soal %d: %+v", i+1, q)
		}
	}

	if res := sendRequest(r, http.MethodPost, base+"/submit", answers, student); res.Code != http.StatusConflict {
		t.Errorf("submit 2x: status %d, seharusnya 409", res.Code)
	}
	// Sudah submit: buka lagi → langsung review (tidak dihitung jatah, tidak potong coin).
	res = sendRequest(r, http.MethodPost, base+"/open", "", student)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"state":"submitted"`) || !strings.Contains(res.Body.String(), `"score":60`) {
		t.Errorf("buka setelah submit: status %d, body %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodGet, base+"/review", "", student)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"score":60`) {
		t.Errorf("review: status %d, body %s", res.Code, res.Body)
	}
	for _, tc := range []struct{ token, url string }{
		{studentToken(t, db, "GRADE_5"), base + "/review"},                 // milik siswa lain
		{student, "/api/v1/daily-quizzes/" + uuid.NewString() + "/review"}, // tidak ada
	} {
		if res := sendRequest(r, http.MethodGet, tc.url, "", tc.token); res.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, seharusnya 404", tc.url, res.Code)
		}
	}
}
