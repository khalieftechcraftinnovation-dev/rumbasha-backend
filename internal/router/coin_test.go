// Test fitur coin: CRUD paket isi coin, top-up coin siswa oleh admin, dan riwayat coin.
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

type coinPackage struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CoinAmount  int      `json:"coin_amount"`
	Price       int64    `json:"price"`
	Description *string  `json:"description"`
	SortOrder   int      `json:"sort_order"`
	Status      string   `json:"status"`
	CreatedBy   *listRef `json:"created_by"`
	UpdatedBy   *listRef `json:"updated_by"`
}

// createPackage membuat paket coin lewat API (harus 201).
func createPackage(t *testing.T, r *gin.Engine, admin, body string) coinPackage {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/coin-packages", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("buat paket: status %d. body: %s", res.Code, res.Body)
	}
	var p coinPackage
	decodeData(t, res, &p)
	return p
}

func TestCoinPackages_CRUD(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	_, student := namedStudent(t, db, "Ani", "GRADE_5")

	p300 := createPackage(t, r, admin, `{"name":" 300 coin ","coin_amount":300,"price":68000,"description":"Hemat 10%","sort_order":2}`)
	if p300.Name != "300 coin" || p300.Status != "active" || p300.SortOrder != 2 || p300.CreatedBy == nil || p300.UpdatedBy != nil {
		t.Errorf("paket baru: %+v", p300)
	}
	p100 := createPackage(t, r, admin, `{"name":"100 coin","coin_amount":100,"price":25000,"sort_order":1}`)
	hidden := createPackage(t, r, admin, `{"name":"Promo lama","coin_amount":50,"price":0,"status":"inactive"}`)

	// Validasi: nama kembar (tidak peka huruf besar/kecil), jumlah coin 0, harga tidak diisi.
	res := sendRequest(r, http.MethodPost, "/api/v1/coin-packages", `{"name":"300 COIN","coin_amount":10,"price":1000}`, admin)
	if errorFields(t, res.Body.String())["name"] != "sudah dipakai paket lain" {
		t.Errorf("nama kembar: status %d, body %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/coin-packages", `{"name":"X","coin_amount":0}`, admin)
	if f := errorFields(t, res.Body.String()); res.Code != http.StatusUnprocessableEntity || f["coin_amount"] == "" || f["price"] == "" {
		t.Errorf("validasi: status %d, errors %v", res.Code, f)
	}
	// Siswa & tutor tidak boleh mengelola paket.
	for _, tok := range []string{student, tokenFor(t, loginmodel.RoleTutor)} {
		if res := sendRequest(r, http.MethodPost, "/api/v1/coin-packages", `{"name":"Y","coin_amount":1,"price":1}`, tok); res.Code != http.StatusForbidden {
			t.Errorf("buat paket bukan admin: status %d", res.Code)
		}
	}

	// List: admin semua paket (urut sort_order); siswa hanya aktif, tanpa data admin.
	var adminList []coinPackage
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/coin-packages", "", admin), &adminList)
	if len(adminList) != 3 || adminList[0].ID != hidden.ID || adminList[1].ID != p100.ID || adminList[2].ID != p300.ID {
		t.Errorf("list admin: %+v", adminList)
	}
	var inactive []coinPackage
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/coin-packages?status=inactive", "", admin), &inactive)
	if len(inactive) != 1 || inactive[0].ID != hidden.ID {
		t.Errorf("filter inactive: %+v", inactive)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/coin-packages?status=inactive", "", student)
	var studentList []coinPackage
	decodeData(t, res, &studentList)
	if len(studentList) != 2 || studentList[0].ID != p100.ID || studentList[0].Status != "" || studentList[0].CreatedBy != nil {
		t.Errorf("list siswa (hanya aktif, tanpa data admin): %s", res.Body)
	}

	// Ubah (ganti penuh).
	res = sendRequest(r, http.MethodPut, "/api/v1/coin-packages/"+p300.ID, `{"name":"300 coin","coin_amount":320,"price":70000}`, admin)
	var updated coinPackage
	decodeData(t, res, &updated)
	if res.Code != http.StatusOK || updated.CoinAmount != 320 || updated.Description != nil || updated.SortOrder != 0 || updated.UpdatedBy == nil {
		t.Errorf("ubah: status %d, %+v", res.Code, updated)
	}
	if res := sendRequest(r, http.MethodPut, "/api/v1/coin-packages/"+p300.ID, `{"name":"100 coin","coin_amount":1,"price":1}`, admin); errorFields(t, res.Body.String())["name"] == "" {
		t.Errorf("ubah ke nama paket lain: %s", res.Body)
	}

	// Hapus: ada ID yang tidak ada → tidak ada yang terhapus.
	missing := uuid.NewString()
	res = sendRequest(r, http.MethodDelete, "/api/v1/coin-packages", fmt.Sprintf(`{"ids":[%q,%q]}`, hidden.ID, missing), admin)
	if errorFields(t, res.Body.String())["ids[1]"] != "paket tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus dengan ID salah: %s", res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/coin-packages/"+hidden.ID, "", admin); res.Code != http.StatusOK {
		t.Errorf("paket seharusnya belum terhapus: status %d", res.Code)
	}
	res = sendRequest(r, http.MethodDelete, "/api/v1/coin-packages", fmt.Sprintf(`{"ids":[%q]}`, hidden.ID), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("hapus: status %d. body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/coin-packages/"+hidden.ID, "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail paket terhapus: status %d, seharusnya 404", res.Code)
	}
	// Nama paket yang sudah dihapus boleh dipakai lagi.
	createPackage(t, r, admin, `{"name":"Promo lama","coin_amount":50,"price":0}`)
}

func TestCoinTopUp_BanyakSiswaSekaligus(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	ani, _ := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_7")
	db.Exec(`UPDATE students SET coin_balance = 20 WHERE user_id = ?`, budi)
	p := createPackage(t, r, admin, `{"name":"300 coin","coin_amount":300,"price":68000}`)
	off := createPackage(t, r, admin, `{"name":"Nonaktif","coin_amount":10,"price":1000,"status":"inactive"}`)

	balance := func(id string) int {
		var b int
		db.Raw(`SELECT coin_balance FROM students WHERE user_id = ?`, id).Scan(&b)
		return b
	}

	// Paket nonaktif & ID bukan siswa → 422 per field, tidak ada coin yang berubah.
	notStudent := tokenOwnerID(t, db, loginmodel.RoleTutor)
	res := sendRequest(r, http.MethodPost, "/api/v1/coin-topups",
		fmt.Sprintf(`{"coin_package_id":%q,"student_ids":[%q,%q]}`, off.ID, ani, notStudent), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["coin_package_id"] != `paket "Nonaktif" tidak aktif` || f["student_ids[1]"] != "siswa tidak ditemukan" || balance(ani) != 0 {
		t.Errorf("validasi top-up: status %d, errors %v, saldo Ani %d", res.Code, f, balance(ani))
	}

	requestID := uuid.NewString()
	body := fmt.Sprintf(`{"coin_package_id":%q,"student_ids":[%q,%q],"note":"Bayar tunai di kasir","request_id":%q}`, p.ID, budi, ani, requestID)
	res = sendRequest(r, http.MethodPost, "/api/v1/coin-topups", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("top-up: status %d. body: %s", res.Code, res.Body)
	}
	var topUp struct {
		TotalCoins int   `json:"total_coins"`
		TotalPrice int64 `json:"total_price"`
		Students   []struct {
			ID           string `json:"id"`
			CoinAdded    int    `json:"coin_added"`
			BalanceAfter int    `json:"balance_after"`
		} `json:"students"`
	}
	decodeData(t, res, &topUp)
	// Urutan hasil = urutan student_ids di request.
	if topUp.TotalCoins != 600 || topUp.TotalPrice != 136000 || len(topUp.Students) != 2 ||
		topUp.Students[0].ID != budi || topUp.Students[0].BalanceAfter != 320 || topUp.Students[1].ID != ani || topUp.Students[1].BalanceAfter != 300 {
		t.Errorf("hasil top-up: %+v", topUp)
	}
	if balance(ani) != 300 || balance(budi) != 320 {
		t.Errorf("saldo: Ani %d, Budi %d", balance(ani), balance(budi))
	}

	// Tercatat sebagai pembelian tunai: harga paket, referensi paket, admin pencatat.
	var trx struct {
		Type, ReferenceType, ReferenceID, Description string
		Amount, BalanceAfter                          int
		PricePaid                                     int64
		CreatedBy                                     *string
	}
	db.Raw(`SELECT type, reference_type, reference_id, description, amount, balance_after, price_paid, created_by
		FROM coin_transactions WHERE student_id = ?`, ani).Scan(&trx)
	if trx.Type != "purchase" || trx.ReferenceType != "coin_package" || trx.ReferenceID != p.ID || trx.Amount != 300 ||
		trx.PricePaid != 68000 || trx.CreatedBy == nil || trx.Description != "Isi coin paket 300 coin (dicatat admin): Bayar tunai di kasir" {
		t.Errorf("transaksi: %+v", trx)
	}

	// Request yang sama terkirim lagi → 409, coin tidak bertambah 2x.
	if res := sendRequest(r, http.MethodPost, "/api/v1/coin-topups", body, admin); res.Code != http.StatusConflict || balance(ani) != 300 {
		t.Errorf("kirim ulang: status %d, saldo Ani %d", res.Code, balance(ani))
	}
	// Tutor tidak boleh top-up.
	if res := sendRequest(r, http.MethodPost, "/api/v1/coin-topups", body, tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("top-up oleh tutor: status %d", res.Code)
	}
}

func TestCoinHistory_SiswaDanAdmin(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_5")
	_, budiToken := namedStudent(t, db, "Budi", "GRADE_5")
	p := createPackage(t, r, admin, `{"name":"100 coin","coin_amount":100,"price":25000}`)
	for i := 0; i < 2; i++ {
		sendRequest(r, http.MethodPost, "/api/v1/coin-topups", fmt.Sprintf(`{"coin_package_id":%q,"student_ids":[%q]}`, p.ID, ani), admin)
	}
	// Pemakaian coin (contoh buka Daily Quiz) dicatat langsung.
	db.Exec(`UPDATE students SET coin_balance = 190 WHERE user_id = ?`, ani)
	db.Exec(`INSERT INTO coin_transactions (student_id, type, amount, balance_after, reference_type, reference_id, description, created_at)
		VALUES (?, 'usage', -10, 190, 'daily_quiz', ?, 'Daily Quiz', NOW() + INTERVAL '1 second')`, ani, uuid.NewString())

	type history struct {
		Student struct {
			ID          string `json:"id"`
			CoinBalance int    `json:"coin_balance"`
		} `json:"student"`
		DailyQuiz *struct {
			FreePerDay    int `json:"free_per_day"`
			RemainingFree int `json:"remaining_free"`
			CoinCostAfter int `json:"coin_cost_after"`
		} `json:"daily_quiz"`
		Transactions []struct {
			Type      string `json:"type"`
			TypeLabel string `json:"type_label"`
			Amount    int    `json:"amount"`
			PricePaid *int64 `json:"price_paid"`
			Reference *struct {
				Type string  `json:"type"`
				Name *string `json:"name"`
			} `json:"reference"`
			CreatedBy *listRef `json:"created_by"`
		} `json:"transactions"`
	}

	// Siswa: saldo + jatah Daily Quiz hari ini + riwayat terbaru dulu.
	res := sendRequest(r, http.MethodGet, "/api/v1/student/coin-transactions?per_page=2", "", aniToken)
	var h history
	decodeData(t, res, &h)
	meta := decodeMeta(t, res)
	if h.Student.ID != ani || h.Student.CoinBalance != 190 || h.DailyQuiz == nil || h.DailyQuiz.FreePerDay != 2 ||
		h.DailyQuiz.RemainingFree != 2 || h.DailyQuiz.CoinCostAfter != 10 || meta.Total != 3 || len(h.Transactions) != 2 {
		t.Fatalf("riwayat siswa: %+v (total %d)", h, meta.Total)
	}
	if tx := h.Transactions[0]; tx.Type != "usage" || tx.TypeLabel != "Pemakaian" || tx.Amount != -10 || tx.CreatedBy != nil {
		t.Errorf("transaksi terbaru: %+v", tx)
	}
	if tx := h.Transactions[1]; tx.Type != "purchase" || tx.TypeLabel != "Isi coin" || tx.PricePaid == nil || *tx.PricePaid != 25000 ||
		tx.Reference == nil || tx.Reference.Name == nil || *tx.Reference.Name != "100 coin" || tx.CreatedBy == nil {
		t.Errorf("transaksi isi coin: %+v", tx)
	}

	// Filter tipe.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/student/coin-transactions?type=purchase", "", aniToken), &h)
	if len(h.Transactions) != 2 {
		t.Errorf("filter purchase: %d transaksi", len(h.Transactions))
	}

	// Admin melihat riwayat siswa (tanpa jatah Daily Quiz). Siswa tidak boleh melihat milik siswa lain.
	var adminView history
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/students/"+ani+"/coin-transactions", "", admin), &adminView)
	if adminView.Student.ID != ani || adminView.DailyQuiz != nil || len(adminView.Transactions) != 3 {
		t.Errorf("riwayat untuk admin: %+v", adminView)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/students/"+ani+"/coin-transactions", "", budiToken); res.Code != http.StatusForbidden {
		t.Errorf("siswa melihat riwayat siswa lain: status %d", res.Code)
	}
	for _, id := range []string{uuid.NewString(), "bukan-uuid", tokenOwnerID(t, db, loginmodel.RoleTutor)} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/students/"+id+"/coin-transactions", "", admin); res.Code != http.StatusNotFound {
			t.Errorf("riwayat %s: status %d, seharusnya 404", id, res.Code)
		}
	}
}

// tokenOwnerID membuat user ber-role tertentu (bukan siswa) dan mengembalikan ID-nya.
func tokenOwnerID(t *testing.T, db *gorm.DB, role string) string {
	t.Helper()
	return createUser(t, db, role, role+"."+uuid.NewString()[:8], loginmodel.UserStatusActive)
}

// decodeMeta membaca field meta (pagination) dari response.
func decodeMeta(t *testing.T, res *httptest.ResponseRecorder) listMeta {
	t.Helper()
	var body struct {
		Meta listMeta `json:"meta"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response bukan JSON: %v", err)
	}
	return body.Meta
}
