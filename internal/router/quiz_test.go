// Test endpoint POST /quizzes (buat quiz Uji Kom / Tryout), bundle soal, kode akses, dan
// penanda "bundle" di list Bank Soal.
package router

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
)

var accessCodeFormat = regexp.MustCompile(`^[A-Z0-9]{6}$`)

// daysFromNowWIB = N hari lagi jam 08:00 WIB, format input "2006-01-02T15:04" (dianggap WIB).
func daysFromNowWIB(days int) (string, time.Time) {
	d := time.Now().In(clock.Jakarta).AddDate(0, 0, days)
	start := time.Date(d.Year(), d.Month(), d.Day(), 8, 0, 0, 0, clock.Jakarta)
	return start.Format("2006-01-02T15:04"), start
}

// quizDay = hari jadwal quiz berikutnya untuk quizBody. Setiap quiz mendapat hari berbeda
// (mulai 10 hari lagi), supaya quiz dengan mapel & kelas sama di 1 test tidak bentrok jadwal.
var quizDay atomic.Int64

// nextSchedule mengembalikan start_at & end_at (JSON) untuk quiz berikutnya: 08:00 - 10:00 WIB.
func nextSchedule() (startAt, endAt string) {
	start, _ := daysFromNowWIB(10 + int(quizDay.Add(1)))
	return fmt.Sprintf("%q", start), fmt.Sprintf("%q", start[:11]+"10:00")
}

