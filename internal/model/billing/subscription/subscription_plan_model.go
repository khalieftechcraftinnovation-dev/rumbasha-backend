package subscription

import (
	"errors"
	"strings"
	"time"
)

// Status paket akses.
const (
	PlanActive   = "active"   // dijual & bisa diberikan ke siswa
	PlanInactive = "inactive" // tidak dijual lagi; siswa yang sudah punya paket ini tetap bisa memakainya
)

// Periode paket.
const (
	PeriodMonthly  = "monthly"
	PeriodSemester = "semester"
	PeriodYearly   = "yearly"
)

// PeriodMonths = lama 1 periode dalam bulan.
var PeriodMonths = map[string]int{PeriodMonthly: 1, PeriodSemester: 6, PeriodYearly: 12}

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Features = akses fitur sebuah paket (kolom has_* di subscription_plans).
type Features struct {
	Module, DailyQuiz, UjiKom, Tryout, Rumath bool
}

// Map mengubah Features menjadi map nama fitur → terbuka (kunci sama dengan AllFeatures).
func (f Features) Map() map[string]bool {
	return map[string]bool{
		FeatureModule: f.Module, FeatureDailyQuiz: f.DailyQuiz, FeatureUjiKom: f.UjiKom,
		FeatureTryout: f.Tryout, FeatureRumath: f.Rumath,
	}
}

// Plan adalah 1 paket akses (tabel subscription_plans) + data tambahan untuk admin.
type Plan struct {
	ID                 string
	Name               string
	Description        *string
	Highlights         []string // poin fitur, 1 baris = 1 poin
	EducationLevel     *Ref     // sasaran jenjang; nil = semua jenjang
	MonthlyCoin        int
	IsFeatured         bool
	PriceMonthly       *int64 // rupiah; nil = periode tidak dijual
	PriceSemester      *int64
	PriceYearly        *int64
	StrikePriceMonthly *int64
	Features           Features
	SortOrder          int
	Status             string

	ActiveStudents int // siswa yang paketnya aktif hari ini (kolom "Pengguna")
	ModuleCount    int // modul pembelajaran yang tersedia di paket ini

	CreatedBy, UpdatedBy *Ref
	CreatedAt, UpdatedAt time.Time
}

// Price mengembalikan harga paket untuk 1 periode (nil = periode tidak dijual).
func (p Plan) Price(period string) *int64 {
	switch period {
	case PeriodMonthly:
		return p.PriceMonthly
	case PeriodSemester:
		return p.PriceSemester
	case PeriodYearly:
		return p.PriceYearly
	}
	return nil
}

// PlanSave adalah isi paket yang disimpan (tambah / ubah, ganti penuh).
type PlanSave struct {
	ID                 string
	Name               string
	Description        *string
	Highlights         *string // sudah digabung dengan "\n"
	EducationLevelID   *string
	MonthlyCoin        int
	IsFeatured         bool
	PriceMonthly       *int64
	PriceSemester      *int64
	PriceYearly        *int64
	StrikePriceMonthly *int64
	Features           Features
	SortOrder          int
	Status             string
}

// PlanFilter adalah filter list paket. Field kosong = tidak difilter.
type PlanFilter struct {
	Status           string
	EducationLevelID string // paket untuk jenjang ini + paket semua jenjang
	Search           string // nama paket
	Today            string // tanggal WIB "2026-11-26", untuk menghitung pengguna aktif
}

// PlanUsage = pemakaian 1 paket yang mencegah paket dihapus.
type PlanUsage struct {
	PlanID string
	// Siswa yang paketnya masih aktif atau belum mulai (tidak dibatalkan).
	CurrentStudents int
	// Modul yang hanya tersedia di paket ini (kalau paket dihapus, modul tidak punya paket lagi).
	OnlyPlanModules int
}

// FieldError & ValidationError: kesalahan data request (dibalas 422).
type FieldError struct {
	Field, Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi paket akses gagal: " + strings.Join(parts, "; ")
}

var (
	ErrPlanNotFound = errors.New("paket akses tidak ditemukan")
	// ErrPlanNameTaken: nama paket sudah dipakai paket lain (index unik subscription_plans_name_unique).
	ErrPlanNameTaken = errors.New("nama paket sudah dipakai")
	// ErrStudentNotFound: bukan akun siswa / tidak ada / sudah dihapus.
	ErrStudentNotFound = errors.New("siswa tidak ditemukan")
	// ErrSubscriptionNotFound: paket siswa (student_subscriptions) tidak ada.
	ErrSubscriptionNotFound = errors.New("paket siswa tidak ditemukan")
	// ErrSubscriptionCancelled: paket siswa sudah dibatalkan sebelumnya.
	ErrSubscriptionCancelled = errors.New("paket siswa sudah dibatalkan")
	// ErrAssignDuplicate: request_id beri paket yang sama sudah pernah diproses (klik 2x).
	ErrAssignDuplicate = errors.New("pemberian paket dengan request_id ini sudah diproses")
	// ErrSubscriptionOverlap: tanggal paket bertumpuk dengan paket lain siswa (constraint no_overlap).
	ErrSubscriptionOverlap = errors.New("tanggal paket bertumpuk dengan paket lain")
)
