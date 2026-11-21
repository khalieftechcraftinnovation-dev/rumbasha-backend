package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
)

// SubscriptionRepository mengakses tabel paket akses siswa
// (student_subscriptions, subscription_plans, student_feature_overrides).
type SubscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository membuat SubscriptionRepository. Dipanggil sekali di main.go.
func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// FindActive mencari paket siswa yang aktif di tanggal tertentu, beserta info
// dan akses fitur paketnya. Mengembalikan nil kalau tidak ada paket aktif. [Native SQL]
//
// date berformat "2026-11-22" (tanggal Asia/Jakarta).
func (r *SubscriptionRepository) FindActive(ctx context.Context, studentID, date string) (*model.ActiveSubscription, error) {
	var subscription model.ActiveSubscription
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT ss.id   AS subscription_id,
			       p.id    AS plan_id,
			       p.name  AS plan_name,
			       ss.channel,
			       ss.start_date,
			       ss.end_date,
			       p.has_module,
			       p.has_daily_quiz,
			       p.has_uji_kom,
			       p.has_tryout,
			       p.has_rumath
			FROM student_subscriptions ss
			JOIN subscription_plans p ON p.id = ss.plan_id
			WHERE ss.student_id = ?
			  AND ss.cancelled_at IS NULL
			  AND ?::date BETWEEN ss.start_date AND ss.end_date
			LIMIT 1`, studentID, date).
		Scan(&subscription).Error
	if err != nil {
		return nil, err
	}

	if subscription.SubscriptionID == "" {
		return nil, nil // tidak ada paket aktif di tanggal ini
	}
	return &subscription, nil
}

// FindOverrides mengambil akses khusus per siswa yang diatur admin. [GORM]
//
//	SELECT * FROM student_feature_overrides WHERE student_id = ?
func (r *SubscriptionRepository) FindOverrides(ctx context.Context, studentID string) ([]model.FeatureOverride, error) {
	overrides := []model.FeatureOverride{}
	err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Find(&overrides).Error
	return overrides, err
}
