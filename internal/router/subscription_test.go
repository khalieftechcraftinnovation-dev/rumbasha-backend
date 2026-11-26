// Test paket akses: CRUD paket (admin), daftar paket untuk siswa, dan akses siswa di menu User
// (beri paket banyak siswa, perpanjangan otomatis, batal, akses khusus, kolom akses di list user).
package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

type planData struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Highlights     []string        `json:"highlights"`
	EducationLevel *listRef        `json:"education_level"`
	MonthlyCoin    int             `json:"monthly_coin"`
	Prices         map[string]any  `json:"prices"`
	Features       map[string]bool `json:"features"`
	FeatureLabels  []string        `json:"feature_labels"`
	Status         string          `json:"status"`
	ActiveStudents int             `json:"active_students"`
	ModuleCount    int             `json:"module_count"`
	CreatedBy      *listRef        `json:"created_by"`
	UpdatedBy      *listRef        `json:"updated_by"`
}

func createPlan(t *testing.T, r *gin.Engine, admin, body string) planData {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/subscription-plans", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("buat paket: status %d. body: %s", res.Code, res.Body)
	}
	var p planData
	decodeData(t, res, &p)
	return p
}

func planNames(t *testing.T, r *gin.Engine, token, query string) string {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/subscription-plans"+query, "", token)
	if res.Code != http.StatusOK {
		t.Fatalf("list paket: status %d. body: %s", res.Code, res.Body)
	}
	var plans []planData
	decodeData(t, res, &plans)
	names := []string{}
	for _, p := range plans {
		names = append(names, p.Name)
	}
	return strings.Join(names, ",")
}

