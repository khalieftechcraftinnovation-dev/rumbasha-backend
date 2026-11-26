package dailyquiz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/dailyquiz"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/coin"
)

// settingsColumns = kolom daily_quiz_settings. prize_rank_N diberi alias tanpa "_" sebelum angka
// supaya cocok dengan nama field GORM (PrizeRank1 → prize_rank1).
const settingsColumns = `
	s.points_easy, s.points_medium, s.points_hard,
	s.prize_rank_1 AS prize_rank1, s.prize_rank_2 AS prize_rank2, s.prize_rank_3 AS prize_rank3,
	s.free_per_day, s.coin_cost,
	s.updated_by, u.real_name AS updated_by_name, s.updated_at`

// FindSettings mengambil pengaturan global Daily Quiz (1 baris). [Native SQL, primary key]
func (r *DailyQuizRepository) FindSettings(ctx context.Context) (dailyquizmodel.Settings, error) {
	var rows []dailyquizmodel.Settings
	err := r.db.WithContext(ctx).
		Raw(`SELECT ` + settingsColumns + `
			FROM daily_quiz_settings s
			LEFT JOIN users u ON u.id = s.updated_by
			WHERE s.id = 1`).
		Scan(&rows).Error
	if err != nil {
		return dailyquizmodel.Settings{}, err
	}
	if len(rows) == 0 {
		return dailyquizmodel.Settings{}, errors.New("baris daily_quiz_settings tidak ada (sudah menjalankan migrate-up?)")
	}
	return rows[0], nil
}

// UpdateSettings mengganti pengaturan global Daily Quiz. [Native SQL, primary key]
func (r *DailyQuizRepository) UpdateSettings(ctx context.Context, s dailyquizmodel.Settings, updatedBy string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE daily_quiz_settings SET
			points_easy = ?, points_medium = ?, points_hard = ?,
			prize_rank_1 = ?, prize_rank_2 = ?, prize_rank_3 = ?,
			free_per_day = ?, coin_cost = ?,
			updated_by = ?, updated_at = NOW()
		WHERE id = 1`,
		s.PointsEasy, s.PointsMedium, s.PointsHard, s.PrizeRank1, s.PrizeRank2, s.PrizeRank3, s.FreePerDay, s.CoinCost, updatedBy).Error
}

// FindPrizePendingPeriods mengambil bulan Daily Quiz yang punya peserta leaderboard tetapi hadiahnya
// belum dibagikan, urut bulan. Service yang memilih bulan mana yang sudah boleh dibagikan.
// [Native SQL, index leaderboard_entries_daily_idx]
func (r *DailyQuizRepository) FindPrizePendingPeriods(ctx context.Context) ([]string, error) {
	periods := []string{}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT DISTINCT e.period
			FROM leaderboard_entries e
			WHERE e.board = 'daily_quiz'
			  AND NOT EXISTS (SELECT 1 FROM daily_quiz_prize_periods p WHERE p.period = e.period)
			ORDER BY e.period`).
		Scan(&periods).Error
	return periods, err
}

// monthlyWinner adalah 1 siswa peringkat 1-3 di kelasnya pada 1 bulan.
type monthlyWinner struct {
	StudentID       string
	GradeID         string
	Points          int
	DurationSeconds int
	Rank            int
}

