// Package accesscode membuat kode akses quiz acak: 6 karakter huruf besar A-Z dan angka 0-9,
// contoh "K7Q2ZD". Dipakai bersama oleh fitur quiz (create) dan scheduler (refresh per jam).
package accesscode

import (
	"crypto/rand"
	"fmt"
)

// Length adalah panjang kode akses.
const Length = 6

// alphabet berisi 36 karakter yang boleh dipakai (sama dengan CHECK di database: ^[A-Z0-9]{6}$).
const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// maxUnbiased = kelipatan 36 terbesar yang <= 256. Byte acak >= 252 dibuang supaya setiap
// karakter punya peluang yang sama persis (252 = 7 x 36; byte % 36 jadi tidak berat sebelah).
const maxUnbiased = 256 - 256%len(alphabet)

// New membuat 1 kode akses acak. Memakai crypto/rand (bukan math/rand) supaya kode tidak bisa ditebak.
// Ada 36^6 ≈ 2,1 miliar kemungkinan kode; keunikan tetap dijaga index unik di database.
func New() (string, error) {
	code := make([]byte, 0, Length)
	buf := make([]byte, Length*2) // cadangan, karena sebagian byte bisa dibuang
	for len(code) < Length {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("gagal membuat kode akses: %w", err)
		}
		for _, b := range buf {
			if int(b) < maxUnbiased && len(code) < Length {
				code = append(code, alphabet[int(b)%len(alphabet)])
			}
		}
	}
	return string(code), nil
}

// NewUnique membuat n kode akses yang berbeda satu sama lain dan tidak ada di exclude.
// Dipakai scheduler saat mengganti banyak kode sekaligus.
func NewUnique(n int, exclude map[string]bool) ([]string, error) {
	seen := make(map[string]bool, n)
	codes := make([]string, 0, n)
	for len(codes) < n {
		code, err := New()
		if err != nil {
			return nil, err
		}
		if seen[code] || exclude[code] {
			continue // sangat jarang terjadi; buat ulang
		}
		seen[code] = true
		codes = append(codes, code)
	}
	return codes, nil
}
