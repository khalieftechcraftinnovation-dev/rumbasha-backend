// Test buka quiz oleh siswa (Uji Kom/Tryout dengan kode akses, Daily Quiz dengan jatah gratis/coin)
// dan API kode akses untuk tutor/admin.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

type openedQuiz struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	StartedAt  time.Time `json:"started_at"`
	DeadlineAt time.Time `json:"deadline_at"`
	EndAt      time.Time `json:"end_at"`
	Questions  []struct {
		No     int            `json:"no"`
		ID     string         `json:"id"`
		Type   string         `json:"type"`
		Answer map[string]any `json:"answer"`
	} `json:"questions"`
}

func openQuiz(r *gin.Engine, accessToken, quizID, body string) (int, string) {
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes/"+quizID+"/open", body, accessToken)
	return res.Code, res.Body.String()
}

func TestOpenQuiz_KodeAksesDanJadwal(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6") // mapel lain, supaya jadwal tidak bentrok
	admin := adminToken(t, db)

	q1 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	q2 := createQuestionID(t, r, admin, questionBody(p, "free_text", freeTextNumberAnswer, nil))
	qOther := createQuestionID(t, r, admin, questionBody(other, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))

	published := map[string]string{"status": `"published"`}
	ongoing := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q2, q1}, mergeExtra(published, map[string]string{"name": `"Uji Kom Berjalan"`})))
	upcoming := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q1}, published))
	finished := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q1}, published))
	draft := createQuiz(t, r, admin, quizBody(basePlacement(other), []string{qOther}, nil))
	endingSoon := createQuiz(t, r, admin, quizBody(basePlacement(other), []string{qOther}, mergeExtra(published, map[string]string{"type": `"tryout"`})))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '30 minutes', end_at = NOW() + INTERVAL '2 hours' WHERE id = ?`, ongoing.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '3 hours', end_at = NOW() - INTERVAL '2 hours' WHERE id = ?`, finished.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '5 hours', end_at = NOW() - INTERVAL '4 hours' WHERE id = ?`, draft.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '50 minutes', end_at = NOW() + INTERVAL '20 minutes' WHERE id = ?`, endingSoon.ID)

	student := studentToken(t, db, "GRADE_6")

	// Pertama kali: kode akses wajib & harus benar.
	for body, want := range map[string]string{
		``:                         "wajib diisi",
		`{}`:                       "wajib diisi",
		`{"access_code":"ZZZZZZ"}`: "kode akses salah atau sudah berganti, minta kode terbaru ke tutor",
	} {
		code, res := openQuiz(r, student, ongoing.ID, body)
		if code != http.StatusUnprocessableEntity || errorFields(t, res)["access_code"] != want {
			t.Errorf("body %q: status %d, body %s, seharusnya access_code %q", body, code, res, want)
		}
	}

	// Kode benar (huruf kecil juga diterima): soal lengkap + kunci jawaban, tanpa pembahasan.
	code, res := openQuiz(r, student, ongoing.ID, fmt.Sprintf(`{"access_code":%q}`, strings.ToLower(ongoing.AccessCode)))
	if code != http.StatusOK {
		t.Fatalf("buka quiz: status %d. body: %s", code, res)
	}
	if strings.Contains(res, "explanation") || strings.Contains(res, "Samakan penyebut") || strings.Contains(res, "access_code") {
		t.Errorf("buka quiz tidak boleh berisi pembahasan / kode akses. body: %s", res)
	}
	var opened openedQuiz
	decodeJSONData(t, res, &opened)
	if len(opened.Questions) != 2 || opened.Questions[0].ID != q2 || opened.Questions[1].ID != q1 ||
		opened.Questions[1].Answer["correct_key"] != "B" || opened.Questions[0].Answer["input_type"] != "number" {
		t.Errorf("soal + kunci jawaban salah: %+v", opened.Questions)
	}
	if got := opened.DeadlineAt.Sub(opened.StartedAt); got != time.Hour {
		t.Errorf("batas mengerjakan = mulai + %s, seharusnya + 1 jam (durasi)", got)
	}

	// Buka ulang tanpa kode (contoh refresh halaman setelah kode berganti): waktu mulai tetap.
	db.Exec(`UPDATE quizzes SET access_code = 'AAAAAA' WHERE id = ?`, ongoing.ID)
	code, res = openQuiz(r, student, ongoing.ID, "")
	var reopened openedQuiz
	decodeJSONData(t, res, &reopened)
	if code != http.StatusOK || !reopened.StartedAt.Equal(opened.StartedAt) {
		t.Errorf("buka ulang: status %d, started_at %s, seharusnya tetap %s", code, reopened.StartedAt, opened.StartedAt)
	}
	var attempts int
	db.Raw(`SELECT COUNT(*) FROM quiz_attempts WHERE quiz_id = ?`, ongoing.ID).Scan(&attempts)
	if attempts != 1 {
		t.Errorf("catatan buka quiz = %d, seharusnya 1", attempts)
	}

	// Batas mengerjakan tidak melewati akhir event.
	code, res = openQuiz(r, student, endingSoon.ID, fmt.Sprintf(`{"access_code":%q}`, endingSoon.AccessCode))
	var soon openedQuiz
	decodeJSONData(t, res, &soon)
	if code != http.StatusOK || !soon.DeadlineAt.Equal(soon.EndAt) {
		t.Errorf("quiz hampir selesai: status %d, deadline %s, seharusnya = end_at %s", code, soon.DeadlineAt, soon.EndAt)
	}

	// Di luar jadwal → 409; draf / kelas lain / tidak ada → 404; bukan siswa → 403.
	for id, want := range map[string]string{upcoming.ID: "Quiz belum dimulai. Mulai ", finished.ID: "Quiz sudah selesai pada "} {
		code, res := openQuiz(r, student, id, fmt.Sprintf(`{"access_code":"%s"}`, upcoming.AccessCode))
		if code != http.StatusConflict || !strings.Contains(res, want) {
			t.Errorf("di luar jadwal: status %d, body %s, seharusnya 409 %q", code, res, want)
		}
	}
	for _, tc := range []struct{ token, id string }{
		{student, draft.ID}, {studentToken(t, db, "GRADE_9"), ongoing.ID}, {student, uuid.NewString()}, {student, "bukan-uuid"},
	} {
		if code, res := openQuiz(r, tc.token, tc.id, `{"access_code":"AAAAAA"}`); code != http.StatusNotFound {
			t.Errorf("quiz %s: status %d, seharusnya 404. body %s", tc.id, code, res)
		}
	}
	if code, _ := openQuiz(r, admin, ongoing.ID, `{"access_code":"AAAAAA"}`); code != http.StatusForbidden {
		t.Errorf("admin buka quiz: status %d, seharusnya 403", code)
	}
}