// DistributeMonthlyPrizes membagikan hadiah coin juara 1-3 PER KELAS untuk 1 bulan Daily Quiz,
// dalam 1 transaksi. [Native SQL]
//
//  1. INSERT daily_quiz_prize_periods (bulan + snapshot hadiah dari pengaturan saat ini).
//     Sudah ada (dibagikan server lain) → done = false, tidak ada yang berubah. Primary key period
//     sekaligus menjadi kunci: server lain yang bersamaan menunggu lalu mendapat konflik.
//  2. Juara per kelas dari leaderboard_entries bulan itu, urutan sama dengan API leaderboard yang
//     difilter kelas (point tertinggi, lalu waktu tersingkat; sama persis → peringkat & hadiah sama).
//     Kelas = snapshot kelas di leaderboard (kelas siswa saat point masuk).
//  3. Per juara yang hadiahnya > 0: tambah coin (coin_transactions type reward, reference
//     daily_quiz_prize → id baris penerima) + catat daily_quiz_prize_awards.
//
// Gagal di tengah jalan → semua dibatalkan dan dicoba lagi di jadwal berikutnya.
func (r *DailyQuizRepository) DistributeMonthlyPrizes(ctx context.Context, period string, now time.Time) (awards []dailyquizmodel.MonthlyPrizeAward, done bool, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, false, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var prizes []struct{ PrizeRank1, PrizeRank2, PrizeRank3 int }
	err = tx.Raw(`
		INSERT INTO daily_quiz_prize_periods (period, prize_rank_1, prize_rank_2, prize_rank_3, distributed_at)
		SELECT ?, prize_rank_1, prize_rank_2, prize_rank_3, ? FROM daily_quiz_settings WHERE id = 1
		ON CONFLICT (period) DO NOTHING
		RETURNING prize_rank_1 AS prize_rank1, prize_rank_2 AS prize_rank2, prize_rank_3 AS prize_rank3`, period, now).
		Scan(&prizes).Error
	if err != nil {
		return nil, false, err
	}
	if len(prizes) == 0 {
		return nil, false, nil // sudah dibagikan
	}
	settings := dailyquizmodel.Settings{PrizeRank1: prizes[0].PrizeRank1, PrizeRank2: prizes[0].PrizeRank2, PrizeRank3: prizes[0].PrizeRank3}

	awards = []dailyquizmodel.MonthlyPrizeAward{}
	if settings.PrizeRank1+settings.PrizeRank2+settings.PrizeRank3 > 0 {
		var winners []monthlyWinner
		err = tx.Raw(`
			SELECT student_id, grade_id, points, duration_seconds, rank
			FROM (
				SELECT e.student_id, e.grade_id, e.points, e.duration_seconds,
				       RANK() OVER (PARTITION BY e.grade_id ORDER BY e.points DESC, e.duration_seconds ASC)::int AS rank
				FROM leaderboard_entries e
				JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
				WHERE e.board = 'daily_quiz' AND e.period = ? AND e.grade_id IS NOT NULL
			) ranked
			WHERE rank <= 3
			ORDER BY grade_id, rank, student_id`, period).
			Scan(&winners).Error
		if err != nil {
			return nil, false, err
		}

		for _, w := range winners {
			coins := settings.PrizeForRank(w.Rank)
			if coins <= 0 {
				continue
			}
			awardID := uuid.NewString()
			transactionID, ok, err := coinrepo.AddReward(tx, coinrepo.Reward{
				StudentID: w.StudentID, Amount: coins,
				ReferenceType: "daily_quiz_prize", ReferenceID: awardID,
				IdempotencyKey: "daily_quiz_prize:" + period + ":" + w.StudentID,
				Description:    fmt.Sprintf("Hadiah juara %d Daily Quiz bulan %s", w.Rank, period),
				At:             now,
			})
			if err != nil {
				return nil, false, fmt.Errorf("hadiah juara %d siswa %s: %w", w.Rank, w.StudentID, err)
			}
			if !ok {
				continue // akun sudah bukan siswa
			}
			err = tx.Exec(`
				INSERT INTO daily_quiz_prize_awards
					(id, period, grade_id, student_id, rank, points, duration_seconds, coin_amount, coin_transaction_id, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				awardID, period, w.GradeID, w.StudentID, w.Rank, w.Points, w.DurationSeconds, coins, transactionID, now).Error
			if err != nil {
				return nil, false, err
			}
			awards = append(awards, dailyquizmodel.MonthlyPrizeAward{
				Period: period, GradeID: w.GradeID, Rank: w.Rank, StudentID: w.StudentID,
				Points: w.Points, DurationSeconds: w.DurationSeconds, CoinAmount: coins, CreatedAt: now,
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, false, err
	}
	return awards, true, nil
}

// FindPrizePeriod mengambil data pembagian hadiah 1 bulan. nil = belum dibagikan. [Native SQL, primary key]
func (r *DailyQuizRepository) FindPrizePeriod(ctx context.Context, period string) (*dailyquizmodel.MonthlyPrizePeriod, error) {
	var rows []dailyquizmodel.MonthlyPrizePeriod
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT period, prize_rank_1 AS prize_rank1, prize_rank_2 AS prize_rank2, prize_rank_3 AS prize_rank3, distributed_at
			FROM daily_quiz_prize_periods WHERE period = ?`, period).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// ListMonthlyAwards mengambil penerima hadiah 1 bulan, urut kelas lalu peringkat. gradeID "" = semua
// kelas. [Native SQL, index daily_quiz_prize_awards_period_idx]
func (r *DailyQuizRepository) ListMonthlyAwards(ctx context.Context, period, gradeID string) ([]dailyquizmodel.MonthlyPrizeAward, error) {
	awards := []dailyquizmodel.MonthlyPrizeAward{}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT a.period, a.grade_id, g.name AS grade_name, a.rank, a.student_id,
			       u.real_name AS student_name, s.student_code,
			       a.points, a.duration_seconds, a.coin_amount, a.created_at
			FROM daily_quiz_prize_awards a
			JOIN      master_data_items g ON g.id = a.grade_id
			LEFT JOIN users             u ON u.id = a.student_id
			LEFT JOIN students          s ON s.user_id = a.student_id
			WHERE a.period = ? AND (? = '' OR a.grade_id::text = ?)
			ORDER BY g.sort_order, a.rank, u.real_name`, period, gradeID, gradeID).
		Scan(&awards).Error
	return awards, err
}
