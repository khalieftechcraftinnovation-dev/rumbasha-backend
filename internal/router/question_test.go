// Test endpoint POST /questions: membuat soal (khusus admin & super admin).
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

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// testPlacement berisi ID master data untuk letak soal: Matematika → Bilangan (SD, Kelas 4) → bab → sub bab → topik.
type testPlacement struct {
	LevelID, GradeID, OtherGradeID, SubjectID, FieldID, ChapterID, SubchapterID, TopicID string
}

// seedPlacement membuat hierarki mapel di database (Kelas 4). Nama diberi akhiran acak supaya
// tidak bentrok dengan test lain (tabel master_data_items tidak dikosongkan antar test).
func seedPlacement(t *testing.T, db *gorm.DB) testPlacement {
	t.Helper()
	return seedPlacementInGrade(t, db, "GRADE_4")
}

// seedPlacementInGrade sama seperti seedPlacement, tapi di kelas tertentu (code master data,
// contoh "GRADE_6" untuk Uji Kom). OtherGradeID selalu Kelas 5.
func seedPlacementInGrade(t *testing.T, db *gorm.DB, gradeCode string) testPlacement {
	t.Helper()
	var p testPlacement
	suffix := uuid.NewString()[:8]

	mustScan := func(target *string, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(target).Error; err != nil || *target == "" {
			t.Fatalf("gagal menyiapkan master data: %v\nSQL: %s", err, sql)
		}
	}
	mustScan(&p.GradeID, `SELECT id FROM master_data_items WHERE type = 'grade' AND code = ?`, gradeCode)
	mustScan(&p.OtherGradeID, `SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_5'`)
	mustScan(&p.LevelID, `SELECT parent_id FROM master_data_items WHERE id = ?`, p.GradeID)
	mustScan(&p.SubjectID, `INSERT INTO master_data_items (type, name) VALUES ('subject', ?) RETURNING id`, "Matematika "+suffix)

	insertChild := func(target *string, itemType, name, parentID string) {
		t.Helper()
		mustScan(target, `
			INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id`, itemType, name, parentID, p.SubjectID, p.LevelID, p.GradeID)
	}
	insertChild(&p.FieldID, "subject_field", "Bilangan", p.SubjectID)
	insertChild(&p.ChapterID, "chapter", "KPK dan FPB", p.FieldID)
	insertChild(&p.SubchapterID, "subchapter", "Menentukan KPK dan FPB", p.ChapterID)
	insertChild(&p.TopicID, "topic", "Menentukan KPK", p.SubchapterID)

	t.Cleanup(func() {
		// Urutan penting: soal & gambar menunjuk ke master data, jadi dihapus lebih dulu.
		db.Exec(`DELETE FROM daily_quizzes`)       // daily_quiz_questions ikut terhapus (ON DELETE CASCADE)
		db.Exec(`DELETE FROM leaderboard_entries`) // leaderboard_point_logs ikut terhapus (ON DELETE CASCADE)
		db.Exec(`DELETE FROM quiz_prize_awards`)   // menunjuk quizzes
		db.Exec(`DELETE FROM daily_quiz_prize_awards`)
		db.Exec(`DELETE FROM daily_quiz_prize_periods`)
		db.Exec(`DELETE FROM quiz_attempts`)
		db.Exec(`DELETE FROM quizzes`) // quiz_questions ikut terhapus (ON DELETE CASCADE)
		db.Exec(`DELETE FROM questions`)
		db.Exec(`DELETE FROM media_assets`)
		db.Exec(`DELETE FROM master_data_items WHERE id IN ?`, []string{p.TopicID, p.SubchapterID, p.ChapterID})
		db.Exec(`DELETE FROM master_data_items WHERE id = ?`, p.FieldID)
		db.Exec(`DELETE FROM master_data_items WHERE id = ?`, p.SubjectID)
	})
	return p
}

