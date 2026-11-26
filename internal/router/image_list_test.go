// Test galeri Image Library: list (filter kategori & hierarki, cari, urutan, pagination), jumlah
// per kategori, detail, dan informasi soal yang memakai gambar.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

type galleryItem struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	FileName  string `json:"file_name"`
	SizeBytes int64  `json:"size_bytes"`
	Hierarchy struct {
		Subject *listRef `json:"subject"`
		Chapter *listRef `json:"chapter"`
		Topic   *listRef `json:"topic"`
	} `json:"hierarchy"`
	Uploader  *listRef `json:"uploader"`
	UsedBy    []string `json:"used_by"`
	CanDelete bool     `json:"can_delete"`
}

func gallery(t *testing.T, r *gin.Engine, token, query string) ([]galleryItem, listMeta) {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/images"+query, "", token)
	if res.Code != http.StatusOK {
		t.Fatalf("galeri %s: status %d. body: %s", query, res.Code, res.Body)
	}
	var items []galleryItem
	decodeData(t, res, &items)
	return items, decodeMeta(t, res)
}

func galleryNames(items []galleryItem) string {
	names := make([]string, 0, len(items))
	for _, i := range items {
		names = append(names, i.Name)
	}
	return strings.Join(names, ",")
}

func TestImageGallery_ListFilterDanPemakaian(t *testing.T) {
	app := setupApp(t)
	r, db := app.Router, app.DB
	p := seedPlacement(t, db)
	admin := adminToken(t, db)

	// Upload berurutan (nama kecil → terlama). Ukuran berbeda untuk urutan "largest".
	q := uploadImages(t, r, admin, map[string]string{
		"category": "question", "name": "Segitiga", "subject_id": p.SubjectID, "subject_field_id": p.FieldID, "chapter_id": p.ChapterID,
	}, uploadFile{"segitiga-a.png", makeImage(t, "png", 4, 4)}, uploadFile{"segitiga-b.png", makeImage(t, "png", 40, 40)})
	banner := uploadImages(t, r, admin, map[string]string{"category": "banner", "name": "Promo Diskon"},
		uploadFile{"promo.jpg", makeImage(t, "jpeg", 30, 10)})
	landing := uploadImages(t, r, admin, map[string]string{"category": "landing", "name": "Hero Alumni"},
		uploadFile{"hero.gif", makeImage(t, "gif", 8, 8)})
	// Urutan created_at dibuat pasti (upload dalam detik yang sama).
	for i, id := range []string{q[0].ID, q[1].ID, banner[0].ID, landing[0].ID} {
		db.Exec(`UPDATE media_assets SET created_at = NOW() - make_interval(mins => ?) WHERE id = ?`, 10-i, id)
	}

	// Respons upload sudah berisi nama hierarki & pengunggah.
	res := sendUpload(r, map[string]string{"category": "question", "name": "Lingkaran", "subject_id": p.SubjectID}, []uploadFile{{"l.png", makeImage(t, "png", 3, 3)}}, admin)
	var uploaded []galleryItem
	decodeData(t, res, &uploaded)
	if res.Code != http.StatusCreated || uploaded[0].Hierarchy.Subject == nil || uploaded[0].Hierarchy.Subject.Name == "" || uploaded[0].Uploader == nil {
		t.Errorf("respons upload: %d %s", res.Code, res.Body)
	}
	lingkaran := uploaded[0].ID

	// Gambar soal dipakai 1 soal aktif.
	createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"image_id": fmt.Sprintf("%q", q[0].ID), "display_order": `["question","image"]`,
	}))

	// Default: terbaru dulu, 15 per halaman.
	items, meta := gallery(t, r, admin, "")
	if galleryNames(items) != "Lingkaran,Hero Alumni,Promo Diskon,Segitiga 2,Segitiga 1" || meta.PerPage != 15 || meta.Total != 5 {
		t.Errorf("galeri default: %s %+v", galleryNames(items), meta)
	}
	seg1 := items[4]
	if seg1.Hierarchy.Chapter == nil || seg1.Hierarchy.Chapter.ID != p.ChapterID || seg1.Hierarchy.Topic != nil ||
		len(seg1.UsedBy) != 1 || !strings.HasPrefix(seg1.UsedBy[0], "Q-") || seg1.CanDelete || !items[3].CanDelete {
		t.Errorf("baris gambar soal: %+v", seg1)
	}

	for query, want := range map[string]string{
		"?category=question":                          "Lingkaran,Segitiga 2,Segitiga 1",
		"?category=banner":                            "Promo Diskon",
		"?search=SEGI":                                "Segitiga 2,Segitiga 1",
		"?search=hero.gif":                            "Hero Alumni", // nama file
		"?chapter_id=" + p.ChapterID:                  "Segitiga 2,Segitiga 1",
		"?subject_id=" + p.SubjectID + "&sort=oldest": "Segitiga 1,Segitiga 2,Lingkaran",
		"?sort=name&category=question":                "Lingkaran,Segitiga 1,Segitiga 2",
		"?topic_id=" + p.TopicID:                      "",
	} {
		if items, _ := gallery(t, r, admin, query); galleryNames(items) != want {
			t.Errorf("filter %s: %q, seharusnya %q", query, galleryNames(items), want)
		}
	}
	db.Exec(`UPDATE media_assets SET size_bytes = 999999 WHERE id = ?`, q[1].ID) // Segitiga 2 terbesar
	if items, _ := gallery(t, r, admin, "?sort=largest&category=question"); items[0].Name != "Segitiga 2" {
		t.Errorf("urut ukuran: %s", galleryNames(items))
	}

	// Pagination.
	items, meta = gallery(t, r, admin, "?per_page=2&page=3")
	if galleryNames(items) != "Segitiga 1" || meta.Total != 5 || meta.Page != 3 {
		t.Errorf("halaman 3: %s %+v", galleryNames(items), meta)
	}
	items, meta = gallery(t, r, admin, "?per_page=2&page=1")
	if len(items) != 2 || meta.Total != 5 {
		t.Errorf("halaman 1: %d %+v", len(items), meta)
	}

	// Jumlah per kategori (filter search ikut, kategori diabaikan).
	var counts map[string]int
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/images/category-counts?category=banner", "", admin), &counts)
	if counts["all"] != 5 || counts["question"] != 3 || counts["banner"] != 1 || counts["landing"] != 1 {
		t.Errorf("jumlah per kategori: %v", counts)
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/images/category-counts?search=segitiga", "", admin), &counts)
	if counts["all"] != 2 || counts["question"] != 2 {
		t.Errorf("jumlah per kategori + cari: %v", counts)
	}

	// Detail.
	var detail galleryItem
	res = sendRequest(r, http.MethodGet, "/api/v1/images/"+q[0].ID, "", admin)
	decodeData(t, res, &detail)
	if res.Code != http.StatusOK || detail.FileName != "segitiga-a.png" || len(detail.UsedBy) != 1 {
		t.Errorf("detail gambar: %d %+v", res.Code, detail)
	}
	for _, id := range []string{uuid.NewString(), "bukan-uuid"} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/images/"+id, "", admin); res.Code != http.StatusNotFound {
			t.Errorf("detail %s: %d", id, res.Code)
		}
	}
	// Gambar yang dihapus tidak tampil lagi.
	sendRequest(r, http.MethodDelete, "/api/v1/images", fmt.Sprintf(`{"ids":[%q]}`, lingkaran), admin)
	if _, meta := gallery(t, r, admin, ""); meta.Total != 4 {
		t.Errorf("total setelah hapus: %d", meta.Total)
	}

	// Validasi & hak akses.
	if res := sendRequest(r, http.MethodGet, "/api/v1/images?category=video&sort=acak&per_page=500", "", admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("query tidak valid: %d", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/images", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor membuka galeri: %d", res.Code)
	}
}
