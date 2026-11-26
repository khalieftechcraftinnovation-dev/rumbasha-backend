package quiz

import (
	"context"
	"log"
	"time"

	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

// prizeBatchSize = jumlah quiz yang diambil per putaran saat membagikan hadiah.
const prizeBatchSize = 200

// DistributePrizes membagikan hadiah coin juara 1-3 untuk semua quiz publish yang event-nya sudah
// selesai (end_at <= sekarang) dan belum dibagikan. Dipanggil scheduler setiap hari 23:59 WIB.
//
// Setiap quiz diproses dalam transaksinya sendiri: 1 quiz gagal tidak membatalkan quiz lain,
// dan quiz yang gagal dicoba lagi di jadwal berikutnya. Aman dijalankan di beberapa server
// sekaligus (baris quiz dikunci + idempotency key coin_transactions), coin tidak bisa dobel.
func (s *QuizService) DistributePrizes(ctx context.Context) (quizmodel.PrizeRunResult, error) {
	var result quizmodel.PrizeRunResult
	now := time.Now()
	skip := map[string]bool{} // quiz gagal / dilewati tidak diproses lagi di putaran berikutnya

	for {
		ids, err := s.repo.FindPrizePendingIDs(ctx, now, prizeBatchSize+len(skip))
		if err != nil {
			return result, err
		}
		processed := 0
		for _, id := range ids {
			if skip[id] {
				continue
			}
			processed++
			awards, done, err := s.repo.DistributePrizes(ctx, id, now)
			if err != nil {
				skip[id] = true
				result.Failed++
				log.Printf("scheduler: GAGAL membagikan hadiah quiz %s: %v", id, err)
				continue
			}
			if !done { // sudah dibagikan server lain
				skip[id] = true
				continue
			}
			result.Quizzes++
			for _, a := range awards {
				result.Awards++
				result.Coins += a.CoinAmount
			}
		}
		if processed == 0 || ctx.Err() != nil {
			return result, ctx.Err()
		}
	}
}
