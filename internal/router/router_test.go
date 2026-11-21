// Test endpoint HTTP dari ujung ke ujung: route → handler → service → repository → PostgreSQL.
// Memakai database test (rumbasha_test), bukan database development.
// Jalankan dengan: make test   (atau: go test -p 1 ./...)
//
// Catatan Go: file test selalu berakhiran _test.go, dan fungsi test
// selalu diawali "Test" dengan parameter (t *testing.T).
package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/testutil"
)

// setupRouter menyusun semua layer persis seperti di main.go, tapi memakai database test.
func setupRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB(t)

	subjectRepository := repository.NewSubjectRepository(db)
	subjectService := service.NewSubjectService(subjectRepository)
	subjectHandler := handler.NewSubjectHandler(subjectService)

	return New(Handlers{Subject: subjectHandler})
}

// sendRequest mengirim request ke router tanpa perlu menjalankan server sungguhan.
func sendRequest(r *gin.Engine, method, url, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	return recorder
}

// createSubject membuat subject lewat API dan mengembalikan ID-nya.
func createSubject(t *testing.T, r *gin.Engine, code, name string) string {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"`+code+`","name":"`+name+`"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("gagal membuat subject: status %d, body: %s", res.Code, res.Body)
	}

	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return body.Data.ID
}

func TestHealth(t *testing.T) {
	res := sendRequest(setupRouter(t), http.MethodGet, "/health", "")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200", res.Code)
	}
}

func TestCreateSubject_KodeDijadikanHurufBesar(t *testing.T) {
	res := sendRequest(setupRouter(t), http.MethodPost, "/api/v1/subjects", `{"code":" mtk ","name":"Matematika"}`)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, seharusnya 201. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"code":"MTK"`) {
		t.Errorf("kode seharusnya MTK. body: %s", res.Body)
	}
}

func TestCreateSubject_ValidasiGagal(t *testing.T) {
	res := sendRequest(setupRouter(t), http.MethodPost, "/api/v1/subjects", `{"name":"Matematika"}`)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"code"`) {
		t.Errorf("seharusnya ada error untuk field code. body: %s", res.Body)
	}
}

func TestCreateSubject_JSONRusak(t *testing.T) {
	res := sendRequest(setupRouter(t), http.MethodPost, "/api/v1/subjects", `{"code":`)

	if res.Code != http.StatusBadRequest {
		t.Errorf("status = %d, seharusnya 400. body: %s", res.Code, res.Body)
	}
}

func TestCreateSubject_KodeKembar(t *testing.T) {
	r := setupRouter(t)
	createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"mtk","name":"Lagi"}`)

	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, seharusnya 409. body: %s", res.Code, res.Body)
	}
}

func TestGetSubject_Berhasil(t *testing.T) {
	r := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodGet, "/api/v1/subjects/"+id, "")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
}

func TestGetSubject_TidakDitemukan(t *testing.T) {
	r := setupRouter(t)

	// ID berformat UUID yang valid, tapi tidak ada di database.
	res := sendRequest(r, http.MethodGet, "/api/v1/subjects/00000000-0000-0000-0000-000000000000", "")
	if res.Code != http.StatusNotFound {
		t.Errorf("UUID tidak ada: status = %d, seharusnya 404. body: %s", res.Code, res.Body)
	}

	// ID yang bukan UUID juga harus 404, bukan 500.
	res = sendRequest(r, http.MethodGet, "/api/v1/subjects/bukan-uuid", "")
	if res.Code != http.StatusNotFound {
		t.Errorf("bukan UUID: status = %d, seharusnya 404. body: %s", res.Code, res.Body)
	}
}

func TestListSubject_Pagination(t *testing.T) {
	r := setupRouter(t)
	createSubject(t, r, "A", "Mapel A")
	createSubject(t, r, "B", "Mapel B")
	createSubject(t, r, "C", "Mapel C")

	res := sendRequest(r, http.MethodGet, "/api/v1/subjects?page=1&per_page=2", "")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	var body struct {
		Data []any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	if len(body.Data) != 2 {
		t.Errorf("jumlah data = %d, seharusnya 2", len(body.Data))
	}
	if body.Meta.Total != 3 {
		t.Errorf("total = %d, seharusnya 3", body.Meta.Total)
	}
}

func TestListSubject_PerPageTerlaluBesar(t *testing.T) {
	res := sendRequest(setupRouter(t), http.MethodGet, "/api/v1/subjects?per_page=500", "")

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"per_page"`) {
		t.Errorf("seharusnya ada error untuk field per_page. body: %s", res.Body)
	}
}

func TestUpdateSubject_Berhasil(t *testing.T) {
	r := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodPut, "/api/v1/subjects/"+id, `{"code":"mtk-1","name":"Matematika Dasar"}`)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"name":"Matematika Dasar"`) {
		t.Errorf("nama seharusnya berubah. body: %s", res.Body)
	}
}

func TestUpdateSubject_KodeDipakaiSubjectLain(t *testing.T) {
	r := setupRouter(t)
	createSubject(t, r, "MTK", "Matematika")
	idIPA := createSubject(t, r, "IPA", "Ilmu Pengetahuan Alam")

	res := sendRequest(r, http.MethodPut, "/api/v1/subjects/"+idIPA, `{"code":"MTK","name":"IPA"}`)

	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, seharusnya 409. body: %s", res.Code, res.Body)
	}
}

func TestDeleteSubject(t *testing.T) {
	r := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodDelete, "/api/v1/subjects/"+id, "")
	if res.Code != http.StatusOK {
		t.Fatalf("hapus: status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	// Setelah dihapus, data tidak boleh ditemukan lagi.
	res = sendRequest(r, http.MethodGet, "/api/v1/subjects/"+id, "")
	if res.Code != http.StatusNotFound {
		t.Errorf("setelah dihapus: status = %d, seharusnya 404", res.Code)
	}
}