func TestQuizAccessCode_TutorAdminSuperAdmin(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	q := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	quiz := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, nil))

	for _, role := range []string{loginmodel.RoleTutor, loginmodel.RoleAdmin, loginmodel.RoleSuperAdmin} {
		res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+quiz.ID+"/access-code", "", tokenFor(t, role))
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"access_code":"`+quiz.AccessCode+`"`) ||
			!strings.Contains(res.Body.String(), `"schedule_status":"upcoming"`) || !strings.Contains(res.Body.String(), `"access_code_updated_at":"`) {
			t.Errorf("%s: status %d, body %s", role, res.Code, res.Body)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+quiz.ID+"/access-code", "", tokenFor(t, loginmodel.RoleStudent)); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status %d, seharusnya 403", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+uuid.NewString()+"/access-code", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusNotFound {
		t.Errorf("quiz tidak ada: status %d, seharusnya 404", res.Code)
	}
}

type openedDaily struct {
	ID    string `json:"id"`
	Usage struct {
		FreePerDay    int  `json:"free_per_day"`
		OpenedToday   int  `json:"opened_today"`
		CoinCharged   int  `json:"coin_charged"`
		CoinBalance   int  `json:"coin_balance"`
		AlreadyOpened bool `json:"already_opened"`
	} `json:"usage"`
	Questions []struct {
		Answer map[string]any `json:"answer"`
	} `json:"questions"`
}

