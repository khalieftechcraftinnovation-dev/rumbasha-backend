// Package subscription (model) berisi struct data untuk fitur paket akses siswa.
//
// Di package lain, import dengan alias subscriptionmodel:
//
//	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
package subscription

import "time"

// Daftar fitur yang aksesnya diatur paket.
// Nilainya sama dengan isi kolom student_feature_overrides.feature.
const (
	FeatureModule    = "module"
	FeatureDailyQuiz = "daily_quiz"
	FeatureUjiKom    = "uji_kom"
	FeatureTryout    = "tryout"
	FeatureRumath    = "rumath"
)

// AllFeatures berisi semua fitur, urut sesuai tampilan.
var AllFeatures = []string{FeatureModule, FeatureDailyQuiz, FeatureUjiKom, FeatureTryout, FeatureRumath}

// ActiveSubscription adalah paket siswa yang aktif di tanggal tertentu,
// lengkap dengan info paket dan akses fiturnya (hasil JOIN, bukan satu tabel).
type ActiveSubscription struct {
	SubscriptionID string
	PlanID         string
	PlanName       string
	Channel        string // offline atau online
	StartDate      time.Time
	EndDate        time.Time

	HasModule    bool
	HasDailyQuiz bool
	HasUjiKom    bool
	HasTryout    bool
	HasRumath    bool
}

// FeatureOverride mewakili satu baris di tabel "student_feature_overrides".
type FeatureOverride struct {
	StudentID string
	Feature   string
	IsEnabled bool
}

// TableName memberi tahu GORM nama tabel untuk model ini.
func (FeatureOverride) TableName() string {
	return "student_feature_overrides"
}
