package user

// File ini berisi pengecekan file Excel tambah user (TIDAK menyimpan ke database).
//
// Alur CheckExcel (sama dengan import soal):
//  1. Buka file, ambil sheet "Template" (atau sheet pertama). Sel dibaca mentah (RawCellValue)
//     supaya tanggal berformat Excel terbaca sebagai angka seri tanggal, bukan teks tampilan.
//  2. Baca header baris 1: kolom wajib harus ada, kolom tidak dikenal ditolak.
//  3. Cocokkan nama cabang (1 query) dan nama jenjang/kelas (1 query) menjadi ID.
//  4. Per baris: ubah sel menjadi item tambah user (CreateUserItem), cek dengan aturan yang SAMA
//     PERSIS dengan API tambah user (tag binding + validateInput + hak role + username kembar).
//  5. Semua error dikembalikan dengan nomor baris & huruf kolom Excel. Ada 1 error → semua ditolak.
//
// Template: docs/Template Upload/template_user.xlsx (contoh terisi: contoh_terisi_user.xlsx).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"

	userdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/validation"
)

// MaxExcelRows adalah batas akun per file (sama dengan batas tambah user sekaligus).
const MaxExcelRows = 100

const excelSheetName = "Template"

// excelHeaders adalah semua kolom template, urut sesuai file.
var excelHeaders = []string{
	"role", "username", "name", "password", "email", "phone", "parent_phone",
	"branch_name", "education_level_name", "grade_name", "school_origin",
	"join_date", "end_date", "status", "date_of_birth", "target_school",
}

// requiredHeaders = kolom yang wajib ada di file (kolom lain boleh tidak ada).
var requiredHeaders = []string{"role", "username", "name", "password", "join_date"}

// fieldHeader memetakan nama field request ke kolom Excel (field lain namanya sama).
var fieldHeader = map[string]string{
	"branch_id":          "branch_name",
	"education_level_id": "education_level_name",
	"grade_id":           "grade_name",
}

// roleAliases: isi kolom role yang diterima (tidak peka huruf besar/kecil).
var roleAliases = map[string]string{
	"student": loginmodel.RoleStudent, "siswa": loginmodel.RoleStudent,
	"tutor": loginmodel.RoleTutor, "guru": loginmodel.RoleTutor,
	"admin":       loginmodel.RoleAdmin,
	"super_admin": loginmodel.RoleSuperAdmin, "super admin": loginmodel.RoleSuperAdmin, "superadmin": loginmodel.RoleSuperAdmin,
}

// statusAliases: isi kolom status yang diterima. Kosong = active.
var statusAliases = map[string]string{
	"active": loginmodel.UserStatusActive, "aktif": loginmodel.UserStatusActive,
	"inactive": loginmodel.UserStatusInactive, "nonaktif": loginmodel.UserStatusInactive, "tidak aktif": loginmodel.UserStatusInactive,
}

// excelErrors mengumpulkan error per sel (baris & kolom Excel).
type excelErrors struct {
	headerIndex map[string]int // nama kolom → index (mulai 0)
	list        []usermodel.FieldError
	cells       map[string]bool // sel yang sudah punya error (1 sel = 1 error, yang pertama)
}

// add mencatat error di baris row, kolom header ("" = error tingkat file / baris).
func (e *excelErrors) add(row int, header, message string) {
	column := ""
	if idx, ok := e.headerIndex[header]; ok {
		column, _ = excelize.ColumnNumberToName(idx + 1)
	}
	// 1 sel cukup 1 error: error pertama (contoh "cabang tidak ditemukan") lebih jelas daripada
	// error lanjutan dari nilai yang sama (contoh "wajib diisi").
	key := fmt.Sprintf("%d|%s", row, header)
	if header != "" && e.cells[key] {
		return
	}
	if e.cells == nil {
		e.cells = map[string]bool{}
	}
	e.cells[key] = true

	field := header
	switch {
	case field == "" && row > 1:
		field = "row"
	case field == "":
		field = "file"
	}
	e.list = append(e.list, usermodel.FieldError{Field: field, Message: message, Row: row, Column: column})
}

// toError mengurutkan error per baris lalu kolom, dan membungkusnya menjadi ValidationError.
func (e *excelErrors) toError() error {
	sort.SliceStable(e.list, func(i, j int) bool {
		a, b := e.list[i], e.list[j]
		if a.Row != b.Row {
			return a.Row < b.Row
		}
		x, _ := excelize.ColumnNameToNumber(a.Column)
		y, _ := excelize.ColumnNameToNumber(b.Column)
		return x < y
	})
	return &usermodel.ValidationError{
		Message: fmt.Sprintf("File Excel berisi %d kesalahan, semua data ditolak", len(e.list)),
		Fields:  e.list,
	}
}

