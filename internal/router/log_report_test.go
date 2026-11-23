// Test endpoint POST /log-reports: kirim log hari ini ke email developer (khusus admin).
package router

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

const logReportBody = `{"description":"Siswa tidak bisa login, muncul error 500","request_id":"abc-123"}`

func TestLogReport_TanpaLoginDitolak(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodPost, "/api/v1/log-reports", logReportBody, "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, seharusnya 401. body: %s", res.Code, res.Body)
	}
}

func TestLogReport_SiswaDanTutorDitolak(t *testing.T) {
	app := setupApp(t)

	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		res := sendRequest(app.Router, http.MethodPost, "/api/v1/log-reports", logReportBody, tokenFor(t, role))
		if res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403. body: %s", role, res.Code, res.Body)
		}
	}
	if len(app.Mail.sent) != 0 {
		t.Errorf("tidak boleh ada email terkirim, ada %d", len(app.Mail.sent))
	}
}

func TestLogReport_DeskripsiWajibDiisi(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodPost, "/api/v1/log-reports", `{}`, tokenFor(t, loginmodel.RoleAdmin))
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), `"field":"description"`) {
		t.Errorf("status = %d, seharusnya 422 dengan error field description. body: %s", res.Code, res.Body)
	}
}

func TestLogReport_AdminBerhasilKirimLaluHarusMenunggu(t *testing.T) {
	app := setupApp(t)

	// Siapkan file log hari ini (di aplikasi, file ini diisi otomatis oleh middleware APILog).
	today := clock.TodayInJakarta(time.Now())
	logContent := `{"request_id":"abc-123","status":500}` + "\n"
	if err := os.WriteFile(apilog.FilePath(app.LogDir, today), []byte(logContent), 0o644); err != nil {
		t.Fatal(err)
	}

	createUser(t, app.DB, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, app.Router, "admin.test", testPassword)

	res := sendRequest(app.Router, http.MethodPost, "/api/v1/log-reports", logReportBody, session.AccessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	var data struct {
		FileName string `json:"file_name"`
	}
	decodeData(t, res, &data)
	if data.FileName != "api-"+today+".txt" {
		t.Errorf("file_name = %q, seharusnya api-%s.txt", data.FileName, today)
	}

	if len(app.Mail.sent) != 1 {
		t.Fatalf("seharusnya 1 email terkirim, ada %d", len(app.Mail.sent))
	}
	msg := app.Mail.sent[0]
	if !strings.Contains(msg.Subject, "admin.test") || !strings.Contains(msg.Body, "Siswa tidak bisa login") || !strings.Contains(msg.Body, "abc-123") {
		t.Errorf("isi email kurang lengkap.\nsubject: %s\nbody: %s", msg.Subject, msg.Body)
	}
	if len(msg.Attachments) != 1 || !strings.HasPrefix(string(msg.Attachments[0].Data), logContent) {
		t.Errorf("lampiran seharusnya berisi file log hari ini")
	}

	// Kirim lagi langsung → ditolak, harus menunggu 1 menit.
	res = sendRequest(app.Router, http.MethodPost, "/api/v1/log-reports", logReportBody, session.AccessToken)
	if res.Code != http.StatusTooManyRequests {
		t.Errorf("kirim ulang: status = %d, seharusnya 429. body: %s", res.Code, res.Body)
	}
}
