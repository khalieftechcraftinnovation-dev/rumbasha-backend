// Test jadwal quiz (end_at, larangan bentrok jadwal), PUT /quizzes/:id, dan GET /quizzes (+ jumlah per jadwal).
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// schedule membuat extra start_at & end_at (JSON) dari input WIB "YYYY-MM-DDTHH:mm".
func schedule(startAt, endAt string) map[string]string {
	return map[string]string{"start_at": fmt.Sprintf("%q", startAt), "end_at": fmt.Sprintf("%q", endAt)}
}

// mergeExtra menggabungkan beberapa map extra quizBody.
func mergeExtra(maps ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// dayWIB = tanggal N hari lagi (WIB), format "2006-01-02T".
func dayWIB(days int) string {
	s, _ := daysFromNowWIB(days)
	return s[:11]
}

func TestCreateQuiz_JadwalEndAtDanBentrok(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6") // mapel lain, kelas sama
	accessToken := adminToken(t, db)
	q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	qOther := createQuestionID(t, r, accessToken, questionBody(other, "single_choice", singleChoiceAnswer, nil))
	d := dayWIB(5)

	// Validasi end_at: wajib, setelah start_at, dan minimal sepanjang durasi.
	for _, tc := range []struct {
		extra map[string]string
		want  string
	}{
		{map[string]string{"end_at": ""}, "wajib diisi"},
		{schedule(d+"10:00", d+"09:00"), "harus setelah start_at"},
		{map[string]string{"end_at": `"kapan-kapan"`}, "format tidak valid, contoh 2026-12-01T08:00 (WIB)"},
	} {
		res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, tc.extra), accessToken)
		if got := errorFields(t, res.Body.String())["end_at"]; res.Code != http.StatusUnprocessableEntity || got != tc.want {
			t.Errorf("end_at %v: status %d, error %q, seharusnya %q", tc.extra, res.Code, got, tc.want)
		}
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, schedule(d+"10:00", d+"10:30")), accessToken)
	wantMin := fmt.Sprintf("minimal %s-%s-%s 11:00 WIB (start_at + durasi 60 menit)", d[8:10], d[5:7], d[0:4])
	if got := errorFields(t, res.Body.String())["end_at"]; got != wantMin {
		t.Errorf("end_at terlalu cepat: %q, seharusnya %q", got, wantMin)
	}

	// Quiz A: 2 hari (hari d 08:00 s/d hari d+1 17:00).
	d2 := dayWIB(6)
	quizA := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, mergeExtra(
		schedule(d+"08:00", d2+"17:00"), map[string]string{"name": `"Uji Kom A"`})))

	// Mapel & kelas sama, jadwal beririsan → ditolak, termasuk beda tipe (Tryout).
	for _, quizType := range []string{"uji_kom", "tryout"} {
		res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, mergeExtra(
			schedule(d2+"09:00", d2+"12:00"), map[string]string{"type": fmt.Sprintf("%q", quizType)})), accessToken)
		want := fmt.Sprintf("jadwal beririsan dengan quiz \"Uji Kom A\" (%s-%s-%s 08:00 WIB - %s-%s-%s 17:00 WIB) untuk mapel & kelas yang sama",
			d[8:10], d[5:7], d[0:4], d2[8:10], d2[5:7], d2[0:4])
		if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["start_at"] != want {
			t.Errorf("%s beririsan: status %d, body %s", quizType, res.Code, res.Body)
		}
	}

	// Boleh: mapel lain di jam yang sama, atau mulai tepat saat quiz A selesai.
	createQuiz(t, r, accessToken, quizBody(basePlacement(other), []string{qOther}, schedule(d2+"09:00", d2+"12:00")))
	quizB := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, schedule(d2+"17:00", d2+"18:00")))

	// Quiz yang dihapus tidak menghalangi.
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, quizA.ID)
	createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, schedule(d+"09:00", d+"12:00")))

	// Database ikut menjaga (exclusion constraint), walaupun service dilewati:
	// geser quiz B (hari d+1 17:00-18:00) ke hari d 10:00-11:00 → bentrok quiz baru (d 09:00-12:00).
	err := db.Exec(`UPDATE quizzes SET start_at = start_at - INTERVAL '31 hours', end_at = end_at - INTERVAL '31 hours' WHERE id = ?`,
		quizB.ID).Error
	if err == nil || !strings.Contains(err.Error(), "quizzes_no_overlap") {
		t.Errorf("database seharusnya menolak jadwal bentrok, error: %v", err)
	}
}

