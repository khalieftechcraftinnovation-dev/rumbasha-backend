// Test endpoint PUT /questions/:id (ubah 1 soal) dan DELETE /questions (soft delete banyak soal).
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// updateBody membuat body PUT: 1 soal + "type" wajib.
func updateBody(p testPlacement, questionType, answer string, extra map[string]string) string {
	fields := map[string]string{"type": fmt.Sprintf("%q", questionType)}
	for k, v := range extra {
		fields[k] = v
	}
	return questionItem(p, answer, fields)
}

func TestUpdateQuestion_GantiTipeIsiDanPlacement(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	// Awal: single choice lengkap sampai topik, dengan tabel.
	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"table": `{"headers":["A"],"rows":[["1"]]}`,
	}))
	var codeBefore string
	db.Raw(`SELECT code FROM questions WHERE id = ?`, id).Scan(&codeBefore)

	// Ubah: jadi free text, tanpa tabel, placement hanya sampai bab, status inactive.
	body := updateBody(p, "free_text", freeTextNumberAnswer, map[string]string{
		"question":    `"Soal sudah diubah"`,
		"explanation": `"Pembahasan baru"`,
		"status":      `"inactive"`,
		"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
			p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID),
	})
	res := sendRequest(r, http.MethodPut, "/api/v1/questions/"+id, body, accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	var data struct {
		ID          string `json:"id"`
		Code        string `json:"code"`
		Type        string `json:"type"`
		Question    string `json:"question"`
		Explanation string `json:"explanation"`
		Status      string `json:"status"`
		Table       any    `json:"table"`
		Placement   struct {
			Subchapter any `json:"subchapter"`
			Topic      any `json:"topic"`
		} `json:"placement"`
		UpdatedBy *struct {
			Name *string `json:"name"`
		} `json:"updated_by"`
	}
	decodeData(t, res, &data)

	if data.ID != id || data.Code != codeBefore {
		t.Errorf("id/code tidak boleh berubah: id=%s code=%s (sebelumnya %s)", data.ID, data.Code, codeBefore)
	}
	if data.Type != "free_text" || data.Question != "Soal sudah diubah" || data.Explanation != "Pembahasan baru" || data.Status != "inactive" {
		t.Errorf("isi tidak berubah: %+v", data)
	}
	// Ganti penuh: field opsional yang tidak dikirim menjadi kosong.
	if data.Table != nil || data.Placement.Subchapter != nil || data.Placement.Topic != nil {
		t.Errorf("table/subchapter/topic seharusnya null: %+v", data)
	}
	if data.UpdatedBy == nil || data.UpdatedBy.Name == nil || *data.UpdatedBy.Name != "User admin.soal" {
		t.Errorf("updated_by tidak terisi: %+v", data.UpdatedBy)
	}

	var answer string
	db.Raw(`SELECT answer::text FROM questions WHERE id = ?`, id).Scan(&answer)
	if !strings.Contains(answer, `"input_type": "number"`) {
		t.Errorf("answer di database belum berubah: %s", answer)
	}
}

func TestUpdateQuestion_Validasi(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)
	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))

	tests := []struct {
		name       string
		body       string
		wantFields []string // field error TANPA awalan questions[i]. (update hanya 1 soal)
	}{
		{
			name:       "type wajib",
			body:       questionItem(p, singleChoiceAnswer, nil),
			wantFields: []string{"type"},
		},
		{
			name:       "type tidak dikenal",
			body:       updateBody(p, "essay", singleChoiceAnswer, nil),
			wantFields: []string{"type"},
		},
		{
			name: "isi answer & pembahasan salah",
			body: updateBody(p, "single_choice", `{"options":[{"key":"A","content":"1"},{"key":"B","content":"2"}],"correct_key":"E"}`,
				map[string]string{"explanation": `"  "`}),
			wantFields: []string{"answer.correct_key", "explanation"},
		},
		{
			name: "placement tidak cocok",
			body: updateBody(p, "single_choice", singleChoiceAnswer, map[string]string{
				"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
					p.LevelID, p.OtherGradeID, p.SubjectID, p.FieldID, p.ChapterID),
			}),
			wantFields: []string{"placement.grade_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendRequest(r, http.MethodPut, "/api/v1/questions/"+id, tt.body, accessToken)
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

	// Soal tidak boleh berubah sedikit pun setelah request yang gagal.
	var updatedBy *string
	db.Raw(`SELECT updated_by FROM questions WHERE id = ?`, id).Scan(&updatedBy)
	if updatedBy != nil {
		t.Errorf("soal tidak boleh berubah setelah validasi gagal")
	}
}

func TestUpdateQuestion_TidakDitemukanDanHakAkses(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)
	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	body := updateBody(p, "single_choice", singleChoiceAnswer, nil)

	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		if res := sendRequest(r, http.MethodPut, "/api/v1/questions/"+id, body, tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403", role, res.Code)
		}
	}

	db.Exec(`UPDATE questions SET deleted_at = NOW() WHERE id = ?`, id)
	for _, badID := range []string{id, uuid.NewString(), "bukan-uuid"} {
		res := sendRequest(r, http.MethodPut, "/api/v1/questions/"+badID, body, accessToken)
		if res.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, seharusnya 404. body: %s", badID, res.Code, res.Body)
		}
	}
}

