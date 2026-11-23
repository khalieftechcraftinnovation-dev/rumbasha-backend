package mailer

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// startFakeSMTP menjalankan server SMTP palsu di localhost yang menerima 1 email,
// lalu mengirim isi email mentahnya ke channel. Port-nya dipilih otomatis oleh OS.
func startFakeSMTP(t *testing.T) (port string, received chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	received = make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			command := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(command, "EHLO"):
				_ = tp.PrintfLine("250-fake")
				_ = tp.PrintfLine("250 AUTH PLAIN")
			case strings.HasPrefix(command, "AUTH"):
				_ = tp.PrintfLine("235 login ok")
			case command == "DATA":
				_ = tp.PrintfLine("354 kirim isinya")
				lines, _ := tp.ReadDotLines()
				received <- strings.Join(lines, "\r\n")
				_ = tp.PrintfLine("250 diterima")
			case command == "QUIT":
				_ = tp.PrintfLine("221 bye")
				return
			default: // MAIL FROM, RCPT TO
				_ = tp.PrintfLine("250 ok")
			}
		}
	}()

	_, port, _ = net.SplitHostPort(listener.Addr().String())
	return port, received
}

func TestSend_EmailDanLampiranTerkirimUtuh(t *testing.T) {
	port, received := startFakeSMTP(t)
	// Host 127.0.0.1: smtp.PlainAuth mengizinkan login tanpa TLS hanya ke localhost.
	m := New("127.0.0.1", port, "user", "rahasia", "noreply@example.com")

	logContent := `{"request_id":"abc","status":500}` + "\n"
	err := m.Send(Message{
		To:      []string{"dev@example.com"},
		Subject: "[Rumbasha] Laporan bug dari admin.test",
		Body:    "Halo developer",
		Attachments: []Attachment{
			{FileName: "api-2026-11-23.txt", ContentType: "text/plain; charset=utf-8", Data: []byte(logContent)},
		},
	})
	if err != nil {
		t.Fatalf("kirim email gagal: %v", err)
	}

	var raw string
	select {
	case raw = <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("server SMTP palsu tidak menerima email")
	}

	// Baca ulang email seperti yang dilakukan aplikasi email (Gmail, Outlook).
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("email tidak bisa dibaca: %v", err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "[Rumbasha] Laporan bug dari admin.test" || msg.Header.Get("To") != "dev@example.com" {
		t.Errorf("header salah: subject=%q to=%q", subject, msg.Header.Get("To"))
	}

	_, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	parts := multipart.NewReader(msg.Body, params["boundary"])

	var bodies []string
	var attachmentName string
	for {
		part, err := parts.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := io.ReadAll(part)
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(encoded), "\r\n", ""))
		if err != nil {
			t.Fatalf("isi part bukan base64 yang valid: %v", err)
		}
		bodies = append(bodies, string(decoded))
		if part.FileName() != "" {
			attachmentName = part.FileName()
		}
	}

	if len(bodies) != 2 || bodies[0] != "Halo developer" || bodies[1] != logContent {
		t.Errorf("isi email / lampiran salah: %q", bodies)
	}
	if attachmentName != "api-2026-11-23.txt" {
		t.Errorf("nama lampiran = %q, seharusnya api-2026-11-23.txt", attachmentName)
	}
}

func TestSend_BelumDikonfigurasi(t *testing.T) {
	m := New("", "587", "", "", "")
	if err := m.Send(Message{To: []string{"dev@example.com"}}); err != ErrNotConfigured {
		t.Errorf("err = %v, seharusnya ErrNotConfigured", err)
	}
}
