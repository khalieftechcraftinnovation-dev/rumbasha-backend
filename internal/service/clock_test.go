// Unit test aturan waktu. Tidak butuh database.
// Jalankan dengan: go test ./internal/service/
package service

import (
	"testing"
	"time"
)

func TestStartOfNextMonth(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "tengah bulan",
			now:  time.Date(2026, 11, 22, 10, 0, 0, 0, jakarta),
			want: time.Date(2026, 12, 1, 0, 0, 0, 0, jakarta),
		},
		{
			name: "detik terakhir bulan Desember pindah ke Januari tahun depan",
			now:  time.Date(2026, 12, 31, 23, 59, 59, 0, jakarta),
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, jakarta),
		},
		{
			name: "tepat tanggal 1 jam 00:00 berarti sesi sampai awal bulan berikutnya",
			now:  time.Date(2026, 12, 1, 0, 0, 0, 0, jakarta),
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, jakarta),
		},
		{
			// 30 Nov 20:00 UTC = 1 Des 03:00 WIB, jadi di WIB sudah bulan Desember.
			name: "waktu UTC dihitung menurut tanggal WIB",
			now:  time.Date(2026, 11, 30, 20, 0, 0, 0, time.UTC),
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, jakarta),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := startOfNextMonth(tt.now)
			if !got.Equal(tt.want) {
				t.Errorf("startOfNextMonth(%v) = %v, seharusnya %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestTodayInJakarta(t *testing.T) {
	// 22 Nov 18:00 UTC = 23 Nov 01:00 WIB → tanggal WIB sudah 23.
	got := todayInJakarta(time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC))
	if got != "2026-11-23" {
		t.Errorf("todayInJakarta = %q, seharusnya 2026-11-23", got)
	}
}
