// Test endpoint Quiz Trial: kelola soal (admin), quiz publik tanpa login, submit, dan report.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// trialFixture = data master untuk test trial: 2 mapel + kelas SD 2 (sd_1_3) & SMP 8 (smp).
type trialFixture struct {
	MathID, EnglishID  string
	Grade2ID, Grade8ID string
	SDLevelID          string
}

func seedTrialFixture(t *testing.T, db *gorm.DB) trialFixture {
	t.Helper()
	var f trialFixture
	suffix := uuid.NewString()[:8]
	scan := func(target *string, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(target).Error; err != nil || *target == "" {
			t.Fatalf("gagal menyiapkan data: %v\nSQL: %s", err, sql)
		}
	}
	scan(&f.MathID, `INSERT INTO master_data_items (type, name) VALUES ('subject', ?) RETURNING id`, "Matematika "+suffix)
	scan(&f.EnglishID, `INSERT INTO master_data_items (type, name) VALUES ('subject', ?) RETURNING id`, "Bahasa Inggris "+suffix)
	scan(&f.Grade2ID, `SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_2'`)
	scan(&f.Grade8ID, `SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_8'`)
	scan(&f.SDLevelID, `SELECT parent_id FROM master_data_items WHERE id = ?`, f.Grade2ID)

	t.Cleanup(func() {
		db.Exec(`DELETE FROM trial_quiz_attempts`)
		db.Exec(`DELETE FROM trial_quiz_questions`)
		db.Exec(`DELETE FROM master_data_items WHERE id IN ?`, []string{f.MathID, f.EnglishID})
	})
	return f
}

// trialQuestionBody membuat body soal trial. extra menimpa/menambah field (nilai JSON mentah; "" = hapus).
func trialQuestionBody(group, subjectID, questionType, answer string, extra map[string]string) string {
	fields := map[string]string{
		"grade_group": fmt.Sprintf("%q", group),
		"subject_id":  fmt.Sprintf("%q", subjectID),
		"type":        fmt.Sprintf("%q", questionType),
		"question":    `"Hasil dari 2 + 2 adalah ..."`,
		"explanation": `"2 + 2 = 4"`,
		"difficulty":  `"easy"`,
		"answer":      answer,
	}
	for k, v := range extra {
		fields[k] = v
	}
	parts := []string{}
	for k, v := range fields {
		if v != "" {
			parts = append(parts, fmt.Sprintf("%q:%s", k, v))
		}
	}
	return "{" + strings.Join(parts, ",") + "}"
}

