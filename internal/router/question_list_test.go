// Test endpoint GET /questions (list Bank Soal): isi kolom, urutan, filter, search, pagination.
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// listItem adalah bentuk 1 baris data GET /questions yang dicek di test.
type listItem struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Type     string `json:"type"`
	Question string `json:"question"`
	Media    struct {
		Formula bool `json:"formula"`
		Image   bool `json:"image"`
		Table   bool `json:"table"`
	} `json:"media"`
	Placement struct {
		Subject      listRef  `json:"subject"`
		SubjectField listRef  `json:"subject_field"`
		Chapter      listRef  `json:"chapter"`
		Grade        listRef  `json:"grade"`
		Topic        *listRef `json:"topic"`
	} `json:"placement"`
	Difficulty string   `json:"difficulty"`
	UsedFor    []string `json:"used_for"`
	Status     string   `json:"status"`
	CreatedBy  listRef  `json:"created_by"`
	UpdatedBy  *listRef `json:"updated_by"`
	Answer     any      `json:"answer"` // tidak boleh ada di list
}

type listRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// listQuestions memanggil GET /questions?query lalu mengembalikan data & meta. Status harus 200.
func listQuestions(t *testing.T, r *gin.Engine, accessToken string, query url.Values) ([]listItem, listMeta) {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/questions?"+query.Encode(), "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("GET /questions?%s: status = %d, seharusnya 200. body: %s", query.Encode(), res.Code, res.Body)
	}
	var body struct {
		Data []listItem `json:"data"`
		Meta listMeta   `json:"meta"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response bukan JSON yang valid: %v. body: %s", err, res.Body)
	}
	if body.Data == nil {
		t.Fatalf("data seharusnya array (bukan null). body: %s", res.Body)
	}
	return body.Data, body.Meta
}

// codesOf mengambil kode soal dari hasil list, contoh [Q-3 Q-2].
func codesOf(items []listItem) []string {
	codes := make([]string, 0, len(items))
	for _, it := range items {
		codes = append(codes, it.Code)
	}
	return codes
}

// createQuestionCodes membuat soal lewat API lalu mengembalikan kode-kodenya (urut sesuai request).
func createQuestionCodes(t *testing.T, r *gin.Engine, accessToken, body string) []string {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/questions", body, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("gagal membuat soal: status %d. body: %s", res.Code, res.Body)
	}
	var list []struct {
		Code string `json:"code"`
	}
	decodeData(t, res, &list)
	codes := make([]string, 0, len(list))
	for _, q := range list {
		codes = append(codes, q.Code)
	}
	return codes
}

func TestListQuestions_IsiKolomDanUrutanTerbaru(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	// 3 soal dalam 1 request (created_at sama persis) → urutan tetap mengikuti request.
	batch := createQuestionCodes(t, r, accessToken, questionsBody("single_choice",
		questionItem(p, singleChoiceAnswer, map[string]string{"question": `"Soal pertama"`}),
		questionItem(p, singleChoiceAnswer, map[string]string{
			"question":      `"Soal kedua"`,
			"formula":       `"x^2"`,
			"table":         `{"headers":["a"],"rows":[["1"]]}`,
			"display_order": `["question","formula","table"]`,
		}),
		questionItem(p, singleChoiceAnswer, map[string]string{
			"question":      `"Soal ketiga"`,
			"image_url":     `"https://cdn.example.com/soal.png"`,
			"display_order": `["question","image"]`,
			"difficulty":    `"hard"`,
		}),
	))
	// Soal terakhir dibuat terpisah, lalu diubah (updated_by terisi).
	lastID := createQuestionID(t, r, accessToken, questionBody(p, "free_text", freeTextTextAnswer,
		map[string]string{"question": `"Soal keempat"`}))
	res := sendRequest(r, http.MethodPut, "/api/v1/questions/"+lastID,
		updateBody(p, "free_text", freeTextTextAnswer, map[string]string{"question": `"Soal keempat (revisi)"`}), accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("update: status = %d. body: %s", res.Code, res.Body)
	}

	items, meta := listQuestions(t, r, accessToken, url.Values{})

	if meta.Page != 1 || meta.PerPage != 20 || meta.Total != 4 {
		t.Errorf("meta = %+v, seharusnya page 1, per_page 20, total 4", meta)
	}
	if len(items) != 4 {
		t.Fatalf("jumlah data = %d, seharusnya 4", len(items))
	}

	// Terbaru dulu: soal keempat, lalu batch dari belakang (ketiga, kedua, pertama).
	wantCodes := []string{items[0].Code, batch[2], batch[1], batch[0]}
	if got := codesOf(items); strings.Join(got, ",") != strings.Join(wantCodes, ",") || items[0].ID != lastID {
		t.Errorf("urutan kode = %v, seharusnya %v (soal %s paling atas)", got, wantCodes, lastID)
	}

	newest, third, second, first := items[0], items[1], items[2], items[3]

	if newest.Question != "Soal keempat (revisi)" || newest.Type != "free_text" ||
		newest.UpdatedBy == nil || newest.UpdatedBy.Name != "User admin.soal" {
		t.Errorf("soal yang diubah: %+v, seharusnya teks revisi & updated_by terisi", newest)
	}
	if first.UpdatedBy != nil {
		t.Errorf("soal yang belum pernah diubah: updated_by = %+v, seharusnya null", first.UpdatedBy)
	}

	// Penanda media di soal.
	if first.Media.Formula || first.Media.Image || first.Media.Table {
		t.Errorf("soal pertama tanpa media: %+v", first.Media)
	}
	if !second.Media.Formula || !second.Media.Table || second.Media.Image {
		t.Errorf("soal kedua (rumus + tabel): %+v", second.Media)
	}
	if !third.Media.Image || third.Media.Formula || third.Media.Table || third.Difficulty != "hard" {
		t.Errorf("soal ketiga (gambar, sulit): media %+v, difficulty %s", third.Media, third.Difficulty)
	}

	// Hierarki lengkap dengan nama, pembuat, peruntukan; kunci jawaban tidak ikut.
	if first.Placement.Subject.ID != p.SubjectID || !strings.HasPrefix(first.Placement.Subject.Name, "Matematika") ||
		first.Placement.SubjectField.Name != "Bilangan" || first.Placement.Chapter.Name != "KPK dan FPB" ||
		first.Placement.Grade.ID != p.GradeID || first.Placement.Topic == nil || first.Placement.Topic.Name != "Menentukan KPK" {
		t.Errorf("placement tidak lengkap: %+v", first.Placement)
	}
	if first.CreatedBy.Name != "User admin.soal" || strings.Join(first.UsedFor, ",") != "daily_quiz,uji_kom" ||
		first.Status != "active" || first.Answer != nil {
		t.Errorf("kolom lain salah: created_by %+v, used_for %v, status %s, answer %v",
			first.CreatedBy, first.UsedFor, first.Status, first.Answer)
	}
}

func TestListQuestions_FilterDanSearch(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	other := seedPlacement(t, db) // mapel lain
	accessToken := adminToken(t, db)

	easySC := createQuestionCodes(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"question": `"Perhatikan GARIS bilangan berikut"`}))[0]
	hardTF := createQuestionCodes(t, r, accessToken, questionBody(p, "true_false", trueFalseAnswer,
		map[string]string{"question": `"Diskon 50% dari harga"`, "difficulty": `"hard"`}))[0]
	// Tanpa sub bab & topik.
	noTopic := createQuestionCodes(t, r, accessToken, questionBody(p, "free_text", freeTextNumberAnswer,
		map[string]string{"question": `"Ibu kota provinsi"`,
			"placement": fmt.Sprintf(`{"education_level_id":%q,"grade_id":%q,"subject_id":%q,"subject_field_id":%q,"chapter_id":%q}`,
				p.LevelID, p.GradeID, p.SubjectID, p.FieldID, p.ChapterID)}))[0]
	otherSubject := createQuestionCodes(t, r, accessToken, questionBody(other, "single_choice", singleChoiceAnswer,
		map[string]string{"question": `"Garis waktu sejarah"`}))[0]

	// Soal yang dihapus tidak pernah muncul.
	deletedID := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer,
		map[string]string{"question": `"Garis yang sudah dihapus"`}))
	if res := sendRequest(r, http.MethodDelete, "/api/v1/questions", fmt.Sprintf(`{"ids":[%q]}`, deletedID), accessToken); res.Code != http.StatusOK {
		t.Fatalf("hapus soal: status = %d. body: %s", res.Code, res.Body)
	}

	cases := []struct {
		name  string
		query url.Values
		want  []string // kode soal, urut terbaru dulu
	}{
		{"tanpa filter (soal terhapus tidak ikut)", url.Values{}, []string{otherSubject, noTopic, hardTF, easySC}},
		{"type", url.Values{"type": {"single_choice"}}, []string{otherSubject, easySC}},
		{"difficulty (level)", url.Values{"difficulty": {"hard"}}, []string{hardTF}},
		{"mapel", url.Values{"subject_id": {other.SubjectID}}, []string{otherSubject}},
		{"bidang materi", url.Values{"subject_field_id": {p.FieldID}}, []string{noTopic, hardTF, easySC}},
		{"bab", url.Values{"chapter_id": {p.ChapterID}}, []string{noTopic, hardTF, easySC}},
		{"sub bab", url.Values{"subchapter_id": {p.SubchapterID}}, []string{hardTF, easySC}},
		{"topik", url.Values{"topic_id": {p.TopicID}}, []string{hardTF, easySC}},
		{"jenjang + kelas", url.Values{"education_level_id": {p.LevelID}, "grade_id": {p.GradeID}}, []string{otherSubject, noTopic, hardTF, easySC}},
		{"kelas lain", url.Values{"grade_id": {p.OtherGradeID}}, []string{}},
		{"search teks, tidak peka huruf besar/kecil", url.Values{"search": {"garis"}}, []string{otherSubject, easySC}},
		{"search kode", url.Values{"search": {strings.ToLower(hardTF)}}, []string{hardTF}},
		{"search % dicari apa adanya", url.Values{"search": {"50%"}}, []string{hardTF}},
		{"search _ bukan wildcard", url.Values{"search": {"_"}}, []string{}},
		{"search \\ dicari apa adanya (bukan karakter escape)", url.Values{"search": {`\o`}}, []string{}},
		{"search + filter digabung", url.Values{"search": {"  garis  "}, "subject_id": {p.SubjectID}}, []string{easySC}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items, meta := listQuestions(t, r, accessToken, tc.query)
			got := codesOf(items)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") || meta.Total != len(tc.want) {
				t.Errorf("kode = %v (total %d), seharusnya %v", got, meta.Total, tc.want)
			}
		})
	}
}

func TestListQuestions_PaginationValidasiDanHakAkses(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	items := make([]string, 0, 5)
	for i := 1; i <= 5; i++ {
		items = append(items, questionItem(p, multipleChoiceAnswer, map[string]string{"question": fmt.Sprintf(`"Soal %d"`, i)}))
	}
	codes := createQuestionCodes(t, r, accessToken, questionsBody("multiple_choice", items...))
	// Terbaru dulu = kebalikan urutan request.
	newestFirst := []string{codes[4], codes[3], codes[2], codes[1], codes[0]}

	pages := []struct {
		page string
		want []string
	}{
		{"1", newestFirst[0:2]},
		{"2", newestFirst[2:4]},
		{"3", newestFirst[4:5]}, // halaman terakhir (tidak penuh)
		{"9", []string{}},       // lewat dari halaman terakhir: kosong, total tetap 5
	}
	for _, pg := range pages {
		got, meta := listQuestions(t, r, accessToken, url.Values{"page": {pg.page}, "per_page": {"2"}})
		if strings.Join(codesOf(got), ",") != strings.Join(pg.want, ",") || meta.Total != 5 || meta.PerPage != 2 {
			t.Errorf("page %s: kode %v meta %+v, seharusnya %v dengan total 5", pg.page, codesOf(got), meta, pg.want)
		}
	}

	// Query tidak valid: semua error dikembalikan sekaligus, per field.
	// (page=0 dianggap kosong → default 1, sama seperti pagination endpoint lain.)
	query := "page=-1&per_page=101&type=essay&difficulty=sulit&chapter_id=bukan-uuid&search=" + strings.Repeat("a", 101)
	res := sendRequest(r, http.MethodGet, "/api/v1/questions?"+query, "", accessToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("query tidak valid: status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	fields := errorFields(t, res.Body.String())
	for _, field := range []string{"page", "per_page", "type", "difficulty", "chapter_id", "search"} {
		if _, ok := fields[field]; !ok {
			t.Errorf("seharusnya ada error untuk field %s. errors: %v", field, fields)
		}
	}

	// Hanya admin & super admin.
	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/questions", "", tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("role %s: status = %d, seharusnya 403", role, res.Code)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/questions", "", tokenFor(t, loginmodel.RoleSuperAdmin)); res.Code != http.StatusOK {
		t.Errorf("super admin: status = %d, seharusnya 200", res.Code)
	}
}

// typeCounts memanggil GET /questions/type-counts?query. Status harus 200.
func typeCounts(t *testing.T, r *gin.Engine, accessToken string, query url.Values) map[string]int {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/questions/type-counts?"+query.Encode(), "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("type-counts?%s: status = %d, seharusnya 200. body: %s", query.Encode(), res.Code, res.Body)
	}
	counts := map[string]int{}
	decodeData(t, res, &counts)
	return counts
}

func TestQuestionTypeCounts_JumlahPerTipeSesuaiFilter(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	other := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	// Belum ada soal: semua tipe tetap ada dengan nilai 0.
	if got := typeCounts(t, r, accessToken, url.Values{}); fmt.Sprint(got) !=
		"map[all:0 free_text:0 multiple_choice:0 single_choice:0 true_false:0]" {
		t.Errorf("tanpa soal: %v, seharusnya semua 0", got)
	}

	createQuestionCodes(t, r, accessToken, questionsBody("single_choice",
		questionItem(p, singleChoiceAnswer, map[string]string{"question": `"Garis bilangan"`}),
		questionItem(p, singleChoiceAnswer, nil),
		questionItem(other, singleChoiceAnswer, map[string]string{"question": `"Garis waktu"`}),
	))
	createQuestionCodes(t, r, accessToken, questionsBody("multiple_choice",
		questionItem(p, multipleChoiceAnswer, map[string]string{"difficulty": `"hard"`})))
	createQuestionCodes(t, r, accessToken, questionsBody("free_text",
		questionItem(p, freeTextNumberAnswer, map[string]string{"question": `"Garis tengah lingkaran"`}),
		questionItem(other, freeTextTextAnswer, nil),
	))
	// Soal terhapus tidak dihitung.
	deletedID := createQuestionID(t, r, accessToken, questionBody(p, "true_false", trueFalseAnswer, nil))
	sendRequest(r, http.MethodDelete, "/api/v1/questions", fmt.Sprintf(`{"ids":[%q]}`, deletedID), accessToken)

	cases := []struct {
		name  string
		query url.Values
		want  string
	}{
		{"semua soal", url.Values{},
			"map[all:6 free_text:2 multiple_choice:1 single_choice:3 true_false:0]"},
		{"filter mapel", url.Values{"subject_id": {other.SubjectID}},
			"map[all:2 free_text:1 multiple_choice:0 single_choice:1 true_false:0]"},
		{"search", url.Values{"search": {"garis"}},
			"map[all:3 free_text:1 multiple_choice:0 single_choice:2 true_false:0]"},
		{"level", url.Values{"difficulty": {"hard"}},
			"map[all:1 free_text:0 multiple_choice:1 single_choice:0 true_false:0]"},
		// Query string list dikirim apa adanya: type & pagination diabaikan.
		{"type & page diabaikan", url.Values{"type": {"free_text"}, "page": {"5"}, "per_page": {"1"}, "subject_id": {p.SubjectID}},
			"map[all:4 free_text:1 multiple_choice:1 single_choice:2 true_false:0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprint(typeCounts(t, r, accessToken, tc.query)); got != tc.want {
				t.Errorf("jumlah = %s, seharusnya %s", got, tc.want)
			}
		})
	}

	// Validasi sama dengan list.
	res := sendRequest(r, http.MethodGet, "/api/v1/questions/type-counts?chapter_id=bukan-uuid&difficulty=sulit", "", accessToken)
	fields := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || fields["chapter_id"] == "" || fields["difficulty"] == "" {
		t.Errorf("query tidak valid: status = %d, errors %v", res.Code, fields)
	}

	// Hanya admin & super admin.
	if res := sendRequest(r, http.MethodGet, "/api/v1/questions/type-counts", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor: status = %d, seharusnya 403", res.Code)
	}
}