func TestOpenDailyQuiz_JatahGratisDanCoin(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	admin := adminToken(t, db)
	for i := 0; i < 5; i++ {
		createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	}
	student := studentToken(t, db, "GRADE_5")
	db.Exec(`UPDATE students SET coin_balance = 15 WHERE user_id = (SELECT id FROM users WHERE role = 'student' ORDER BY created_at DESC LIMIT 1)`)

	placement := fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q}`, p.SubjectID, p.FieldID)
	ids := make([]string, 4)
	for i := range ids {
		code, body := generateDaily(r, student, dailyBody(placement, "easy", 5))
		if code != http.StatusCreated {
			t.Fatalf("generate: status %d. body: %s", code, body)
		}
		var d struct {
			ID string `json:"id"`
		}
		decodeJSONData(t, body, &d)
		ids[i] = d.ID
	}

	open := func(id string) (int, openedDaily, string) {
		res := sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+id+"/open", "", student)
		var d openedDaily
		if res.Code == http.StatusOK {
			decodeJSONData(t, res.Body.String(), &d)
		}
		return res.Code, d, res.Body.String()
	}

	// Buka ke-1 & ke-2 hari ini: gratis. Buka ulang tidak dihitung.
	code, d, body := open(ids[0])
	if code != http.StatusOK || d.Usage.CoinCharged != 0 || d.Usage.OpenedToday != 1 || d.Usage.FreePerDay != 2 || d.Usage.CoinBalance != 15 {
		t.Fatalf("buka ke-1: status %d, usage %+v. body: %s", code, d.Usage, body)
	}
	if len(d.Questions) != 5 || d.Questions[0].Answer["correct_key"] != "B" || strings.Contains(body, "explanation") {
		t.Errorf("soal Daily Quiz yang dibuka harus berisi kunci jawaban tanpa pembahasan. body: %s", body)
	}
	if _, d, _ = open(ids[0]); !d.Usage.AlreadyOpened || d.Usage.OpenedToday != 1 || d.Usage.CoinCharged != 0 {
		t.Errorf("buka ulang: %+v, seharusnya already_opened & tidak dihitung", d.Usage)
	}
	if _, d, _ = open(ids[1]); d.Usage.CoinCharged != 0 || d.Usage.OpenedToday != 2 {
		t.Errorf("buka ke-2: %+v, seharusnya masih gratis", d.Usage)
	}

	// Buka ke-3: jatah habis → potong 10 coin + tercatat di coin_transactions.
	if _, d, _ = open(ids[2]); d.Usage.CoinCharged != 10 || d.Usage.CoinBalance != 5 || d.Usage.OpenedToday != 3 {
		t.Errorf("buka ke-3: %+v, seharusnya potong 10 coin, saldo 5", d.Usage)
	}
	var trx struct {
		Amount, BalanceAfter int
		ReferenceType        string
		ReferenceID          string
	}
	db.Raw(`SELECT amount, balance_after, reference_type, reference_id FROM coin_transactions WHERE type = 'usage'`).Scan(&trx)
	if trx.Amount != -10 || trx.BalanceAfter != 5 || trx.ReferenceType != "daily_quiz" || trx.ReferenceID != ids[2] {
		t.Errorf("transaksi coin: %+v", trx)
	}

	// Buka ke-4: coin kurang → 402, tidak ada yang berubah.
	code, _, body = open(ids[3])
	if code != http.StatusPaymentRequired || !strings.Contains(body, "butuh 10 coin, saldo 5 coin") {
		t.Errorf("coin kurang: status %d, body %s", code, body)
	}
	var balance int
	var opened bool
	db.Raw(`SELECT coin_balance FROM students WHERE coin_balance IS NOT NULL ORDER BY created_at DESC LIMIT 1`).Scan(&balance)
	db.Raw(`SELECT opened_at IS NOT NULL FROM daily_quizzes WHERE id = ?`, ids[3]).Scan(&opened)
	if balance != 5 || opened {
		t.Errorf("setelah gagal: saldo %d, dibuka %v; seharusnya saldo tetap 5 & belum dibuka", balance, opened)
	}

	// Hari berganti: yang dibuka kemarin tidak dihitung → gratis lagi.
	db.Exec(`UPDATE daily_quizzes SET opened_at = opened_at - INTERVAL '1 day' WHERE opened_at IS NOT NULL`)
	if code, d, _ = open(ids[3]); code != http.StatusOK || d.Usage.CoinCharged != 0 || d.Usage.OpenedToday != 1 {
		t.Errorf("hari berikutnya: status %d, %+v, seharusnya gratis", code, d.Usage)
	}

	// Admin mengubah pengaturan global: tanpa jatah gratis, 3 coin per Daily Quiz → langsung berlaku.
	res := sendRequest(r, http.MethodPut, "/api/v1/daily-quiz-settings",
		`{"points":{"easy":10,"medium":15,"hard":20},"prize_coins":{"rank_1":0,"rank_2":0,"rank_3":0},"free_per_day":0,"coin_cost":3}`, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah pengaturan: status %d. body: %s", res.Code, res.Body)
	}
	code, body = generateDaily(r, student, dailyBody(placement, "easy", 5))
	if code != http.StatusCreated {
		t.Fatalf("generate: status %d", code)
	}
	var extra struct {
		ID string `json:"id"`
	}
	decodeJSONData(t, body, &extra)
	if code, d, body = open(extra.ID); code != http.StatusOK || d.Usage.CoinCharged != 3 || d.Usage.FreePerDay != 0 || d.Usage.CoinBalance != 2 {
		t.Errorf("setelah pengaturan diubah: status %d, %+v, seharusnya potong 3 coin, saldo 2. body: %s", code, d.Usage, body)
	}

	// Daily Quiz siswa lain / tidak ada → 404; bukan siswa → 403.
	for _, tc := range []struct {
		token, id string
		want      int
	}{
		{studentToken(t, db, "GRADE_5"), ids[0], http.StatusNotFound},
		{student, uuid.NewString(), http.StatusNotFound},
		{admin, ids[0], http.StatusForbidden},
	} {
		if res := sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+tc.id+"/open", "", tc.token); res.Code != tc.want {
			t.Errorf("buka %s: status %d, seharusnya %d", tc.id, res.Code, tc.want)
		}
	}
}
