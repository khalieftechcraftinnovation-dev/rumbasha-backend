// Unit test kode akses. Tidak butuh database.
// Jalankan dengan: go test ./internal/accesscode/
package accesscode

import (
	"regexp"
	"testing"
)

var codeFormat = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func TestNew_FormatDanAcak(t *testing.T) {
	seen := map[string]bool{}
	chars := map[rune]bool{}
	for i := 0; i < 2000; i++ {
		code, err := New()
		if err != nil {
			t.Fatal(err)
		}
		if !codeFormat.MatchString(code) {
			t.Fatalf("kode %q tidak sesuai format A-Z0-9 6 karakter", code)
		}
		seen[code] = true
		for _, c := range code {
			chars[c] = true
		}
	}
	// 2000 kode dari 2,1 miliar kemungkinan: praktis tidak mungkin ada yang kembar.
	if len(seen) < 1995 {
		t.Errorf("kode unik hanya %d dari 2000, seharusnya hampir semua berbeda", len(seen))
	}
	// 12.000 karakter acak pasti memakai ke-36 karakter (huruf besar & angka).
	if len(chars) != 36 {
		t.Errorf("karakter yang muncul %d, seharusnya 36 (A-Z dan 0-9)", len(chars))
	}
}

func TestNewUnique_TidakKembarDanMenghindariExclude(t *testing.T) {
	exclude := map[string]bool{"AAAAAA": true}
	codes, err := NewUnique(500, exclude)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if seen[c] || exclude[c] || !codeFormat.MatchString(c) {
			t.Fatalf("kode %q kembar / ada di exclude / format salah", c)
		}
		seen[c] = true
	}
	if len(codes) != 500 {
		t.Errorf("jumlah kode %d, seharusnya 500", len(codes))
	}
}
