package subscription

import (
	"testing"
	"time"
)

// Tanggal berakhir paket = mulai + n bulan - 1 hari, dengan tanggal dijepit ke akhir bulan.
func TestAddMonths_TanggalBerakhirPaket(t *testing.T) {
	cases := []struct {
		start  string
		months int
		end    string
	}{
		{"2026-11-26", 1, "2026-12-25"},
		{"2026-12-04", 1, "2027-01-03"},
		{"2027-01-31", 1, "2027-02-27"}, // 31 Jan + 1 bulan = 28 Feb (bukan 3 Maret)
		{"2028-01-31", 1, "2028-02-28"}, // tahun kabisat: 29 Feb - 1
		{"2026-08-31", 6, "2027-02-27"},
		{"2026-11-01", 12, "2027-10-31"},
	}
	for _, c := range cases {
		start, _ := time.Parse("2006-01-02", c.start)
		got := addMonths(start, c.months).AddDate(0, 0, -1).Format("2006-01-02")
		if got != c.end {
			t.Errorf("%s + %d bulan: berakhir %s, seharusnya %s", c.start, c.months, got, c.end)
		}
	}
}
