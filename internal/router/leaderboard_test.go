// Test leaderboard: Daily Quiz per bulan (filter jenjang/kelas) dan per quiz Uji Kom / Tryout.
package router

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// namedStudent membuat siswa bernama (real_name = name) di kelas tertentu, mengembalikan id & token.
func namedStudent(t *testing.T, db *gorm.DB, name, gradeCode string) (string, string) {
	t.Helper()
	id := createUser(t, db, loginmodel.RoleStudent, "siswa."+uuid.NewString()[:8], loginmodel.UserStatusActive)
	db.Exec(`
		UPDATE users SET real_name = ?, grade_id = g.id, education_level_id = g.parent_id
		FROM master_data_items g WHERE g.type = 'grade' AND g.code = ? AND users.id = ?`, name, gradeCode, id)
	accessToken, err := token.NewAccessToken(id, loginmodel.RoleStudent, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return id, accessToken
}

// submittedDaily menambah point 1 Daily Quiz yang sudah di-submit ke leaderboard_entries (langsung
// di database, sama seperti yang dilakukan submit: baris siswa per bulan dijumlahkan).
// submittedAt = ekspresi SQL waktu submit, contoh NOW().
func submittedDaily(t *testing.T, db *gorm.DB, studentID string, points, duration int, submittedAt string) {
	t.Helper()
	err := db.Exec(`
		INSERT INTO leaderboard_entries (board, period, student_id, education_level_id, grade_id,
			points, duration_seconds, quiz_count, last_submitted_at)
		SELECT 'daily_quiz', to_char(`+submittedAt+` AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM'), u.id,
		       u.education_level_id, u.grade_id, ?, ?, 1, `+submittedAt+`
		FROM users u WHERE u.id = ?
		ON CONFLICT ON CONSTRAINT leaderboard_entries_unique DO UPDATE SET
			points = leaderboard_entries.points + EXCLUDED.points,
			duration_seconds = leaderboard_entries.duration_seconds + EXCLUDED.duration_seconds,
			quiz_count = leaderboard_entries.quiz_count + 1`,
		points, duration, studentID).Error
	if err != nil {
		t.Fatal(err)
	}
}

type boardEntry struct {
	Rank    int `json:"rank"`
	Student struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"student"`
	Grade           *listRef `json:"grade"`
	Points          int      `json:"points"`
	DurationSeconds int      `json:"duration_seconds"`
	QuizCount       *int     `json:"quiz_count"`
	Score           *float64 `json:"score"`
}

type board struct {
	Month   string       `json:"month"`
	Entries []boardEntry `json:"entries"`
	Me      *boardEntry  `json:"me"`
}

func getBoard(t *testing.T, r *gin.Engine, accessToken, url string) (board, listMeta) {
	t.Helper()
	res := sendRequest(r, http.MethodGet, url, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d. body: %s", url, res.Code, res.Body)
	}
	var body struct {
		Data board    `json:"data"`
		Meta listMeta `json:"meta"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response bukan JSON: %v", err)
	}
	return body.Data, body.Meta
}

// ranking mengubah entries menjadi "rank:nama" supaya mudah dibandingkan.
func ranking(entries []boardEntry) string {
	parts := []string{}
	for _, e := range entries {
		parts = append(parts, strconv.Itoa(e.Rank)+":"+e.Student.Name)
	}
	return strings.Join(parts, ",")
}