func fileError(message string) error {
	return &usermodel.ValidationError{Message: message, Fields: []usermodel.FieldError{{Field: "file", Message: message}}}
}

// CheckExcel membaca file Excel tambah user lalu mengecek semua barisnya tanpa menyimpan.
// Hasilnya akun-akun siap dikirim ke POST /api/v1/users (cabang/jenjang/kelas sudah berisi ID).
// actorRole dipakai untuk aturan role (admin hanya boleh menambah tutor & siswa).
func (s *UserService) CheckExcel(ctx context.Context, file io.Reader, actorRole string) ([]usermodel.ImportedUser, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fileError("File bukan Excel (.xlsx) yang valid")
	}
	defer f.Close()

	sheet := excelSheetName
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		sheet = f.GetSheetName(0)
	}
	rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil || len(rows) == 0 {
		return nil, fileError("File kosong, gunakan template tambah user")
	}

	// ---- Header ---------------------------------------------------------------
	errs := &excelErrors{headerIndex: map[string]int{}}
	known := map[string]bool{}
	for _, h := range excelHeaders {
		known[h] = true
	}
	for i, h := range rows[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		column, _ := excelize.ColumnNumberToName(i + 1)
		switch {
		case !known[h]:
			errs.list = append(errs.list, usermodel.FieldError{Field: h, Message: "kolom tidak dikenal, gunakan template tambah user", Row: 1, Column: column})
		case has(errs.headerIndex, h):
			errs.list = append(errs.list, usermodel.FieldError{Field: h, Message: "nama kolom kembar", Row: 1, Column: column})
		default:
			errs.headerIndex[h] = i
		}
	}
	for _, h := range requiredHeaders {
		if !has(errs.headerIndex, h) {
			errs.list = append(errs.list, usermodel.FieldError{Field: h, Message: "kolom wajib tidak ada di baris 1", Row: 1})
		}
	}
	if len(errs.list) > 0 {
		return nil, errs.toError()
	}

	// ---- Baris data -----------------------------------------------------------
	type dataRow struct {
		Number int
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
		return nil, fileError("File tidak berisi data user (baris 2 dan seterusnya kosong)")
	case len(dataRows) > MaxExcelRows:
		return nil, fileError(fmt.Sprintf("Maksimal %d user per file, file ini berisi %d user", MaxExcelRows, len(dataRows)))
	}

	// ---- Nama cabang, jenjang, kelas → ID (2 query untuk semua baris) -----------
	branchNames := []string{}
	for _, r := range dataRows {
		if name := cellOf(r.Cells, errs.headerIndex, "branch_name"); name != "" {
			branchNames = append(branchNames, name)
		}
	}
	branches, err := s.repo.FindBranchesByNames(ctx, branchNames)
	if err != nil {
		return nil, err
	}
	levelsGrades, err := s.repo.FindLevelsAndGrades(ctx)
	if err != nil {
		return nil, err
	}
	resolver := newPlacementResolver(levelsGrades)

	// ---- Ubah setiap baris menjadi item tambah user -----------------------------
	items := make([]userdto.CreateUserItem, len(dataRows))
	imported := make([]usermodel.ImportedUser, len(dataRows))
	for i, r := range dataRows {
		cell := func(h string) string { return cellOf(r.Cells, errs.headerIndex, h) }
		addErr := func(h, msg string) { errs.add(r.Number, h, msg) }
		items[i], imported[i] = rowToItem(r.Number, cell, branches, resolver, addErr)
	}

	// ---- Aturan yang sama dengan API tambah user ---------------------------------
	inputs := make([]userInput, len(items))
	for i, item := range items {
		inputs[i] = inputFromCreateItem(item)
	}
	lk, err := s.loadLookups(ctx, inputs, true)
	if err != nil {
		return nil, err
	}
	firstRow := map[string]int{} // username → baris pertama yang memakainya
	for i, item := range items {
		rowNum := dataRows[i].Number
		// Tag binding (wajib, panjang, format email, panjang password, ...).
		var vErrs validator.ValidationErrors
		if err := validation.Struct(item); errors.As(err, &vErrs) {
			for _, fe := range vErrs {
				errs.add(rowNum, headerOf(validation.Path(fe)), validation.Message(fe))
			}
		}
		if item.Role != "" && !usermodel.CanManage(actorRole, item.Role) {
			errs.add(rowNum, "role", fmt.Sprintf("%s tidak boleh menambah akun %s", roleNames[actorRole], roleNames[item.Role]))
		}
		rowErrs := &fieldErrors{}
		u := validateInput(inputs[i], "", lk, rowErrs)
		for _, fe := range rowErrs.list {
			errs.add(rowNum, headerOf(fe.Field), fe.Message)
		}
		if j, dup := firstRow[u.Username]; dup && u.Username != "" {
			errs.add(rowNum, "username", fmt.Sprintf("sama dengan username di baris %d", j))
		} else {
			firstRow[u.Username] = rowNum
		}
		// Hasil dirapikan sama seperti yang akan disimpan (username huruf kecil, dll).
		imported[i].Username = u.Username
		imported[i].Status = u.Status
	}
	if len(errs.list) > 0 {
		return nil, errs.toError()
	}
	return imported, nil
}