// quizBody membuat body POST /quizzes. placement berisi JSON mentah; extra menimpa field lain
// (nilai JSON mentah; "" = field dihapus).
func quizBody(placement string, questionIDs []string, extra map[string]string) string {
	ids := make([]string, 0, len(questionIDs))
	for _, id := range questionIDs {
		ids = append(ids, fmt.Sprintf("%q", id))
	}
	startAt, endAt := nextSchedule()
	fields := map[string]string{
		"name":             `"Uji Kompetensi Matematika - KPK"`,
		"type":             `"uji_kom"`,
		"status":           `"draft"`,
		"placement":        placement,
		"start_at":         startAt,
		"end_at":           endAt,
		"duration_minutes": "60",
		"total_questions":  fmt.Sprint(len(questionIDs)),
		"question_ids":     "[" + strings.Join(ids, ",") + "]",
	}
	for k, v := range extra {
		fields[k] = v
	}
	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		if v != "" {
			parts = append(parts, fmt.Sprintf("%q:%s", k, v))
		}
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// basePlacement = letak quiz minimal: jenjang, kelas, mapel.
func basePlacement(p testPlacement) string {
	return fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q}`, p.LevelID, p.GradeID, p.SubjectID)
}

// jsonIDs mengubah daftar ID menjadi array JSON, contoh ["a","b"].
func jsonIDs(ids ...string) string {
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, fmt.Sprintf("%q", id))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

type quizDetail struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	StartAt         time.Time `json:"start_at"`
	EndAt           time.Time `json:"end_at"`
	ScheduleStatus  string    `json:"schedule_status"`
	DurationMinutes int       `json:"duration_minutes"`
	AccessCode      string    `json:"access_code"`
	CoinCost        int       `json:"coin_cost"`
	TotalQuestions  int       `json:"total_questions"`
	Status          string    `json:"status"`
	Placement       struct {
		Grade        listRef  `json:"grade"`
		Subject      listRef  `json:"subject"`
		SubjectField *listRef `json:"subject_field"`
		Chapter      *listRef `json:"chapter"`
	} `json:"placement"`
	Questions []struct {
		No   int    `json:"no"`
		ID   string `json:"id"`
		Code string `json:"code"`
	} `json:"questions"`
	CreatedBy listRef  `json:"created_by"`
	UpdatedBy *listRef `json:"updated_by"`
}

// createQuiz membuat quiz lewat API (harus 201) lalu mengembalikan detailnya.
func createQuiz(t *testing.T, r *gin.Engine, accessToken, body string) quizDetail {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", body, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("buat quiz: status = %d, seharusnya 201. body: %s", res.Code, res.Body)
	}
	var quiz quizDetail
	decodeData(t, res, &quiz)
	return quiz
}

// questionIDsOf mengambil ID soal quiz (urut nomor soal).
func questionIDsOf(q quizDetail) []string {
	ids := make([]string, 0, len(q.Questions))
	for _, item := range q.Questions {
		ids = append(ids, item.ID)
	}
	return ids
}

// questionCode mengambil kode soal langsung dari database.
func questionCode(t *testing.T, r *gin.Engine, accessToken, id string) string {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", accessToken)
	var q struct {
		Code string `json:"code"`
	}
	decodeData(t, res, &q)
	return q.Code
}

func TestCreateQuiz_BerhasilDenganUrutanSoalJadwalDanKodeAkses(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	q1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	q2 := createQuestionID(t, r, accessToken, questionBody(p, "free_text", freeTextNumberAnswer, nil))
	q3 := createQuestionID(t, r, accessToken, questionBody(p, "true_false", trueFalseAnswer, nil))

	// Urutan di request (q3, q1, q2) = nomor soal di quiz. coin_cost tidak dikirim → 0.
	// Event 3 hari: besok 08:00 s/d lusa+1 17:00 WIB (input tanpa zona = WIB).
	startInput, wantStart := daysFromNowWIB(1)
	endInput, wantEnd := daysFromNowWIB(3)
	endInput = endInput[:11] + "17:00"
	wantEnd = wantEnd.Add(9 * time.Hour)
	quiz := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q3, q1, q2}, map[string]string{
		"start_at": fmt.Sprintf("%q", startInput), "end_at": fmt.Sprintf("%q", endInput),
	}))

	if quiz.Name != "Uji Kompetensi Matematika - KPK" || quiz.Type != "uji_kom" || quiz.Status != "draft" ||
		quiz.DurationMinutes != 60 || quiz.CoinCost != 0 || quiz.TotalQuestions != 3 {
		t.Errorf("info quiz salah: %+v", quiz)
	}
	if quiz.Placement.Grade.Name != "Kelas 6" || !strings.HasPrefix(quiz.Placement.Subject.Name, "Matematika") ||
		quiz.Placement.SubjectField != nil || quiz.Placement.Chapter != nil {
		t.Errorf("placement salah: %+v", quiz.Placement)
	}
	if quiz.CreatedBy.Name != "User admin.soal" || quiz.UpdatedBy != nil {
		t.Errorf("pembuat salah: created_by %+v updated_by %+v", quiz.CreatedBy, quiz.UpdatedBy)
	}
	if got := strings.Join(questionIDsOf(quiz), ","); got != strings.Join([]string{q3, q1, q2}, ",") {
		t.Errorf("urutan soal = %s", got)
	}
	for i, q := range quiz.Questions {
		if q.No != i+1 || !strings.HasPrefix(q.Code, "Q-") {
			t.Errorf("soal ke-%d = %+v", i, q)
		}
	}

	// Jadwal: input "YYYY-MM-DDTHH:mm" dianggap WIB, ditampilkan dengan +07:00, event boleh beberapa hari.
	if !quiz.StartAt.Equal(wantStart) || !quiz.EndAt.Equal(wantEnd) || quiz.ScheduleStatus != "upcoming" {
		t.Errorf("start_at %s / end_at %s / %s, seharusnya %s / %s / upcoming", quiz.StartAt, quiz.EndAt, quiz.ScheduleStatus, wantStart, wantEnd)
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q1}, map[string]string{
		"start_at": fmt.Sprintf("%q", endInput), "end_at": fmt.Sprintf("%q", endInput[:11]+"23:00"), // tepat setelah quiz pertama selesai: boleh
	}), accessToken)
	if !strings.Contains(res.Body.String(), `"start_at":"`+wantEnd.Format("2006-01-02")+`T17:00:00+07:00"`) {
		t.Errorf("start_at seharusnya ditampilkan dalam WIB. body: %s", res.Body)
	}

	// Kode akses: 6 karakter A-Z/0-9, berbeda tiap quiz, tersimpan di database.
	var other quizDetail
	decodeData(t, res, &other)
	if !accessCodeFormat.MatchString(quiz.AccessCode) || quiz.AccessCode == other.AccessCode {
		t.Errorf("kode akses %q / %q, seharusnya format A-Z0-9 6 karakter dan berbeda", quiz.AccessCode, other.AccessCode)
	}
	var stored struct {
		AccessCode   string
		HasUpdatedAt bool
	}
	db.Raw(`SELECT access_code, access_code_updated_at IS NOT NULL AS has_updated_at FROM quizzes WHERE id = ?`, quiz.ID).Scan(&stored)
	if stored.AccessCode != quiz.AccessCode || !stored.HasUpdatedAt {
		t.Errorf("kode akses tersimpan %+v, seharusnya %s", stored, quiz.AccessCode)
	}
}

func TestCreateQuiz_TryoutKelasBebasDanLetakLebihDalam(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db) // Kelas 4: boleh untuk Tryout
	accessToken := adminToken(t, db)

	q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"used_for": `["tryout"]`}))

	placement := fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q,"subchapter_id":%q,"topic_id":%q}`,
		p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID, p.SubchapterID, p.TopicID)
	quiz := createQuiz(t, r, accessToken, quizBody(placement, []string{q}, map[string]string{
		"type": `"tryout"`, "name": `"  Tryout Bersama  "`, "coin_cost": "25", "duration_minutes": "120",
		"start_at": `"2099-01-01T01:00:00Z"`, // RFC3339 dengan zona waktu juga diterima
		"end_at":   `"2099-01-02T10:00:00Z"`,
	}))

	if quiz.Type != "tryout" || quiz.Name != "Tryout Bersama" || quiz.CoinCost != 25 || quiz.DurationMinutes != 120 ||
		quiz.Placement.Grade.Name != "Kelas 4" || !accessCodeFormat.MatchString(quiz.AccessCode) ||
		quiz.Placement.SubjectField == nil || quiz.Placement.SubjectField.Name != "Bilangan" ||
		quiz.Placement.Chapter == nil || quiz.Placement.Chapter.Name != "KPK dan FPB" {
		t.Errorf("quiz tryout salah: %+v", quiz)
	}
	if want := time.Date(2099, 1, 1, 8, 0, 0, 0, clock.Jakarta); !quiz.StartAt.Equal(want) || quiz.StartAt.Format("-07:00") != "+07:00" {
		t.Errorf("start_at = %s, seharusnya 2099-01-01 08:00 WIB", quiz.StartAt)
	}
}