// adminToken membuat user admin sungguhan (dibutuhkan untuk kolom created_by) lalu token-nya.
func adminToken(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := createUser(t, db, loginmodel.RoleAdmin, "admin.soal", loginmodel.UserStatusActive)
	accessToken, err := token.NewAccessToken(id, loginmodel.RoleAdmin, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return accessToken
}

// questionBody membuat body request berisi 1 soal: {"type": ..., "questions": [soal]}.
// extra menimpa/menambah field soal (nilai JSON mentah).
func questionBody(p testPlacement, questionType, answer string, extra map[string]string) string {
	return questionsBody(questionType, questionItem(p, answer, extra))
}

// questionsBody membungkus beberapa soal (JSON) menjadi body request create.
func questionsBody(questionType string, items ...string) string {
	return fmt.Sprintf(`{"type":%q,"questions":[%s]}`, questionType, strings.Join(items, ","))
}

// questionItem membuat 1 soal (JSON). extra menimpa/menambah field (nilai JSON mentah; "" = hapus field).
func questionItem(p testPlacement, answer string, extra map[string]string) string {
	fields := map[string]string{
		"question":    `"Hasil dari $\\frac{1}{2} + \\frac{1}{4}$ adalah ..."`,
		"explanation": `"Samakan penyebut: $\\frac{2}{4} + \\frac{1}{4} = \\frac{3}{4}$"`,
		"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q,"subchapter_id":%q,"topic_id":%q}`,
			p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID, p.SubchapterID, p.TopicID),
		"difficulty": `"easy"`,
		"used_for":   `["daily_quiz","uji_kom"]`,
		"answer":     answer,
	}
	for k, v := range extra {
		fields[k] = v
	}

	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		if v == "" { // nilai kosong = field dihapus dari body
			continue
		}
		parts = append(parts, fmt.Sprintf("%q:%s", k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

const (
	singleChoiceAnswer   = `{"options":[{"key":"A","content":"$\\frac{2}{6}$","image_id":null},{"key":"B","content":" $\\frac{3}{4}$ "},{"key":"C","content":"1"}],"correct_key":"B"}`
	multipleChoiceAnswer = `{"options":[{"key":"A","content":"2"},{"key":"B","content":"4"},{"key":"C","content":"7"}],"correct_keys":["A","C"]}`
	trueFalseAnswer      = `{"options":[{"key":"A","content":"2 bilangan prima"},{"key":"B","content":"4 bilangan prima"}],"correct_keys":[{"key":"B","correct":false},{"key":"A","correct":true}]}`
	freeTextNumberAnswer = `{"input_type":"number","accepted_answers":["0.75","3/4","0,75"]}`
	freeTextTextAnswer   = `{"input_type":"text","accepted_answers":["kelipatan persekutuan"]}`
)

// errorFields membaca daftar errors dari response menjadi map field → pesan.
func errorFields(t *testing.T, body string) map[string]string {
	t.Helper()
	var parsed struct {
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("response bukan JSON: %v. body: %s", err, body)
	}
	result := map[string]string{}
	for _, e := range parsed.Errors {
		result[e.Field] = e.Message
	}
	return result
}

func TestCreateQuestion_HanyaAdminDanSuperAdmin(t *testing.T) {
	r, _ := setupRouter(t)
	body := `{}`

	if res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, ""); res.Code != http.StatusUnauthorized {
		t.Errorf("tanpa login: status = %d, seharusnya 401", res.Code)
	}
	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		if res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403", role, res.Code)
		}
	}
}

