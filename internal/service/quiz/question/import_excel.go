package question

// File ini berisi pengecekan file Excel upload soal (TIDAK menyimpan ke database).
//
// Alur CheckExcel:
//  1. Buka file, ambil sheet "Template" (atau sheet pertama).
//  2. Baca header baris 1 → tentukan tipe soal dari kolom khasnya, cek kolom lengkap.
//  3. Ambil semua nama placement di file dengan 1 query.
//  4. Per baris: ubah sel menjadi request create (CreateQuestionRequest), lalu cek dengan
//     aturan yang SAMA PERSIS dengan create (tag binding + buildQuestion).
//  5. Setiap error dikembalikan dengan nomor baris & huruf kolom Excel-nya.
//     Ada 1 error saja → semua data ditolak.
//
// Template ada di docs/Template Upload/template_<tipe>.xlsx.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/validation"
)

// MaxExcelRows adalah batas jumlah soal per file (sama dengan batas create banyak soal).
const MaxExcelRows = 100

// excelSheetName adalah nama sheet data di template. Kalau tidak ada, dipakai sheet pertama.
const excelSheetName = "Template"

// ----------------------------------------------------------------------------
// Definisi template
// ----------------------------------------------------------------------------

// optionLetters adalah huruf opsi di template (A sampai E).
var optionLetters = []string{"a", "b", "c", "d", "e"}

// commonHeaders adalah kolom yang sama di semua template.
var commonHeaders = []string{
	"question", "formula", "image_link", "table_headers", "table_rows", "display_order", "explanation",
	"education_level_name", "grade_name", "subject_name", "subject_field_name", "chapter_name",
	"subchapter_name", "topic_name", "difficulty", "used_for", "status",
}

// excelTemplate menjelaskan 1 template: kolom khusus tipenya & jumlah opsi wajib.
type excelTemplate struct {
	Type          string
	OptionPrefix  string   // "option" atau "statement"; kosong untuk free_text
	RequiredCount int      // jumlah opsi wajib dari depan (contoh single choice: A-D = 4)
	ExtraHeaders  []string // kolom setelah opsi
}

// excelTemplates mengikuti kolom "Wajib" di sheet Petunjuk masing-masing template.
var excelTemplates = map[string]excelTemplate{
	questionmodel.TypeSingleChoice:   {Type: questionmodel.TypeSingleChoice, OptionPrefix: "option", RequiredCount: 4, ExtraHeaders: []string{"correct_key"}},
	questionmodel.TypeMultipleChoice: {Type: questionmodel.TypeMultipleChoice, OptionPrefix: "option", RequiredCount: 3, ExtraHeaders: []string{"correct_keys", "scoring"}},
	questionmodel.TypeTrueFalse:      {Type: questionmodel.TypeTrueFalse, OptionPrefix: "statement", RequiredCount: 2},
	questionmodel.TypeFreeText:       {Type: questionmodel.TypeFreeText, ExtraHeaders: []string{"input_type", "accepted_answers", "tolerance"}},
}

// headers mengembalikan semua kolom template sesuai urutan di file.
func (t excelTemplate) headers() []string {
	result := append([]string{}, commonHeaders...)
	for _, letter := range optionLetters {
		if t.OptionPrefix == "" {
			break
		}
		result = append(result, t.OptionPrefix+"_"+letter+"_content", t.OptionPrefix+"_"+letter+"_image_link")
		if t.Type == questionmodel.TypeTrueFalse {
			result = append(result, t.OptionPrefix+"_"+letter+"_correct")
		}
	}
	return append(result, t.ExtraHeaders...)
}

// detectTemplate menentukan tipe soal dari kolom khas masing-masing template.
func detectTemplate(headerIndex map[string]int) (excelTemplate, bool) {
	switch {
	case has(headerIndex, "correct_key"):
		return excelTemplates[questionmodel.TypeSingleChoice], true
	case has(headerIndex, "correct_keys"):
		return excelTemplates[questionmodel.TypeMultipleChoice], true
	case has(headerIndex, "statement_a_content"):
		return excelTemplates[questionmodel.TypeTrueFalse], true
	case has(headerIndex, "input_type"):
		return excelTemplates[questionmodel.TypeFreeText], true
	}
	return excelTemplate{}, false
}

