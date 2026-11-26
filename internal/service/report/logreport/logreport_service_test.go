package logreport

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	logreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/logreport"
)

// fakeSender pura-pura mengirim email: pesan yang "dikirim" disimpan di sent.
type fakeSender struct {
	configured bool
	err        error // kalau diisi, Send mengembalikan error ini
	sent       []mailer.Message
}

func (f *fakeSender) IsConfigured() bool { return f.configured }

func (f *fakeSender) Send(msg mailer.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
}

var testReporter = Reporter{Username: "admin.test", RealName: "Admin Test", Role: "admin"}

// writeTodayLog membuat file log hari ini di folder dir.
func writeTodayLog(t *testing.T, dir string, content []byte) {
	t.Helper()
	today := clock.TodayInJakarta(time.Now())
	if err := os.WriteFile(apilog.FilePath(dir, today), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSendToday_Berhasil(t *testing.T) {
	dir := t.TempDir()
	logContent := []byte(`{"request_id":"abc-123","status":500}` + "\n")
	writeTodayLog(t, dir, logContent)
	sender := &fakeSender{configured: true}
	service := NewLogReportService(sender, dir, "dev@example.com", "development")

	result, err := service.SendToday(testReporter, "Login siswa error 500", "abc-123")
	if err != nil {
		t.Fatal(err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("seharusnya 1 email terkirim, ada %d", len(sender.sent))
	}
	msg := sender.sent[0]
	if len(msg.To) != 1 || msg.To[0] != "dev@example.com" {
		t.Errorf("tujuan email salah: %v", msg.To)
	}
	if !strings.Contains(msg.Subject, "admin.test") {
		t.Errorf("subject seharusnya berisi username pelapor: %q", msg.Subject)
	}
	for _, text := range []string{"Admin Test", "Login siswa error 500", "abc-123"} {
		if !strings.Contains(msg.Body, text) {
			t.Errorf("isi email seharusnya berisi %q:\n%s", text, msg.Body)
		}
	}
	if len(msg.Attachments) != 1 || !bytes.Equal(msg.Attachments[0].Data, logContent) {
		t.Errorf("lampiran seharusnya berisi file log hari ini")
	}
	if result.Compressed || result.FileName != "api-"+clock.TodayInJakarta(time.Now())+".txt" {
		t.Errorf("result salah: %+v", result)
	}
}

func TestSendToday_FileBesarDikompres(t *testing.T) {
	dir := t.TempDir()
	logContent := bytes.Repeat([]byte(`{"request_id":"abc","status":200}`+"\n"), compressAbove/30)
	writeTodayLog(t, dir, logContent)
	sender := &fakeSender{configured: true}
	service := NewLogReportService(sender, dir, "dev@example.com", "development")

	result, err := service.SendToday(testReporter, "tes", "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Compressed || !strings.HasSuffix(result.FileName, ".txt.gz") || result.FileSize >= len(logContent) {
		t.Fatalf("file besar seharusnya dikompres: %+v (asli %d byte)", result, len(logContent))
	}

	// Pastikan isi gzip bisa diekstrak kembali menjadi log aslinya.
	reader, err := gzip.NewReader(bytes.NewReader(sender.sent[0].Attachments[0].Data))
	if err != nil {
		t.Fatal(err)
	}
	extracted, _ := io.ReadAll(reader)
	if !bytes.Equal(extracted, logContent) {
		t.Error("isi gzip tidak sama dengan log asli")
	}
}

func TestSendToday_BelumDikonfigurasi(t *testing.T) {
	dir := t.TempDir()
	writeTodayLog(t, dir, []byte("x\n"))

	notConfigured := NewLogReportService(&fakeSender{configured: false}, dir, "dev@example.com", "development")
	if _, err := notConfigured.SendToday(testReporter, "tes", ""); !errors.Is(err, logreportmodel.ErrNotConfigured) {
		t.Errorf("SMTP kosong: err = %v, seharusnya ErrNotConfigured", err)
	}

	noRecipient := NewLogReportService(&fakeSender{configured: true}, dir, "", "development")
	if _, err := noRecipient.SendToday(testReporter, "tes", ""); !errors.Is(err, logreportmodel.ErrNotConfigured) {
		t.Errorf("LOG_REPORT_EMAIL kosong: err = %v, seharusnya ErrNotConfigured", err)
	}
}

func TestSendToday_FileLogBelumAda(t *testing.T) {
	service := NewLogReportService(&fakeSender{configured: true}, t.TempDir(), "dev@example.com", "development")

	if _, err := service.SendToday(testReporter, "tes", ""); !errors.Is(err, logreportmodel.ErrLogFileNotFound) {
		t.Errorf("err = %v, seharusnya ErrLogFileNotFound", err)
	}
}

func TestSendToday_TidakBolehKirimBeruntunTapiBolehUlangSetelahGagal(t *testing.T) {
	dir := t.TempDir()
	writeTodayLog(t, dir, []byte("x\n"))
	sender := &fakeSender{configured: true, err: errors.New("SMTP mati")}
	service := NewLogReportService(sender, dir, "dev@example.com", "development")

	// Percobaan 1 gagal (SMTP mati) → error ErrSendFailed.
	if _, err := service.SendToday(testReporter, "tes", ""); !errors.Is(err, logreportmodel.ErrSendFailed) {
		t.Fatalf("err = %v, seharusnya ErrSendFailed", err)
	}

	// Percobaan 2 langsung setelah gagal → boleh (tidak kena jeda).
	sender.err = nil
	if _, err := service.SendToday(testReporter, "tes", ""); err != nil {
		t.Fatalf("setelah gagal seharusnya boleh langsung coba lagi: %v", err)
	}

	// Percobaan 3 langsung setelah berhasil → ditolak (jeda 1 menit).
	if _, err := service.SendToday(testReporter, "tes", ""); !errors.Is(err, logreportmodel.ErrTooSoon) {
		t.Errorf("err = %v, seharusnya ErrTooSoon", err)
	}
	if len(sender.sent) != 1 {
		t.Errorf("seharusnya hanya 1 email terkirim, ada %d", len(sender.sent))
	}
}
