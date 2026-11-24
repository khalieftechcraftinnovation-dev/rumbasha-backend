// Unit test scheduler. Tidak butuh database (memakai refresher palsu).
// Jalankan dengan: go test ./internal/scheduler/
package scheduler

import (
	"context"
	"testing"
	"time"
)

type fakeRefresher struct{ calls int }

func (f *fakeRefresher) RefreshAccessCodes(context.Context) (int, bool, error) {
	f.calls++
	return 3, false, nil
}

func TestStart_JadwalValidOffDanSalah(t *testing.T) {
	f := &fakeRefresher{}

	c, err := Start("0 * * * *", f)
	if err != nil {
		t.Fatalf("jadwal setiap jam seharusnya valid: %v", err)
	}
	entries := c.Entries()
	if len(entries) != 1 {
		t.Fatalf("jumlah pekerjaan = %d, seharusnya 1", len(entries))
	}
	// Jadwal berikutnya tepat di menit 0, dihitung dalam WIB.
	next := entries[0].Next
	if next.Minute() != 0 || next.Second() != 0 || next.Location().String() != "WIB" || next.Sub(time.Now()) > time.Hour {
		t.Errorf("jadwal berikutnya %s, seharusnya jam tepat berikutnya (WIB)", next)
	}
	<-c.Stop().Done()

	off, err := Start("off", f)
	if err != nil || len(off.Entries()) != 0 {
		t.Errorf("off: err %v, pekerjaan %d, seharusnya tidak ada pekerjaan", err, len(off.Entries()))
	}
	<-off.Stop().Done()

	if _, err := Start("tiap jam", f); err == nil {
		t.Error("jadwal salah seharusnya error")
	}
}

func TestRunAccessCodeRefresh_MemanggilRefresher(t *testing.T) {
	f := &fakeRefresher{}
	RunAccessCodeRefresh(f)
	if f.calls != 1 {
		t.Errorf("refresher dipanggil %d kali, seharusnya 1", f.calls)
	}
}
