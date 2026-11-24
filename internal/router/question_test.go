// Test endpoint POST /questions: membuat soal (khusus admin & super admin).
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// testPlacement berisi ID master data untuk letak soal: Matematika → Bilangan (SD, Kelas 4) → bab → sub bab → topik.
type testPlacement struct {
	LevelID, GradeID, OtherGradeID, SubjectID, FieldID, ChapterID, SubchapterID, TopicID string
}

// seedPlacement membuat hierarki mapel di database test. Nama diberi akhiran acak supaya
// tidak bentrok dengan test lain (tabel master_data_items tidak dikosongkan antar test).
func seedPlacement(t *testing.T, db *gorm.DB) testPlacement {
	t.Helper()
	var p testPlacement
	suffix := uuid.NewString()[:8]

	mustScan := func(target *string, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(target).Error; err != nil || *target == "" {
			t.Fatalf("gagal menyiapkan master data: %v\nSQL: %s", err, sql)
		}
	}
	mustScan(&p.GradeID, `SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_4'`)
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
		db.Exec(`DELETE FROM questions`)
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

// questionBody membuat body request soal. extra menimpa/menambah field level atas (nilai JSON mentah).
func questionBody(p testPlacement, questionType, answer string, extra map[string]string) string {
	fields := map[string]string{
		"type":     `"` + questionType + `"`,
		"question": `"Hasil dari $\\frac{1}{2} + \\frac{1}{4}$ adalah ..."`,
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
	freeTextStringAnswer = `{"input_type":"string","accepted_answers":["kelipatan persekutuan"]}`
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
		{"free_text", freeTextStringAnswer,
			`{"input_type":"string","accepted_answers":["kelipatan persekutuan"],"case_sensitive":false}`},
	}

	for _, tt := range tests {
		res := sendRequest(r, http.MethodPost, "/api/v1/questions", questionBody(p, tt.questionType, tt.answer, nil), accessToken)
		if res.Code != http.StatusCreated {
			t.Errorf("%s: status = %d, seharusnya 201. body: %s", tt.questionType, res.Code, res.Body)
			continue
		}

		var data struct {
			Code         string          `json:"code"`
			Status       string          `json:"status"`
			DisplayOrder []string        `json:"display_order"`
			Answer       json.RawMessage `json:"answer"`
		}
		decodeData(t, res, &data)
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
	db.Raw(`INSERT INTO media_assets (file_name, storage_path, mime_type, size_bytes) VALUES ('a.png', 'img/a.png', 'image/png', 100) RETURNING id`).Scan(&imageID)

	body := questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"formula":       `"\\frac{a}{b}"`,
		"image_id":      fmt.Sprintf("%q", imageID),
		"table":         `{"headers":["Nama","Nilai"],"rows":[["Andi","80"],["Budi","75"]]}`,
		"display_order": `["question","table","formula","image"]`,
	})
	res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, seharusnya 201. body: %s", res.Code, res.Body)
	}
}

func TestCreateQuestion_ValidasiField(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		name       string
		body       string
		wantFields []string // field yang harus muncul di errors
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
