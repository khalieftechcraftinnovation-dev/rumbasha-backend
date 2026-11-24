// Package scheduler menjalankan pekerjaan terjadwal (cron) di dalam proses API, contoh
// mengganti kode akses quiz setiap jam.
//
// Aman dijalankan di beberapa server sekaligus: setiap pekerjaan memakai advisory lock
// PostgreSQL (lihat repository), jadi di jam yang sama hanya 1 server yang benar-benar bekerja.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

// AccessCodeRefresher adalah pekerjaan ganti kode akses (dipenuhi oleh quizservice.QuizService).
type AccessCodeRefresher interface {
	RefreshAccessCodes(ctx context.Context) (refreshed int, skipped bool, err error)
}

// jobTimeout = batas waktu 1 kali jalan, supaya pekerjaan yang macet tidak menumpuk.
const jobTimeout = 5 * time.Minute

// Start membuat & menjalankan scheduler. accessCodeSpec = jadwal cron 5 kolom dalam WIB
// (contoh "0 * * * *" = setiap jam tepat); "off" atau kosong = tidak dijadwalkan.
// Hentikan saat aplikasi berhenti: <-s.Stop().Done() (menunggu pekerjaan yang sedang jalan).
func Start(accessCodeSpec string, refresher AccessCodeRefresher) (*cron.Cron, error) {
	c := cron.New(cron.WithLocation(clock.Jakarta))

	if accessCodeSpec != "" && accessCodeSpec != "off" {
		_, err := c.AddFunc(accessCodeSpec, func() { RunAccessCodeRefresh(refresher) })
		if err != nil {
			return nil, fmt.Errorf("jadwal ACCESS_CODE_REFRESH_CRON %q tidak valid: %w", accessCodeSpec, err)
		}
		log.Printf("scheduler: ganti kode akses quiz dijadwalkan %q (WIB)", accessCodeSpec)
	} else {
		log.Println("scheduler: ganti kode akses quiz DIMATIKAN (ACCESS_CODE_REFRESH_CRON=off)")
	}

	c.Start()
	return c, nil
}

// RunAccessCodeRefresh menjalankan 1 kali ganti kode akses lalu mencatat hasilnya ke log terminal.
// Error tidak menghentikan scheduler; jam berikutnya dicoba lagi.
func RunAccessCodeRefresh(refresher AccessCodeRefresher) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	start := time.Now()
	refreshed, skipped, err := refresher.RefreshAccessCodes(ctx)
	switch {
	case err != nil:
		log.Printf("scheduler: GAGAL ganti kode akses quiz: %v", err)
	case skipped:
		log.Println("scheduler: ganti kode akses dilewati (sedang dikerjakan server lain)")
	default:
		log.Printf("scheduler: kode akses %d quiz diganti (%s)", refreshed, time.Since(start).Round(time.Millisecond))
	}
}
