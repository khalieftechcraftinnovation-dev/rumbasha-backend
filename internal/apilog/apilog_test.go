package apilog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

// dateOffset mengembalikan tanggal WIB hari ini + offset hari, format "2026-11-23".
func dateOffset(offset int) string {
	return time.Now().In(clock.Jakarta).AddDate(0, 0, offset).Format(time.DateOnly)
}

func TestWriter_MenulisKeFileHariIni(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, 15)
	if err != nil {
		t.Fatal(err)
	}

	w.Write([]byte(`{"request_id":"a"}` + "\n"))
	w.Write([]byte(`{"request_id":"b"}` + "\n"))
	w.Close() // menunggu antrean selesai ditulis

	content, err := os.ReadFile(filepath.Join(dir, "api-"+dateOffset(0)+".txt"))
	if err != nil {
		t.Fatalf("file log hari ini seharusnya ada: %v", err)
	}
	if string(content) != `{"request_id":"a"}`+"\n"+`{"request_id":"b"}`+"\n" {
		t.Errorf("isi file salah: %q", content)
	}
}

func TestWriter_HanyaMenyimpan15HariTerakhir(t *testing.T) {
	dir := t.TempDir()
	files := map[string]bool{ // nama file → seharusnya masih ada?
		"api-" + dateOffset(0) + ".txt":   true,  // hari ini
		"api-" + dateOffset(-14) + ".txt": true,  // hari ke-15 (paling lama yang disimpan)
		"api-" + dateOffset(-15) + ".txt": false, // hari ke-16 → dihapus
		"api-2020-01-01.txt":              false,
		"catatan.txt":                     true, // bukan file log → tidak disentuh
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	w, err := NewWriter(dir, 15) // file lama dihapus saat writer dibuat
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	for name, shouldExist := range files {
		_, err := os.Stat(filepath.Join(dir, name))
		if exists := err == nil; exists != shouldExist {
			t.Errorf("%s: ada = %v, seharusnya %v", name, exists, shouldExist)
		}
	}
}

func TestBodyForLog_PasswordDanTokenDisamarkan(t *testing.T) {
	body := []byte(`{"username":"budi","password":"rahasia123","data":{"access_token":"abc","refresh_token":"def","items":[{"Token":"ghi"}]}}`)

	result := string(BodyForLog(body, len(body)))

	for _, secret := range []string{"rahasia123", "abc", "def", "ghi"} {
		if strings.Contains(result, secret) {
			t.Errorf("%q seharusnya disamarkan. hasil: %s", secret, result)
		}
	}
	if !strings.Contains(result, `"username":"budi"`) {
		t.Errorf("field biasa seharusnya tetap tercatat. hasil: %s", result)
	}
}

func TestBodyForLog_BukanJSONYangBerisiPasswordTidakDicatat(t *testing.T) {
	body := []byte(`{"username":"budi","password":"rahasia123"`) // JSON rusak (kurang "}")

	result := string(BodyForLog(body, len(body)))

	if strings.Contains(result, "rahasia123") {
		t.Errorf("password di body rusak tidak boleh tercatat. hasil: %s", result)
	}
}

func TestBodyForLog_BodyBesarHanyaDicatatUkurannya(t *testing.T) {
	result := string(BodyForLog([]byte("{}"), MaxBodySize+1))

	if !strings.Contains(result, "terlalu besar") {
		t.Errorf("body besar seharusnya hanya dicatat ukurannya. hasil: %s", result)
	}
}
