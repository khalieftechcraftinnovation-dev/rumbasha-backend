package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// responseBody adalah bentuk JSON standar semua response (lihat package response).
// Data memakai json.RawMessage supaya bisa dicek apakah isinya "null".
type responseBody struct {
	StatusCode int             `json:"status_code"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
	Errors     []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

// decodeBody membaca seluruh response JSON, sekaligus memastikan field "data" selalu ada.
func decodeBody(t *testing.T, res *httptest.ResponseRecorder) responseBody {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response bukan JSON yang valid: %v. body: %s", err, res.Body)
	}
	if _, ok := raw["data"]; !ok {
		t.Errorf("field data seharusnya selalu ada. body: %s", res.Body)
	}

	var body responseBody
	_ = json.Unmarshal(res.Body.Bytes(), &body)

	if body.StatusCode != res.Code {
		t.Errorf("status_code = %d, seharusnya sama dengan HTTP status %d. body: %s", body.StatusCode, res.Code, res.Body)
	}
	if body.Message == "" {
		t.Errorf("message tidak boleh kosong. body: %s", res.Body)
	}
	return body
}

func TestResponseFormat_Sukses(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodGet, "/health", "", "")
	body := decodeBody(t, res)

	if body.StatusCode != http.StatusOK {
		t.Errorf("status_code = %d, seharusnya 200", body.StatusCode)
	}
}

func TestResponseFormat_ValidasiGagal(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{}`, "")
	body := decodeBody(t, res)

	if body.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status_code = %d, seharusnya 422. body: %s", body.StatusCode, res.Body)
	}
	if string(body.Data) != "null" {
		t.Errorf("data seharusnya null saat gagal. body: %s", res.Body)
	}
	if len(body.Errors) == 0 {
		t.Errorf("seharusnya ada daftar errors per field. body: %s", res.Body)
	}
}

func TestResponseFormat_EndpointTidakAda(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodGet, "/api/v1/tidak-ada", "", "")
	body := decodeBody(t, res)

	if body.StatusCode != http.StatusNotFound {
		t.Errorf("status_code = %d, seharusnya 404. body: %s", body.StatusCode, res.Body)
	}
}
