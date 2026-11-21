// Test endpoint /subjects (contoh CRUD), termasuk pembatasan akses per role.
package router

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// createSubject membuat subject lewat API (sebagai admin) dan mengembalikan ID-nya.
func createSubject(t *testing.T, r *gin.Engine, code, name string) string {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/subjects",
		`{"code":"`+code+`","name":"`+name+`"}`, tokenFor(t, loginmodel.RoleAdmin))
	if res.Code != http.StatusCreated {
		t.Fatalf("gagal membuat subject: status %d, body: %s", res.Code, res.Body)
	}

	var subject struct {
		ID string `json:"id"`
	}
	decodeData(t, res, &subject)
	return subject.ID
}

func TestHealth(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodGet, "/health", "", "")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200", res.Code)
	}
}

func TestSubject_TanpaLoginDitolak(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodGet, "/api/v1/subjects", "", "")

	if res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, seharusnya 401. body: %s", res.Code, res.Body)
	}
}

func TestSubject_SiswaBolehMelihatTapiTidakBolehMengubah(t *testing.T) {
	r, _ := setupRouter(t)
	studentToken := tokenFor(t, loginmodel.RoleStudent)

	res := sendRequest(r, http.MethodGet, "/api/v1/subjects", "", studentToken)
	if res.Code != http.StatusOK {
		t.Errorf("melihat: status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"MTK","name":"Matematika"}`, studentToken)
	if res.Code != http.StatusForbidden {
		t.Errorf("membuat: status = %d, seharusnya 403. body: %s", res.Code, res.Body)
	}
}

func TestCreateSubject_KodeDijadikanHurufBesar(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":" mtk ","name":"Matematika"}`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, seharusnya 201. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"code":"MTK"`) {
		t.Errorf("kode seharusnya MTK. body: %s", res.Body)
	}
}

func TestCreateSubject_ValidasiGagal(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"name":"Matematika"}`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"code"`) {
		t.Errorf("seharusnya ada error untuk field code. body: %s", res.Body)
	}
}

func TestCreateSubject_JSONRusak(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusBadRequest {
		t.Errorf("status = %d, seharusnya 400. body: %s", res.Code, res.Body)
	}
}

func TestCreateSubject_KodeKembar(t *testing.T) {
	r, _ := setupRouter(t)
	createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodPost, "/api/v1/subjects", `{"code":"mtk","name":"Lagi"}`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, seharusnya 409. body: %s", res.Code, res.Body)
	}
}

func TestGetSubject_Berhasil(t *testing.T) {
	r, _ := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodGet, "/api/v1/subjects/"+id, "", tokenFor(t, loginmodel.RoleStudent))

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
}

func TestGetSubject_TidakDitemukan(t *testing.T) {
	r, _ := setupRouter(t)
	studentToken := tokenFor(t, loginmodel.RoleStudent)

	// ID berformat UUID yang valid, tapi tidak ada di database.
	res := sendRequest(r, http.MethodGet, "/api/v1/subjects/00000000-0000-0000-0000-000000000000", "", studentToken)
	if res.Code != http.StatusNotFound {
		t.Errorf("UUID tidak ada: status = %d, seharusnya 404. body: %s", res.Code, res.Body)
	}

	// ID yang bukan UUID juga harus 404, bukan 500.
	res = sendRequest(r, http.MethodGet, "/api/v1/subjects/bukan-uuid", "", studentToken)
	if res.Code != http.StatusNotFound {
		t.Errorf("bukan UUID: status = %d, seharusnya 404. body: %s", res.Code, res.Body)
	}
}

func TestListSubject_Pagination(t *testing.T) {
	r, _ := setupRouter(t)
	createSubject(t, r, "A", "Mapel A")
	createSubject(t, r, "B", "Mapel B")
	createSubject(t, r, "C", "Mapel C")

	res := sendRequest(r, http.MethodGet, "/api/v1/subjects?page=1&per_page=2", "", tokenFor(t, loginmodel.RoleStudent))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	var subjects []any
	decodeData(t, res, &subjects)
	if len(subjects) != 2 {
		t.Errorf("jumlah data = %d, seharusnya 2", len(subjects))
	}
	if !strings.Contains(res.Body.String(), `"total":3`) {
		t.Errorf("total seharusnya 3. body: %s", res.Body)
	}
}

func TestListSubject_PerPageTerlaluBesar(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodGet, "/api/v1/subjects?per_page=500", "", tokenFor(t, loginmodel.RoleStudent))

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"per_page"`) {
		t.Errorf("seharusnya ada error untuk field per_page. body: %s", res.Body)
	}
}

func TestUpdateSubject_Berhasil(t *testing.T) {
	r, _ := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")

	res := sendRequest(r, http.MethodPut, "/api/v1/subjects/"+id, `{"code":"mtk-1","name":"Matematika Dasar"}`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"name":"Matematika Dasar"`) {
		t.Errorf("nama seharusnya berubah. body: %s", res.Body)
	}
}

func TestUpdateSubject_KodeDipakaiSubjectLain(t *testing.T) {
	r, _ := setupRouter(t)
	createSubject(t, r, "MTK", "Matematika")
	idIPA := createSubject(t, r, "IPA", "Ilmu Pengetahuan Alam")

	res := sendRequest(r, http.MethodPut, "/api/v1/subjects/"+idIPA, `{"code":"MTK","name":"IPA"}`, tokenFor(t, loginmodel.RoleAdmin))

	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, seharusnya 409. body: %s", res.Code, res.Body)
	}
}

func TestDeleteSubject(t *testing.T) {
	r, _ := setupRouter(t)
	id := createSubject(t, r, "MTK", "Matematika")
	adminToken := tokenFor(t, loginmodel.RoleSuperAdmin)

	res := sendRequest(r, http.MethodDelete, "/api/v1/subjects/"+id, "", adminToken)
	if res.Code != http.StatusOK {
		t.Fatalf("hapus: status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	// Setelah dihapus, data tidak boleh ditemukan lagi.
	res = sendRequest(r, http.MethodGet, "/api/v1/subjects/"+id, "", adminToken)
	if res.Code != http.StatusNotFound {
		t.Errorf("setelah dihapus: status = %d, seharusnya 404", res.Code)
	}
}