func TestLeaderboard_DailyQuizPerBulan(t *testing.T) {
	r, db := setupRouter(t)
	p5 := seedPlacementInGrade(t, db, "GRADE_5")

	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_5")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_5")
	dodi, _ := namedStudent(t, db, "Dodi", "GRADE_4")
	eko, _ := namedStudent(t, db, "Eko", "GRADE_7")
	fani, _ := namedStudent(t, db, "Fani", "GRADE_5")

	now := `NOW()`
	lastMonth := `((date_trunc('month', NOW() AT TIME ZONE 'Asia/Jakarta') - INTERVAL '1 day') AT TIME ZONE 'Asia/Jakarta')`
	// Bulan ini: Ani 2 quiz (30+20 = 50 point, 200 detik); Budi 50 point tapi lebih cepat (150);
	// Cici sama persis dengan Budi (peringkat sama); Dodi (Kelas 4) 60; Eko (SMP) 10.
	submittedDaily(t, db, ani, 30, 100, now)
	submittedDaily(t, db, ani, 20, 100, now)
	submittedDaily(t, db, budi, 50, 150, now)
	submittedDaily(t, db, cici, 50, 150, now)
	submittedDaily(t, db, dodi, 60, 500, now)
	submittedDaily(t, db, eko, 10, 50, now)
	// Bulan lalu: Fani 100 point → tidak ikut bulan ini.
	submittedDaily(t, db, fani, 100, 10, lastMonth)

	admin := adminToken(t, db)
	all, meta := getBoard(t, r, admin, "/api/v1/leaderboards/daily-quiz")
	if got := ranking(all.Entries); got != "1:Dodi,2:Budi,2:Cici,4:Ani,5:Eko" || meta.Total != 5 || all.Me != nil {
		t.Errorf("semua: %s (total %d, me %v)", got, meta.Total, all.Me)
	}
	if e := all.Entries[3]; e.Points != 50 || e.DurationSeconds != 200 || e.QuizCount == nil || *e.QuizCount != 2 || e.Grade == nil || e.Grade.Name != "Kelas 5" {
		t.Errorf("baris Ani: %+v", e)
	}
	if all.Month != time.Now().In(clock.Jakarta).Format("2006-01") {
		t.Errorf("month default = %s, seharusnya bulan ini", all.Month)
	}

	// Filter jenjang SD & kelas 5: peringkat dihitung ulang di dalam kelompoknya.
	sd, _ := getBoard(t, r, admin, "/api/v1/leaderboards/daily-quiz?"+url.Values{"education_level_id": {p5.LevelID}}.Encode())
	if got := ranking(sd.Entries); got != "1:Dodi,2:Budi,2:Cici,4:Ani" {
		t.Errorf("jenjang SD: %s", got)
	}
	grade5, _ := getBoard(t, r, aniToken, "/api/v1/leaderboards/daily-quiz?"+url.Values{"grade_id": {p5.GradeID}}.Encode())
	if got := ranking(grade5.Entries); got != "1:Budi,1:Cici,3:Ani" || grade5.Me == nil || grade5.Me.Rank != 3 || grade5.Me.Student.Name != "Ani" {
		t.Errorf("kelas 5: %s, me %+v", got, grade5.Me)
	}

	// Bulan lalu: hanya Fani.
	prev := time.Now().In(clock.Jakarta).AddDate(0, 0, -time.Now().In(clock.Jakarta).Day()).Format("2006-01")
	last, _ := getBoard(t, r, admin, "/api/v1/leaderboards/daily-quiz?month="+prev)
	if got := ranking(last.Entries); got != "1:Fani" {
		t.Errorf("bulan %s: %s", prev, got)
	}

	// Pagination & validasi.
	page2, meta := getBoard(t, r, admin, "/api/v1/leaderboards/daily-quiz?page=2&per_page=2")
	if got := ranking(page2.Entries); got != "2:Cici,4:Ani" || meta.Total != 5 {
		t.Errorf("halaman 2: %s total %d", got, meta.Total)
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/leaderboards/daily-quiz?month=2026-13", "", admin)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["month"] != "format tidak valid, contoh 2026-11" {
		t.Errorf("month salah: status %d, body %s", res.Code, res.Body)
	}
}

func TestLeaderboard_PerQuizUjiKomTryout(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	q := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	ujiKom := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"status": `"published"`}))
	tryout := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"status": `"published"`, "type": `"tryout"`}))
	draft := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, nil))

	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_6")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_6")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_6")
	// attempt = hasil submit 1 siswa di 1 quiz, langsung ke leaderboard_entries (belum submit = tidak ada baris).
	attempt := func(quizID, studentID string, points, duration int, submitted bool) {
		t.Helper()
		if !submitted {
			return
		}
		err := db.Exec(`
			INSERT INTO leaderboard_entries (board, quiz_id, student_id, education_level_id, grade_id,
				points, duration_seconds, quiz_count, score, last_submitted_at)
			SELECT z.type, z.id, u.id, u.education_level_id, u.grade_id, ?, ?, 1, 100, NOW()
			FROM quizzes z, users u WHERE z.id = ? AND u.id = ?`, points, duration, quizID, studentID).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	attempt(ujiKom.ID, ani, 20, 300, true)
	attempt(ujiKom.ID, budi, 20, 120, true) // point sama, lebih cepat → di atas Ani
	attempt(ujiKom.ID, cici, 0, 0, false)   // belum submit → tidak ikut
	attempt(tryout.ID, cici, 15, 60, true)  // leaderboard Tryout terpisah

	b, meta := getBoard(t, r, aniToken, "/api/v1/leaderboards/quizzes/"+ujiKom.ID)
	if got := ranking(b.Entries); got != "1:Budi,2:Ani" || meta.Total != 2 || b.Me == nil || b.Me.Rank != 2 ||
		b.Entries[0].Score == nil || *b.Entries[0].Score != 100 || b.Entries[0].QuizCount != nil {
		t.Errorf("uji kom: %s total %d me %+v", got, meta.Total, b.Me)
	}
	if b2, _ := getBoard(t, r, admin, "/api/v1/leaderboards/quizzes/"+tryout.ID); ranking(b2.Entries) != "1:Cici" {
		t.Errorf("tryout: %s", ranking(b2.Entries))
	}

	// Siswa: hanya quiz publish untuk kelasnya. Tutor/admin: semua (termasuk draf).
	for _, tc := range []struct {
		token, id string
		want      int
	}{
		{aniToken, draft.ID, http.StatusNotFound},
		{studentToken(t, db, "GRADE_9"), ujiKom.ID, http.StatusNotFound},
		{aniToken, uuid.NewString(), http.StatusNotFound},
		{tokenFor(t, loginmodel.RoleTutor), draft.ID, http.StatusOK},
		{admin, ujiKom.ID, http.StatusOK},
	} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/leaderboards/quizzes/"+tc.id, "", tc.token); res.Code != tc.want {
			t.Errorf("quiz %s: status %d, seharusnya %d", tc.id, res.Code, tc.want)
		}
	}
}

