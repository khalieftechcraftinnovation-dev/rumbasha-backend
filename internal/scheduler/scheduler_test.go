// Unit test scheduler. Tidak butuh database (memakai refresher palsu).
// Jalankan dengan: go test ./internal/scheduler/
package scheduler

import (
	"context"
	"testing"
	"time"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/dailyquiz"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

type fakeRefresher struct{ calls, prizeCalls int }

func (f *fakeRefresher) RefreshAccessCodes(context.Context) (int, bool, error) {
	f.calls++
	return 3, false, nil
}

type fakeDaily struct{ calls int }

func (f *fakeDaily) DistributeMonthlyPrizes(context.Context) (dailyquizmodel.MonthlyPrizeResult, error) {
	f.calls++
	return dailyquizmodel.MonthlyPrizeResult{}, nil
}

func (f *fakeRefresher) DistributePrizes(context.Context) (quizmodel.PrizeRunResult, error) {
	f.prizeCalls++
	return quizmodel.PrizeRunResult{Quizzes: 1, Awards: 3, Coins: 900}, nil
}

func TestStart_JadwalValidOffDanSalah(t *testing.T) {
	f := &fakeRefresher{}

	c, err := Start(Specs{AccessCode: "0 * * * *", QuizPrize: "off"}, f, &fakeDaily{})
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

	off, err := Start(Specs{AccessCode: "off", QuizPrize: "off"}, f, &fakeDaily{})
	if err != nil || len(off.Entries()) != 0 {
		t.Errorf("off: err %v, pekerjaan %d, seharusnya tidak ada pekerjaan", err, len(off.Entries()))
	}
	<-off.Stop().Done()

	if _, err := Start(Specs{AccessCode: "tiap jam"}, f, &fakeDaily{}); err == nil {
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

func TestStart_JadwalHadiahSetiapHari2359(t *testing.T) {
	f := &fakeRefresher{}
	c, err := Start(Specs{AccessCode: "off", QuizPrize: "59 23 * * *"}, f, &fakeDaily{})
	if err != nil {
		t.Fatalf("jadwal hadiah seharusnya valid: %v", err)
	}
	defer func() { <-c.Stop().Done() }()
	entries := c.Entries()
	if len(entries) != 1 {
		t.Fatalf("jumlah pekerjaan = %d, seharusnya 1", len(entries))
	}
	next := entries[0].Next
	if next.Hour() != 23 || next.Minute() != 59 || next.Location().String() != "WIB" {
		t.Errorf("jadwal berikutnya %s, seharusnya 23:59 WIB", next)
	}
}

func TestRunPrizeDistribution_MemanggilService(t *testing.T) {
	f := &fakeRefresher{}
	d := &fakeDaily{}
	RunPrizeDistribution(f, d)
	if f.prizeCalls != 1 || d.calls != 1 {
		t.Errorf("bagi hadiah quiz dipanggil %d kali, Daily Quiz bulanan %d kali, seharusnya 1 & 1", f.prizeCalls, d.calls)
	}
}