func TestCreateQuiz_UjiKomHanyaKelas6_9_12(t *testing.T) {
	r, db := setupRouter(t)
	accessToken := adminToken(t, db)

	for _, tc := range []struct {
		grade string
		ok    bool
	}{
		{"GRADE_4", false}, {"GRADE_6", true}, {"GRADE_7", false}, {"GRADE_9", true}, {"GRADE_12", true},
	} {
		p := seedPlacementInGrade(t, db, tc.grade)
		q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
		res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, nil), accessToken)

		if tc.ok && res.Code != http.StatusCreated {
			t.Errorf("%s: status = %d, seharusnya 201. body: %s", tc.grade, res.Code, res.Body)
		}
		if !tc.ok && (res.Code != http.StatusUnprocessableEntity ||
			errorFields(t, res.Body.String())["placement.grade_id"] != "Uji Kom hanya untuk Kelas 6, 9, dan 12") {
			t.Errorf("%s: seharusnya ditolak. status %d, body: %s", tc.grade, res.Code, res.Body)
		}
	}
}

func TestCreateQuiz_ValidasiSemuaKesalahanSekaligus(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6") // mapel lain
	accessToken := adminToken(t, db)

	inactive := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"status": `"inactive"`}))
	tryoutOnly := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"used_for": `["tryout"]`}))
	otherSubject := createQuestionID(t, r, accessToken, questionBody(other, "single_choice", singleChoiceAnswer, nil))
	deleted := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	deletedCode := questionCode(t, r, accessToken, deleted)
	sendRequest(r, http.MethodDelete, "/api/v1/questions", fmt.Sprintf(`{"ids":[%q]}`, deleted), accessToken)
	missing := uuid.NewString()

	ids := []string{inactive, tryoutOnly, otherSubject, deleted, missing}
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), ids, map[string]string{
		"total_questions": "4", "name": `"   "`, "start_at": `"2020-01-01T08:00"`,
	}), accessToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	body := res.Body.String()
	for _, want := range []string{
		`"field":"name","message":"wajib diisi"`,
		`"field":"start_at","message":"harus setelah waktu sekarang"`,
		`"field":"total_questions","message":"harus sama dengan jumlah soal yang dipilih (5)"`,
		`"field":"question_ids[0]","message":"` + questionCode(t, r, accessToken, inactive) + `: soal tidak aktif"`,
		`"field":"question_ids[1]","message":"` + questionCode(t, r, accessToken, tryoutOnly) + `: soal tidak ditandai untuk Uji Kom"`,
		`"field":"question_ids[2]","message":"` + questionCode(t, r, accessToken, otherSubject) + `: mapel soal berbeda dengan quiz"`,
		`"field":"question_ids[3]","message":"` + deletedCode + `: soal sudah dihapus"`,
		`"field":"question_ids[4]","message":"soal tidak ditemukan"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("seharusnya ada error %s. body: %s", want, body)
		}
	}

	// Format waktu salah, tanpa soal & bundle, start_at kosong.
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), nil, map[string]string{
		"start_at": `"besok pagi"`, "question_ids": "", "total_questions": "1",
	}), accessToken)
	fields := errorFields(t, res.Body.String())
	if fields["start_at"] != "format tidak valid, contoh 2026-12-01T08:00 (WIB)" ||
		fields["question_ids"] != "pilih minimal 1 soal atau 1 bundle" {
		t.Errorf("format waktu / tanpa soal: body: %s", res.Body)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{tryoutOnly},
		map[string]string{"start_at": ""}), accessToken)
	if errorFields(t, res.Body.String())["start_at"] != "wajib diisi" {
		t.Errorf("start_at kosong seharusnya wajib diisi. body: %s", res.Body)
	}

	// Letak quiz salah: kelas SMP + jenjang SD, bab tanpa bidang materi. Soal tidak dibandingkan
	// ke letak yang salah (tidak ada pesan "berbeda dengan quiz").
	var smpGrade string
	db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_9'`).Scan(&smpGrade)
	placement := fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"chapter_id":%q}`,
		p.LevelID, smpGrade, p.SubjectID, p.ChapterID)
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(placement, []string{tryoutOnly}, nil), accessToken)
	fields = errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity ||
		fields["placement.grade_id"] != "kelas bukan bagian dari jenjang yang dipilih" ||
		fields["placement.subject_field_id"] != "wajib diisi jika chapter_id diisi" ||
		strings.Contains(res.Body.String(), "berbeda dengan quiz") {
		t.Errorf("letak salah: status %d, body: %s", res.Code, res.Body)
	}

	// Letak lebih dalam: soal tanpa sub bab → berbeda dengan quiz yang mengisi sub bab.
	noSub := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
			p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID)}))
	deep := fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q,"subchapter_id":%q}`,
		p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID, p.SubchapterID)
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(deep, []string{noSub}, nil), accessToken)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), ": sub bab soal berbeda dengan quiz") {
		t.Errorf("sub bab berbeda: status %d, body: %s", res.Code, res.Body)
	}

	// Binding: tipe selain uji_kom/tryout, soal kembar, durasi 0.
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{noSub, noSub},
		map[string]string{"type": `"daily_quiz"`, "duration_minutes": "0"}), accessToken)
	fields = errorFields(t, res.Body.String())
	for _, f := range []string{"type", "question_ids", "duration_minutes"} {
		if fields[f] == "" {
			t.Errorf("seharusnya ada error untuk %s. body: %s", f, res.Body)
		}
	}

	// Tidak ada quiz yang tersimpan dari semua request gagal di atas.
	var count int
	db.Raw(`SELECT COUNT(*) FROM quizzes`).Scan(&count)
	if count != 0 {
		t.Errorf("jumlah quiz = %d, seharusnya 0", count)
	}

	// Hanya admin & super admin.
	if res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", "{}", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor: status = %d, seharusnya 403", res.Code)
	}
}

