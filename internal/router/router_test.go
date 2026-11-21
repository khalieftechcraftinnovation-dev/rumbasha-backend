// Test endpoint HTTP dari ujung ke ujung: route → handler → service → repository.
// Jalankan dengan: go test ./internal/router/
package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service"
)

// setupRouter menyusun semua layer persis seperti di main.go.
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	subjectRepository := repository.NewSubjectMemoryRepository()
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

func TestHealth(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodGet, "/health", "")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200", res.Code)
	}
}

func TestCreateSubject_Berhasil(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodPost, "/api/v1/subjects", `{"code":"mtk","name":"Matematika"}`)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, seharusnya 201. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"code":"MTK"`) {
		t.Errorf("kode seharusnya MTK. body: %s", res.Body)
	}
}

func TestCreateSubject_ValidasiGagal(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodPost, "/api/v1/subjects", `{"name":"Matematika"}`)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"code"`) {
		t.Errorf("seharusnya ada error untuk field code. body: %s", res.Body)
	}
}

func TestCreateSubject_JSONRusak(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodPost, "/api/v1/subjects", `{"code":`)

	if res.Code != http.StatusBadRequest {
		t.Errorf("status = %d, seharusnya 400. body: %s", res.Code, res.Body)
	}
}

func TestCreateSubject_KodeKembar(t *testing.T) {
	r := setupRouter()
	sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"MTK","name":"Matematika"}`)

	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"MTK","name":"Lagi"}`)

	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, seharusnya 409. body: %s", res.Code, res.Body)
	}
}

func TestGetSubject_TidakDitemukan(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodGet, "/api/v1/subjects/tidak-ada", "")

	if res.Code != http.StatusNotFound {
		t.Errorf("status = %d, seharusnya 404. body: %s", res.Code, res.Body)
	}
}

func TestListSubject_PerPageTerlaluBesar(t *testing.T) {
	res := sendRequest(setupRouter(), http.MethodGet, "/api/v1/subjects?per_page=500", "")

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"per_page"`) {
		t.Errorf("seharusnya ada error untuk field per_page. body: %s", res.Body)
	}
}
