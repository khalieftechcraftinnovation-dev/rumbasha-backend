package quizreport

import (
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

func TestOverviewPeriod_DefaultDanBatas(t *testing.T) {
	now := time.Date(2026, 11, 9, 23, 30, 0, 0, clock.Jakarta) // Senin 9 Nov 2026 WIB

	cases := []struct {
		from, to         string
		wantFrom, wantTo string // inklusif
		wantErr          bool
	}{
		{"", "", "2026-09-11", "2026-11-09", false}, // default 60 hari s/d hari ini
		{"2026-11-01", "", "2026-11-01", "2026-11-09", false},
		{"", "2026-10-31", "2026-09-02", "2026-10-31", false}, // 60 hari s/d date_to
		{"2026-10-01", "2026-10-31", "2026-10-01", "2026-10-31", false},
		{"2026-11-10", "2026-11-01", "", "", true},
		{"2025-01-01", "2026-01-02", "", "", true}, // lebih dari 366 hari
		{"2025-01-01", "2026-01-01", "2025-01-01", "2026-01-01", false},
	}
	for _, c := range cases {
		from, to, err := overviewPeriod(c.from, c.to, 60, now)
		if (err != nil) != c.wantErr {
			t.Errorf("%q-%q: err %v", c.from, c.to, err)
			continue
		}
		if err != nil {
			continue
		}
		gotFrom, gotTo := from.Format(time.DateOnly), to.AddDate(0, 0, -1).Format(time.DateOnly)
		if gotFrom != c.wantFrom || gotTo != c.wantTo {
			t.Errorf("%q-%q: %s s/d %s, seharusnya %s s/d %s", c.from, c.to, gotFrom, gotTo, c.wantFrom, c.wantTo)
		}
	}
}

func TestMondayOf(t *testing.T) {
	for in, want := range map[string]string{
		"2026-11-09": "2026-11-09", // Senin
		"2026-11-15": "2026-11-09", // Minggu
		"2026-11-11": "2026-11-09",
	} {
		d, _ := time.ParseInLocation(time.DateOnly, in, clock.Jakarta)
		if got := mondayOf(d).Format(time.DateOnly); got != want {
			t.Errorf("mondayOf(%s) = %s, seharusnya %s", in, got, want)
		}
	}
}