func TestCreateQuiz_DariBundle(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	q := make([]string, 6)
	for i := range q {
		q[i] = createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	}
	bundleA := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q[0], q[1]}, map[string]string{"name": `"Bundle A"`}))
	bundleB := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q[2], q[1]}, map[string]string{"name": `"Bundle B"`}))

	// Bundle A (q0, q1) + bundle B (q2, q1 → q1 sudah ada) + soal satuan q3, q0 (q0 sudah ada).
	quiz := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q[3], q[0]}, map[string]string{
		"bundle_quiz_ids": jsonIDs(bundleA.ID, bundleB.ID), "total_questions": "4",
	}))
	if got := strings.Join(questionIDsOf(quiz), ","); got != strings.Join([]string{q[0], q[1], q[2], q[3]}, ",") {
		t.Errorf("soal quiz dari bundle = %s, seharusnya q0,q1,q2,q3", got)
	}

	// Hanya bundle, tanpa soal satuan.
	onlyBundle := createQuiz(t, r, accessToken, quizBody(basePlacement(p), nil, map[string]string{
		"bundle_quiz_ids": jsonIDs(bundleB.ID), "question_ids": "", "total_questions": "2",
	}))
	if got := strings.Join(questionIDsOf(onlyBundle), ","); got != q[2]+","+q[1] {
		t.Errorf("quiz dari 1 bundle = %s, seharusnya q2,q1", got)
	}

	// Soal di bundle yang sekarang nonaktif / dihapus → error di bundle_quiz_ids[i] dengan kode soal.
	db.Exec(`UPDATE questions SET status = 'inactive' WHERE id = ?`, q[0])
	db.Exec(`UPDATE questions SET deleted_at = NOW() WHERE id = ?`, q[2])
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q[4]}, map[string]string{
		"bundle_quiz_ids": jsonIDs(bundleA.ID, uuid.NewString(), bundleB.ID), "total_questions": "9",
	}), accessToken)
	codeOf := func(id string) string { // dari database, karena soal terhapus tidak bisa dibuka lewat API
		var code string
		db.Raw(`SELECT code FROM questions WHERE id = ?`, id).Scan(&code)
		return code
	}
	body := res.Body.String()
	// Soal: q0, q1 (bundle A), q2 (bundle B; q1 sudah ada), q4 (satuan) = 4 soal.
	for _, want := range []string{
		`"field":"bundle_quiz_ids[0]","message":"` + codeOf(q[0]) + `: soal tidak aktif"`,
		`"field":"bundle_quiz_ids[1]","message":"bundle tidak ditemukan"`,
		`"field":"bundle_quiz_ids[2]","message":"` + codeOf(q[2]) + `: soal sudah dihapus"`,
		`"field":"total_questions","message":"harus sama dengan jumlah soal yang dipilih (4, termasuk soal dari bundle)"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("seharusnya ada error %s. body: %s", want, body)
		}
	}
	if strings.Contains(body, codeOf(q[1])+":") || strings.Contains(body, "question_ids[0]") {
		t.Errorf("soal yang valid tidak boleh error. body: %s", body)
	}
}

func TestQuizBundles_ListDanIsiBundle(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	q1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	q2 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	q3 := createQuestionID(t, r, accessToken, questionBody(other, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["tryout"]`}))

	ujiKom := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q1, q2}, map[string]string{"name": `"Uji Kom Pecahan"`}))
	tryout := createQuiz(t, r, accessToken, quizBody(basePlacement(other), []string{q3}, map[string]string{"name": `"Tryout 100%"`, "type": `"tryout"`}))
	deletedQuiz := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q1}, map[string]string{"name": `"Uji Kom Lama"`}))
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, deletedQuiz.ID)

	type bundle struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		Grade          listRef `json:"grade"`
		Subject        listRef `json:"subject"`
		TotalQuestions int     `json:"total_questions"`
	}
	list := func(query string) []bundle {
		res := sendRequest(r, http.MethodGet, "/api/v1/quiz-bundles?"+query, "", accessToken)
		if res.Code != http.StatusOK {
			t.Fatalf("quiz-bundles?%s: status %d. body: %s", query, res.Code, res.Body)
		}
		var items []bundle
		decodeData(t, res, &items)
		return items
	}
	names := func(items []bundle) string {
		parts := []string{}
		for _, b := range items {
			parts = append(parts, b.Name)
		}
		return strings.Join(parts, ",")
	}

	all := list("")
	if names(all) != "Tryout 100%,Uji Kom Pecahan" { // terbaru dulu, quiz terhapus tidak ikut
		t.Errorf("semua bundle = %s", names(all))
	}
	if b := all[1]; b.ID != ujiKom.ID || b.Type != "uji_kom" || b.TotalQuestions != 2 || b.Grade.Name != "Kelas 6" ||
		!strings.HasPrefix(b.Subject.Name, "Matematika") {
		t.Errorf("isi bundle salah: %+v", b)
	}
	for query, want := range map[string]string{
		"type=tryout":                    "Tryout 100%",
		"subject_id=" + p.SubjectID:      "Uji Kom Pecahan",
		"search=" + url.QueryEscape("%"): "Tryout 100%", // % dicari apa adanya
		"search=pecahan":                 "Uji Kom Pecahan",
	} {
		if got := names(list(query)); got != want {
			t.Errorf("filter %s = %q, seharusnya %q", query, got, want)
		}
	}
	if tryout.ID == "" {
		t.Fatal("tryout tidak terbuat")
	}

	// Isi bundle = list soal dengan filter quiz_id (semua kolom & filter list soal tetap berlaku).
	res := sendRequest(r, http.MethodGet, "/api/v1/questions?quiz_id="+ujiKom.ID, "", accessToken)
	var items []struct {
		ID string `json:"id"`
	}
	decodeData(t, res, &items)
	got := map[string]bool{}
	for _, it := range items {
		got[it.ID] = true
	}
	if len(items) != 2 || !got[q1] || !got[q2] {
		t.Errorf("isi bundle Uji Kom = %v, seharusnya q1 & q2", items)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/questions?used_for=tryout&quiz_id="+ujiKom.ID, "", accessToken)
	decodeData(t, res, &items)
	if len(items) != 1 || items[0].ID != q2 {
		t.Errorf("isi bundle + filter tryout = %v, seharusnya hanya q2", items)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/questions?quiz_id="+deletedQuiz.ID, "", accessToken)
	decodeData(t, res, &items)
	if len(items) != 0 {
		t.Errorf("isi bundle yang sudah dihapus = %v, seharusnya kosong", items)
	}

	// Hanya admin.
	if res := sendRequest(r, http.MethodGet, "/api/v1/quiz-bundles", "", tokenFor(t, loginmodel.RoleStudent)); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status = %d, seharusnya 403", res.Code)
	}
}

