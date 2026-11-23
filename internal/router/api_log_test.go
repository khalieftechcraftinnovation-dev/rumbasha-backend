// Test log API: setiap request tercatat di file log harian, tanpa membocorkan password/token.
package router

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// logEntry adalah isi satu baris file log (hanya field yang dicek di test).
type logEntry struct {
	Time      string          `json:"time"`
	RequestID string          `json:"request_id"`
	Method    string          `json:"method"`
	Endpoint  string          `json:"endpoint"`
	Status    int             `json:"status"`
	UserID    string          `json:"user_id"`
	Payload   json.RawMessage `json:"payload"`
	Process   []string        `json:"process"`
	Response  json.RawMessage `json:"response"`
}

// readLogEntries membaca semua baris file log hari ini.
func readLogEntries(t *testing.T, logDir string) []logEntry {
	t.Helper()
	today := time.Now().In(clock.Jakarta).Format(time.DateOnly)
	content, err := os.ReadFile(filepath.Join(logDir, "api-"+today+".txt"))
	if err != nil {
		t.Fatalf("file log hari ini seharusnya ada: %v", err)
	}

	var entries []logEntry
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("baris log bukan JSON yang valid: %v. baris: %s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestAPILog_LoginTercatatTanpaPasswordDanToken(t *testing.T) {
	app := setupApp(t)
	r, db, apiLog, logDir := app.Router, app.DB, app.APILog, app.LogDir
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login",
		`{"username":"admin.test","password":"`+testPassword+`"}`, "")
	if res.Code != http.StatusOK {
		t.Fatalf("login gagal: status %d, body: %s", res.Code, res.Body)
	}
	var session sessionData
	decodeData(t, res, &session)

	apiLog.Close() // tunggu semua log selesai ditulis ke file
	entries := readLogEntries(t, logDir)
	if len(entries) != 1 {
		t.Fatalf("seharusnya ada 1 baris log, ada %d", len(entries))
	}
	entry := entries[0]

	if entry.Method != http.MethodPost || entry.Endpoint != "/api/v1/auth/login" || entry.Status != http.StatusOK || entry.Time == "" {
		t.Errorf("data request di log salah: %+v", entry)
	}
	if entry.RequestID == "" || entry.RequestID != res.Header().Get("X-Request-ID") {
		t.Errorf("request_id di log (%q) seharusnya sama dengan header X-Request-ID (%q)", entry.RequestID, res.Header().Get("X-Request-ID"))
	}
	if !strings.Contains(string(entry.Payload), `"username":"admin.test"`) {
		t.Errorf("payload seharusnya tercatat. payload: %s", entry.Payload)
	}
	if !strings.Contains(string(entry.Response), `"Login berhasil"`) {
		t.Errorf("response seharusnya tercatat. response: %s", entry.Response)
	}

	// Rahasia tidak boleh ada di mana pun di file log.
	today := time.Now().In(clock.Jakarta).Format(time.DateOnly)
	content, _ := os.ReadFile(filepath.Join(logDir, "api-"+today+".txt"))
	for _, secret := range []string{testPassword, session.AccessToken, session.RefreshToken} {
		if strings.Contains(string(content), secret) {
			t.Errorf("rahasia %q tidak boleh tercatat di log", secret[:8]+"...")
		}
	}
}

func TestAPILog_ProsesValidasiDanUserIDTercatat(t *testing.T) {
	app := setupApp(t)
	r, db, apiLog, logDir := app.Router, app.DB, app.APILog, app.LogDir
	userID := createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, r, "admin.test", testPassword)

	// Validasi gagal: password kosong.
	sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"admin.test"}`, "")
	// Request yang sudah login: user_id harus tercatat.
	sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", session.AccessToken)
	// Token palsu: alasan penolakannya harus tercatat.
	sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", "token-palsu")

	apiLog.Close()
	entries := readLogEntries(t, logDir)
	if len(entries) != 4 {
		t.Fatalf("seharusnya ada 4 baris log, ada %d", len(entries))
	}

	validation := entries[1]
	if validation.Status != http.StatusUnprocessableEntity || len(validation.Process) == 0 || !strings.Contains(validation.Process[0], "password") {
		t.Errorf("error validasi seharusnya tercatat di process: %+v", validation)
	}

	me := entries[2]
	if me.UserID != userID {
		t.Errorf("user_id = %q, seharusnya %q", me.UserID, userID)
	}

	fakeToken := entries[3]
	if fakeToken.Status != http.StatusUnauthorized || len(fakeToken.Process) == 0 {
		t.Errorf("alasan token ditolak seharusnya tercatat di process: %+v", fakeToken)
	}
}