func TestUpdateQuiz_GantiPenuh(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	q1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	q2 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	q3 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["tryout"]`}))

	d := dayWIB(20)
	quiz := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q1, q2}, schedule(d+"08:00", d+"12:00")))
	other := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q1}, schedule(dayWIB(21)+"08:00", dayWIB(21)+"12:00")))

	// Ganti semua: nama, tipe, jadwal (tetap beririsan dengan jadwal lamanya sendiri: boleh), soal.
	body := quizBody(basePlacement(p), []string{q3, q1}, mergeExtra(schedule(d+"09:00", d+"15:00"), map[string]string{
		"name": `"Tryout Revisi"`, "type": `"tryout"`, "duration_minutes": "90", "coin_cost": "15",
	}))
	res := sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+quiz.ID, body, accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("update: status %d. body: %s", res.Code, res.Body)
	}
	var updated quizDetail
	decodeData(t, res, &updated)
	if updated.ID != quiz.ID || updated.Name != "Tryout Revisi" || updated.Type != "tryout" || updated.DurationMinutes != 90 ||
		updated.CoinCost != 15 || updated.TotalQuestions != 2 || updated.Status != "draft" ||
		updated.AccessCode != quiz.AccessCode || updated.UpdatedBy == nil || updated.UpdatedBy.Name != "User admin.soal" ||
		updated.StartAt.Format("15:04") != "09:00" || updated.EndAt.Format("15:04") != "15:00" {
		t.Errorf("hasil update salah: %+v", updated)
	}
	if got := strings.Join(questionIDsOf(updated), ","); got != q3+","+q1 {
		t.Errorf("soal setelah update = %s, seharusnya q3,q1", got)
	}
	var rows int
	db.Raw(`SELECT COUNT(*) FROM quiz_questions WHERE quiz_id = ?`, quiz.ID).Scan(&rows)
	if rows != 2 {
		t.Errorf("baris quiz_questions = %d, seharusnya 2 (soal lama diganti)", rows)
	}

	// Bentrok dengan quiz lain → 422; aturan lain sama dengan create.
	res = sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+quiz.ID, quizBody(basePlacement(p), []string{q1},
		schedule(dayWIB(21)+"10:00", dayWIB(21)+"13:00")), accessToken)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "jadwal beririsan dengan quiz") {
		t.Errorf("update bentrok: status %d, body %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+quiz.ID, quizBody(basePlacement(p), []string{q1},
		map[string]string{"total_questions": "3"}), accessToken)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["total_questions"] == "" {
		t.Errorf("update total salah: status %d, body %s", res.Code, res.Body)
	}

	// Quiz yang sudah dimulai tidak bisa diubah.
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '1 hour', end_at = NOW() + INTERVAL '1 hour' WHERE id = ?`, other.ID)
	res = sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+other.ID, quizBody(basePlacement(p), []string{q1}, nil), accessToken)
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "Quiz sudah dimulai, tidak bisa diubah") {
		t.Errorf("quiz sudah dimulai: status %d, body %s", res.Code, res.Body)
	}

	// Tidak ditemukan, bukan UUID, sudah dihapus → 404. Bukan admin → 403.
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, other.ID)
	for _, id := range []string{uuid.NewString(), "bukan-uuid", other.ID} {
		res := sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+id, quizBody(basePlacement(p), []string{q1}, nil), accessToken)
		if res.Code != http.StatusNotFound {
			t.Errorf("update %s: status %d, seharusnya 404", id, res.Code)
		}
	}
	if res := sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+quiz.ID, "{}", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor: status %d, seharusnya 403", res.Code)
	}
}

// quizListItem adalah 1 baris GET /quizzes yang dicek di test.
type quizListItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	ScheduleStatus string `json:"schedule_status"`
	AccessCode     string `json:"access_code"`
	TotalQuestions int    `json:"total_questions"`
	Placement      struct {
		Grade   listRef `json:"grade"`
		Subject listRef `json:"subject"`
	} `json:"placement"`
	Questions any `json:"questions"` // tidak boleh ada di list
}