func TestCreateQuestion_SemuaTipeBerhasil(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		questionType string
		answer       string
		wantAnswer   string // answer yang tersimpan (sudah dirapikan)
	}{
		{"single_choice", singleChoiceAnswer,
			`{"options":[{"key":"A","content":"$\\frac{2}{6}$","image_id":null},{"key":"B","content":"$\\frac{3}{4}$","image_id":null},{"key":"C","content":"1","image_id":null}],"correct_key":"B"}`},
		{"multiple_choice", multipleChoiceAnswer,
			`{"options":[{"key":"A","content":"2","image_id":null},{"key":"B","content":"4","image_id":null},{"key":"C","content":"7","image_id":null}],"correct_keys":["A","C"],"scoring":"all_or_nothing"}`},
		{"true_false", trueFalseAnswer,
			`{"options":[{"key":"A","content":"2 bilangan prima","image_id":null},{"key":"B","content":"4 bilangan prima","image_id":null}],"correct_keys":[{"key":"A","correct":true},{"key":"B","correct":false}]}`},
		{"free_text", freeTextNumberAnswer,
			`{"input_type":"number","accepted_answers":["0.75","3/4","0,75"],"tolerance":0}`},
		{"free_text", freeTextTextAnswer,
			`{"input_type":"text","accepted_answers":["kelipatan persekutuan"],"case_sensitive":false}`},
	}

	for _, tt := range tests {
		res := sendRequest(r, http.MethodPost, "/api/v1/questions", questionBody(p, tt.questionType, tt.answer, nil), accessToken)
		if res.Code != http.StatusCreated {
			t.Errorf("%s: status = %d, seharusnya 201. body: %s", tt.questionType, res.Code, res.Body)
			continue
		}

		var list []struct {
			Code         string          `json:"code"`
			Status       string          `json:"status"`
			DisplayOrder []string        `json:"display_order"`
			Answer       json.RawMessage `json:"answer"`
		}
		decodeData(t, res, &list)
		if len(list) != 1 {
			t.Fatalf("%s: data berisi %d soal, seharusnya 1", tt.questionType, len(list))
		}
		data := list[0]
		if !strings.HasPrefix(data.Code, "Q-") || data.Status != "active" || strings.Join(data.DisplayOrder, ",") != "question" {
			t.Errorf("%s: code=%q status=%q display_order=%v tidak sesuai", tt.questionType, data.Code, data.Status, data.DisplayOrder)
		}

		// Bandingkan dengan isi di database (jsonb), supaya yang tersimpan juga benar.
		var stored string
		db.Raw(`SELECT answer::text FROM questions WHERE code = ?`, data.Code).Scan(&stored)
		if !sameJSON(t, stored, tt.wantAnswer) {
			t.Errorf("%s: answer tersimpan = %s\nseharusnya %s", tt.questionType, stored, tt.wantAnswer)
		}
	}
}

func TestCreateQuestion_DenganTabelFormulaDanGambar(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	var imageID string
	db.Raw(`INSERT INTO media_assets (category, name, url, file_name, storage_path, mime_type, size_bytes) VALUES ('question', 'Gambar', 'http://api.test/storage/images/a.png', 'a.png', 'images/a.png', 'image/png', 100) RETURNING id`).Scan(&imageID)

	// Gambar, tabel, dan rumus ada di level soal (bukan di answer), jadi berlaku untuk SEMUA tipe:
	// boleh salah satu, beberapa, atau ketiganya sekaligus.
	content := map[string]string{
		"formula":       `"\\frac{a}{b}"`,
		"image_id":      fmt.Sprintf("%q", imageID),
		"table":         `{"headers":["Nama","Nilai"],"rows":[["Andi","80"],["Budi","75"]]}`,
		"display_order": `["question","table","formula","image"]`,
	}
	answers := map[string]string{
		"single_choice":   singleChoiceAnswer,
		"multiple_choice": multipleChoiceAnswer,
		"true_false":      trueFalseAnswer,
		"free_text":       freeTextNumberAnswer,
	}

	for questionType, answer := range answers {
		res := sendRequest(r, http.MethodPost, "/api/v1/questions", questionBody(p, questionType, answer, content), accessToken)
		if res.Code != http.StatusCreated {
			t.Errorf("%s: status = %d, seharusnya 201. body: %s", questionType, res.Code, res.Body)
			continue
		}

		var stored struct {
			Formula      *string
			ImageID      *string
			HasTable     bool
			DisplayOrder string
		}
		db.Raw(`SELECT formula, image_id, table_data IS NOT NULL AS has_table, display_order::text AS display_order
			FROM questions WHERE type = ? ORDER BY created_at DESC LIMIT 1`, questionType).Scan(&stored)
		if stored.Formula == nil || stored.ImageID == nil || *stored.ImageID != imageID || !stored.HasTable ||
			stored.DisplayOrder != `["question", "table", "formula", "image"]` {
			t.Errorf("%s: isi tersimpan tidak lengkap: %+v", questionType, stored)
		}
	}
}