type trialQuestionResp struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	GradeGroup string `json:"grade_group"`
	Subject    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"subject"`
	Type   string         `json:"type"`
	Answer map[string]any `json:"answer"`
}

func createTrialQuestion(t *testing.T, r *gin.Engine, admin, body string) trialQuestionResp {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/trial-quiz/questions", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("buat soal trial: status = %d. body: %s", res.Code, res.Body.String())
	}
	var q trialQuestionResp
	decodeData(t, res, &q)
	return q
}

func TestTrialQuiz_KelolaSoal(t *testing.T) {
	r, db := setupRouter(t)
	f := seedTrialFixture(t, db)
	admin := adminToken(t, db)

	// Hak akses: siswa tidak boleh.
	student := tokenFor(t, loginmodel.RoleStudent)
	if res := sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/questions", "", student); res.Code != http.StatusForbidden {
		t.Fatalf("siswa: status = %d, want 403", res.Code)
	}

	// Semua tipe soal bisa dibuat (struktur sama dengan bank soal), jawaban dirapikan.
	answers := map[string]string{
		"single_choice":   singleChoiceAnswer,
		"multiple_choice": multipleChoiceAnswer,
		"true_false":      trueFalseAnswer,
		"free_text":       freeTextNumberAnswer,
	}
	for qType, answer := range answers {
		q := createTrialQuestion(t, r, admin, trialQuestionBody("sd_1_3", f.MathID, qType, answer, nil))
		if q.Type != qType || q.GradeGroup != "sd_1_3" || !strings.HasPrefix(q.Code, "TQ-") || q.Subject.ID != f.MathID {
			t.Errorf("%s: soal = %+v", qType, q)
		}
	}

	// Validasi isi soal sama dengan bank soal.
	cases := []struct {
		name, body, wantField string
	}{
		{"kunci salah", trialQuestionBody("sd_1_3", f.MathID, "single_choice",
			`{"options":[{"key":"A","content":"1"},{"key":"B","content":"2"}],"correct_key":"Z"}`, nil), "answer.correct_key"},
		{"golongan salah", trialQuestionBody("tk", f.MathID, "single_choice", singleChoiceAnswer, nil), "grade_group"},
		{"mapel tidak ada", trialQuestionBody("sd_1_3", uuid.NewString(), "single_choice", singleChoiceAnswer, nil), "subject_id"},
		{"bukan mapel", trialQuestionBody("sd_1_3", f.Grade2ID, "single_choice", singleChoiceAnswer, nil), "subject_id"},
		{"gambar tidak ada", trialQuestionBody("sd_1_3", f.MathID, "single_choice", singleChoiceAnswer,
			map[string]string{"image_id": fmt.Sprintf("%q", uuid.NewString())}), "image_id"},
		{"pembahasan wajib", trialQuestionBody("sd_1_3", f.MathID, "single_choice", singleChoiceAnswer,
			map[string]string{"explanation": `"   "`}), "explanation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := sendRequest(r, http.MethodPost, "/api/v1/trial-quiz/questions", tc.body, admin)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422. body: %s", res.Code, res.Body.String())
			}
			if _, ok := errorFields(t, res.Body.String())[tc.wantField]; !ok {
				t.Errorf("error field %q tidak ada. body: %s", tc.wantField, res.Body.String())
			}
		})
	}

	// Maksimal 10 soal per golongan.
	for i := 0; i < 6; i++ {
		createTrialQuestion(t, r, admin, trialQuestionBody("sd_1_3", f.EnglishID, "single_choice", singleChoiceAnswer, nil))
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/trial-quiz/questions",
		trialQuestionBody("sd_1_3", f.MathID, "single_choice", singleChoiceAnswer, nil), admin)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["grade_group"] == "" {
		t.Fatalf("soal ke-11: status = %d, want 422 grade_group. body: %s", res.Code, res.Body.String())
	}

	// List + jumlah per golongan.
	var list struct {
		MaxPerGroup int `json:"max_per_group"`
		GroupCounts []struct {
			GradeGroup string `json:"grade_group"`
			Total      int    `json:"total"`
		} `json:"group_counts"`
		Questions []trialQuestionResp `json:"questions"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/questions?grade_group=sd_1_3", "", admin), &list)
	if list.MaxPerGroup != 10 || len(list.Questions) != 10 || len(list.GroupCounts) != 5 || list.GroupCounts[1].Total != 10 {
		t.Fatalf("list = max %d, %d soal, counts %+v", list.MaxPerGroup, len(list.Questions), list.GroupCounts)
	}

	// Ubah: pindah ke SMP (belum penuh) boleh.
	moved := list.Questions[0]
	res = sendRequest(r, http.MethodPut, "/api/v1/trial-quiz/questions/"+moved.ID,
		trialQuestionBody("smp", f.EnglishID, "free_text", freeTextTextAnswer, nil), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah: status = %d. body: %s", res.Code, res.Body.String())
	}
	var updated trialQuestionResp
	decodeData(t, res, &updated)
	if updated.GradeGroup != "smp" || updated.Type != "free_text" || updated.Code != moved.Code {
		t.Errorf("setelah ubah = %+v", updated)
	}

	// Detail, hapus, lalu detail lagi → 404.
	if res := sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/questions/"+moved.ID, "", admin); res.Code != http.StatusOK {
		t.Fatalf("detail: status = %d", res.Code)
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/trial-quiz/questions/"+moved.ID, "", admin); res.Code != http.StatusOK {
		t.Fatalf("hapus: status = %d. body: %s", res.Code, res.Body.String())
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/questions/"+moved.ID, "", admin); res.Code != http.StatusNotFound {
		t.Fatalf("detail setelah hapus: status = %d, want 404", res.Code)
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/trial-quiz/questions/bukan-uuid", "", admin); res.Code != http.StatusNotFound {
		t.Fatalf("hapus id asal: status = %d, want 404", res.Code)
	}

	// Mapel yang dipakai soal trial tercatat sebagai pemakaian (tidak bisa dihapus dari master data).
	var used int
	db.Raw(`SELECT COUNT(*) FROM trial_quiz_questions WHERE subject_id = ? AND deleted_at IS NULL`, f.EnglishID).Scan(&used)
	if used == 0 {
		t.Fatal("soal B. Inggris seharusnya ada")
	}
	res = sendRequest(r, http.MethodDelete, "/api/v1/master-data", fmt.Sprintf(`{"ids":[%q]}`, f.EnglishID), admin)
	if res.Code == http.StatusOK {
		t.Errorf("hapus mapel yang dipakai soal trial seharusnya ditolak. body: %s", res.Body.String())
	}
}

