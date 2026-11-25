package dailyquiz

import (
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// Settings adalah pengaturan global Daily Quiz (tabel daily_quiz_settings, selalu 1 baris).
// Berlaku untuk SELURUH Daily Quiz; diubah admin / super admin.
type Settings struct {
	PointsEasy, PointsMedium, PointsHard int // point 1 soal BENAR per level soal
	PrizeRank1, PrizeRank2, PrizeRank3   int // hadiah coin juara 1-3 bulanan per kelas (0 = tanpa hadiah)
	FreePerDay                           int // Daily Quiz gratis per siswa per hari (WIB)
	CoinCost                             int // coin per Daily Quiz setelah jatah gratis habis (0 = selalu gratis)
	UpdatedBy                            *string
	UpdatedByName                        *string
	UpdatedAt                            time.Time
}

// PointTable mengubah point per level menjadi map level → point (dipakai scoring.Grade).
func (s Settings) PointTable() map[string]int {
	return map[string]int{
		questionmodel.DifficultyEasy:   s.PointsEasy,
		questionmodel.DifficultyMedium: s.PointsMedium,
		questionmodel.DifficultyHard:   s.PointsHard,
	}
}

// PrizeForRank = hadiah coin untuk peringkat rank (1-3); peringkat lain 0.
func (s Settings) PrizeForRank(rank int) int {
	switch rank {
	case 1:
		return s.PrizeRank1
	case 2:
		return s.PrizeRank2
	case 3:
		return s.PrizeRank3
	}
	return 0
}

// MonthlyPrizePeriod adalah 1 bulan yang hadiahnya sudah dibagikan (snapshot besar hadiah).
type MonthlyPrizePeriod struct {
	Period                             string
	PrizeRank1, PrizeRank2, PrizeRank3 int
	DistributedAt                      time.Time
}

// MonthlyPrizeAward adalah 1 siswa penerima hadiah juara bulanan Daily Quiz (per kelas).
type MonthlyPrizeAward struct {
	Period          string
	GradeID         string
	GradeName       string
	Rank            int
	StudentID       string
	StudentName     *string
	StudentCode     *string
	Points          int
	DurationSeconds int
	CoinAmount      int
	CreatedAt       time.Time
}

// MonthlyPrizeResult adalah hasil 1 kali scheduler membagikan hadiah bulanan.
type MonthlyPrizeResult struct {
	Periods int // bulan yang selesai diproses
	Awards  int // jumlah siswa yang menerima coin
	Coins   int // total coin yang dibagikan
	Failed  int // bulan yang gagal (dicoba lagi di jadwal berikutnya)
}
