// Test Daily Quiz (pilihan kelas & generate soal acak), status publish quiz, dan pengumuman quiz siswa.
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// studentToken membuat akun siswa di kelas tertentu (code master data, contoh "GRADE_5";
// "" = kelas belum diisi) lalu mengembalikan access token-nya.
func studentToken(t *testing.T, db *gorm.DB, gradeCode string) string {
	t.Helper()
	id := createUser(t, db, loginmodel.RoleStudent, "siswa."+uuid.NewString()[:8], loginmodel.UserStatusActive)
	if gradeCode != "" {
		err := db.Exec(`
			UPDATE users SET grade_id = g.id, education_level_id = g.parent_id
			FROM master_data_items g
			WHERE g.type = 'grade' AND g.code = ? AND users.id = ?`, gradeCode, id).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	accessToken, err := token.NewAccessToken(id, loginmodel.RoleStudent, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return accessToken
}

func TestDailyQuiz_PilihanKelas(t *testing.T) {
	r, db := setupRouter(t)

	type option struct {
		Value     string `json:"value"`
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
	}
	for _, tc := range []struct{ grade, want, def string }{
		{"GRADE_2", "Kelas 1,Kelas 2,Kelas 3", "Kelas 2"},
		{"GRADE_5", "Kelas 4,Kelas 5,Kelas 6", "Kelas 5"},
		{"GRADE_8", "Kelas 7,Kelas 8,Kelas 9", "Kelas 8"},
		{"GRADE_11", "Kelas 10,Kelas 11,Kelas 12", "Kelas 11"},
	} {
		res := sendRequest(r, http.MethodGet, "/api/v1/daily-quizzes/grade-options", "", studentToken(t, db, tc.grade))
		var options []option
		decodeData(t, res, &options)
		names, def := []string{}, ""
		for _, o := range options {
			names = append(names, o.Name)
			if o.IsDefault {
				def = o.Name
			}
		}
		if strings.Join(names, ",") != tc.want || def != tc.def {
			t.Errorf("%s: pilihan %v (default %s), seharusnya %s (default %s)", tc.grade, names, def, tc.want, tc.def)
		}
	}

	// Siswa tanpa kelas → 422; bukan siswa → 403.
	res := sendRequest(r, http.MethodGet, "/api/v1/daily-quizzes/grade-options", "", studentToken(t, db, ""))
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "Kelas kamu belum diisi") {
		t.Errorf("siswa tanpa kelas: status %d, body %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quizzes/grade-options", "", adminToken(t, db)); res.Code != http.StatusForbidden {
		t.Errorf("admin: status %d, seharusnya 403", res.Code)
	}
}

// decodeJSONData membaca field "data" dari body response (string).
func decodeJSONData(t *testing.T, body string, target any) {
	t.Helper()
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		t.Fatalf("response bukan JSON: %v. body: %s", err, body)
	}
	if err := json.Unmarshal(wrapper.Data, target); err != nil {
		t.Fatalf("gagal membaca data: %v. body: %s", err, body)
	}
}

// dailyBody membuat body POST /daily-quizzes.
func dailyBody(placement string, difficulty string, total int) string {
	return fmt.Sprintf(`{"placement":%s,"difficulty":%q,"total_questions":%d}`, placement, difficulty, total)
}

// generateDaily memanggil POST /daily-quizzes.
func generateDaily(r *gin.Engine, accessToken, body string) (int, string) {
	res := sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes", body, accessToken)
	return res.Code, res.Body.String()
}

func TestDailyQuiz_GenerateSoalAcak(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	admin := adminToken(t, db)
	student := studentToken(t, db, "GRADE_5")

	var imageID string
	db.Raw(`INSERT INTO media_assets (category, name, url, file_name, storage_path, mime_type, size_bytes)
		VALUES ('question', 'Gambar', 'http://api.test/storage/images/opsi.png', 'opsi.png', 'images/opsi.png', 'image/png', 100) RETURNING id`).Scan(&imageID)

	// Soal yang cocok (easy, daily_quiz, aktif): 5 pilihan ganda + 2 isian = 7, tipe campur.
	eligible := map[string]bool{}
	for i := 0; i < 5; i++ {
		eligible[createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))] = true
	}
	eligible[createQuestionID(t, r, admin, questionBody(p, "free_text", freeTextNumberAnswer, nil))] = true
	withImage := createQuestionID(t, r, admin, questionBody(p, "single_choice",
		fmt.Sprintf(`{"options":[{"key":"A","content":"","image_id":%q},{"key":"B","content":"12"}],"correct_key":"B"}`, imageID),
		map[string]string{"image_url": `"https://cdn.example.com/soal.png"`, "display_order": `["question","image"]`}))
	eligible[withImage] = true
	// Tidak cocok: level lain, nonaktif, tidak bertanda daily_quiz.
	createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"difficulty": `"medium"`}))
	createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"status": `"inactive"`}))
	createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom"]`}))

	placement := fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q}`, p.SubjectID, p.FieldID)

	// 7 soal cocok dari 7 → semua soal cocok, tanpa kunci jawaban & pembahasan.
	code, body := generateDaily(r, student, dailyBody(placement, "easy", 5))
	if code != http.StatusCreated {
		t.Fatalf("generate: status %d. body: %s", code, body)
	}
	for _, secret := range []string{"correct_key", "correct_keys", "accepted_answers", "explanation", "access_code", "duration"} {
		if strings.Contains(body, secret) {
			t.Errorf("response Daily Quiz tidak boleh berisi %q. body: %s", secret, body)
		}
	}
	var quiz struct {
		ID         string `json:"id"`
		Difficulty string `json:"difficulty"`
		Total      int    `json:"total_questions"`
		Placement  struct {
			Grade        listRef `json:"grade"`
			SubjectField listRef `json:"subject_field"`
		} `json:"placement"`
		Questions []struct {
			No        int     `json:"no"`
			ID        string  `json:"id"`
			Type      string  `json:"type"`
			ImageURL  *string `json:"image_url"`
			InputType *string `json:"input_type"`
			Options   []struct {
				Key      string  `json:"key"`
				ImageURL *string `json:"image_url"`
			} `json:"options"`
		} `json:"questions"`
	}
	decodeJSONData(t, body, &quiz)
	if quiz.Difficulty != "easy" || quiz.Total != 5 || len(quiz.Questions) != 5 ||
		quiz.Placement.Grade.Name != "Kelas 5" || quiz.Placement.SubjectField.Name != "Bilangan" {
		t.Fatalf("daily quiz salah: %+v", quiz)
	}
	seen := map[string]bool{}
	for i, q := range quiz.Questions {
		if q.No != i+1 || !eligible[q.ID] || seen[q.ID] {
			t.Errorf("soal %d (%s) tidak memenuhi syarat atau kembar", q.No, q.ID)
		}
		seen[q.ID] = true
		switch q.Type {
		case "free_text":
			if q.InputType == nil || *q.InputType != "number" || len(q.Options) != 0 {
				t.Errorf("isian: input_type %v, options %v", q.InputType, q.Options)
			}
		case "single_choice":
			if len(q.Options) != 3 && len(q.Options) != 2 {
				t.Errorf("pilihan ganda: opsi %v", q.Options)
			}
		}
	}
	var saved int
	db.Raw(`SELECT COUNT(*) FROM daily_quiz_questions WHERE daily_quiz_id = ?`, quiz.ID).Scan(&saved)
	if saved != 5 {
		t.Errorf("soal tersimpan %d, seharusnya 5", saved)
	}

	// Ambil semua 7 (tidak bisa: hanya 5/10/15). Minta 10 → tersedia hanya 7.
	code, body = generateDaily(r, student, dailyBody(placement, "easy", 10))
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "soal yang tersedia hanya 7, pilih total soal yang lebih kecil atau ubah pilihan") {
		t.Errorf("soal kurang: status %d, body %s", code, body)
	}

	// Gambar Image Library di opsi diubah menjadi link: cek dengan topik yang hanya berisi soal bergambar.
	db.Exec(`UPDATE questions SET subchapter_id = NULL, topic_id = NULL WHERE id <> ?`, withImage)
	db.Exec(`UPDATE questions SET difficulty = 'hard' WHERE id = ?`, withImage)
	for i := 0; i < 4; i++ { // 4 soal hard lain di topik → total 5
		createQuestionID(t, r, admin, questionBody(p, "true_false", trueFalseAnswer, map[string]string{"difficulty": `"hard"`}))
	}
	topicPlacement := fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q,"chapter_id":%q,"subchapter_id":%q,"topic_id":%q}`,
		p.SubjectID, p.FieldID, p.ChapterID, p.SubchapterID, p.TopicID)
	code, body = generateDaily(r, student, dailyBody(topicPlacement, "hard", 5))
	if code != http.StatusCreated || !strings.Contains(body, `"image_url":"https://cdn.example.com/soal.png"`) ||
		!strings.Contains(body, `{"key":"A","content":"","image_url":"http://api.test/storage/images/opsi.png"}`) {
		t.Errorf("soal bergambar: status %d, body %s", code, body)
	}

	// Level tanpa soal → pesan "belum ada soal".
	code, body = generateDaily(r, student, dailyBody(placement, "medium", 5))
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "soal yang tersedia hanya 1") {
		t.Errorf("level medium: status %d, body %s", code, body)
	}
}

