package quiz

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/coin"
)

// FindPrizePendingIDs mengambil quiz yang hadiahnya siap dibagikan: sudah publish, belum dihapus,
// event sudah selesai (end_at <= now), dan belum dibagikan. Urut end_at (yang paling lama selesai dulu).
// [Native SQL, partial index quizzes_prize_pending_idx]
func (r *QuizRepository) FindPrizePendingIDs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	ids := []string{}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id FROM quizzes
			WHERE prizes_distributed_at IS NULL AND deleted_at IS NULL AND status = 'published'
			  AND end_at <= ?
			ORDER BY end_at, id
			LIMIT ?`, now, limit).
		Scan(&ids).Error
	return ids, err
}

// prizeWinner adalah 1 siswa peringkat 1-3 di leaderboard quiz.
type prizeWinner struct {
	StudentID       string
	Points          int
	DurationSeconds int
	Rank            int
}

// DistributePrizes membagikan hadiah coin juara 1-3 untuk 1 quiz, dalam 1 transaksi. [Native SQL]
//
//  1. Kunci baris quiz (FOR UPDATE) lalu cek ulang: masih ada, publish, event selesai, belum
//     dibagikan. Kalau tidak memenuhi (contoh sudah dibagikan server lain) → done = false, tidak ada
//     yang berubah.
//  2. Ambil juara dari leaderboard_entries quiz ini, peringkat sama persis dengan API leaderboard
//     (point tertinggi, lalu waktu tersingkat; sama persis → peringkat sama, jadi hadiahnya sama).
//  3. Per juara yang hadiahnya > 0: tambah coin_balance siswa, catat coin_transactions (type reward,
//     idempotency_key quiz_prize:<quiz>:<siswa> → tidak mungkin dobel), catat quiz_prize_awards.
//  4. Tandai quiz sudah dibagikan (prizes_distributed_at), termasuk quiz tanpa hadiah / tanpa
//     peserta, supaya tidak diproses lagi.
//
// Gagal di tengah jalan → semua dibatalkan (ROLLBACK) dan dicoba lagi di jadwal berikutnya.
func (r *QuizRepository) DistributePrizes(ctx context.Context, quizID string, now time.Time) (awards []quizmodel.PrizeAward, done bool, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, false, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var quizzes []struct {
		Name                               string
		PrizeRank1, PrizeRank2, PrizeRank3 int
	}
	err = tx.Raw(`
		SELECT name, prize_rank_1 AS prize_rank1, prize_rank_2 AS prize_rank2, prize_rank_3 AS prize_rank3
		FROM quizzes
		WHERE id = ? AND prizes_distributed_at IS NULL AND deleted_at IS NULL
		  AND status = 'published' AND end_at <= ?
		FOR UPDATE`, quizID, now).
		Scan(&quizzes).Error
	if err != nil {
		return nil, false, err
	}
	if len(quizzes) == 0 {
		return nil, false, nil // sudah dibagikan / berubah sejak dicari: lewati
	}
	q := quizzes[0]
	prizes := quizmodel.Prizes{Rank1: q.PrizeRank1, Rank2: q.PrizeRank2, Rank3: q.PrizeRank3}

	awards = []quizmodel.PrizeAward{}
	if prizes.Rank1+prizes.Rank2+prizes.Rank3 > 0 {
		var winners []prizeWinner
		err = tx.Raw(`
			SELECT student_id, points, duration_seconds, rank
			FROM (
				SELECT e.student_id, e.points, e.duration_seconds,
				       RANK() OVER (ORDER BY e.points DESC, e.duration_seconds ASC)::int AS rank
				FROM leaderboard_entries e
				JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
				WHERE e.quiz_id = ?
			) ranked
			WHERE rank <= 3
			ORDER BY rank, student_id`, quizID).
			Scan(&winners).Error
		if err != nil {
			return nil, false, err
		}

		for _, w := range winners {
			coins := prizes.ForRank(w.Rank)
			if coins <= 0 {
				continue
			}
			award, ok, err := giveReward(tx, quizID, q.Name, w, coins, now)
			if err != nil {
				return nil, false, fmt.Errorf("hadiah juara %d siswa %s: %w", w.Rank, w.StudentID, err)
			}
			if ok {
				awards = append(awards, award)
			}
		}
	}

	if err := tx.Exec(`UPDATE quizzes SET prizes_distributed_at = ? WHERE id = ?`, now, quizID).Error; err != nil {
		return nil, false, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, false, err
	}
	return awards, true, nil
}

// giveReward menambah coin 1 juara + mencatat riwayatnya (di dalam transaksi tx). ok = false kalau
// akun sudah bukan siswa (tidak ada baris students): dilewati.
func giveReward(tx *gorm.DB, quizID, quizName string, w prizeWinner, coins int, now time.Time) (quizmodel.PrizeAward, bool, error) {
	transactionID, ok, err := coinrepo.AddReward(tx, coinrepo.Reward{
		StudentID: w.StudentID, Amount: coins,
		ReferenceType: "quiz_prize", ReferenceID: quizID,
		IdempotencyKey: "quiz_prize:" + quizID + ":" + w.StudentID,
		Description:    fmt.Sprintf("Hadiah juara %d: %s", w.Rank, quizName),
		At:             now,
	})
	if err != nil || !ok {
		return quizmodel.PrizeAward{}, false, err
	}

	err = tx.Exec(`
		INSERT INTO quiz_prize_awards
			(quiz_id, student_id, rank, points, duration_seconds, coin_amount, coin_transaction_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		quizID, w.StudentID, w.Rank, w.Points, w.DurationSeconds, coins, transactionID, now).Error
	if err != nil {
		return quizmodel.PrizeAward{}, false, err
	}
	return quizmodel.PrizeAward{
		Rank: w.Rank, StudentID: w.StudentID, Points: w.Points, DurationSeconds: w.DurationSeconds,
		CoinAmount: coins, CreatedAt: now,
	}, true, nil
}