// rowToItem mengubah 1 baris menjadi item tambah user (+ data preview). Nilai yang tidak bisa dibaca
// (role/status tidak dikenal, nama cabang/kelas tidak ada, tanggal tidak valid) dicatat lewat addErr.
func rowToItem(rowNum int, cell func(string) string, branches map[string]userrepo.BranchRef,
	resolver placementResolver, addErr func(header, message string)) (userdto.CreateUserItem, usermodel.ImportedUser) {

	item := userdto.CreateUserItem{
		Username:     cell("username"),
		Name:         cell("name"),
		Password:     cell("password"),
		Email:        optionalCell(cell("email")),
		Phone:        optionalCell(normalizePhone(cell("phone"))),
		ParentPhone:  optionalCell(normalizePhone(cell("parent_phone"))),
		SchoolOrigin: optionalCell(cell("school_origin")),
		TargetSchool: optionalCell(cell("target_school")),
	}

	if v := strings.ToLower(cell("role")); v != "" {
		if role, ok := roleAliases[v]; ok {
			item.Role = role
		} else {
			addErr("role", "isi salah satu: student (siswa), tutor, admin, super_admin")
		}
	}
	if v := strings.ToLower(cell("status")); v != "" {
		if status, ok := statusAliases[v]; ok {
			item.Status = status
		} else {
			addErr("status", "isi active (aktif) atau inactive (nonaktif)")
		}
	}

	for _, d := range []struct {
		header string
		target *string
	}{{"join_date", &item.JoinDate}} {
		if v := cell(d.header); v != "" {
			if t, ok := parseExcelDate(v); ok {
				*d.target = t
			} else {
				addErr(d.header, "tanggal tidak valid, contoh 2026-01-15")
			}
		}
	}
	for _, d := range []struct {
		header string
		target **string
	}{{"end_date", &item.EndDate}, {"date_of_birth", &item.DateOfBirth}} {
		if v := cell(d.header); v != "" {
			if t, ok := parseExcelDate(v); ok {
				*d.target = &t
			} else {
				addErr(d.header, "tanggal tidak valid, contoh 2026-01-15")
			}
		}
	}

	imported := usermodel.ImportedUser{Row: rowNum}
	if name := cell("branch_name"); name != "" {
		imported.BranchName = &name
		if b, ok := branches[strings.ToLower(name)]; ok {
			item.BranchID = &b.ID
		} else {
			addErr("branch_name", fmt.Sprintf("cabang %q tidak ditemukan", name))
		}
	}
	levelName, gradeName := cell("education_level_name"), cell("grade_name")
	if levelName != "" {
		imported.EducationLevelName = &levelName
	}
	if gradeName != "" {
		imported.GradeName = &gradeName
	}
	levelID, gradeID := resolver.resolve(levelName, gradeName, addErr)
	item.EducationLevelID, item.GradeID = levelID, gradeID

	imported.Role, imported.Name, imported.Password = item.Role, strings.TrimSpace(item.Name), item.Password
	imported.Email, imported.Phone, imported.ParentPhone = item.Email, item.Phone, item.ParentPhone
	imported.SchoolOrigin, imported.TargetSchool = item.SchoolOrigin, item.TargetSchool
	imported.BranchID, imported.EducationLevelID, imported.GradeID = item.BranchID, item.EducationLevelID, item.GradeID
	imported.JoinDate, imported.EndDate, imported.DateOfBirth = item.JoinDate, item.EndDate, item.DateOfBirth
	return item, imported
}

// inputFromCreateItem mengubah item tambah user menjadi userInput (dipakai API tambah & cek Excel).
func inputFromCreateItem(item userdto.CreateUserItem) userInput {
	return userInput{
		Role: item.Role, Username: item.Username, Name: item.Name, Status: item.Status,
		Email: item.Email, Phone: item.Phone, ParentPhone: item.ParentPhone, SchoolOrigin: item.SchoolOrigin,
		BranchID: item.BranchID, EducationLevelID: item.EducationLevelID, GradeID: item.GradeID,
		DateOfBirth: item.DateOfBirth, TargetSchool: item.TargetSchool,
		JoinDate: item.JoinDate, EndDate: item.EndDate,
	}
}

