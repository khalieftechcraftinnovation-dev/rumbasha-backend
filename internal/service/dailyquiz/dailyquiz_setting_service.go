package dailyquiz

import (
	"context"
	"log"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/dailyquiz"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/leaderboard"
)

// GetSettings mengambil pengaturan global Daily Quiz (point per level & hadiah juara bulanan).
func (s *DailyQuizService) GetSettings(ctx context.Context) (dailyquizmodel.Settings, error) {
	return s.repo.FindSettings(ctx)
}

// UpdateSettings mengganti pengaturan global Daily Quiz. Berlaku untuk SELURUH Daily Quiz:
//   - point: untuk jawaban yang di-submit SETELAH diubah (point yang sudah tercatat tidak berubah);
//   - hadiah: dipakai saat hadiah bulan berjalan dibagikan (hari terakhir bulan 23:59 WIB).
//   - jatah gratis & biaya coin: langsung berlaku untuk Daily Quiz yang dibuka setelahnya.
func (s *DailyQuizService) UpdateSettings(ctx context.Context, req dailyquizdto.SettingsRequest, updatedBy string) (dailyquizmodel.Settings, error) {
	settings := dailyquizmodel.Settings{
		PointsEasy: *req.Points.Easy, PointsMedium: *req.Points.Medium, PointsHard: *req.Points.Hard,
		PrizeRank1: *req.PrizeCoins.Rank1, PrizeRank2: *req.PrizeCoins.Rank2, PrizeRank3: *req.PrizeCoins.Rank3,
		FreePerDay: *req.FreePerDay, CoinCost: *req.CoinCost,
	}
	if err := s.repo.UpdateSettings(ctx, settings, updatedBy); err != nil {
		return dailyquizmodel.Settings{}, err
	}
	return s.repo.FindSettings(ctx)
}

// PrizeCloseAt = saat hadiah 1 bulan (period "YYYY-MM") boleh dibagikan: hari terakhir bulan itu
// jam 23:59 WIB (sama dengan jadwal scheduler hadiah).
func PrizeCloseAt(period string) (time.Time, bool) {
	start, err := time.ParseInLocation(leaderboardmodel.PeriodLayout, period, clock.Jakarta)
	if err != nil {
		return time.Time{}, false
	}
	return start.AddDate(0, 1, 0).Add(-time.Minute), true
}

// DistributeMonthlyPrizes membagikan hadiah coin juara 1-3 per kelas untuk setiap bulan Daily Quiz
// yang sudah ditutup (hari terakhir bulan 23:59 WIB sudah lewat) dan belum dibagikan, termasuk bulan
// lalu yang terlewat (contoh server mati). Dipanggil scheduler setiap hari 23:59 WIB; di hari lain
// tidak ada bulan yang ditutup, jadi tidak ada yang dikerjakan.
//
// Setiap bulan diproses dalam transaksinya sendiri; yang gagal dicoba lagi di jadwal berikutnya.
func (s *DailyQuizService) DistributeMonthlyPrizes(ctx context.Context) (dailyquizmodel.MonthlyPrizeResult, error) {
	var result dailyquizmodel.MonthlyPrizeResult
	now := time.Now()

	periods, err := s.repo.FindPrizePendingPeriods(ctx)
	if err != nil {
		return result, err
	}
	for _, period := range periods {
		closeAt, ok := PrizeCloseAt(period)
		if !ok || now.Before(closeAt) {
			continue // bulan berjalan (belum ditutup) atau period rusak
		}
		awards, done, err := s.repo.DistributeMonthlyPrizes(ctx, period, now)
		if err != nil {
			result.Failed++
			log.Printf("scheduler: GAGAL membagikan hadiah Daily Quiz bulan %s: %v", period, err)
			continue
		}
		if !done {
			continue // sudah dibagikan server lain
		}
		result.Periods++
		for _, a := range awards {
			result.Awards++
			result.Coins += a.CoinAmount
		}
	}
	return result, nil
}

// MonthlyPrizes mengambil data pembagian hadiah 1 bulan (nil = belum dibagikan) dan penerimanya
// (opsional difilter kelas). month kosong = bulan ini (WIB).
func (s *DailyQuizService) MonthlyPrizes(ctx context.Context, q dailyquizdto.MonthlyPrizesQuery) (string, *dailyquizmodel.MonthlyPrizePeriod, []dailyquizmodel.MonthlyPrizeAward, error) {
	month := q.Month
	if month == "" {
		month = time.Now().In(clock.Jakarta).Format(leaderboardmodel.PeriodLayout)
	}
	if _, ok := PrizeCloseAt(month); !ok {
		return month, nil, nil, &dailyquizmodel.ValidationError{Fields: []dailyquizmodel.FieldError{{
			Field: "month", Message: "format tidak valid, contoh 2026-11",
		}}}
	}
	period, err := s.repo.FindPrizePeriod(ctx, month)
	if err != nil {
		return month, nil, nil, err
	}
	awards := []dailyquizmodel.MonthlyPrizeAward{}
	if period != nil {
		awards, err = s.repo.ListMonthlyAwards(ctx, month, q.GradeID)
	}
	return month, period, awards, err
}