func TestCreateQuestion_ValidasiField(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		name       string
		body       string
		wantFields []string // field yang harus muncul di errors (tanpa awalan questions[0].)
	}{
		{
			name:       "field wajib & enum",
			body:       questionBody(p, "essay", singleChoiceAnswer, map[string]string{"difficulty": "", "used_for": `["ujian"]`}),
			wantFields: []string{"type", "difficulty", "used_for[0]"},
		},
		{
			name: "placement tanpa bab",
			body: questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
				"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q}`, p.LevelID, p.GradeID, p.SubjectID, p.FieldID),
			}),
			wantFields: []string{"placement.chapter_id"},
		},
		{
			name: "single_choice: kunci tidak ada & key tidak urut & opsi kosong",
			body: questionBody(p, "single_choice",
				`{"options":[{"key":"A","content":"1"},{"key":"C","content":""}],"correct_key":"E"}`, nil),
			wantFields: []string{"answer.correct_key", "answer.options[1].key", "answer.options[1].content"},
		},
		{
			name:       "multiple_choice: salah ketik nama field ditolak",
			body:       questionBody(p, "multiple_choice", `{"options":[{"key":"A","content":"1"},{"key":"B","content":"2"}],"correct_key":"A"}`, nil),
			wantFields: []string{"answer"},
		},
		{
			name: "true_false: pernyataan B belum diberi kunci",
			body: questionBody(p, "true_false",
				`{"options":[{"key":"A","content":"x"},{"key":"B","content":"y"}],"correct_keys":[{"key":"A","correct":true}]}`, nil),
			wantFields: []string{"answer.correct_keys"},
		},
		{
			name:       "free_text number: jawaban bukan angka",
			body:       questionBody(p, "free_text", `{"input_type":"number","accepted_answers":["tiga per empat"]}`, nil),
			wantFields: []string{"answer.accepted_answers[0]"},
		},
		{
			name:       "pembahasan tidak dikirim",
			body:       questionBody(p, "multiple_choice", multipleChoiceAnswer, map[string]string{"explanation": ""}),
			wantFields: []string{"explanation"},
		},
		{
			name:       "pembahasan hanya spasi",
			body:       questionBody(p, "free_text", freeTextNumberAnswer, map[string]string{"explanation": `"   "`}),
			wantFields: []string{"explanation"},
		},
		{
			name:       "display_order mencantumkan tabel yang tidak diisi",
			body:       questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"display_order": `["question","table"]`}),
			wantFields: []string{"display_order[1]"},
		},
		{
			name: "tabel jumlah kolom tidak sama",
			body: questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
				"table": `{"headers":["A","B"],"rows":[["1","2"],["3"]]}`,
			}),
			wantFields: []string{"table.rows[1]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendRequest(r, http.MethodPost, "/api/v1/questions", tt.body, accessToken)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
			}
			got := errorFields(t, res.Body.String())
			for _, field := range tt.wantFields {
				field = firstQuestionField(field)
				if _, ok := got[field]; !ok {
					t.Errorf("error untuk field %q tidak ada. errors: %v", field, got)
				}
			}
		})
	}
}