// Alur asli: generate → buka → submit Daily Quiz 2x → baris leaderboard bulan ini dijumlahkan dan
// setiap submit tercatat di riwayat (leaderboard_point_logs). Uji Kom: 1 baris per quiz.
func TestLeaderboard_TercatatSaatSubmit(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	admin := adminToken(t, db)
	for i := 0; i < 5; i++ { // 5 soal easy (10 point per soal benar)
		createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	}
	studentID, student := namedStudent(t, db, "Ani", "GRADE_5")
	placement := `{"subject_id":"` + p.SubjectID + `","subject_field_id":"` + p.FieldID + `"}`

	dailyIDs := []string{}
	for round, correct := range []int{3, 5} { // submit ke-1: 3 benar (30 point), ke-2: 5 benar (50 point)
		code, body := generateDaily(r, student, dailyBody(placement, "easy", 5))
		if code != http.StatusCreated {
			t.Fatalf("generate %d: status %d", round, code)
		}
		var generated struct {
			ID        string `json:"id"`
			Questions []struct {
				ID string `json:"id"`
			} `json:"questions"`
		}
		decodeJSONData(t, body, &generated)
		dailyIDs = append(dailyIDs, generated.ID)
		sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+generated.ID+"/open", "", student)
		items := []string{}
		for i, q := range generated.Questions {
			items = append(items, `{"question_id":"`+q.ID+`","answer":"B","is_correct":`+strconv.FormatBool(i < correct)+`}`)
		}
		res := sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+generated.ID+"/submit", `{"answers":[`+strings.Join(items, ",")+`]}`, student)
		if res.Code != http.StatusOK {
			t.Fatalf("submit %d: status %d. body: %s", round, res.Code, res.Body)
		}
	}

	// 1 baris per siswa per bulan: point 30 + 50, 2 quiz, snapshot kelas siswa.
	var entry struct {
		ID, Period, GradeID string
		Points, QuizCount   int
	}
	db.Raw(`SELECT id, period, grade_id, points, quiz_count FROM leaderboard_entries WHERE board = 'daily_quiz' AND student_id = ?`, studentID).Scan(&entry)
	if entry.Points != 80 || entry.QuizCount != 2 || entry.GradeID != p.GradeID ||
		entry.Period != time.Now().In(clock.Jakarta).Format("2006-01") {
		t.Errorf("leaderboard_entries: %+v", entry)
	}

	// Riwayat: 1 log per submit, menunjuk ke Daily Quiz asalnya.
	var logs []struct {
		SourceType, SourceID string
		Points               int
	}
	db.Raw(`SELECT source_type, source_id, points FROM leaderboard_point_logs WHERE entry_id = ? ORDER BY created_at`, entry.ID).Scan(&logs)
	if len(logs) != 2 || logs[0].SourceType != "daily_quiz" || logs[0].SourceID != dailyIDs[0] || logs[0].Points != 30 ||
		logs[1].SourceID != dailyIDs[1] || logs[1].Points != 50 {
		t.Errorf("leaderboard_point_logs: %+v", logs)
	}

	// API leaderboard membaca dari tabel ini.
	b, _ := getBoard(t, r, student, "/api/v1/leaderboards/daily-quiz")
	if b.Me == nil || b.Me.Rank != 1 || b.Me.Points != 80 || b.Me.QuizCount == nil || *b.Me.QuizCount != 2 {
		t.Errorf("leaderboard: me %+v", b.Me)
	}
}