func TestSubscriptionPlans_CRUD(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	sdID := masterID(t, db, "education_level", "SD")
	smpID := masterID(t, db, "education_level", "SMP")

	// Validasi dikumpulkan sekaligus.
	res := sendRequest(r, http.MethodPost, "/api/v1/subscription-plans",
		fmt.Sprintf(`{"name":"  ","strike_price_monthly":1000,"features":{},"education_level_id":%q,"highlights":["ok",""]}`, uuid.NewString()), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["name"] == "" || f["price_monthly"] == "" || f["strike_price_monthly"] == "" ||
		f["features"] == "" || f["education_level_id"] == "" || f["highlights[1]"] == "" {
		t.Errorf("validasi paket: status %d, errors %v", res.Code, f)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/subscription-plans", `{"name":"X","price_monthly":5000,"strike_price_monthly":5000,"features":{"module":true}}`, admin)
	if f := errorFields(t, res.Body.String()); f["strike_price_monthly"] != "harus lebih mahal dari harga bulanan" {
		t.Errorf("harga coret: %v", f)
	}
	// features wajib dikirim.
	if res := sendRequest(r, http.MethodPost, "/api/v1/subscription-plans", `{"name":"X","price_monthly":5000}`, admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("tanpa features: status %d", res.Code)
	}

	sd := createPlan(t, r, admin, fmt.Sprintf(`{"name":"Reguler Uji SD","education_level_id":%q,"monthly_coin":20,
		"highlights":["Modul"," Daily Quiz "],"price_monthly":150000,"strike_price_monthly":200000,
		"features":{"module":true,"daily_quiz":true},"sort_order":1}`, sdID))
	if sd.EducationLevel == nil || sd.EducationLevel.Name != "SD" || strings.Join(sd.FeatureLabels, ",") != "Modul,Daily Quiz" ||
		strings.Join(sd.Highlights, "|") != "Modul|Daily Quiz" || sd.Status != "active" || sd.CreatedBy == nil || sd.MonthlyCoin != 20 {
		t.Errorf("paket baru: %+v", sd)
	}
	all := createPlan(t, r, admin, `{"name":"Semua Jenjang Uji","price_yearly":1000000,"features":{"rumath":true},"sort_order":2}`)
	createPlan(t, r, admin, `{"name":"Lama Uji","price_monthly":1000,"features":{"module":true},"sort_order":3,"status":"inactive"}`)

	// Nama kembar (tidak peka huruf besar/kecil).
	res = sendRequest(r, http.MethodPost, "/api/v1/subscription-plans", `{"name":"reguler uji sd","price_monthly":1,"features":{"module":true}}`, admin)
	if f := errorFields(t, res.Body.String()); f["name"] != "sudah dipakai paket lain" {
		t.Errorf("nama kembar: %v", f)
	}

	// List admin + filter.
	if got := planNames(t, r, admin, ""); got != "Reguler Uji SD,Semua Jenjang Uji,Lama Uji" {
		t.Errorf("list semua: %s", got)
	}
	if got := planNames(t, r, admin, "?education_level_id="+smpID); got != "Semua Jenjang Uji,Lama Uji" {
		t.Errorf("filter jenjang SMP: %s", got)
	}
	if got := planNames(t, r, admin, "?status=inactive"); got != "Lama Uji" {
		t.Errorf("filter nonaktif: %s", got)
	}
	if got := planNames(t, r, admin, "?search=semua"); got != "Semua Jenjang Uji" {
		t.Errorf("cari: %s", got)
	}

	// Siswa SMP: hanya paket aktif untuk jenjangnya (+ semua jenjang), tanpa data admin.
	smpStudent, smpToken := namedStudent(t, db, "Sinta", "GRADE_7")
	res = sendRequest(r, http.MethodGet, "/api/v1/subscription-plans", "", smpToken)
	var studentView struct {
		Current struct {
			Plan *listRef `json:"plan"`
		} `json:"current"`
		Plans []map[string]any `json:"plans"`
	}
	decodeData(t, res, &studentView)
	if res.Code != http.StatusOK || len(studentView.Plans) != 1 || studentView.Plans[0]["name"] != "Semua Jenjang Uji" ||
		studentView.Plans[0]["status"] != nil || studentView.Current.Plan != nil {
		t.Errorf("paket untuk siswa SMP: status %d, %+v", res.Code, studentView)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/subscription-plans/"+sd.ID, "", smpToken); res.Code != http.StatusForbidden {
		t.Errorf("detail paket oleh siswa: status %d", res.Code)
	}

	// Ubah (ganti penuh): fitur & harga berubah, pengubah tercatat.
	res = sendRequest(r, http.MethodPut, "/api/v1/subscription-plans/"+all.ID,
		`{"name":"Semua Jenjang Uji","price_monthly":99000,"price_yearly":900000,"features":{"module":true,"rumath":true},"status":"active"}`, admin)
	var updated planData
	decodeData(t, res, &updated)
	if res.Code != http.StatusOK || !updated.Features["module"] || updated.UpdatedBy == nil || updated.Prices["monthly"] != float64(99000) {
		t.Errorf("ubah paket: status %d, %+v", res.Code, updated)
	}
	if res := sendRequest(r, http.MethodPut, "/api/v1/subscription-plans/"+uuid.NewString(), `{"name":"A","price_monthly":1,"features":{"module":true}}`, admin); res.Code != http.StatusNotFound {
		t.Errorf("ubah paket tidak ada: status %d", res.Code)
	}

	// Paket dipakai siswa SMP → sasaran jenjang tidak bisa diubah ke SD, dan paket tidak bisa dihapus.
	subscribe(t, db, smpStudent, all.ID)
	res = sendRequest(r, http.MethodPut, "/api/v1/subscription-plans/"+all.ID,
		fmt.Sprintf(`{"name":"Semua Jenjang Uji","education_level_id":%q,"price_monthly":99000,"features":{"module":true}}`, sdID), admin)
	if f := errorFields(t, res.Body.String()); res.Code != http.StatusUnprocessableEntity || !strings.Contains(f["education_level_id"], "1 siswa jenjang lain") {
		t.Errorf("ubah jenjang paket terpakai: status %d, %v", res.Code, f)
	}
	var detail planData
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/subscription-plans/"+all.ID, "", admin), &detail)
	if detail.ActiveStudents != 1 {
		t.Errorf("pengguna aktif: %d", detail.ActiveStudents)
	}

	// Modul yang hanya tersedia di paket SD → paket SD tidak bisa dihapus sendirian.
	p := seedPlacementInGrade(t, db, "GRADE_5")
	var moduleID string
	if err := db.Raw(`INSERT INTO learning_modules (name, education_level_id, grade_id, subject_id, subject_field_id,
		file_name, storage_path, mime_type, size_bytes, created_by) VALUES ('Modul Uji', ?, ?, ?, ?, 'a.pdf', 'modules/a.pdf', 'application/pdf', 10, ?)
		RETURNING id`, p.LevelID, p.GradeID, p.SubjectID, p.FieldID, tokenOwnerID(t, db, loginmodel.RoleAdmin)).Scan(&moduleID).Error; err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO learning_module_plans (module_id, plan_id) VALUES (?, ?)`, moduleID, sd.ID)

	missing := uuid.NewString()
	res = sendRequest(r, http.MethodDelete, "/api/v1/subscription-plans", fmt.Sprintf(`{"ids":[%q,%q,%q]}`, sd.ID, all.ID, missing), admin)
	f = errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["ids[2]"] != "paket tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus dengan id tidak ada: status %d, %v", res.Code, f)
	}
	res = sendRequest(r, http.MethodDelete, "/api/v1/subscription-plans", fmt.Sprintf(`{"ids":[%q,%q]}`, sd.ID, all.ID), admin)
	f = errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(f["ids[0]"], "satu-satunya paket di 1 modul") || !strings.Contains(f["ids[1]"], "masih dipakai 1 siswa") {
		t.Errorf("hapus paket terpakai: status %d, %v", res.Code, f)
	}

	// Modul juga ada di paket lain → paket SD boleh dihapus; hubungan modulnya ikut dihapus.
	db.Exec(`INSERT INTO learning_module_plans (module_id, plan_id) VALUES (?, ?)`, moduleID, all.ID)
	res = sendRequest(r, http.MethodDelete, "/api/v1/subscription-plans", fmt.Sprintf(`{"ids":[%q]}`, sd.ID), admin)
	var links int
	db.Raw(`SELECT COUNT(*) FROM learning_module_plans WHERE plan_id = ?`, sd.ID).Scan(&links)
	if res.Code != http.StatusOK || links != 0 {
		t.Errorf("hapus paket: status %d, sisa hubungan modul %d. body: %s", res.Code, links, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/subscription-plans/"+sd.ID, "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail paket terhapus: status %d", res.Code)
	}
}

type accessData struct {
	Plan           *listRef        `json:"plan"`
	SubscriptionID *string         `json:"subscription_id"`
	StartDate      *string         `json:"start_date"`
	EndDate        *string         `json:"end_date"`
	DaysLeft       *int            `json:"days_left"`
	Features       map[string]bool `json:"features"`
	FeatureLabels  []string        `json:"feature_labels"`
	Overrides      map[string]bool `json:"overrides"`
}

type assignData struct {
	TotalPrice int64 `json:"total_price"`
	Students   []struct {
		ID             string `json:"id"`
		SubscriptionID string `json:"subscription_id"`
		StartDate      string `json:"start_date"`
		EndDate        string `json:"end_date"`
		Status         string `json:"status"`
		PricePaid      int64  `json:"price_paid"`
		CoinAdded      int    `json:"coin_added"`
		CoinBalance    int    `json:"coin_balance"`
	} `json:"students"`
}

// dbDate menghitung tanggal di PostgreSQL (WIB), contoh expr "+ INTERVAL '1 month' - 1".
func dbDate(t *testing.T, db *gorm.DB, expr string) string {
	t.Helper()
	var d string
	if err := db.Raw(`SELECT TO_CHAR(((NOW() AT TIME ZONE 'Asia/Jakarta')::date ` + expr + `)::date, 'YYYY-MM-DD')`).Scan(&d).Error; err != nil {
		t.Fatal(err)
	}
	return d
}

func TestStudentAccess_BeriPaketBatalDanAksesKhusus(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	sdID := masterID(t, db, "education_level", "SD")
	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_7")
	tutor := tokenOwnerID(t, db, loginmodel.RoleTutor)

	pro := createPlan(t, r, admin, `{"name":"Pro Uji","monthly_coin":50,"price_monthly":199000,
		"features":{"module":true,"daily_quiz":true,"uji_kom":true}}`)
	sdOnly := createPlan(t, r, admin, fmt.Sprintf(`{"name":"Khusus SD Uji","education_level_id":%q,"price_monthly":1000,"features":{"module":true}}`, sdID))
	off := createPlan(t, r, admin, `{"name":"Nonaktif Uji","price_monthly":1000,"features":{"module":true},"status":"inactive"}`)

	assign := func(body string) (int, string) {
		res := sendRequest(r, http.MethodPost, "/api/v1/student-subscriptions", body, admin)
		return res.Code, res.Body.String()
	}

	// Validasi: paket nonaktif, siswa tidak ada, jenjang tidak cocok, periode tidak dijual.
	code, body := assign(fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q],"period":"monthly"}`, off.ID, ani))
	if f := errorFields(t, body); code != 422 || f["plan_id"] != `paket "Nonaktif Uji" tidak aktif` {
		t.Errorf("paket nonaktif: %d %v", code, f)
	}
	code, body = assign(fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q,%q,%q],"period":"monthly"}`, sdOnly.ID, ani, budi, tutor))
	if f := errorFields(t, body); code != 422 || !strings.Contains(f["student_ids[1]"], "bukan siswa jenjang SD") || f["student_ids[2]"] != "siswa tidak ditemukan" || f["student_ids[0]"] != "" {
		t.Errorf("jenjang & siswa: %d %v", code, f)
	}
	code, body = assign(fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q],"period":"semester"}`, pro.ID, ani))
	if f := errorFields(t, body); code != 422 || !strings.Contains(f["period"], "tidak dijual") {
		t.Errorf("periode tidak dijual: %d %v", code, f)
	}
	var count int
	db.Raw(`SELECT COUNT(*) FROM student_subscriptions`).Scan(&count)
	if count != 0 {
		t.Fatalf("validasi gagal tapi paket tersimpan: %d", count)
	}

	// Beri paket Pro bulanan ke Ani & Budi sekaligus: mulai hari ini, bonus coin bulan pertama.
	today, monthEnd := dbDate(t, db, ""), dbDate(t, db, "+ INTERVAL '1 month' - INTERVAL '1 day'")
	requestID := uuid.NewString()
	first := fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q,%q],"period":"monthly","notes":"Bayar tunai","request_id":%q}`, pro.ID, budi, ani, requestID)
	code, body = assign(first)
	var a assignData
	decodeData(t, sendRequestBody(code, body), &a)
	if code != http.StatusCreated || len(a.Students) != 2 || a.Students[0].ID != budi || a.TotalPrice != 398000 ||
		a.Students[1].StartDate != today || a.Students[1].EndDate != monthEnd || a.Students[1].Status != "active" ||
		a.Students[1].CoinAdded != 50 || a.Students[1].CoinBalance != 50 {
		t.Fatalf("beri paket: %d %+v. body: %s", code, a, body)
	}
	var grant struct{ Type, ReferenceType, ReferenceID string }
	db.Raw(`SELECT type, reference_type, reference_id FROM coin_transactions WHERE student_id = ?`, ani).Scan(&grant)
	if grant.Type != "plan_grant" || grant.ReferenceType != "student_subscription" || grant.ReferenceID != a.Students[1].SubscriptionID {
		t.Errorf("transaksi bonus coin: %+v", grant)
	}

	// Kirim ulang request yang sama → 409, tidak dobel.
	if code, _ := assign(first); code != http.StatusConflict {
		t.Errorf("kirim ulang: status %d", code)
	}

	// Beri lagi tanpa start_date → perpanjangan otomatis mulai sehari setelah paket aktif berakhir.
	code, body = assign(fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q],"period":"monthly","price_paid":150000}`, pro.ID, ani))
	var ext assignData
	decodeData(t, sendRequestBody(code, body), &ext)
	nextStart := dbDate(t, db, "+ INTERVAL '1 month'")
	if code != http.StatusCreated || ext.Students[0].StartDate != nextStart || ext.Students[0].Status != "upcoming" || ext.Students[0].PricePaid != 150000 {
		t.Errorf("perpanjangan: %d %+v", code, ext)
	}
	// start_date yang bertumpuk dengan paket yang ada → 422.
	code, body = assign(fmt.Sprintf(`{"plan_id":%q,"student_ids":[%q],"period":"monthly","start_date":%q}`, pro.ID, ani, today))
	if f := errorFields(t, body); code != 422 || !strings.Contains(f["student_ids[0]"], "masih punya paket sampai") {
		t.Errorf("tanggal bertumpuk: %d %v", code, f)
	}

	// Menu User: kolom akses + filter paket.
	items, _ := listUsers(t, r, admin, "?plan_id="+pro.ID)
	if userNames(items) != "Ani,Budi" && userNames(items) != "Budi,Ani" {
		t.Errorf("filter plan_id: %s", userNames(items))
	}
	res := sendRequest(r, http.MethodGet, "/api/v1/users/"+ani, "", admin)
	var aniUser struct {
		Access *accessData `json:"access"`
	}
	decodeData(t, res, &aniUser)
	if aniUser.Access == nil || aniUser.Access.Plan == nil || aniUser.Access.Plan.Name != "Pro Uji" || *aniUser.Access.EndDate != monthEnd ||
		strings.Join(aniUser.Access.FeatureLabels, ",") != "Modul,Daily Quiz,Uji Kom" {
		t.Errorf("akses di detail user: %+v", aniUser.Access)
	}

	// Akses khusus: buka RUMATH & tutup modul untuk Ani.
	res = sendRequest(r, http.MethodPut, "/api/v1/student-access-overrides",
		fmt.Sprintf(`{"student_ids":[%q],"features":{"rumath":true,"module":false}}`, ani), admin)
	var changed []struct {
		Access accessData `json:"access"`
	}
	decodeData(t, res, &changed)
	if res.Code != http.StatusOK || len(changed) != 1 || !changed[0].Access.Features["rumath"] || changed[0].Access.Features["module"] ||
		len(changed[0].Access.Overrides) != 2 {
		t.Errorf("akses khusus: %d %+v", res.Code, changed)
	}
	// Fitur tidak dikenal → 422; null = kembali ikut paket.
	if res := sendRequest(r, http.MethodPut, "/api/v1/student-access-overrides", fmt.Sprintf(`{"student_ids":[%q],"features":{"game":true}}`, ani), admin); res.Code != 422 {
		t.Errorf("fitur tidak dikenal: status %d", res.Code)
	}
	sendRequest(r, http.MethodPut, "/api/v1/student-access-overrides", fmt.Sprintf(`{"student_ids":[%q],"features":{"module":null}}`, ani), admin)

	// /auth/me siswa ikut berubah.
	res = sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", aniToken)
	if !strings.Contains(res.Body.String(), `"rumath":true`) || !strings.Contains(res.Body.String(), `"module":true`) {
		t.Errorf("/auth/me setelah akses khusus: %s", res.Body)
	}

	// Detail akses siswa: rincian fitur + riwayat paket (terbaru dulu).
	res = sendRequest(r, http.MethodGet, "/api/v1/students/"+ani+"/access", "", admin)
	var detail struct {
		Features []struct {
			Feature  string `json:"feature"`
			FromPlan bool   `json:"from_plan"`
			Override *bool  `json:"override"`
			Enabled  bool   `json:"enabled"`
		} `json:"features"`
		Subscriptions []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"subscriptions"`
	}
	decodeData(t, res, &detail)
	rumath := detail.Features[4]
	if res.Code != http.StatusOK || len(detail.Subscriptions) != 2 || detail.Subscriptions[0].Status != "upcoming" || detail.Subscriptions[1].Status != "active" ||
		rumath.Feature != "rumath" || rumath.FromPlan || rumath.Override == nil || !rumath.Enabled || detail.Features[0].Override != nil {
		t.Errorf("detail akses: %d %+v", res.Code, detail)
	}

	// Batalkan paket aktif Ani → tidak punya paket aktif (filter subscription=none), batal 2x → 409.
	activeID := a.Students[1].SubscriptionID
	res = sendRequest(r, http.MethodPost, "/api/v1/student-subscriptions/"+activeID+"/cancel", `{"reason":"Salah input"}`, admin)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"status":"cancelled"`) || !strings.Contains(res.Body.String(), "Dibatalkan admin: Salah input") {
		t.Errorf("batal paket: %d %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/student-subscriptions/"+activeID+"/cancel", "", admin); res.Code != http.StatusConflict {
		t.Errorf("batal 2x: status %d", res.Code)
	}
	items, _ = listUsers(t, r, admin, "?subscription=none&role=student")
	if !strings.Contains(userNames(items), "Ani") || strings.Contains(userNames(items), "Budi") {
		t.Errorf("filter tanpa paket: %s", userNames(items))
	}
	items, _ = listUsers(t, r, admin, "?subscription=expiring")
	if len(items) != 0 {
		t.Errorf("filter hampir berakhir: %s", userNames(items))
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/student-subscriptions", first, tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor beri paket: status %d", res.Code)
	}
}

// sendRequestBody membungkus status + body menjadi recorder supaya bisa dibaca decodeData.
func sendRequestBody(code int, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rec.Code = code
	rec.Body.WriteString(body)
	return rec
}
