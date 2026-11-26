// Package scheduler menjalankan pekerjaan terjadwal (cron) di dalam proses API:
//   - ganti kode akses quiz setiap jam
//   - bagikan hadiah coin juara quiz setiap hari 23:59 WIB, dan hadiah juara bulanan Daily Quiz
//     (per kelas) pada hari terakhir bulan 23:59 WIB
//
// Aman dijalankan di beberapa server sekaligus: ganti kode akses memakai advisory lock PostgreSQL,
// bagi hadiah mengunci baris quiz + idempotency key coin (lihat repository), jadi tidak dobel.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/dailyquiz"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

// QuizJobs adalah pekerjaan terjadwal quiz (dipenuhi oleh quizservice.QuizService).
type QuizJobs interface {
	RefreshAccessCodes(ctx context.Context) (refreshed int, skipped bool, err error)
	DistributePrizes(ctx context.Context) (quizmodel.PrizeRunResult, error)
}

// DailyQuizJobs adalah pekerjaan terjadwal Daily Quiz (dipenuhi oleh dailyquizservice.DailyQuizService).
type DailyQuizJobs interface {
	DistributeMonthlyPrizes(ctx context.Context) (dailyquizmodel.MonthlyPrizeResult, error)
}

// Specs adalah jadwal cron 5 kolom dalam WIB. "off" atau kosong = pekerjaan itu tidak dijadwalkan.
type Specs struct {
	AccessCode string // contoh "0 * * * *" = setiap jam tepat
	QuizPrize  string // contoh "59 23 * * *" = setiap hari 23:59 (hadiah quiz + hadiah bulanan Daily Quiz)
}

// jobTimeout = batas waktu 1 kali jalan, supaya pekerjaan yang macet tidak menumpuk.
const jobTimeout = 5 * time.Minute

// Start membuat & menjalankan scheduler.
// Hentikan saat aplikasi berhenti: <-s.Stop().Done() (menunggu pekerjaan yang sedang jalan).
func Start(specs Specs, jobs QuizJobs, daily DailyQuizJobs) (*cron.Cron, error) {
	c := cron.New(cron.WithLocation(clock.Jakarta))

	add := func(name, env, spec string, run func()) error {
		if spec == "" || spec == "off" {
			log.Printf("scheduler: %s DIMATIKAN (%s=off)", name, env)
			return nil
		}
		if _, err := c.AddFunc(spec, run); err != nil {
			return fmt.Errorf("jadwal %s %q tidak valid: %w", env, spec, err)
		}
		log.Printf("scheduler: %s dijadwalkan %q (WIB)", name, spec)
		return nil
	}
	if err := add("ganti kode akses quiz", "ACCESS_CODE_REFRESH_CRON", specs.AccessCode, func() { RunAccessCodeRefresh(jobs) }); err != nil {
		return nil, err
	}
	if err := add("bagi hadiah juara quiz & Daily Quiz bulanan", "QUIZ_PRIZE_CRON", specs.QuizPrize, func() { RunPrizeDistribution(jobs, daily) }); err != nil {
		return nil, err
	}

	c.Start()
	return c, nil
}

// RunAccessCodeRefresh menjalankan 1 kali ganti kode akses lalu mencatat hasilnya ke log terminal.
// Error tidak menghentikan scheduler; jam berikutnya dicoba lagi.
func RunAccessCodeRefresh(jobs QuizJobs) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	start := time.Now()
	refreshed, skipped, err := jobs.RefreshAccessCodes(ctx)
	switch {
	case err != nil:
		log.Printf("scheduler: GAGAL ganti kode akses quiz: %v", err)
	case skipped:
		log.Println("scheduler: ganti kode akses dilewati (sedang dikerjakan server lain)")
	default:
		log.Printf("scheduler: kode akses %d quiz diganti (%s)", refreshed, time.Since(start).Round(time.Millisecond))
	}
}

// RunPrizeDistribution menjalankan 1 kali bagi hadiah juara quiz lalu mencatat hasilnya ke log terminal.
// Quiz yang gagal dicoba lagi di jadwal berikutnya.
func RunPrizeDistribution(jobs QuizJobs, daily DailyQuizJobs) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	start := time.Now()
	r, err := jobs.DistributePrizes(ctx)
	if err != nil {
		log.Printf("scheduler: GAGAL bagi hadiah quiz: %v", err)
	}
	log.Printf("scheduler: hadiah quiz dibagikan: %d quiz, %d siswa, %d coin, %d quiz gagal (%s)",
		r.Quizzes, r.Awards, r.Coins, r.Failed, time.Since(start).Round(time.Millisecond))

	// Hadiah bulanan Daily Quiz: hanya ada yang dibagikan di hari terakhir bulan (atau bulan lalu
	// yang terlewat); hari lain langsung selesai.
	start = time.Now()
	m, err := daily.DistributeMonthlyPrizes(ctx)
	if err != nil {
		log.Printf("scheduler: GAGAL bagi hadiah Daily Quiz bulanan: %v", err)
	}
	if m.Periods > 0 || m.Failed > 0 {
		log.Printf("scheduler: hadiah Daily Quiz bulanan dibagikan: %d bulan, %d siswa, %d coin, %d bulan gagal (%s)",
			m.Periods, m.Awards, m.Coins, m.Failed, time.Since(start).Round(time.Millisecond))
	}
}
