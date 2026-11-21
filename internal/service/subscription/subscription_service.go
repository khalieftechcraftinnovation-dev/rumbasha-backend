// Package subscription (service) berisi logika paket akses siswa.
//
// Di package lain, import dengan alias subscriptionservice:
//
//	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subscription"
package subscription

import (
	"context"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
	subscriptionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subscription"
)

// SubscriptionService berisi logika paket akses siswa.
type SubscriptionService struct {
	repo *subscriptionrepo.SubscriptionRepository
}

// NewSubscriptionService membuat SubscriptionService. Dipanggil sekali di main.go.
func NewSubscriptionService(repo *subscriptionrepo.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

// StudentAccess adalah akses siswa pada hari ini.
type StudentAccess struct {
	Subscription *subscriptionmodel.ActiveSubscription // nil = tidak ada paket aktif hari ini
	Features     map[string]bool                       // contoh: {"module": true, "rumath": false}
}

// GetTodayAccess mengecek paket siswa yang aktif HARI INI (tanggal WIB) dan
// menghitung fitur apa saja yang terbuka.
//
// Pengecekan selalu memakai tanggal hari ini, jadi paket online yang habis
// tanggal 15 otomatis tidak terhitung mulai tanggal 16, tanpa perlu login ulang
// dan tanpa job pengubah status.
func (s *SubscriptionService) GetTodayAccess(ctx context.Context, studentID string) (StudentAccess, error) {
	subscription, err := s.repo.FindActive(ctx, studentID, clock.TodayInJakarta(time.Now()))
	if err != nil {
		return StudentAccess{}, err
	}

	overrides, err := s.repo.FindOverrides(ctx, studentID)
	if err != nil {
		return StudentAccess{}, err
	}

	return StudentAccess{
		Subscription: subscription,
		Features:     resolveFeatures(subscription, overrides),
	}, nil
}

// resolveFeatures menghitung akses akhir:
//  1. mulai dari fitur paket aktif (semua false kalau tidak ada paket aktif),
//  2. lalu ditimpa akses khusus dari admin (student_feature_overrides).
func resolveFeatures(subscription *subscriptionmodel.ActiveSubscription, overrides []subscriptionmodel.FeatureOverride) map[string]bool {
	features := map[string]bool{}
	for _, feature := range subscriptionmodel.AllFeatures {
		features[feature] = false
	}

	if subscription != nil {
		features[subscriptionmodel.FeatureModule] = subscription.HasModule
		features[subscriptionmodel.FeatureDailyQuiz] = subscription.HasDailyQuiz
		features[subscriptionmodel.FeatureUjiKom] = subscription.HasUjiKom
		features[subscriptionmodel.FeatureTryout] = subscription.HasTryout
		features[subscriptionmodel.FeatureRumath] = subscription.HasRumath
	}

	for _, override := range overrides {
		features[override.Feature] = override.IsEnabled
	}
	return features
}
