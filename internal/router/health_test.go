// Test endpoint /health (cek server hidup).
package router

import (
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodGet, "/health", "", "")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, seharusnya 200", res.Code)
	}
}
