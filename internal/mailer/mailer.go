// Package mailer mengirim email lewat SMTP (contoh: Gmail), memakai paket bawaan Go (net/smtp).
// Dipakai bersama oleh beberapa fitur, jadi tidak ditaruh di folder fitur tertentu.
//
// Setting SMTP diisi di .env (SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM).
package mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// timeout = batas waktu seluruh proses kirim (konek + login + kirim data).
// Supaya request tidak menggantung lama kalau server SMTP lambat/tidak bisa dihubungi.
const timeout = 30 * time.Second

// ErrNotConfigured dikembalikan kalau SMTP_HOST / SMTP_FROM belum diisi di .env.
var ErrNotConfigured = errors.New("SMTP belum dikonfigurasi")

// Attachment adalah satu file lampiran.
type Attachment struct {
	FileName    string // nama file yang terlihat di email, contoh api-2026-11-23.txt
	ContentType string // contoh text/plain; charset=utf-8
	Data        []byte
}

// Message adalah isi satu email.
type Message struct {
	To          []string
	Subject     string
	Body        string // teks biasa (bukan HTML)
	Attachments []Attachment
}

// Mailer menyimpan setting SMTP.
type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

// New membuat Mailer. Dipanggil sekali di main.go.
func New(host, port, username, password, from string) *Mailer {
	return &Mailer{host: host, port: port, username: username, password: password, from: from}
}

// IsConfigured bernilai true kalau setting minimal SMTP sudah diisi.
func (m *Mailer) IsConfigured() bool {
	return m.host != "" && m.port != "" && m.from != ""
}

// Send mengirim email. Prosesnya mengikuti percakapan SMTP standar:
// konek → (STARTTLS) → login → MAIL FROM → RCPT TO → DATA → QUIT.
func (m *Mailer) Send(msg Message) error {
	if !m.IsConfigured() {
		return ErrNotConfigured
	}
	if len(msg.To) == 0 {
		return errors.New("alamat tujuan email kosong")
	}

	data, err := buildMessage(m.from, msg, time.Now())
	if err != nil {
		return err
	}

	conn, err := m.dial()
	if err != nil {
		return fmt.Errorf("gagal konek ke SMTP %s:%s: %w", m.host, m.port, err)
	}
	// Seluruh percakapan SMTP harus selesai sebelum batas waktu ini.
	_ = conn.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("gagal memulai SMTP: %w", err)
	}
	defer client.Close()

	// Port 587: koneksi awalnya biasa, lalu dienkripsi dengan STARTTLS (Gmail mendukung).
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("gagal STARTTLS: %w", err)
		}
	}

	// smtp.PlainAuth otomatis menolak mengirim password lewat koneksi yang tidak terenkripsi
	// (kecuali ke localhost), jadi password tidak akan bocor.
	if m.username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return fmt.Errorf("gagal login SMTP (cek SMTP_USERNAME / SMTP_PASSWORD): %w", err)
		}
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM ditolak: %w", err)
	}
	for _, to := range msg.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s ditolak: %w", to, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA ditolak: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("gagal mengirim isi email: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("server SMTP menolak email: %w", err)
	}
	return client.Quit()
}

// dial membuka koneksi ke server SMTP.
// Port 465 langsung memakai TLS; port lain (587, 25) memakai koneksi biasa + STARTTLS.
func (m *Mailer) dial() (net.Conn, error) {
	address := net.JoinHostPort(m.host, m.port)
	dialer := &net.Dialer{Timeout: timeout}
	if m.port == "465" {
		return tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: m.host})
	}
	return dialer.Dial("tcp", address)
}

// buildMessage menyusun email lengkap (header + isi + lampiran) dalam format MIME.
//
// Bentuknya kira-kira:
//
//	From: ...
//	To: ...
//	Subject: ...
//	Content-Type: multipart/mixed; boundary="xyz"
//
//	--xyz
//	(isi email)
//	--xyz
//	(lampiran 1, base64)
//	--xyz--
func buildMessage(from string, msg Message, now time.Time) ([]byte, error) {
	var buf bytes.Buffer
	parts := multipart.NewWriter(&buf)

	// mime.QEncoding: subject dengan karakter non-ASCII / enter tetap aman di header email.
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	buf.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	buf.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(`Content-Type: multipart/mixed; boundary="` + parts.Boundary() + `"` + "\r\n")
	buf.WriteString("\r\n")

	// Bagian 1: isi email.
	body, err := parts.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		return nil, err
	}
	writeBase64(body, []byte(msg.Body))

	// Bagian 2 dst: lampiran.
	for _, file := range msg.Attachments {
		part, err := parts.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {file.ContentType},
			"Content-Disposition":       {mime.FormatMediaType("attachment", map[string]string{"filename": file.FileName})},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return nil, err
		}
		writeBase64(part, file.Data)
	}

	if err := parts.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeBase64 menulis data dalam bentuk base64, dipotong per 76 karakter per baris
// (aturan format email: satu baris tidak boleh terlalu panjang).
func writeBase64(w io.Writer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		_, _ = w.Write([]byte(encoded[:76] + "\r\n"))
		encoded = encoded[76:]
	}
	_, _ = w.Write([]byte(encoded + "\r\n"))
}