func has(m map[string]int, k string) bool {
	_, ok := m[k]
	return ok
}

// ----------------------------------------------------------------------------
// Pengumpul error Excel
// ----------------------------------------------------------------------------

// excelErrors mengumpulkan error beserta lokasi sel. Error yang sama persis di sel yang
// sama hanya dicatat sekali (contoh "wajib diisi" dari 2 tahap pengecekan).
type excelErrors struct {
	headerIndex map[string]int // nama kolom → index (mulai 0)
	list        []questionmodel.FieldError
	seen        map[string]bool
}

// add mencatat error di baris row, kolom header. header boleh kosong (error tanpa kolom).
func (e *excelErrors) add(row int, header, message string) {
	column := ""
	if idx, ok := e.headerIndex[header]; ok {
		column, _ = excelize.ColumnNumberToName(idx + 1)
	}
	k := fmt.Sprintf("%d|%s|%s", row, header, message)
	if e.seen[k] {
		return
	}
	e.seen[k] = true

	// Error tanpa kolom: "file" untuk masalah tingkat file/header, "row" untuk masalah 1 baris.
	field := header
	switch {
	case field == "" && row > 1:
		field = "row"
	case field == "":
		field = "file"
	}
	e.list = append(e.list, questionmodel.FieldError{Field: field, Message: message, Row: row, Column: column})
}

// toError mengurutkan error per baris lalu per kolom, dan membungkusnya menjadi ValidationError.
func (e *excelErrors) toError() error {
	sort.SliceStable(e.list, func(i, j int) bool {
		a, b := e.list[i], e.list[j]
		if a.Row != b.Row {
			return a.Row < b.Row
		}
		return columnNumber(a.Column) < columnNumber(b.Column)
	})
	return &questionmodel.ValidationError{
		Message: fmt.Sprintf("File Excel berisi %d kesalahan, semua data ditolak", len(e.list)),
		Fields:  e.list,
	}
}

func columnNumber(column string) int {
	if column == "" {
		return 0
	}
	n, _ := excelize.ColumnNameToNumber(column)
	return n
}

// fileError membuat error 1 pesan untuk masalah tingkat file (bukan per sel).
func fileError(message string) error {
	return &questionmodel.ValidationError{
		Message: message,
		Fields:  []questionmodel.FieldError{{Field: "file", Message: message}},
	}
}

// ----------------------------------------------------------------------------
// CheckExcel
// ----------------------------------------------------------------------------