func TestDailyQuiz_ValidasiPilihan(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	student := studentToken(t, db, "GRADE_5")
	base := func(extra string) string {
		return fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q%s}`, p.SubjectID, p.FieldID, extra)
	}

	var grade4, grade7 string
	db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_4'`).Scan(&grade4)
	db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_7'`).Scan(&grade7)

	cases := []struct {
		body, field, want string
	}{
		{dailyBody(base(fmt.Sprintf(`,"grade_id":%q`, grade7)), "easy", 5), "placement.grade_id", "kelas hanya boleh: Kelas 4, Kelas 5, Kelas 6"},
		{dailyBody(base(fmt.Sprintf(`,"grade_id":%q`, grade4)), "easy", 5), "placement.subject_field_id", "bidang materi bukan untuk kelas yang dipilih"},
		{dailyBody(base(fmt.Sprintf(`,"topic_id":%q`, p.TopicID)), "easy", 5), "placement.subchapter_id", "wajib diisi jika topic_id diisi"},
		{dailyBody(fmt.Sprintf(`{"subject_id":%q,"subject_field_id":%q}`, uuid.NewString(), p.FieldID), "easy", 5), "placement.subject_id", "mapel tidak ditemukan"},
		{dailyBody(base(""), "easy", 7), "total_questions", "harus salah satu dari: 5, 10, 15"},
		{dailyBody(base(""), "sulit", 5), "difficulty", "harus salah satu dari: easy, medium, hard"},
		{`{"placement":{"subject_id":"` + p.SubjectID + `"},"difficulty":"easy","total_questions":5}`, "placement.subject_field_id", "wajib diisi"},
		{dailyBody(base(""), "easy", 5), "total_questions", "belum ada soal untuk pilihan ini, ubah pilihan mapel/bab/level"},
	}
	for _, tc := range cases {
		code, body := generateDaily(r, student, tc.body)
		if got := errorFields(t, body)[tc.field]; code != http.StatusUnprocessableEntity || got != tc.want {
			t.Errorf("%s: status %d, error %q, seharusnya %q. body: %s", tc.field, code, got, tc.want, body)
		}
	}
	if code, _ := generateDaily(r, adminToken(t, db), dailyBody(base(""), "easy", 5)); code != http.StatusForbidden {
		t.Errorf("admin generate: status %d, seharusnya 403", code)
	}
}

