package dailyquiz

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
)

// SettingsRequest adalah body PUT /api/v1/daily-quiz-settings (ganti penuh, semua wajib).
//
//	{
//	  "points": {"easy": 10, "medium": 15, "hard": 20},
//	  "prize_coins": {"rank_1": 300, "rank_2": 200, "rank_3": 100},
//	  "free_per_day": 2, "coin_cost": 10
//	}
type SettingsRequest struct {
	Points     SettingsPointsRequest `json:"points" binding:"required"`
	PrizeCoins SettingsPrizesRequest `json:"prize_coins" binding:"required"`
	// Jatah Daily Quiz gratis per siswa per hari (WIB), 0-100. 0 = tidak ada jatah gratis.
	FreePerDay *int `json:"free_per_day" binding:"required,min=0,max=100"`
	// Biaya coin per Daily Quiz setelah jatah gratis habis, 0-1.000.000. 0 = selalu gratis.
	CoinCost *int `json:"coin_cost" binding:"required,min=0,max=1000000"`
}

// SettingsPointsRequest = point 1 soal BENAR per level soal (0-1000).
type SettingsPointsRequest struct {
	Easy   *int `json:"easy" binding:"required,min=0,max=1000"`
	Medium *int `json:"medium" binding:"required,min=0,max=1000"`
	Hard   *int `json:"hard" binding:"required,min=0,max=1000"`
}

// SettingsPrizesRequest = hadiah coin juara 1-3 bulanan per kelas (0 = tanpa hadiah).
type SettingsPrizesRequest struct {
	Rank1 *int `json:"rank_1" binding:"required,min=0,max=1000000"`
	Rank2 *int `json:"rank_2" binding:"required,min=0,max=1000000"`
	Rank3 *int `json:"rank_3" binding:"required,min=0,max=1000000"`
}

// SettingsResponse adalah data pengaturan global Daily Quiz.
type SettingsResponse struct {
	Points     PointsResponse   `json:"points"`
	PrizeCoins PrizesResponse   `json:"prize_coins"`
	FreePerDay int              `json:"free_per_day"` // Daily Quiz gratis per siswa per hari (WIB)
	CoinCost   int              `json:"coin_cost"`    // coin per Daily Quiz setelah jatah gratis habis
	UpdatedBy  *UserRefResponse `json:"updated_by"`   // null = belum pernah diubah
	UpdatedAt  time.Time        `json:"updated_at"`   // WIB
}

// PointsResponse = point 1 soal benar per level.
type PointsResponse struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

// PrizesResponse = hadiah coin juara 1-3.
type PrizesResponse struct {
	Rank1 int `json:"rank_1"`
	Rank2 int `json:"rank_2"`
	Rank3 int `json:"rank_3"`
}

// UserRefResponse = id + nama user (nama bisa null).
type UserRefResponse struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

// ToSettingsResponse mengubah pengaturan menjadi response.
func ToSettingsResponse(s dailyquizmodel.Settings) SettingsResponse {
	r := SettingsResponse{
		Points:     PointsResponse{Easy: s.PointsEasy, Medium: s.PointsMedium, Hard: s.PointsHard},
		PrizeCoins: PrizesResponse{Rank1: s.PrizeRank1, Rank2: s.PrizeRank2, Rank3: s.PrizeRank3},
		FreePerDay: s.FreePerDay,
		CoinCost:   s.CoinCost,
		UpdatedAt:  s.UpdatedAt.In(clock.Jakarta),
	}
	if s.UpdatedBy != nil {
		r.UpdatedBy = &UserRefResponse{ID: *s.UpdatedBy, Name: s.UpdatedByName}
	}
	return r
}

// MonthlyPrizesQuery adalah query string GET /api/v1/daily-quiz-prizes.
//
//	?month=2026-11&grade_id=...
type MonthlyPrizesQuery struct {
	Month   string `form:"month" binding:"omitempty,max=7"` // YYYY-MM (WIB), default bulan ini
	GradeID string `form:"grade_id" binding:"omitempty,uuid"`
}

// MonthlyPrizesResponse adalah hasil pembagian hadiah juara Daily Quiz 1 bulan.
type MonthlyPrizesResponse struct {
	Month string `json:"month"`
	// Kapan hadiah dibagikan (WIB); null = belum (dibagikan hari terakhir bulan 23:59 WIB).
	DistributedAt *time.Time `json:"distributed_at"`
	// Hadiah yang dipakai saat dibagikan; null kalau belum dibagikan (lihat pengaturan).
	PrizeCoins *PrizesResponse        `json:"prize_coins"`
	Awards     []MonthlyAwardResponse `json:"awards"` // urut kelas, lalu peringkat
}

// MonthlyAwardResponse = 1 penerima hadiah juara bulanan.
type MonthlyAwardResponse struct {
	Grade           RefResponse    `json:"grade"`
	Rank            int            `json:"rank"`
	Student         StudentRefResp `json:"student"`
	Points          int            `json:"points"`
	DurationSeconds int            `json:"duration_seconds"`
	CoinAmount      int            `json:"coin_amount"`
	AwardedAt       time.Time      `json:"awarded_at"` // WIB
}

// StudentRefResp = data siswa penerima hadiah.
type StudentRefResp struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	StudentCode *string `json:"student_code"`
}

// ToMonthlyPrizesResponse mengubah hasil pembagian hadiah 1 bulan menjadi response.
func ToMonthlyPrizesResponse(month string, p *dailyquizmodel.MonthlyPrizePeriod, awards []dailyquizmodel.MonthlyPrizeAward) MonthlyPrizesResponse {
	r := MonthlyPrizesResponse{Month: month, Awards: make([]MonthlyAwardResponse, 0, len(awards))}
	if p != nil {
		at := p.DistributedAt.In(clock.Jakarta)
		r.DistributedAt = &at
		r.PrizeCoins = &PrizesResponse{Rank1: p.PrizeRank1, Rank2: p.PrizeRank2, Rank3: p.PrizeRank3}
	}
	for _, a := range awards {
		r.Awards = append(r.Awards, MonthlyAwardResponse{
			Grade:           RefResponse{ID: a.GradeID, Name: a.GradeName},
			Rank:            a.Rank,
			Student:         StudentRefResp{ID: a.StudentID, Name: a.StudentName, StudentCode: a.StudentCode},
			Points:          a.Points,
			DurationSeconds: a.DurationSeconds,
			CoinAmount:      a.CoinAmount,
			AwardedAt:       a.CreatedAt.In(clock.Jakarta),
		})
	}
	return r
}