// CheckExcel membaca file Excel upload soal lalu mengecek semua barisnya tanpa menyimpan.
// Hasilnya: tipe soal + daftar soal siap dikirim ke create (placement sudah berisi ID & nama).
// Kalau ada error, dikembalikan *questionmodel.ValidationError berisi SEMUA error (baris & kolom).
func (s *QuestionService) CheckExcel(ctx context.Context, file io.Reader) (string, []questionmodel.ImportedQuestion, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return "", nil, fileError("File bukan Excel (.xlsx) yang valid")
	}
	defer f.Close()

	sheet := excelSheetName
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		sheet = f.GetSheetName(0)
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", nil, fileError("Sheet " + sheet + " tidak bisa dibaca")
	}
	if len(rows) == 0 {
		return "", nil, fileError("File kosong, gunakan template upload soal")
	}

	// ---- Header & tipe soal ---------------------------------------------------
	errs := &excelErrors{headerIndex: map[string]int{}, seen: map[string]bool{}}
	tmpl, err := readHeaders(rows[0], errs)
	if err != nil {
		return "", nil, err
	}
	if len(errs.list) > 0 {
		return "", nil, errs.toError()
	}

	// ---- Baris data (baris kosong dilewati) ------------------------------------
	type dataRow struct {
		Number int // nomor baris Excel (mulai 1)
		Cells  []string
	}
	var dataRows []dataRow
	for i, cells := range rows[1:] {
		if !isEmptyRow(cells) {
			dataRows = append(dataRows, dataRow{Number: i + 2, Cells: cells})
		}
	}
	switch {
	case len(dataRows) == 0:
		return "", nil, fileError("Belum ada soal di file. Isi mulai baris 2 sheet " + sheet)
	case len(dataRows) > MaxExcelRows:
		return "", nil, fileError(fmt.Sprintf("Maksimal %d soal per file, file ini berisi %d soal", MaxExcelRows, len(dataRows)))
	}

	// ---- Placement: 1 query untuk semua nama di file ---------------------------
	allCells := make([][]string, 0, len(dataRows))
	for _, r := range dataRows {
		allCells = append(allCells, r.Cells)
	}
	resolver, err := s.loadPlacementResolver(ctx, errs.headerIndex, allCells)
	if err != nil {
		return "", nil, err
	}

	// ---- Cek per baris -----------------------------------------------------------
	result := make([]questionmodel.ImportedQuestion, 0, len(dataRows))
	for _, r := range dataRows {
		cell := func(header string) string {
			idx := errs.headerIndex[header]
			if idx < len(r.Cells) {
				return strings.TrimSpace(r.Cells[idx])
			}
			return ""
		}
		imported, ok := checkExcelRow(tmpl, r.Number, cell, resolver, errs)
		if ok {
			result = append(result, imported)
		}
	}

	if len(errs.list) > 0 {
		return "", nil, errs.toError()
	}
	return tmpl.Type, result, nil
}

// readHeaders membaca header baris 1, menentukan template, dan mencatat kolom yang
// hilang / tidak dikenal / kembar. Error tipe tidak dikenali langsung dikembalikan.
func readHeaders(headerRow []string, errs *excelErrors) (excelTemplate, error) {
	for i, h := range headerRow {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if _, dup := errs.headerIndex[h]; dup {
			column, _ := excelize.ColumnNumberToName(i + 1)
			errs.list = append(errs.list, questionmodel.FieldError{Field: h, Message: "nama kolom kembar", Row: 1, Column: column})
			continue
		}
		errs.headerIndex[h] = i
	}

	tmpl, ok := detectTemplate(errs.headerIndex)
	if !ok {
		return tmpl, fileError("Header tidak dikenali. Gunakan template upload soal (single_choice, multiple_choice, true_false, atau free_text) tanpa mengubah nama kolom")
	}

	expected := map[string]bool{}
	for _, h := range tmpl.headers() {
		expected[h] = true
		if !has(errs.headerIndex, h) {
			errs.add(1, "", fmt.Sprintf("kolom %q tidak ada (template %s)", h, tmpl.Type))
		}
	}
	for h := range errs.headerIndex {
		if !expected[h] {
			errs.add(1, h, fmt.Sprintf("kolom %q tidak dikenal di template %s", h, tmpl.Type))
		}
	}
	return tmpl, nil
}

func isEmptyRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// loadPlacementResolver mengumpulkan semua nama placement di file lalu mengambilnya
// dengan 1 query. rows berisi sel setiap baris data.
func (s *QuestionService) loadPlacementResolver(ctx context.Context, headerIndex map[string]int, rows [][]string) (*placementResolver, error) {
	nameHeaders := []string{"education_level_name", "grade_name", "subject_name", "subject_field_name", "chapter_name", "subchapter_name", "topic_name"}
	unique := map[string]bool{}
	for _, cells := range rows {
		for _, h := range nameHeaders {
			if idx := headerIndex[h]; idx < len(cells) {
				if name := strings.ToLower(strings.TrimSpace(cells[idx])); name != "" {
					unique[name] = true
				}
			}
		}
	}
	names := make([]string, 0, len(unique))
	for n := range unique {
		names = append(names, n)
	}

	items, err := s.repo.FindMasterDataByNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca master data placement: %w", err)
	}
	return newPlacementResolver(items), nil
}

// ----------------------------------------------------------------------------
// Cek 1 baris
// ----------------------------------------------------------------------------