func TestQuizPublishDanPengumumanSiswa(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	q := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))

	// Tombol simpan: status wajib draft / published, bisa diubah lewat update.
	published := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Publish"`, "status": `"published"`}))
	if published.Status != "published" {
		t.Errorf("status = %s, seharusnya published", published.Status)
	}
	draft := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Draf"`}))
	ongoing := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Tryout Berjalan"`, "type": `"tryout"`, "status": `"published"`}))
	finished := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Lewat"`, "status": `"published"`}))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '1 hour', end_at = NOW() + INTERVAL '1 hour' WHERE id = ?`, ongoing.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '3 hours', end_at = NOW() - INTERVAL '2 hours' WHERE id = ?`, finished.ID)

	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, map[string]string{"status": `"arsip"`}), admin)
	if errorFields(t, res.Body.String())["status"] == "" {
		t.Errorf("status tidak valid seharusnya 422. body: %s", res.Body)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, map[string]string{"status": ""}), admin)
	if errorFields(t, res.Body.String())["status"] != "wajib diisi" {
		t.Errorf("status kosong seharusnya wajib diisi. body: %s", res.Body)
	}

	// Siswa Kelas 6 melihat quiz publish yang belum selesai, urut paling dekat dimulai, tanpa kode akses.
	student := studentToken(t, db, "GRADE_6")
	res = sendRequest(r, http.MethodGet, "/api/v1/student/quiz-announcements", "", student)
	if res.Code != http.StatusOK {
		t.Fatalf("pengumuman: status %d. body: %s", res.Code, res.Body)
	}
	if strings.Contains(res.Body.String(), "access_code") || strings.Contains(res.Body.String(), published.AccessCode) {
		t.Errorf("pengumuman tidak boleh berisi kode akses. body: %s", res.Body)
	}
	var items []quizListItem
	decodeData(t, res, &items)
	if quizNames(items) != "Tryout Berjalan,Uji Kom Publish" || items[0].ScheduleStatus != "ongoing" || items[1].ScheduleStatus != "upcoming" {
		t.Errorf("pengumuman = %s %+v, seharusnya Tryout Berjalan (ongoing), Uji Kom Publish (upcoming)", quizNames(items), items)
	}

	// Draf dipublish lewat update → ikut tampil; kelas lain & siswa tanpa kelas tidak melihat apa pun.
	body := quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Uji Kom Draf"`, "status": `"published"`})
	if res := sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+draft.ID, body, admin); res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"status":"published"`) {
		t.Fatalf("publish lewat update: status %d, body %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/student/quiz-announcements", "", student)
	decodeData(t, res, &items)
	if len(items) != 3 {
		t.Errorf("setelah draf dipublish: %s, seharusnya 3 quiz", quizNames(items))
	}
	for _, grade := range []string{"GRADE_9", ""} {
		res := sendRequest(r, http.MethodGet, "/api/v1/student/quiz-announcements", "", studentToken(t, db, grade))
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"data":[]`) {
			t.Errorf("siswa kelas %q: status %d, body %s, seharusnya data []", grade, res.Code, res.Body)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/quiz-announcements", "", admin); res.Code != http.StatusForbidden {
		t.Errorf("admin: status %d, seharusnya 403", res.Code)
	}
}
