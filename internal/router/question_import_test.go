// Test endpoint POST /questions/import/check: cek file Excel upload soal (tidak menyimpan).
// File Excel dibuat dari template asli di docs/Template Upload/, jadi kalau template berubah
// (nama kolom, urutan), test ini ikut mendeteksinya.
package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

const importCheckURL = "/api/v1/questions/import/check"

// excelRow adalah isi 1 baris Excel: nama kolom → nilai sel.
type excelRow map[string]string

// buildExcel membuat file .xlsx dari template asli: baris contoh (baris 2 dst.) dihapus,
// lalu rows ditulis mulai baris 2. Baris nil = baris kosong (untuk mengetes baris yang dilewati).
func buildExcel(t *testing.T, questionType string, rows []excelRow, editHeader func(f *excelize.File)) []byte {
	t.Helper()
	f, err := excelize.OpenFile("../../docs/Template Upload/template_" + questionType + ".xlsx")
	if err != nil {
		t.Fatalf("gagal membuka template %s: %v", questionType, err)
	}
	defer f.Close()

	sheet := "Template"
	all, _ := f.GetRows(sheet)
	headers := all[0]
	for rowNum := len(all); rowNum >= 2; rowNum-- {
		if err := f.RemoveRow(sheet, rowNum); err != nil {
			t.Fatal(err)
		}
	}

	for i, row := range rows {
		for col, header := range headers {
			value, ok := row[header]
			if !ok {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(col+1, i+2)
			if err := f.SetCellStr(sheet, cell, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	if editHeader != nil {
		editHeader(f)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// sendExcel mengirim file lewat multipart (field "file").
func sendExcel(r *gin.Engine, fileName string, data []byte, accessToken string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	_, _ = part.Write(data)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, importCheckURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	return recorder
}

// placementRow mengisi kolom nama placement sesuai data seedPlacement (SD, Kelas 4, ...).
func placementRow(t *testing.T, db *gorm.DB, p testPlacement, withTopic bool) excelRow {
	t.Helper()
	var subjectName string
	db.Raw(`SELECT name FROM master_data_items WHERE id = ?`, p.SubjectID).Scan(&subjectName)
	row := excelRow{
		"education_level_name": "sd", // tidak peka huruf besar/kecil
		"grade_name":           "Kelas 4",
		"subject_name":         subjectName,
		"subject_field_name":   "Bilangan",
		"chapter_name":         "KPK dan FPB",
	}
	if withTopic {
		row["subchapter_name"] = "Menentukan KPK dan FPB"
		row["topic_name"] = "Menentukan KPK"
	}
	return row
}

// merge menggabungkan beberapa excelRow (yang belakang menimpa yang depan).
func merge(rows ...excelRow) excelRow {
	result := excelRow{}
	for _, r := range rows {
		for k, v := range r {
			result[k] = v
		}
	}
	return result
}

// baseRow adalah kolom umum yang valid (tanpa placement & jawaban).
var baseRow = excelRow{
	"question":      "KPK dari 4 dan 6 adalah ...",
	"display_order": "question",
	"explanation":   "Kelipatan persekutuan terkecil dari 4 dan 6 adalah 12",
	"difficulty":    "easy",
	"used_for":      "daily_quiz, uji_kom",
}

var singleChoiceCells = excelRow{
	"option_a_content": "2", "option_b_content": "12", "option_c_content": "24", "option_d_content": "10",
	"correct_key": "B",
}

// excelErrorList membaca daftar errors (dengan baris & kolom) dari response.
type excelErr struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Row     int    `json:"row"`
	Column  string `json:"column"`
}

func excelErrors(t *testing.T, body string) []excelErr {
	t.Helper()
	var parsed struct {
		Message string     `json:"message"`
		Errors  []excelErr `json:"errors"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("response bukan JSON: %v. body: %s", err, body)
	}
	return parsed.Errors
}

// hasExcelError mengecek ada error di sel (row, column).
func hasExcelError(list []excelErr, row int, column string) bool {
	for _, e := range list {
		if e.Row == row && e.Column == column {
			return true
		}
	}
	return false
}

func TestImportCheck_SingleChoiceValidLaluBisaLangsungDisimpan(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	rows := []excelRow{
		merge(baseRow, placementRow(t, db, p, true), singleChoiceCells, excelRow{
			"formula":       `\frac{a}{b}`,
			"image_link":    "https://cdn.example.com/kpk.png",
			"table_headers": "Bilangan|Kelipatan",
			"table_rows":    "4|4, 8, 12, 16\n6|6, 12, 18",
			"display_order": "question, table, formula, image",
		}),
		nil, // baris kosong dilewati
		merge(baseRow, placementRow(t, db, p, false), singleChoiceCells, excelRow{
			"question": "Soal kedua", "option_e_content": "36", "option_e_image_link": "https://cdn.example.com/e.png",
		}),
	}
	res := sendExcel(r, "soal.xlsx", buildExcel(t, "single_choice", rows, nil), accessToken)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "File valid: 2 soal siap disimpan") {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	var data struct {
		Type      string            `json:"type"`
		Total     int               `json:"total"`
		Questions []json.RawMessage `json:"questions"`
	}
	decodeData(t, res, &data)
	if data.Type != "single_choice" || data.Total != 2 || len(data.Questions) != 2 {
		t.Fatalf("type=%s total=%d", data.Type, data.Total)
	}

	var first struct {
		Row       int     `json:"row"`
		ImageURL  *string `json:"image_url"`
		Placement struct {
			GradeID        string  `json:"grade_id"`
			GradeName      string  `json:"grade_name"`
			EducationLevel string  `json:"education_level_name"`
			TopicID        *string `json:"topic_id"`
			TopicName      *string `json:"topic_name"`
		} `json:"placement"`
		Table struct {
			Rows [][]string `json:"rows"`
		} `json:"table"`
	}
	_ = json.Unmarshal(data.Questions[0], &first)
	if first.Row != 2 || first.Placement.GradeID != p.GradeID || first.Placement.GradeName != "Kelas 4" ||
		first.Placement.EducationLevel != "SD" || first.Placement.TopicID == nil || *first.Placement.TopicID != p.TopicID ||
		first.ImageURL == nil || len(first.Table.Rows) != 2 || first.Table.Rows[1][1] != "6, 12, 18" {
		t.Errorf("soal pertama tidak sesuai: %s", data.Questions[0])
	}
	var second struct {
		Row int `json:"row"`
	}
	_ = json.Unmarshal(data.Questions[1], &second)
	if second.Row != 4 {
		t.Errorf("row soal kedua = %d, seharusnya 4 (baris 3 kosong)", second.Row)
	}

	// Cek Excel tidak menyimpan apa pun.
	var count int64
	db.Raw(`SELECT COUNT(*) FROM questions`).Scan(&count)
	if count != 0 {
		t.Fatalf("cek Excel tidak boleh menyimpan soal, ada %d", count)
	}

	// Hasil cek bisa langsung dikirim ke create (field tambahan row & *_name diabaikan).
	createBody := fmt.Sprintf(`{"type":%q,"questions":[%s]}`, data.Type, strings.Join([]string{string(data.Questions[0]), string(data.Questions[1])}, ","))
	res = sendRequest(r, http.MethodPost, "/api/v1/questions", createBody, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("hasil cek Excel gagal disimpan lewat create: status %d. body: %s", res.Code, res.Body)
	}
}

func TestImportCheck_TipeLainValid(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		questionType string
		cells        excelRow
		wantAnswer   string
	}{
		{"multiple_choice", excelRow{
			"option_a_content": "2", "option_b_content": "4", "option_c_content": "7", "option_e_content": "11", "option_d_content": "9",
			"correct_keys": "a, c,E", "scoring": "all_or_nothing",
		}, `"correct_keys":["A","C","E"],"scoring":"all_or_nothing"`},
		{"true_false", excelRow{
			"statement_a_content": "2 prima", "statement_a_correct": "TRUE",
			"statement_b_content": "9 prima", "statement_b_correct": "false",
		}, `"correct_keys":[{"key":"A","correct":true},{"key":"B","correct":false}]`},
		{"free_text", excelRow{"input_type": "number", "accepted_answers": "0.75|0,75|3/4", "tolerance": "0,01"},
			`"answer":{"input_type":"number","accepted_answers":["0.75","0,75","3/4"],"tolerance":0.01}`},
		{"free_text", excelRow{"input_type": "text", "accepted_answers": "kelipatan|persekutuan"},
			`"answer":{"input_type":"text","accepted_answers":["kelipatan","persekutuan"],"case_sensitive":false}`},
	}
	for _, tt := range tests {
		rows := []excelRow{merge(baseRow, placementRow(t, db, p, false), tt.cells)}
		res := sendExcel(r, "soal.xlsx", buildExcel(t, tt.questionType, rows, nil), accessToken)
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), tt.wantAnswer) {
			t.Errorf("%s: status = %d, seharusnya 200 dengan %s. body: %s", tt.questionType, res.Code, tt.wantAnswer, res.Body)
		}
	}
}

func TestImportCheck_SemuaErrorDenganBarisDanKolom(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	rows := []excelRow{
		// Baris 2: kunci tidak ada, kesulitan salah, opsi D (wajib di template) kosong, used_for ke-2 salah.
		merge(baseRow, placementRow(t, db, p, false), singleChoiceCells, excelRow{
			"correct_key": "F", "difficulty": "sulit", "option_d_content": "", "used_for": "daily_quiz,ujian",
		}),
		// Baris 3: kelas tidak ada di jenjang SD, tabel tanpa baris, display_order kosong.
		merge(baseRow, placementRow(t, db, p, false), singleChoiceCells, excelRow{
			"grade_name": "Kelas 9", "table_headers": "A|B", "display_order": "",
		}),
		// Baris 4: valid, tapi tetap ditolak karena ada baris lain yang salah.
		merge(baseRow, placementRow(t, db, p, false), singleChoiceCells),
		// Baris 5: bab tidak ada, link gambar salah.
		merge(baseRow, placementRow(t, db, p, false), singleChoiceCells, excelRow{
			"chapter_name": "Bab Tidak Ada", "image_link": "bukan-link",
		}),
	}
	res := sendExcel(r, "soal.xlsx", buildExcel(t, "single_choice", rows, nil), accessToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), "kesalahan, semua data ditolak") {
		t.Errorf("pesan tidak sesuai. body: %s", res.Body)
	}

	list := excelErrors(t, res.Body.String())
	want := []struct {
		row    int
		column string
	}{
		{2, "O"},  // difficulty
		{2, "P"},  // used_for (nilai ke-2)
		{2, "X"},  // option_d_content
		{2, "AB"}, // correct_key
		{3, "E"},  // table_rows
		{3, "F"},  // display_order
		{3, "I"},  // grade_name
		{5, "C"},  // image_link
		{5, "L"},  // chapter_name
	}
	for _, w := range want {
		if !hasExcelError(list, w.row, w.column) {
			t.Errorf("error di %s%d tidak ada. errors: %+v", w.column, w.row, list)
		}
	}
	for _, e := range list {
		if e.Row == 4 {
			t.Errorf("baris 4 valid, tidak boleh ada error: %+v", e)
		}
	}
	// Urut per baris lalu kolom.
	for i := 1; i < len(list); i++ {
		if list[i].Row < list[i-1].Row {
			t.Errorf("error tidak urut per baris: %+v", list)
			break
		}
	}
}

func TestImportCheck_TrueFalseNilaiBenarSalahTidakValid(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	rows := []excelRow{merge(baseRow, placementRow(t, db, p, false), excelRow{
		"statement_a_content": "2 prima", "statement_a_correct": "YA",
		"statement_b_content": "9 prima", // statement_b_correct kosong
	})}
	res := sendExcel(r, "soal.xlsx", buildExcel(t, "true_false", rows, nil), accessToken)
	list := excelErrors(t, res.Body.String())

	// T2 = statement_a_correct, W2 = statement_b_correct. Masing-masing cukup 1 pesan.
	got := map[string][]string{}
	for _, e := range list {
		got[fmt.Sprintf("%s%d", e.Column, e.Row)] = append(got[fmt.Sprintf("%s%d", e.Column, e.Row)], e.Message)
	}
	if len(got["T2"]) != 1 || got["T2"][0] != "harus TRUE atau FALSE" || len(got["W2"]) != 1 {
		t.Errorf("error tidak sesuai: %+v", got)
	}
}

func TestImportCheck_MasalahFile(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)
	valid := merge(baseRow, placementRow(t, db, p, false), singleChoiceCells)

	tooMany := make([]excelRow, 101)
	for i := range tooMany {
		tooMany[i] = valid
	}

	tests := []struct {
		name     string
		fileName string
		data     []byte
		want     string
	}{
		{"bukan file xlsx", "soal.csv", []byte("a,b,c"), "harus file .xlsx"},
		{"isi bukan Excel", "soal.xlsx", []byte("bukan excel"), "File bukan Excel (.xlsx) yang valid"},
		{"tanpa data", "soal.xlsx", buildExcel(t, "single_choice", nil, nil), "Belum ada soal di file"},
		{"lebih dari 100 soal", "soal.xlsx", buildExcel(t, "single_choice", tooMany, nil), "Maksimal 100 soal per file"},
		{"header diubah", "soal.xlsx", buildExcel(t, "single_choice", []excelRow{valid}, func(f *excelize.File) {
			_ = f.SetCellStr("Template", "A1", "soal")
		}), `kolom \"question\" tidak ada`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendExcel(r, tt.fileName, tt.data, accessToken)
			if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), tt.want) {
				t.Errorf("status = %d, seharusnya 422 dengan %q. body: %s", res.Code, tt.want, res.Body)
			}
		})
	}

	// Tanpa file sama sekali.
	req := httptest.NewRequest(http.MethodPost, importCheckURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), `"field":"file"`) {
		t.Errorf("tanpa file: status = %d. body: %s", rec.Code, rec.Body)
	}

	// Siswa tidak boleh.
	if res := sendExcel(r, "soal.xlsx", []byte("x"), tokenFor(t, loginmodel.RoleStudent)); res.Code != http.StatusForbidden {
		t.Errorf("siswa: status = %d, seharusnya 403", res.Code)
	}
}