// checkExcelRow mengubah 1 baris menjadi request create, lalu mengeceknya dengan aturan create.
// ok = false kalau baris ini punya error (error sudah dicatat di errs).
func checkExcelRow(tmpl excelTemplate, rowNum int, cell func(string) string, resolver *placementResolver, errs *excelErrors) (questionmodel.ImportedQuestion, bool) {
	errCountBefore := len(errs.list)

	// Tahap 1 (baca sel & cocokkan placement) mencatat kolom yang sudah salah. Error dari
	// tahap 2 & 3 di kolom yang sama dilewati, supaya 1 sel tidak mendapat 2 pesan berbeda
	// (contoh "YA" di kolom TRUE/FALSE: cukup "harus TRUE atau FALSE").
	stage1Headers := map[string]bool{}
	addErr := func(header, message string) {
		stage1Headers[header] = true
		errs.add(rowNum, header, message)
	}
	addCreateErr := func(header, message string) {
		if header == "" || !stage1Headers[header] {
			errs.add(rowNum, header, message)
		}
	}

	// 1. Sel → request create.
	req := questiondto.CreateQuestionRequest{
		Type:         tmpl.Type,
		Question:     cell("question"),
		Formula:      optional(cell("formula")),
		ImageURL:     optional(cell("image_link")),
		Explanation:  cell("explanation"),
		DisplayOrder: splitList(strings.ToLower(cell("display_order")), ","),
		Difficulty:   strings.ToLower(cell("difficulty")),
		UsedFor:      splitList(strings.ToLower(cell("used_for")), ","),
		Status:       strings.ToLower(cell("status")),
	}
	if len(req.DisplayOrder) == 0 {
		addErr("display_order", "wajib diisi") // wajib di template (di API opsional)
	}
	req.Table = parseTable(cell("table_headers"), cell("table_rows"), addErr)

	placement, placementOK := resolver.resolve(placementNames{
		EducationLevel: cell("education_level_name"), Grade: cell("grade_name"), Subject: cell("subject_name"),
		SubjectField: cell("subject_field_name"), Chapter: cell("chapter_name"),
		Subchapter: cell("subchapter_name"), Topic: cell("topic_name"),
	}, addErr)
	req.Placement = placement.Request

	answer, err := json.Marshal(parseExcelAnswer(tmpl, cell, addErr))
	if err != nil {
		addErr("", "gagal membaca jawaban: "+err.Error())
	}
	req.Answer = answer

	// 2. Aturan tag binding (sama dengan create). Placement dilewati: sudah dicek resolver
	//    (kalau namanya tidak ketemu, ID-nya kosong dan error-nya sudah dicatat di atas).
	var vErrs validator.ValidationErrors
	if err := validation.Struct(req); errors.As(err, &vErrs) {
		for _, fe := range vErrs {
			path := validation.Path(fe)
			if strings.HasPrefix(path, "placement.") {
				continue
			}
			header, message := excelLocation(tmpl, path, validation.Message(fe))
			addCreateErr(header, message)
		}
	}

	// 3. Aturan isi soal & jawaban (sama dengan create).
	contentErrs := &fieldErrors{}
	q, err := buildQuestion(tmpl.Type, req, "", contentErrs, map[string]string{})
	if err != nil {
		addCreateErr("", err.Error())
	}
	for _, fe := range contentErrs.list {
		header, message := excelLocation(tmpl, fe.Field, fe.Message)
		addCreateErr(header, message)
	}

	if !placementOK || len(errs.list) > errCountBefore {
		return questionmodel.ImportedQuestion{}, false
	}

	return questionmodel.ImportedQuestion{
		Row: rowNum,
		QuestionDetail: questionmodel.QuestionDetail{
			Question:           q,
			EducationLevelName: placement.Names.EducationLevel,
			GradeName:          placement.Names.Grade,
			SubjectName:        placement.Names.Subject,
			SubjectFieldName:   placement.Names.SubjectField,
			ChapterName:        placement.Names.Chapter,
			SubchapterName:     optional(placement.Names.Subchapter),
			TopicName:          optional(placement.Names.Topic),
		},
	}, true
}

