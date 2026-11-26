// Test halaman siswa: Uji Kom & Tryout ("Akan datang", "Sudah dikerjakan", statistik) dan riwayat &
// statistik Daily Quiz.
package router

import (
	"net/http"
	"strings"
	"testing"
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

func TestStudentQuizPages_AkanDatangSudahDikerjakanStatistik(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_9")
	opts := map[string]string{"used_for": `["uji_kom","tryout"]`}
	q1 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, opts))
	q2 := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, opts))
	q9 := createQuestionID(t, r, admin, questionBody(other, "single_choice", singleChoiceAnswer, opts))
	ids := []string{q1, q2}
	pub := func(typ, name string) string {
		return createQuiz(t, r, admin, quizBody(basePlacement(p), ids, map[string]string{"status": `"published"`, "type": `"` + typ + `"`, "name": `"` + name + `"`})).ID
	}
	ongoing := pub("uji_kom", "Uji Kom Pecahan")
	upcoming := pub("tryout", "Tryout Akhir Semester")
	doneTryout := pub("tryout", "Tryout Bulan Lalu")
	doneUji := pub("uji_kom", "Uji Kom Lama")
	createQuiz(t, r, admin, quizBody(basePlacement(p), ids, map[string]string{"name": `"Draf"`}))                                      // draf: tidak tampil
	createQuiz(t, r, admin, quizBody(basePlacement(other), []string{q9}, map[string]string{"status": `"published"`, "name": `"SMP"`})) // kelas lain
	// Jadwal tidak boleh bertumpuk (mapel & kelas sama), jadi dipisah.
	for id, window := range map[string]string{
		ongoing:    "NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour'",
		upcoming:   "NOW() + INTERVAL '2 days', NOW() + INTERVAL '2 days 2 hours'",
		doneTryout: "NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days' + INTERVAL '2 hours'",
		doneUji:    "NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days' + INTERVAL '2 hours'",
	} {
		parts := strings.Split(window, ", ")
		if err := db.Exec(`UPDATE quizzes SET start_at = `+parts[0]+`, end_at = `+parts[1]+` WHERE id = ?`, id).Error; err != nil {
			t.Fatal(err)
		}
	}

	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_6")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_6")
	// Ani sudah membuka Uji Kom yang sedang berjalan (belum dikumpulkan).
	db.Exec(`INSERT INTO quiz_attempts (quiz_id, student_id, started_at) VALUES (?, ?, NOW() - INTERVAL '10 minutes')`, ongoing, ani)
	// Tryout bulan lalu: Ani #1 dari 2; Uji Kom lama: Ani #2 dari 2.
	attemptAt(t, db, doneTryout, ani, ids, "TT", 100, "NOW() - INTERVAL '10 days' + INTERVAL '30 minutes'", 20)
	attemptAt(t, db, doneTryout, budi, ids, "TF", 50, "NOW() - INTERVAL '10 days' + INTERVAL '40 minutes'", 30)
	attemptAt(t, db, doneUji, ani, ids, "TF", 50, "NOW() - INTERVAL '5 days' + INTERVAL '30 minutes'", 20)
	attemptAt(t, db, doneUji, budi, ids, "TT", 100, "NOW() - INTERVAL '5 days' + INTERVAL '30 minutes'", 20)
	db.Exec(`UPDATE quiz_attempts SET points = ROUND(score)::int WHERE submitted_at IS NOT NULL`)

	// ---- Akan datang ----
	var up []struct {
		ID                string    `json:"id"`
		Type              string    `json:"type"`
		ScheduleStatus    string    `json:"schedule_status"`
		Locked            bool      `json:"locked"`
		SecondsUntilStart int64     `json:"seconds_until_start"`
		MyStatus          string    `json:"my_status"`
		StartedAt         *string   `json:"started_at"`
		StartAt           time.Time `json:"start_at"`
		Subject           listRef   `json:"subject"`
		TotalQuestions    int       `json:"total_questions"`
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/upcoming", "", aniToken)
	decodeData(t, res, &up)
	if res.Code != http.StatusOK || len(up) != 2 || up[0].ID != ongoing || up[1].ID != upcoming {
		t.Fatalf("akan datang: %d %s", res.Code, res.Body)
	}
	if u := up[0]; u.ScheduleStatus != "ongoing" || u.Locked || u.MyStatus != "in_progress" || u.StartedAt == nil || u.TotalQuestions != 2 || u.Subject.Name == "" {
		t.Errorf("quiz berjalan: %+v", u)
	}
	if u := up[1]; u.ScheduleStatus != "upcoming" || !u.Locked || u.MyStatus != "not_started" || u.SecondsUntilStart < 47*3600 || u.SecondsUntilStart > 48*3600 {
		t.Errorf("quiz terkunci: %+v", u)
	}
	for query, want := range map[string]int{"?type=tryout": 1, "?search=pecahan": 1, "?subject_id=" + other.SubjectID: 0} {
		var items []map[string]any
		decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/upcoming"+query, "", aniToken), &items)
		if len(items) != want {
			t.Errorf("akan datang %s: %d, seharusnya %d", query, len(items), want)
		}
	}

	// ---- Sudah dikerjakan ----
	var done []struct {
		QuizID       string  `json:"quiz_id"`
		Type         string  `json:"type"`
		Score        float64 `json:"score"`
		CorrectCount int     `json:"correct_count"`
		Points       int     `json:"points"`
		Rank         struct {
			Rank, Total int
		} `json:"rank"`
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/history", "", aniToken)
	decodeData(t, res, &done)
	if res.Code != http.StatusOK || len(done) != 2 || done[0].QuizID != doneUji || done[1].QuizID != doneTryout {
		t.Fatalf("sudah dikerjakan: %d %s", res.Code, res.Body)
	}
	if d := done[0]; d.Score != 50 || d.CorrectCount != 1 || d.Rank.Rank != 2 || d.Rank.Total != 2 {
		t.Errorf("uji kom lama: %+v", d)
	}
	if d := done[1]; d.Score != 100 || d.Points != 100 || d.Rank.Rank != 1 || d.Rank.Total != 2 {
		t.Errorf("tryout lama: %+v", d)
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/history?type=tryout", "", aniToken), &done)
	if len(done) != 1 {
		t.Errorf("sudah dikerjakan tab tryout: %d", len(done))
	}

	// ---- Statistik ----
	var stats struct {
		Sessions     int      `json:"sessions"`
		AverageScore *float64 `json:"average_score"`
		TotalPoints  int      `json:"total_points"`
		LastTryout   *struct {
			QuizID string `json:"quiz_id"`
			Rank   struct{ Rank, Total int }
		} `json:"last_tryout"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/stats", "", aniToken), &stats)
	if stats.Sessions != 2 || stats.AverageScore == nil || *stats.AverageScore != 75 || stats.TotalPoints != 150 ||
		stats.LastTryout == nil || stats.LastTryout.QuizID != doneTryout || stats.LastTryout.Rank.Rank != 1 {
		t.Errorf("statistik: %+v", stats)
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/stats?type=uji_kom", "", aniToken), &stats)
	if stats.Sessions != 1 || *stats.AverageScore != 50 || stats.LastTryout == nil {
		t.Errorf("statistik tab uji kom: %+v", stats)
	}

	// Siswa baru: kosong, tanpa Tryout terakhir.
	_, newToken := namedStudent(t, db, "Cici", "GRADE_6")
	res = sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/stats", "", newToken)
	if !strings.Contains(res.Body.String(), `"sessions":0`) || !strings.Contains(res.Body.String(), `"last_tryout":null`) || !strings.Contains(res.Body.String(), `"average_score":null`) {
		t.Errorf("statistik siswa baru: %s", res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/upcoming?type=daily_quiz", "", aniToken); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("type tidak valid: %d", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/quizzes/history", "", admin); res.Code != http.StatusForbidden {
		t.Errorf("admin membuka halaman siswa: %d", res.Code)
	}
}

func TestStudentDailyQuizPages_RiwayatDanStatistik(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_5")
	month := "TO_CHAR(NOW() AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')"

	// Bulan ini: 2 Daily Quiz Ani (easy 80, hard 60); bulan lalu: 1 (medium 40). Budi tidak ikut riwayat Ani.
	submittedDailyQuiz(t, db, ani, p, "easy", 5, 4, 80, 40, 100, "NOW() - INTERVAL '1 minute'")
	submittedDailyQuiz(t, db, ani, p, "hard", 5, 3, 60, 60, 200, "NOW()")
	submittedDailyQuiz(t, db, ani, p, "medium", 5, 2, 40, 30, 150, "date_trunc('month', NOW() AT TIME ZONE 'Asia/Jakarta') AT TIME ZONE 'Asia/Jakarta' - INTERVAL '2 days'")
	submittedDailyQuiz(t, db, budi, p, "easy", 5, 5, 100, 200, 100, "NOW()")
	// Daily Quiz yang belum dikumpulkan tidak masuk riwayat.
	db.Exec(`INSERT INTO daily_quizzes (student_id, education_level_id, grade_id, subject_id, subject_field_id, difficulty, total_questions)
		VALUES (?, ?, ?, ?, ?, 'easy', 5)`, ani, p.LevelID, p.GradeID, p.SubjectID, p.FieldID)
	// Leaderboard bulan ini (Kelas 5): Budi 200 point, Ani 100 → Ani #2 dari 2.
	for id, points := range map[string]int{ani: 100, budi: 200} {
		db.Exec(`INSERT INTO leaderboard_entries (board, period, student_id, education_level_id, grade_id, points, duration_seconds, quiz_count, last_submitted_at)
			SELECT 'daily_quiz', `+month+`, u.id, u.education_level_id, u.grade_id, ?, 100, 1, NOW() FROM users u WHERE u.id = ?`, points, id)
	}

	var items []struct {
		ID              string  `json:"id"`
		Subject         listRef `json:"subject"`
		SubjectField    listRef `json:"subject_field"`
		Difficulty      string  `json:"difficulty"`
		DifficultyLabel string  `json:"difficulty_label"`
		Score           float64 `json:"score"`
		Points          int     `json:"points"`
		DurationSeconds int     `json:"duration_seconds"`
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes", "", aniToken)
	decodeData(t, res, &items)
	meta := decodeMeta(t, res)
	if res.Code != http.StatusOK || len(items) != 3 || meta.Total != 3 || items[0].Difficulty != "hard" || items[0].DifficultyLabel != "Sulit" ||
		items[0].Score != 60 || items[0].Points != 60 || items[0].DurationSeconds != 200 || items[0].SubjectField.Name == "" {
		t.Fatalf("riwayat: %d %s", res.Code, res.Body)
	}
	for query, want := range map[string]int{"?difficulty=easy": 1, "?subject_id=" + p.SubjectID: 3, "?per_page=1&page=3": 1} {
		var got []map[string]any
		decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes"+query, "", aniToken), &got)
		if len(got) != want {
			t.Errorf("riwayat %s: %d, seharusnya %d", query, len(got), want)
		}
	}
	thisMonth := time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01")
	var monthItems []map[string]any
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes?month="+thisMonth, "", aniToken), &monthItems)
	if len(monthItems) != 2 {
		t.Errorf("riwayat bulan ini: %d", len(monthItems))
	}

	var stats struct {
		Month     string `json:"month"`
		ThisMonth struct {
			Quizzes      int      `json:"quizzes"`
			AverageScore *float64 `json:"average_score"`
			Points       int      `json:"points"`
		} `json:"this_month"`
		AllTime struct {
			Quizzes      int      `json:"quizzes"`
			AverageScore *float64 `json:"average_score"`
			Points       int      `json:"points"`
		} `json:"all_time"`
		Rank *struct{ Rank, Total int } `json:"rank"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes/stats", "", aniToken), &stats)
	if stats.Month != thisMonth || stats.ThisMonth.Quizzes != 2 || *stats.ThisMonth.AverageScore != 70 || stats.ThisMonth.Points != 100 ||
		stats.AllTime.Quizzes != 3 || *stats.AllTime.AverageScore != 60 || stats.AllTime.Points != 130 ||
		stats.Rank == nil || stats.Rank.Rank != 2 || stats.Rank.Total != 2 {
		t.Errorf("statistik Daily Quiz: %+v rank %+v", stats, stats.Rank)
	}
	// Bulan tanpa Daily Quiz: angka 0, rata-rata & peringkat null.
	res = sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes/stats?month=2020-01", "", aniToken)
	if !strings.Contains(res.Body.String(), `"this_month":{"quizzes":0,"average_score":null,"points":0}`) || !strings.Contains(res.Body.String(), `"rank":null`) {
		t.Errorf("statistik bulan kosong: %s", res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes?month=11-2026", "", aniToken); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("bulan salah format: %d", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/daily-quizzes", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor membuka riwayat siswa: %d", res.Code)
	}
}
