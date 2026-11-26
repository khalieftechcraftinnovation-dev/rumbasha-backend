// Test endpoint Landing Page: draft (ubah isi, tampil, urutan) → Publish → halaman publik.
package router

import (
	"net/http"
	"testing"

	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// resetLandingPages mengembalikan data Landing Page ke data awal migration (tabel landing_pages &
// landing_sections tidak dikosongkan antar test). Dipanggil di awal & akhir test.
func resetLandingPages(t *testing.T, db *gorm.DB) {
	t.Helper()
	reset := func() {
		db.Exec(`DELETE FROM landing_changes`)
		db.Exec(`DELETE FROM landing_publish_logs`)
		db.Exec(`UPDATE landing_pages SET published_content = NULL, published_at = NULL, published_by = NULL`)
		db.Exec(`
			UPDATE landing_sections SET content = '{}', is_visible = TRUE, updated_by = NULL,
				sort_order = CASE key
					WHEN 'hero' THEN 1 WHEN 'video_profile' THEN 2 WHEN 'achievements' THEN 3
					WHEN 'programs' THEN 4 WHEN 'facilities' THEN 5 WHEN 'testimonials' THEN 6
					WHEN 'promo' THEN 7 WHEN 'faq' THEN 8 ELSE 1 END`)
	}
	reset()
	t.Cleanup(reset)
}

type landingDraft struct {
	Page struct {
		Code         string  `json:"code"`
		PublishedAt  *string `json:"published_at"`
		PendingCount int     `json:"pending_count"`
		PublishedBy  *struct {
			Name *string `json:"name"`
		} `json:"published_by"`
	} `json:"page"`
	Sections []struct {
		Key       string         `json:"key"`
		Content   map[string]any `json:"content"`
		IsVisible bool           `json:"is_visible"`
		SortOrder int            `json:"sort_order"`
	} `json:"sections"`
	PendingChanges []struct {
		SectionKey  *string `json:"section_key"`
		Description string  `json:"description"`
	} `json:"pending_changes"`
}

type landingPublic struct {
	Code     string `json:"code"`
	Sections []struct {
		Key     string         `json:"key"`
		Content map[string]any `json:"content"`
	} `json:"sections"`
	PublishedAt *string `json:"published_at"`
}

func TestLandingPage_HakAkses(t *testing.T) {
	r, db := setupRouter(t)
	resetLandingPages(t, db)

	if res := sendRequest(r, http.MethodGet, "/api/v1/landing-pages/home", "", ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa login: status = %d, want 401", res.Code)
	}
	student := tokenFor(t, loginmodel.RoleStudent)
	if res := sendRequest(r, http.MethodPost, "/api/v1/landing-pages/home/publish", "", student); res.Code != http.StatusForbidden {
		t.Fatalf("siswa: status = %d, want 403", res.Code)
	}
	// Halaman publik boleh tanpa login.
	if res := sendRequest(r, http.MethodGet, "/api/v1/public/landing-pages/home", "", ""); res.Code != http.StatusOK {
		t.Fatalf("publik: status = %d, want 200. body: %s", res.Code, res.Body.String())
	}
}

func TestLandingPage_DraftLaluPublish(t *testing.T) {
	r, db := setupRouter(t)
	resetLandingPages(t, db)
	admin := adminToken(t, db)

	// Daftar tab: 3 halaman bawaan.
	res := sendRequest(r, http.MethodGet, "/api/v1/landing-pages", "", admin)
	var pages []struct{ Code string }
	decodeData(t, res, &pages)
	if len(pages) != 3 || pages[0].Code != "home" {
		t.Fatalf("pages = %+v, want home, pricing, achievers", pages)
	}

	// Sebelum publish: halaman publik kosong.
	var public landingPublic
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/public/landing-pages/home", "", ""), &public)
	if len(public.Sections) != 0 || public.PublishedAt != nil {
		t.Fatalf("sebelum publish: %+v, want kosong", public)
	}

	// 1. Ubah isi Hero.
	res = sendRequest(r, http.MethodPut, "/api/v1/landing-pages/home/sections/hero",
		`{"content": {"title": "Belajar seru di Rumbasha", "button": "Daftar"}}`, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah hero: status = %d. body: %s", res.Code, res.Body.String())
	}
	// Isi yang sama persis (beda spasi) tidak dicatat sebagai perubahan.
	sendRequest(r, http.MethodPut, "/api/v1/landing-pages/home/sections/hero",
		`{"content": {"title":"Belajar seru di Rumbasha","button":"Daftar"}}`, admin)

	// 2. Sembunyikan Promo.
	res = sendRequest(r, http.MethodPatch, "/api/v1/landing-pages/home/sections/promo/visibility", `{"is_visible": false}`, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("sembunyikan promo: status = %d. body: %s", res.Code, res.Body.String())
	}

	// 3. Pindahkan FAQ ke paling atas.
	res = sendRequest(r, http.MethodPut, "/api/v1/landing-pages/home/sections/order",
		`{"keys": ["faq","hero","video_profile","achievements","programs","facilities","testimonials","promo"]}`, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("urutan: status = %d. body: %s", res.Code, res.Body.String())
	}

	var draft landingDraft
	decodeData(t, res, &draft)
	wantChanges := []string{"Hero · isi diubah", "Promo · disembunyikan", "Urutan section diubah"}
	if len(draft.PendingChanges) != len(wantChanges) || draft.Page.PendingCount != 3 {
		t.Fatalf("pending = %+v (count %d), want %v", draft.PendingChanges, draft.Page.PendingCount, wantChanges)
	}
	for i, want := range wantChanges {
		if draft.PendingChanges[i].Description != want {
			t.Errorf("pending[%d] = %q, want %q", i, draft.PendingChanges[i].Description, want)
		}
	}
	if draft.Sections[0].Key != "faq" || draft.Sections[1].Content["title"] != "Belajar seru di Rumbasha" {
		t.Errorf("draft sections = %+v", draft.Sections[:2])
	}

	// Draft belum tayang.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/public/landing-pages/home", "", ""), &public)
	if len(public.Sections) != 0 {
		t.Fatalf("draft sudah tayang sebelum publish: %+v", public.Sections)
	}

	// Publish.
	res = sendRequest(r, http.MethodPost, "/api/v1/landing-pages/home/publish", "", admin)
	if res.Code != http.StatusOK {
		t.Fatalf("publish: status = %d. body: %s", res.Code, res.Body.String())
	}
	var published struct {
		Changes []string `json:"changes"`
	}
	decodeData(t, res, &published)
	if len(published.Changes) != 3 {
		t.Errorf("changes = %v, want 3", published.Changes)
	}

	// Halaman publik: 7 section (promo disembunyikan), FAQ di atas, isi hero terbaru.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/public/landing-pages/home", "", ""), &public)
	if len(public.Sections) != 7 || public.Sections[0].Key != "faq" || public.PublishedAt == nil {
		t.Fatalf("publik = %+v, want 7 section diawali faq", public)
	}
	for _, s := range public.Sections {
		if s.Key == "promo" {
			t.Errorf("promo seharusnya disembunyikan")
		}
		if s.Key == "hero" && s.Content["title"] != "Belajar seru di Rumbasha" {
			t.Errorf("hero content = %v", s.Content)
		}
	}

	// Panel admin: tidak ada perubahan tertunda, pelaku publish tercatat.
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/landing-pages/home", "", admin), &draft)
	if draft.Page.PendingCount != 0 || len(draft.PendingChanges) != 0 || draft.Page.PublishedBy == nil {
		t.Errorf("setelah publish: %+v", draft.Page)
	}

	// Publish lagi tanpa perubahan → 409.
	if res := sendRequest(r, http.MethodPost, "/api/v1/landing-pages/home/publish", "", admin); res.Code != http.StatusConflict {
		t.Errorf("publish tanpa perubahan: status = %d, want 409", res.Code)
	}

	// Riwayat publish tersimpan.
	var logs int
	db.Raw(`SELECT COUNT(*) FROM landing_publish_logs WHERE page_code = 'home'`).Scan(&logs)
	if logs != 1 {
		t.Errorf("publish logs = %d, want 1", logs)
	}
}

func TestLandingPage_Validasi(t *testing.T) {
	r, db := setupRouter(t)
	resetLandingPages(t, db)
	admin := adminToken(t, db)

	cases := []struct {
		name, method, url, body string
		wantStatus              int
		wantField               string
	}{
		{"content bukan object", http.MethodPut, "/api/v1/landing-pages/home/sections/hero", `{"content": [1,2]}`, 422, "content"},
		{"halaman tidak ada", http.MethodGet, "/api/v1/landing-pages/blog", "", 404, ""},
		{"section tidak ada", http.MethodPut, "/api/v1/landing-pages/home/sections/footer", `{"content": {}}`, 404, ""},
		{"section halaman lain", http.MethodPatch, "/api/v1/landing-pages/pricing/sections/hero/visibility", `{"is_visible": false}`, 404, ""},
		{"is_visible wajib", http.MethodPatch, "/api/v1/landing-pages/home/sections/hero/visibility", `{}`, 422, "is_visible"},
		{"urutan kurang", http.MethodPut, "/api/v1/landing-pages/home/sections/order", `{"keys": ["hero","faq"]}`, 422, "keys"},
		{"urutan key asing", http.MethodPut, "/api/v1/landing-pages/home/sections/order",
			`{"keys": ["footer","hero","video_profile","achievements","programs","facilities","testimonials","promo","faq"]}`, 422, "keys[0]"},
		{"publik halaman tidak ada", http.MethodGet, "/api/v1/public/landing-pages/blog", "", 404, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := sendRequest(r, tc.method, tc.url, tc.body, admin)
			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. body: %s", res.Code, tc.wantStatus, res.Body.String())
			}
			if tc.wantField != "" {
				if _, ok := errorFields(t, res.Body.String())[tc.wantField]; !ok {
					t.Errorf("error field %q tidak ada. body: %s", tc.wantField, res.Body.String())
				}
			}
		})
	}

	// Tidak ada perubahan yang tercatat dari request yang gagal.
	var pending int
	db.Raw(`SELECT COUNT(*) FROM landing_changes`).Scan(&pending)
	if pending != 0 {
		t.Errorf("pending = %d, want 0", pending)
	}
}