// parseTable membaca tabel: header dipisah "|", baris dipisah baris baru (Alt+Enter),
// kolom dipisah "|". Sesuai template, kalau salah satu diisi, keduanya wajib.
func parseTable(headersCell, rowsCell string, addErr func(header, message string)) *questiondto.TableRequest {
	if headersCell == "" && rowsCell == "" {
		return nil
	}
	if headersCell == "" {
		addErr("table_headers", "wajib diisi jika table_rows diisi")
	}
	if rowsCell == "" {
		addErr("table_rows", "wajib diisi jika table_headers diisi")
	}

	table := &questiondto.TableRequest{Headers: splitCells(headersCell)}
	for _, line := range strings.Split(strings.ReplaceAll(rowsCell, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			table.Rows = append(table.Rows, splitCells(line))
		}
	}
	return table
}

// parseExcelAnswer menyusun answer sesuai tipe, dengan bentuk JSON yang sama dengan create.
func parseExcelAnswer(tmpl excelTemplate, cell func(string) string, addErr func(header, message string)) any {
	switch tmpl.Type {
	case questionmodel.TypeSingleChoice:
		return questionmodel.SingleChoiceAnswer{
			Options:    parseOptions(tmpl, cell),
			CorrectKey: strings.ToUpper(cell("correct_key")),
		}

	case questionmodel.TypeMultipleChoice:
		scoring := strings.ToLower(cell("scoring"))
		if scoring == "" {
			addErr("scoring", "wajib diisi") // wajib di template (di API default all_or_nothing)
		}
		return questionmodel.MultipleChoiceAnswer{
			Options:     parseOptions(tmpl, cell),
			CorrectKeys: splitList(strings.ToUpper(cell("correct_keys")), ","),
			Scoring:     scoring,
		}

	case questionmodel.TypeTrueFalse:
		options := parseOptions(tmpl, cell)
		keys := make([]questionmodel.TrueFalseKey, 0, len(options))
		for i, opt := range options {
			header := tmpl.OptionPrefix + "_" + optionLetters[i] + "_correct"
			keys = append(keys, questionmodel.TrueFalseKey{Key: opt.Key, Correct: parseBool(cell(header), header, addErr)})
		}
		return questionmodel.TrueFalseAnswer{Options: options, CorrectKeys: keys}

	default: // free_text
		return questionmodel.FreeTextAnswer{
			InputType:       strings.ToLower(cell("input_type")),
			AcceptedAnswers: splitList(cell("accepted_answers"), "|"),
			Tolerance:       parseFloat(cell("tolerance"), "tolerance", addErr),
		}
	}
}

// parseOptions membaca opsi A-E. Jumlah opsi = sampai opsi terakhir yang terisi, minimal
// sebanyak opsi wajib template. Opsi kosong di tengah tetap dimasukkan supaya pengecekan
// create melaporkannya ("wajib diisi jika opsi tidak punya gambar") di sel yang benar.
func parseOptions(tmpl excelTemplate, cell func(string) string) []questionmodel.ChoiceOption {
	last := -1
	for i, letter := range optionLetters {
		prefix := tmpl.OptionPrefix + "_" + letter
		if cell(prefix+"_content") != "" || cell(prefix+"_image_link") != "" ||
			(tmpl.Type == questionmodel.TypeTrueFalse && cell(prefix+"_correct") != "") {
			last = i
		}
	}
	count := max(last+1, tmpl.RequiredCount)

	options := make([]questionmodel.ChoiceOption, 0, count)
	for i := range count {
		prefix := tmpl.OptionPrefix + "_" + optionLetters[i]
		options = append(options, questionmodel.ChoiceOption{
			Key:      strings.ToUpper(optionLetters[i]),
			Content:  cell(prefix + "_content"),
			ImageURL: optional(cell(prefix + "_image_link")),
		})
	}
	return options
}