func TestDeleteQuestions_BanyakSekaligusSoftDelete(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	id1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	id2 := createQuestionID(t, r, accessToken, questionBody(p, "free_text", freeTextNumberAnswer, nil))
	id3 := createQuestionID(t, r, accessToken, questionBody(p, "true_false", trueFalseAnswer, nil))

	body := fmt.Sprintf(`{"ids":[%q,%q]}`, id1, id2)
	res := sendRequest(r, http.MethodDelete, "/api/v1/questions", body, accessToken)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"2 soal berhasil dihapus"`) {
		t.Fatalf("status = %d, seharusnya 200 dengan pesan 2 soal. body: %s", res.Code, res.Body)
	}

	// Soft delete: baris tetap ada, deleted_at & deleted_by terisi.
	var rows []struct {
		ID        string
		Deleted   bool
		DeletedBy *string
	}
	db.Raw(`SELECT id, deleted_at IS NOT NULL AS deleted, deleted_by FROM questions WHERE id IN ?`,
		[]string{id1, id2, id3}).Scan(&rows)
	if len(rows) != 3 {
		t.Fatalf("baris soal seharusnya tetap ada (3), ada %d", len(rows))
	}
	for _, row := range rows {
		wantDeleted := row.ID != id3
		if row.Deleted != wantDeleted || (wantDeleted && row.DeletedBy == nil) {
			t.Errorf("soal %s: deleted=%v deleted_by=%v, seharusnya deleted=%v", row.ID, row.Deleted, row.DeletedBy, wantDeleted)
		}
	}

	// Soal yang dihapus tidak bisa dibuka lagi; yang tidak dihapus tetap bisa.
	if res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id1, "", accessToken); res.Code != http.StatusNotFound {
		t.Errorf("detail soal terhapus: status = %d, seharusnya 404", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id3, "", accessToken); res.Code != http.StatusOK {
		t.Errorf("detail soal lain: status = %d, seharusnya 200", res.Code)
	}
}

func TestDeleteQuestions_SemuaAtauTidakSamaSekali(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	id1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	deletedID := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	db.Exec(`UPDATE questions SET deleted_at = NOW() WHERE id = ?`, deletedID)

	ids101 := make([]string, 101)
	for i := range ids101 {
		ids101[i] = fmt.Sprintf("%q", uuid.NewString())
	}

	tests := []struct {
		name       string
		body       string
		wantFields []string
	}{
		{"ID tidak ada & sudah dihapus", fmt.Sprintf(`{"ids":[%q,%q,%q]}`, id1, uuid.NewString(), deletedID), []string{"ids[1]", "ids[2]"}},
		{"daftar kosong", `{"ids":[]}`, []string{"ids"}},
		{"lebih dari 100", `{"ids":[` + strings.Join(ids101, ",") + `]}`, []string{"ids"}},
		{"bukan UUID", `{"ids":["abc"]}`, []string{"ids[0]"}},
		{"ID kembar", fmt.Sprintf(`{"ids":[%q,%q]}`, id1, id1), []string{"ids"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendRequest(r, http.MethodDelete, "/api/v1/questions", tt.body, accessToken)
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

	// id1 tidak boleh ikut terhapus karena request-nya gagal.
	var deleted bool
	db.Raw(`SELECT deleted_at IS NOT NULL FROM questions WHERE id = ?`, id1).Scan(&deleted)
	if deleted {
		t.Error("soal id1 tidak boleh terhapus karena ada ID lain yang salah")
	}

	// Siswa & tutor tidak boleh menghapus.
	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		res := sendRequest(r, http.MethodDelete, "/api/v1/questions", fmt.Sprintf(`{"ids":[%q]}`, id1), tokenFor(t, role))
		if res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403", role, res.Code)
		}
	}
}