// placementResolver mencocokkan nama jenjang & kelas (tidak peka huruf besar/kecil) ke ID.
type placementResolver struct {
	levels map[string]usermodel.MasterDataItem   // nama kecil → jenjang
	grades map[string][]usermodel.MasterDataItem // nama kecil → kelas (bisa >1 di jenjang berbeda)
}

func newPlacementResolver(items []usermodel.MasterDataItem) placementResolver {
	r := placementResolver{levels: map[string]usermodel.MasterDataItem{}, grades: map[string][]usermodel.MasterDataItem{}}
	for _, it := range items {
		key := strings.ToLower(it.Name)
		if it.Type == "education_level" {
			r.levels[key] = it
		} else {
			r.grades[key] = append(r.grades[key], it)
		}
	}
	return r
}

// resolve mengembalikan ID jenjang & kelas. Jenjang boleh dikosongkan kalau nama kelasnya hanya ada
// di 1 jenjang (contoh "Kelas 5" → SD). Aturan wajib/tidak (siswa vs role lain) dicek validateInput.
func (r placementResolver) resolve(levelName, gradeName string, addErr func(header, message string)) (*string, *string) {
	var levelID, gradeID *string
	var level *usermodel.MasterDataItem
	if levelName != "" {
		l, ok := r.levels[strings.ToLower(levelName)]
		if !ok {
			addErr("education_level_name", fmt.Sprintf("jenjang %q tidak ditemukan", levelName))
			return nil, nil
		}
		level, levelID = &l, &l.ID
	}
	if gradeName == "" {
		return levelID, nil
	}
	candidates := r.grades[strings.ToLower(gradeName)]
	var match []usermodel.MasterDataItem
	for _, g := range candidates {
		if level == nil || (g.ParentID != nil && *g.ParentID == level.ID) {
			match = append(match, g)
		}
	}
	switch {
	case len(candidates) == 0:
		addErr("grade_name", fmt.Sprintf("kelas %q tidak ditemukan", gradeName))
	case len(match) == 0:
		addErr("grade_name", fmt.Sprintf("%s bukan bagian dari jenjang %s", gradeName, level.Name))
	case len(match) > 1:
		addErr("education_level_name", fmt.Sprintf("wajib diisi: kelas %q ada di lebih dari 1 jenjang", gradeName))
	default:
		g := match[0]
		gradeID = &g.ID
		if levelID == nil && g.ParentID != nil {
			levelID = g.ParentID // jenjang diambil dari kelasnya
		}
	}
	return levelID, gradeID
}

// excelDateLayouts adalah format tanggal teks yang diterima (selain tanggal berformat Excel).
var excelDateLayouts = []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006"}

var excelSerial = regexp.MustCompile(`^\d+(\.\d+)?$`)

// parseExcelDate membaca tanggal dari sel: angka seri tanggal Excel (sel berformat tanggal) atau
// teks "2026-01-15" / "15/01/2026" / "15-01-2026". Hasil "YYYY-MM-DD".
func parseExcelDate(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if excelSerial.MatchString(v) {
		serial, err := strconv.ParseFloat(v, 64)
		if err != nil || serial < 1 || serial > 2958465 { // batas tanggal Excel (tahun 9999)
			return "", false
		}
		t, err := excelize.ExcelDateToTime(serial, false)
		if err != nil {
			return "", false
		}
		return t.Format(time.DateOnly), true
	}
	for _, layout := range excelDateLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format(time.DateOnly), true
		}
	}
	return "", false
}

// normalizePhone: nomor HP yang diketik sebagai angka di Excel kehilangan "0" di depan
// (0812... → 812...). Kalau isinya hanya angka & diawali 8, "0" ditambahkan lagi.
func normalizePhone(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && v[0] == '8' && strings.Trim(v, "0123456789") == "" {
		return "0" + v
	}
	return v
}

// headerOf mengubah nama field request ("grade_id", "users[0].x" tidak dipakai di sini) ke kolom Excel.
func headerOf(field string) string {
	if h, ok := fieldHeader[field]; ok {
		return h
	}
	return field
}

func cellOf(cells []string, headerIndex map[string]int, header string) string {
	idx, ok := headerIndex[header]
	if !ok || idx >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[idx])
}

func optionalCell(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func isEmptyRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func has(m map[string]int, k string) bool {
	_, ok := m[k]
	return ok
}