// parseBool membaca TRUE/FALSE (tidak peka huruf besar/kecil). Kosong → nil
// (pengecekan create melaporkan "wajib diisi").
func parseBool(value, header string, addErr func(header, message string)) *bool {
	switch strings.ToUpper(value) {
	case "":
		return nil
	case "TRUE":
		v := true
		return &v
	case "FALSE":
		v := false
		return &v
	}
	addErr(header, "harus TRUE atau FALSE")
	return nil
}

// parseFloat membaca angka (koma boleh dipakai sebagai desimal). Kosong → nil.
func parseFloat(value, header string, addErr func(header, message string)) *float64 {
	if value == "" {
		return nil
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	if err != nil {
		addErr(header, "harus berupa angka")
		return nil
	}
	return &f
}

// splitList memecah "a, b,c" menjadi ["a","b","c"] (nilai kosong dibuang).
func splitList(value, sep string) []string {
	var result []string
	for _, part := range strings.Split(value, sep) {
		if p := strings.TrimSpace(part); p != "" {
			result = append(result, p)
		}
	}
	return result
}

// splitCells memecah 1 baris tabel / header "a|b|c" (sel kosong tetap dihitung sebagai kolom).
func splitCells(line string) []string {
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ----------------------------------------------------------------------------
// Field API → kolom Excel
// ----------------------------------------------------------------------------

// fieldPathPattern memecah path error, contoh "answer.options[2].content" →
// base "answer.options", index "2", rest ".content".
var fieldPathPattern = regexp.MustCompile(`^([a-z_.]+)(?:\[(\d+)\])?(.*)$`)

// listColumns adalah kolom yang berisi beberapa nilai dalam 1 sel. Error per nilai
// diberi keterangan "nilai ke-N" supaya admin tahu nilai mana yang salah.
var listColumns = map[string]string{
	"display_order": "nilai", "used_for": "nilai", "correct_keys": "nilai",
	"accepted_answers": "jawaban", "table_headers": "kolom", "table_rows": "baris tabel",
}

// excelLocation mengubah path field API (dari pengecekan create) menjadi nama kolom Excel.
// Kolom yang tidak bisa ditentukan dikembalikan kosong (error tetap dilaporkan di barisnya).
func excelLocation(tmpl excelTemplate, path, message string) (string, string) {
	m := fieldPathPattern.FindStringSubmatch(path)
	if m == nil {
		return "", path + ": " + message
	}
	base, indexStr, rest := m[1], m[2], m[3]
	index := -1
	if indexStr != "" {
		index, _ = strconv.Atoi(indexStr)
	}

	header := excelHeaderFor(tmpl, base, index, rest)
	if kind, isList := listColumns[header]; isList && index >= 0 {
		message = fmt.Sprintf("%s ke-%d: %s", kind, index+1, message)
	}
	return header, message
}

// excelHeaderFor berisi tabel pemetaan field API → kolom template.
func excelHeaderFor(tmpl excelTemplate, base string, index int, rest string) string {
	switch base {
	case "question", "formula", "explanation", "display_order", "difficulty", "used_for", "status":
		return base
	case "image_url", "image_id":
		return "image_link"
	case "table", "table.headers":
		return "table_headers"
	case "table.rows":
		return "table_rows"
	case "answer.options":
		letter := optionLetters[0]
		if index >= 0 && index < len(optionLetters) {
			letter = optionLetters[index]
		}
		prefix := tmpl.OptionPrefix + "_" + letter
		if strings.HasPrefix(rest, ".image") {
			return prefix + "_image_link"
		}
		return prefix + "_content"
	case "answer.correct_key":
		return "correct_key"
	case "answer.correct_keys":
		if tmpl.Type == questionmodel.TypeTrueFalse {
			if index < 0 || index >= len(optionLetters) {
				index = 0
			}
			return tmpl.OptionPrefix + "_" + optionLetters[index] + "_correct"
		}
		return "correct_keys"
	case "answer.scoring":
		return "scoring"
	case "answer.input_type":
		return "input_type"
	case "answer.accepted_answers":
		return "accepted_answers"
	case "answer.tolerance":
		return "tolerance"
	}
	return ""
}