func TestTrialQuiz_PublikDanSubmit(t *testing.T) {
	r, db := setupRouter(t)
	f := seedTrialFixture(t, db)
	admin := adminToken(t, db)

	q1 := createTrialQuestion(t, r, admin, trialQuestionBody("sd_1_3", f.MathID, "single_choice", singleChoiceAnswer,
		map[string]string{"sort_order": "2"}))
	q2 := createTrialQuestion(t, r, admin, trialQuestionBody("sd_1_3", f.EnglishID, "true_false", trueFalseAnswer,
		map[string]string{"sort_order": "1"}))
	q3 := createTrialQuestion(t, r, admin, trialQuestionBody("sd_1_3", f.MathID, "free_text", freeTextNumberAnswer,
		map[string]string{"sort_order": "3"}))

	// 1. Jenjang & kelas (tanpa login).
	res := sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/grades", "", "")
	if res.Code != http.StatusOK {
		t.Fatalf("grades: status = %d. body: %s", res.Code, res.Body.String())
	}
	var levels []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Grades []struct {
			ID         string `json:"id"`
			GradeGroup string `json:"grade_group"`
		} `json:"grades"`
	}
	decodeData(t, res, &levels)
	groupOf := map[string]string{}
	for _, l := range levels {
		for _, g := range l.Grades {
			groupOf[g.ID] = g.GradeGroup
		}
	}
	if len(levels) < 4 || groupOf[f.Grade2ID] != "sd_1_3" || groupOf[f.Grade8ID] != "smp" {
		t.Fatalf("grades = %+v", levels)
	}

	// 2. Soal untuk Kelas 2: urut sort_order, berisi kunci & pembahasan.
	res = sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/questions?grade_id="+f.Grade2ID, "", "")
	if res.Code != http.StatusOK {
		t.Fatalf("questions: status = %d. body: %s", res.Code, res.Body.String())
	}
	var quiz struct {
		GradeGroup     string `json:"grade_group"`
		TotalQuestions int    `json:"total_questions"`
		EducationLevel struct {
			ID string `json:"id"`
		} `json:"education_level"`
		Questions []struct {
			No          int            `json:"no"`
			ID          string         `json:"id"`
			Answer      map[string]any `json:"answer"`
			Explanation string         `json:"explanation"`
			Subject     struct {
				Name string `json:"name"`
			} `json:"subject"`
		} `json:"questions"`
	}
	decodeData(t, res, &quiz)
	if quiz.GradeGroup != "sd_1_3" || quiz.TotalQuestions != 3 || quiz.EducationLevel.ID != f.SDLevelID {
		t.Fatalf("quiz = %+v", quiz)
	}
	if quiz.Questions[0].ID != q2.ID || quiz.Questions[1].ID != q1.ID || quiz.Questions[2].ID != q3.ID {
		t.Errorf("urutan soal salah: %+v", quiz.Questions)
	}
	if quiz.Questions[1].Answer["correct_key"] != "B" || quiz.Questions[1].Explanation == "" || quiz.Questions[0].Subject.Name == "" {
		t.Errorf("soal publik harus berisi kunci, pembahasan, mapel: %+v", quiz.Questions[1])
	}

	// Kelas SMP belum punya soal → 404; kelas asal → 422.
	if res := sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/questions?grade_id="+f.Grade8ID, "", ""); res.Code != http.StatusNotFound {
		t.Errorf("SMP tanpa soal: status = %d, want 404", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/questions?grade_id="+uuid.NewString(), "", ""); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("kelas asal: status = %d, want 422", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/questions", "", ""); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("tanpa grade_id: status = %d, want 422", res.Code)
	}

	// 3. Submit: data peserta wajib & dicek.
	res = sendRequest(r, http.MethodPost, "/api/v1/public/trial-quiz/submit",
		fmt.Sprintf(`{"name":"  ","whatsapp":"12345","grade_id":%q,"answers":[]}`, uuid.NewString()), "")
	fields := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || fields["name"] == "" || fields["whatsapp"] == "" || fields["grade_id"] == "" {
		t.Fatalf("submit data salah: status = %d. body: %s", res.Code, res.Body.String())
	}

	// Soal yang bukan bagian quiz ditolak.
	res = sendRequest(r, http.MethodPost, "/api/v1/public/trial-quiz/submit",
		fmt.Sprintf(`{"name":"Budi","whatsapp":"0812-3456-7890","grade_id":%q,"answers":[{"question_id":%q,"answer":"B","is_correct":true}]}`,
			f.Grade2ID, uuid.NewString()), "")
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("soal asing: status = %d, want 422. body: %s", res.Code, res.Body.String())
	}

	// Submit benar: 2 dari 3 benar (q3 tidak dijawab) → nilai 66.67.
	res = sendRequest(r, http.MethodPost, "/api/v1/public/trial-quiz/submit", fmt.Sprintf(`{
		"name": "  Budi   Santoso ", "whatsapp": "+62 812-3456-7890", "grade_id": %q, "duration_seconds": 300,
		"answers": [
			{"question_id": %q, "answer": "B", "is_correct": true},
			{"question_id": %q, "answer": [{"key":"A","correct":true},{"key":"B","correct":false}], "is_correct": true}
		]}`, f.Grade2ID, q1.ID, q2.ID), "")
	if res.Code != http.StatusCreated {
		t.Fatalf("submit: status = %d. body: %s", res.Code, res.Body.String())
	}
	var result struct {
		Score          float64 `json:"score"`
		CorrectCount   int     `json:"correct_count"`
		TotalQuestions int     `json:"total_questions"`
	}
	decodeData(t, res, &result)
	if result.Score != 66.67 || result.CorrectCount != 2 || result.TotalQuestions != 3 {
		t.Fatalf("hasil = %+v, want 66.67, 2/3", result)
	}

	// Submit kedua: semua salah.
	sendRequest(r, http.MethodPost, "/api/v1/public/trial-quiz/submit", fmt.Sprintf(`{
		"name": "Ani", "whatsapp": "081299998888", "grade_id": %q,
		"answers": [{"question_id": %q, "answer": "A", "is_correct": false}]}`, f.Grade2ID, q1.ID), "")

	// Data peserta tersimpan rapi.
	var saved struct {
		Name, Whatsapp, GradeGroup string
		EducationLevelID           string
	}
	db.Raw(`SELECT name, whatsapp, grade_group, education_level_id FROM trial_quiz_attempts WHERE name = 'Budi Santoso'`).Scan(&saved)
	if saved.Whatsapp != "6281234567890" || saved.GradeGroup != "sd_1_3" || saved.EducationLevelID != f.SDLevelID {
		t.Errorf("tersimpan = %+v", saved)
	}

	// 4. Report admin: list (terbaru dulu) + pencarian.
	res = sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/attempts?grade_group=sd_1_3", "", admin)
	var attempts []struct {
		Name     string  `json:"name"`
		WhatsApp string  `json:"whatsapp"`
		Score    float64 `json:"score"`
		Grade    struct {
			Name string `json:"name"`
		} `json:"grade"`
	}
	decodeData(t, res, &attempts)
	if len(attempts) != 2 || attempts[0].Name != "Ani" || attempts[1].Score != 66.67 || attempts[1].Grade.Name == "" || attempts[1].WhatsApp != "6281234567890" {
		t.Fatalf("attempts = %+v", attempts)
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/attempts?search=6281234", "", admin), &attempts)
	if len(attempts) != 1 || attempts[0].Name != "Budi Santoso" {
		t.Errorf("search WhatsApp = %+v", attempts)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/attempts?date_from=2026-12-02&date_to=2026-12-01", "", admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("rentang terbalik: status = %d, want 422", res.Code)
	}

	// Ringkasan: 2 peserta, rata-rata (66.67 + 0) / 2, soal paling sering salah = q3 (2x tidak dijawab).
	var summary struct {
		Participants int     `json:"participants"`
		AverageScore float64 `json:"average_score"`
		ByGroup      []struct {
			GradeGroup   string `json:"grade_group"`
			Participants int    `json:"participants"`
		} `json:"by_group"`
		MostWrong []struct {
			QuestionID string  `json:"question_id"`
			Wrong      int     `json:"wrong"`
			WrongRate  float64 `json:"wrong_rate"`
		} `json:"most_wrong"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/trial-quiz/attempts/summary", "", admin), &summary)
	if summary.Participants != 2 || summary.AverageScore != 33.34 || len(summary.ByGroup) != 5 || summary.ByGroup[1].Participants != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.MostWrong) == 0 || summary.MostWrong[0].Wrong != 2 || summary.MostWrong[0].WrongRate != 100 {
		t.Errorf("most_wrong = %+v", summary.MostWrong)
	}

	// 5. Soal dihapus admin → langsung hilang dari quiz publik (cache dihapus).
	sendRequest(r, http.MethodDelete, "/api/v1/trial-quiz/questions/"+q3.ID, "", admin)
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/public/trial-quiz/questions?grade_id="+f.Grade2ID, "", ""), &quiz)
	if quiz.TotalQuestions != 2 {
		t.Errorf("setelah hapus: total = %d, want 2", quiz.TotalQuestions)
	}
}

func TestTrialQuiz_SubmitDibatasiPerIP(t *testing.T) {
	r, db := setupRouter(t)
	seedTrialFixture(t, db)

	var last int
	for i := 0; i < 11; i++ {
		last = sendRequest(r, http.MethodPost, "/api/v1/public/trial-quiz/submit", `{}`, "").Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("request ke-11: status = %d, want 429", last)
	}
}
