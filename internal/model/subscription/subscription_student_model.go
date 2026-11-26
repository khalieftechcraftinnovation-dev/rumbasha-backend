package subscription

import "time"

// Status paket siswa (dihitung dari tanggal, tidak disimpan).
const (
	SubscriptionActive    = "active"    // hari ini di antara start_date dan end_date
	SubscriptionUpcoming  = "upcoming"  // belum mulai (perpanjangan / dijadwalkan)
	SubscriptionExpired   = "expired"   // sudah lewat
	SubscriptionCancelled = "cancelled" // dibatalkan admin
)

// StudentSubscription adalah 1 periode paket siswa (tabel student_subscriptions) + nama paket.
type StudentSubscription struct {
	ID          string
	StudentID   string
	Plan        Ref
	Channel     string // offline (dicatat admin) / online (beli di aplikasi)
	Period      string
	PricePaid   int64
	StartDate   time.Time
	EndDate     time.Time
	CancelledAt *time.Time
	CancelledBy *Ref
	Notes       *string
	CreatedBy   *Ref
	CreatedAt   time.Time
}

// Status menghitung status paket pada tanggal today ("2026-11-26", WIB).
func (s StudentSubscription) Status(today string) string {
	switch {
	case s.CancelledAt != nil:
		return SubscriptionCancelled
	case s.StartDate.Format("2006-01-02") > today:
		return SubscriptionUpcoming
	case s.EndDate.Format("2006-01-02") < today:
		return SubscriptionExpired
	}
	return SubscriptionActive
}

// AccessStudent adalah siswa yang dicek sebelum diberi paket / akses khusus.
type AccessStudent struct {
	ID               string
	Name             string
	StudentCode      string
	EducationLevelID *string
	CoinBalance      int
	// Hari terakhir paket siswa yang belum dibatalkan dan belum lewat (nil = tidak ada). Dipakai
	// untuk tanggal mulai otomatis: perpanjangan mulai sehari setelahnya.
	LastEndDate *time.Time
}

// NewSubscription adalah 1 paket siswa yang siap disimpan.
type NewSubscription struct {
	StudentID      string
	StartDate      time.Time
	EndDate        time.Time
	IdempotencyKey *string
}

// AssignInput = isi pemberian paket yang sama untuk semua siswa di 1 request.
type AssignInput struct {
	Plan      Plan
	Period    string
	PricePaid int64
	Notes     *string
	AdminID   string
	Now       time.Time
}

// AssignResult adalah hasil beri paket ke 1 siswa.
type AssignResult struct {
	Subscription StudentSubscription
	StudentName  string
	StudentCode  string
	CoinAdded    int // bonus coin bulan pertama (monthly_coin paket); 0 = tidak ada
	CoinBalance  int // saldo coin setelahnya
}

// Access adalah akses 1 siswa pada hari ini: paket aktif + akses khusus + hasil akhirnya.
type Access struct {
	Subscription *ActiveSubscription // nil = tidak ada paket aktif hari ini
	Overrides    []FeatureOverride
	Features     map[string]bool // akses akhir (paket lalu ditimpa akses khusus)
}

// ResolveFeatures menghitung akses akhir siswa:
//  1. mulai dari fitur paket aktif (semua false kalau tidak ada paket aktif),
//  2. lalu ditimpa akses khusus dari admin (student_feature_overrides).
func ResolveFeatures(subscription *ActiveSubscription, overrides []FeatureOverride) map[string]bool {
	features := map[string]bool{}
	for _, feature := range AllFeatures {
		features[feature] = false
	}
	if subscription != nil {
		features[FeatureModule] = subscription.HasModule
		features[FeatureDailyQuiz] = subscription.HasDailyQuiz
		features[FeatureUjiKom] = subscription.HasUjiKom
		features[FeatureTryout] = subscription.HasTryout
		features[FeatureRumath] = subscription.HasRumath
	}
	for _, override := range overrides {
		features[override.Feature] = override.IsEnabled
	}
	return features
}