func TestCreateQuestion_PlacementDanGambarDicekKeDatabase(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		name       string
		placement  string
		extra      map[string]string
		wantFields []string
	}{
		{
			name: "kelas tidak sesuai bab",
			placement: fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
				p.LevelID, p.OtherGradeID, p.SubjectID, p.FieldID, p.ChapterID),
			wantFields: []string{"placement.grade_id"},
		},
		{
			name: "topik tanpa sub bab",
			placement: fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q,"topic_id":%q}`,
				p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID, p.TopicID),
			wantFields: []string{"placement.subchapter_id"},
		},
		{
			name: "bab tidak ada",
			placement: fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
				p.LevelID, p.GradeID, p.SubjectID, p.FieldID, uuid.NewString()),
			wantFields: []string{"placement.chapter_id"},
		},
		{
			name:       "gambar soal & opsi tidak ada",
			extra:      map[string]string{"image_id": fmt.Sprintf("%q", uuid.NewString())},
			wantFields: []string{"image_id", "answer.options[0].image_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extra := map[string]string{}
			for k, v := range tt.extra {
				extra[k] = v
			}
			if tt.placement != "" {
				extra["placement"] = tt.placement
			}
			answer := singleChoiceAnswer
			if _, ok := extra["image_id"]; ok {
				answer = fmt.Sprintf(`{"options":[{"key":"A","content":"","image_id":%q},{"key":"B","content":"2"}],"correct_key":"A"}`, uuid.NewString())
				extra["display_order"] = `["question","image"]`
			}

			res := sendRequest(r, http.MethodPost, "/api/v1/questions", questionBody(p, "single_choice", answer, extra), accessToken)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
			}
			got := errorFields(t, res.Body.String())
			for _, field := range tt.wantFields {
				field = firstQuestionField(field)
				if _, ok := got[field]; !ok {
					t.Errorf("error untuk field %q tidak ada. errors: %v", field, got)
				}
			}
		})
	}

	var count int64
	db.Raw(`SELECT COUNT(*) FROM questions`).Scan(&count)
	if count != 0 {
		t.Errorf("tidak boleh ada soal tersimpan, ada %d", count)
	}
}

// sameJSON membandingkan dua JSON tanpa peduli spasi & urutan key object.
func sameJSON(t *testing.T, a, b string) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		t.Fatalf("JSON tidak valid: %v: %s", err, a)
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		t.Fatalf("JSON tidak valid: %v: %s", err, b)
	}
	ja, _ := json.Marshal(va)
	jb, _ := json.Marshal(vb)
	return string(ja) == string(jb)
}

func TestCreateQuestion_BanyakSoalSekaligus(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	body := questionsBody("single_choice",
		questionItem(p, singleChoiceAnswer, map[string]string{"question": `"Soal 1"`}),
		questionItem(p, singleChoiceAnswer, map[string]string{"question": `"Soal 2"`, "type": `"single_choice"`}), // type per soal boleh, asal sama
		questionItem(p, singleChoiceAnswer, map[string]string{"question": `"Soal 3"`, "difficulty": `"hard"`}),
	)
	res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, accessToken)
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), `"3 soal berhasil dibuat"`) {
		t.Fatalf("status = %d, seharusnya 201 dengan pesan 3 soal. body: %s", res.Code, res.Body)
	}

	var list []struct {
		ID       string `json:"id"`
		Code     string `json:"code"`
		Question string `json:"question"`
	}
	decodeData(t, res, &list)
	if len(list) != 3 {
		t.Fatalf("data berisi %d soal, seharusnya 3", len(list))
	}

	codes := map[string]bool{}
	for i, q := range list {
		// Urutan response sama dengan urutan request, dan setiap soal dapat kode berbeda.
		if want := fmt.Sprintf("Soal %d", i+1); q.Question != want {
			t.Errorf("data[%d].question = %q, seharusnya %q", i, q.Question, want)
		}
		if q.ID == "" || !strings.HasPrefix(q.Code, "Q-") || codes[q.Code] {
			t.Errorf("data[%d]: id=%q code=%q tidak valid / kembar", i, q.ID, q.Code)
		}
		codes[q.Code] = true

		var stored string
		db.Raw(`SELECT question_text FROM questions WHERE id = ? AND code = ?`, q.ID, q.Code).Scan(&stored)
		if stored != q.Question {
			t.Errorf("data[%d]: id & code di response tidak cocok dengan database", i)
		}
	}
}

func TestCreateQuestion_BanyakSoalSemuaAtauTidakSamaSekali(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		name       string
		body       string
		wantFields []string
	}{
		{
			name: "tipe bercampur ditolak",
			body: questionsBody("single_choice",
				questionItem(p, singleChoiceAnswer, nil),
				questionItem(p, freeTextNumberAnswer, map[string]string{"type": `"free_text"`}),
			),
			wantFields: []string{"questions[1].type"},
		},
		{
			name: "1 soal salah, soal lain tidak ikut tersimpan",
			body: questionsBody("single_choice",
				questionItem(p, singleChoiceAnswer, nil),
				questionItem(p, singleChoiceAnswer, nil),
				questionItem(p, `{"options":[{"key":"A","content":"1"},{"key":"B","content":"2"}],"correct_key":"E"}`, nil),
			),
			wantFields: []string{"questions[2].answer.correct_key"},
		},
		{
			name: "placement soal kedua salah",
			body: questionsBody("single_choice",
				questionItem(p, singleChoiceAnswer, nil),
				questionItem(p, singleChoiceAnswer, map[string]string{
					"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
						p.LevelID, p.OtherGradeID, p.SubjectID, p.FieldID, p.ChapterID),
				}),
			),
			wantFields: []string{"questions[1].placement.grade_id"},
		},
		{
			name:       "daftar soal kosong",
			body:       `{"type":"single_choice","questions":[]}`,
			wantFields: []string{"questions"},
		},
		{
			name:       "lebih dari 100 soal",
			body:       questionsBody("single_choice", strings.Repeat(questionItem(p, singleChoiceAnswer, nil)+",", 100)+questionItem(p, singleChoiceAnswer, nil)),
			wantFields: []string{"questions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendRequest(r, http.MethodPost, "/api/v1/questions", tt.body, accessToken)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
			}
			got := errorFields(t, res.Body.String())
			for _, field := range tt.wantFields {
				if _, ok := got[field]; !ok {
					t.Errorf("error untuk field %q tidak ada. errors: %v", field, got)
				}
			}
		})
	}

	var count int64
	db.Raw(`SELECT COUNT(*) FROM questions`).Scan(&count)
	if count != 0 {
		t.Errorf("tidak boleh ada soal tersimpan, ada %d", count)
	}
}

// firstQuestionField menambah awalan "questions[0]." ke nama field soal pertama.
// Field level atas ("type", "questions") dibiarkan apa adanya.
func firstQuestionField(field string) string {
	if field == "type" || strings.HasPrefix(field, "questions") {
		return field
	}
	return "questions[0]." + field
}

// createQuestionID membuat 1 soal lewat API lalu mengembalikan ID-nya.
func createQuestionID(t *testing.T, r *gin.Engine, accessToken, body string) string {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("gagal membuat soal: status %d. body: %s", res.Code, res.Body)
	}
	var list []struct {
		ID string `json:"id"`
	}
	decodeData(t, res, &list)
	return list[0].ID
}

func TestGetQuestionDetail_LengkapDenganKunciDanPembahasan(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"formula": `"\\frac{a}{b}"`,
		"table":   `{"headers":["Nama","Nilai"],"rows":[["Andi","80"]]}`,
	}))

	res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	type ref struct {
		ID   string  `json:"id"`
		Name *string `json:"name"`
	}
	var data struct {
		ID           string          `json:"id"`
		Code         string          `json:"code"`
		Type         string          `json:"type"`
		Formula      *string         `json:"formula"`
		Explanation  string          `json:"explanation"`
		DisplayOrder []string        `json:"display_order"`
		UsedFor      []string        `json:"used_for"`
		Answer       json.RawMessage `json:"answer"`
		Table        *struct {
			Rows [][]string `json:"rows"`
		} `json:"table"`
		Placement struct {
			EducationLevel ref  `json:"education_level"`
			Grade          ref  `json:"grade"`
			SubjectField   ref  `json:"subject_field"`
			Chapter        ref  `json:"chapter"`
			Subchapter     *ref `json:"subchapter"`
			Topic          *ref `json:"topic"`
		} `json:"placement"`
		CreatedBy ref `json:"created_by"`
	}
	decodeData(t, res, &data)

	// Kunci jawaban: urutan field sama dengan saat create (options dulu, baru correct_key).
	wantAnswer := `{"options":[{"key":"A","content":"$\\frac{2}{6}$","image_id":null},{"key":"B","content":"$\\frac{3}{4}$","image_id":null},{"key":"C","content":"1","image_id":null}],"correct_key":"B"}`
	if data.Formula == nil || *data.Formula != `\frac{a}{b}` {
		t.Errorf("formula = %v, seharusnya \\frac{a}{b}", data.Formula)
	}
	if string(data.Answer) != wantAnswer {
		t.Errorf("answer = %s\nseharusnya %s", data.Answer, wantAnswer)
	}
	if !strings.HasPrefix(data.Explanation, "Samakan penyebut") {
		t.Errorf("explanation = %q", data.Explanation)
	}
	if data.ID != id || data.Type != "single_choice" || data.Formula == nil || data.Table == nil || len(data.Table.Rows) != 1 {
		t.Errorf("isi soal tidak lengkap: %+v", data)
	}
	if strings.Join(data.DisplayOrder, ",") != "question,formula,table" || strings.Join(data.UsedFor, ",") != "daily_quiz,uji_kom" {
		t.Errorf("display_order = %v, used_for = %v", data.DisplayOrder, data.UsedFor)
	}

	nameOf := func(r *ref) string {
		if r == nil || r.Name == nil {
			return "<null>"
		}
		return *r.Name
	}
	pl := data.Placement
	got := []string{nameOf(&pl.EducationLevel), nameOf(&pl.Grade), nameOf(&pl.SubjectField), nameOf(&pl.Chapter), nameOf(pl.Subchapter), nameOf(pl.Topic)}
	want := []string{"SD", "Kelas 4", "Bilangan", "KPK dan FPB", "Menentukan KPK dan FPB", "Menentukan KPK"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("nama placement = %v, seharusnya %v", got, want)
	}
	if pl.Grade.ID != p.GradeID || pl.Topic.ID != p.TopicID {
		t.Errorf("id placement tidak sesuai")
	}
	if nameOf(&data.CreatedBy) != "User admin.soal" {
		t.Errorf("created_by = %+v", data.CreatedBy)
	}
}

func TestGetQuestionDetail_TanpaSubBabDanTopik(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	id := createQuestionID(t, r, accessToken, questionBody(p, "free_text", freeTextTextAnswer, map[string]string{
		"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
			p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID),
	}))

	res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	body := res.Body.String()
	if !strings.Contains(body, `"subchapter":null`) || !strings.Contains(body, `"topic":null`) || !strings.Contains(body, `"table":null`) {
		t.Errorf("sub bab, topik, dan tabel seharusnya null. body: %s", body)
	}
	if !strings.Contains(body, `"answer":{"input_type":"text","accepted_answers":["kelipatan persekutuan"],"case_sensitive":false}`) {
		t.Errorf("answer free_text tidak sesuai. body: %s", body)
	}
}

func TestGetQuestionDetail_TidakDitemukanDanHakAkses(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)
	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))

	// Siswa & tutor tidak boleh melihat kunci jawaban.
	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403", role, res.Code)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("tanpa login: status = %d, seharusnya 401", res.Code)
	}

	// Soal yang sudah dihapus (soft delete) dianggap tidak ada.
	db.Exec(`UPDATE questions SET deleted_at = NOW() WHERE id = ?`, id)

	for _, badID := range []string{id, uuid.NewString(), "bukan-uuid"} {
		res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+badID, "", accessToken)
		if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "Soal tidak ditemukan") {
			t.Errorf("%s: status = %d, seharusnya 404. body: %s", badID, res.Code, res.Body)
		}
	}
}