func listQuizzes(t *testing.T, r *gin.Engine, accessToken string, query url.Values) ([]quizListItem, listMeta) {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/quizzes?"+query.Encode(), "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("GET /quizzes?%s: status %d. body: %s", query.Encode(), res.Code, res.Body)
	}
	var body struct {
		Data []quizListItem `json:"data"`
		Meta listMeta       `json:"meta"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response bukan JSON: %v. body: %s", err, res.Body)
	}
	return body.Data, body.Meta
}

func quizNames(items []quizListItem) string {
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.Name)
	}
	return strings.Join(names, ",")
}

func TestListQuizzes_StatusJadwalFilterDanJumlah(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	other := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)
	q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))
	qOther := createQuestionID(t, r, accessToken, questionBody(other, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["tryout"]`}))

	// Urutan dibuat: Selesai, Berjalan, Akan Datang, Tryout Lain, (Dihapus).
	finished := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Selesai"`}))
	ongoing := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Berjalan"`}))
	createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Akan Datang"`}))
	createQuiz(t, r, accessToken, quizBody(basePlacement(other), []string{qOther}, map[string]string{"name": `"Tryout Lain"`, "type": `"tryout"`}))
	deleted := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Dihapus"`}))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '3 hours', end_at = NOW() - INTERVAL '2 hours' WHERE id = ?`, finished.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '30 minutes', end_at = NOW() + INTERVAL '1 day' WHERE id = ?`, ongoing.ID)
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, deleted.ID)

	items, meta := listQuizzes(t, r, accessToken, url.Values{})
	if got := quizNames(items); got != "Tryout Lain,Uji Kom Akan Datang,Uji Kom Berjalan,Uji Kom Selesai" || meta.Total != 4 {
		t.Fatalf("list = %s (total %d), seharusnya 4 quiz terbaru dibuat dulu", got, meta.Total)
	}
	wantStatus := []string{"upcoming", "upcoming", "ongoing", "finished"}
	for i, it := range items {
		if it.ScheduleStatus != wantStatus[i] || !accessCodeFormat.MatchString(it.AccessCode) || it.TotalQuestions != 1 ||
			it.Placement.Grade.Name != "Kelas 6" || it.Questions != nil {
			t.Errorf("%s: %+v, seharusnya schedule_status %s, tanpa daftar soal", it.Name, it, wantStatus[i])
		}
	}

	for query, want := range map[string]string{
		"schedule_status=upcoming":          "Tryout Lain,Uji Kom Akan Datang",
		"schedule_status=ongoing":           "Uji Kom Berjalan",
		"schedule_status=finished":          "Uji Kom Selesai",
		"type=tryout":                       "Tryout Lain",
		"subject_id=" + p.SubjectID:         "Uji Kom Akan Datang,Uji Kom Berjalan,Uji Kom Selesai",
		"search=" + url.QueryEscape("BERJ"): "Uji Kom Berjalan",
		"status=published":                  "",
		"grade_id=" + p.OtherGradeID:        "",
	} {
		values, _ := url.ParseQuery(query)
		got, meta := listQuizzes(t, r, accessToken, values)
		if quizNames(got) != want || meta.Total != len(got) {
			t.Errorf("filter %s = %q (total %d), seharusnya %q", query, quizNames(got), meta.Total, want)
		}
	}

	// Pagination: total tetap 4.
	page2, meta := listQuizzes(t, r, accessToken, url.Values{"page": {"2"}, "per_page": {"3"}})
	if quizNames(page2) != "Uji Kom Selesai" || meta.Total != 4 {
		t.Errorf("halaman 2 = %s (total %d)", quizNames(page2), meta.Total)
	}

	// Jumlah per status jadwal (angka tab), ikut filter lain; schedule_status diabaikan.
	for query, want := range map[string]string{
		"":                          `{"all":4,"upcoming":2,"ongoing":1,"finished":1}`,
		"subject_id=" + p.SubjectID: `{"all":3,"upcoming":1,"ongoing":1,"finished":1}`,
		"type=tryout&schedule_status=finished&page=9": `{"all":1,"upcoming":1,"ongoing":0,"finished":0}`,
	} {
		res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/schedule-counts?"+query, "", accessToken)
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"data":`+want) {
			t.Errorf("schedule-counts?%s: status %d, body %s, seharusnya data %s", query, res.Code, res.Body, want)
		}
	}

	// Validasi & hak akses.
	res := sendRequest(r, http.MethodGet, "/api/v1/quizzes?schedule_status=kemarin&type=ujian", "", accessToken)
	fields := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || fields["schedule_status"] == "" || fields["type"] == "" {
		t.Errorf("query tidak valid: status %d, body %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes", "", tokenFor(t, loginmodel.RoleStudent)); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status %d, seharusnya 403", res.Code)
	}
}