func TestRefreshAccessCodes_HanyaQuizYangBelumSelesai(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)
	service := quizservice.NewQuizService(quizrepo.NewQuizRepository(db), questionrepo.NewQuestionRepository(db))

	q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	upcoming := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, nil))
	running := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, nil))
	ended := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, nil))
	deleted := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, nil))

	// Atur waktu langsung di database: sedang berjalan (mulai 30 menit lalu, durasi 60),
	// sudah selesai (mulai 2 jam lalu, durasi 60), dan quiz yang dihapus.
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '30 minutes', end_at = NOW() + INTERVAL '30 minutes' WHERE id = ?`, running.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '2 hours', end_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, ended.ID)
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, deleted.ID)
	db.Exec(`UPDATE quizzes SET access_code_updated_at = NOW() - INTERVAL '1 hour'`)

	codeOf := func(id string) (code string, refreshedRecently bool) {
		var row struct {
			AccessCode string
			Recent     bool
		}
		db.Raw(`SELECT access_code, access_code_updated_at > NOW() - INTERVAL '1 minute' AS recent FROM quizzes WHERE id = ?`, id).Scan(&row)
		return row.AccessCode, row.Recent
	}

	refreshed, skipped, err := service.RefreshAccessCodes(context.Background())
	if err != nil || skipped || refreshed != 2 {
		t.Fatalf("refresh: %d quiz, skipped %v, err %v; seharusnya 2 quiz (akan datang & sedang berjalan)", refreshed, skipped, err)
	}
	for _, tc := range []struct {
		name    string
		quiz    quizDetail
		changed bool
	}{
		{"akan datang", upcoming, true}, {"sedang berjalan", running, true},
		{"sudah selesai", ended, false}, {"dihapus", deleted, false},
	} {
		code, recent := codeOf(tc.quiz.ID)
		if (code != tc.quiz.AccessCode) != tc.changed || recent != tc.changed || !accessCodeFormat.MatchString(code) {
			t.Errorf("%s: kode %s → %s (baru diganti %v), seharusnya diganti = %v", tc.name, tc.quiz.AccessCode, code, recent, tc.changed)
		}
	}

	// Server lain sedang mengerjakan (advisory lock dipegang) → dilewati, tidak ada yang diganti.
	tx := db.Begin()
	defer tx.Rollback()
	tx.Exec(`SELECT pg_advisory_xact_lock(?)`, quizrepo.RefreshLockKey)
	before, _ := codeOf(upcoming.ID)
	refreshed, skipped, err = service.RefreshAccessCodes(context.Background())
	after, _ := codeOf(upcoming.ID)
	if err != nil || !skipped || refreshed != 0 || before != after {
		t.Errorf("saat lock dipegang server lain: %d quiz, skipped %v, err %v, kode %s → %s", refreshed, skipped, err, before, after)
	}
}

func TestListQuestions_PenandaQuizDanFilterPeruntukan(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	used := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	unused := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"used_for": `["tryout"]`, "status": `"inactive"`}))

	first := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{used}, map[string]string{"name": `"Uji Kom A"`}))
	second := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{used}, map[string]string{"name": `"Uji Kom B"`}))

	type quizRef struct{ ID, Name, Type string }
	quizzesOf := func(query url.Values) map[string][]quizRef {
		res := sendRequest(r, http.MethodGet, "/api/v1/questions?"+query.Encode(), "", accessToken)
		var items []struct {
			ID      string    `json:"id"`
			Quizzes []quizRef `json:"quizzes"`
		}
		decodeData(t, res, &items)
		result := map[string][]quizRef{}
		for _, it := range items {
			if it.Quizzes == nil {
				t.Fatalf("quizzes seharusnya array (bukan null). body: %s", res.Body)
			}
			result[it.ID] = it.Quizzes
		}
		return result
	}

	got := quizzesOf(url.Values{})
	if q := got[used]; len(q) != 2 || q[0].ID != second.ID || q[0].Name != "Uji Kom B" || q[0].Type != "uji_kom" || q[1].ID != first.ID {
		t.Errorf("soal dipakai 2 quiz: %+v, seharusnya [Uji Kom B, Uji Kom A]", q)
	}
	if q := got[unused]; len(q) != 0 {
		t.Errorf("soal belum dipakai: %+v, seharusnya []", q)
	}

	// Quiz yang dihapus tidak dihitung.
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, second.ID)
	if q := quizzesOf(url.Values{})[used]; len(q) != 1 || q[0].ID != first.ID {
		t.Errorf("setelah quiz B dihapus: %+v, seharusnya hanya Uji Kom A", q)
	}

	// Filter untuk memilih soal quiz: peruntukan & status.
	cases := []struct {
		query url.Values
		want  string
	}{
		{url.Values{"used_for": {"uji_kom"}}, used},
		{url.Values{"used_for": {"tryout"}}, unused},
		{url.Values{"status": {"active"}}, used},
		{url.Values{"status": {"inactive"}, "used_for": {"tryout"}}, unused},
	}
	for _, tc := range cases {
		got := quizzesOf(tc.query)
		if _, ok := got[tc.want]; !ok || len(got) != 1 {
			t.Errorf("filter %s: %d soal, seharusnya hanya %s", tc.query.Encode(), len(got), tc.want)
		}
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/questions?used_for=olimpiade&status=draft&quiz_id=abc", "", accessToken)
	fields := errorFields(t, res.Body.String())
	if fields["used_for"] == "" || fields["status"] == "" || fields["quiz_id"] == "" {
		t.Errorf("filter tidak valid seharusnya 422: %s", res.Body)
	}
}
