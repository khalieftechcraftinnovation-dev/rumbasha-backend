// Package subscription (dto) berisi bentuk request dan response JSON untuk paket akses: CRUD paket
// (admin), daftar paket untuk siswa, dan pengaturan akses siswa (beri paket, batal, akses khusus).
//
// Di package lain, import dengan alias subscriptiondto:
//
//	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subscription"
package subscription

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// DateLayout = format tanggal "2026-11-26".
const DateLayout = "2006-01-02"

// ---------------------------------------------------------------------------
// Paket akses (CRUD admin)
// ---------------------------------------------------------------------------

// PlanRequest adalah body POST /api/v1/subscription-plans dan PUT /api/v1/subscription-plans/:id
// (ganti penuh). Contoh:
//
//	{
//	  "name": "Pro", "description": "Ringkasan materi + latihan soal",
//	  "highlights": ["Modul ringkasan + latihan soal", "Daily Quiz & Uji Kom"],
//	  "education_level_id": null, "monthly_coin": 50, "is_featured": true,
//	  "price_monthly": 199000, "price_semester": 1080000, "price_yearly": 2040000, "strike_price_monthly": 249000,
//	  "features": {"module": true, "daily_quiz": true, "uji_kom": true, "tryout": false, "rumath": false},
//	  "sort_order": 2, "status": "active"
//	}
type PlanRequest struct {
	Name        string   `json:"name" binding:"required,max=100"`
	Description *string  `json:"description" binding:"omitempty,max=1000"`
	Highlights  []string `json:"highlights" binding:"omitempty,max=20,dive,max=150"` // poin fitur, 1 item = 1 poin
	// Sasaran jenjang; null = semua jenjang. Siswa hanya melihat paket jenjangnya + paket semua jenjang.
	EducationLevelID *string `json:"education_level_id" binding:"omitempty,uuid"`
	MonthlyCoin      *int    `json:"monthly_coin" binding:"omitempty,min=0,max=100000"` // bonus coin tiap bulan, default 0
	IsFeatured       bool    `json:"is_featured"`                                       // label "Unggulan" / "Paling banyak dipilih"
	// Harga per periode (rupiah). null = periode itu tidak dijual; minimal 1 periode diisi.
	PriceMonthly  *int64 `json:"price_monthly" binding:"omitempty,min=0,max=1000000000"`
	PriceSemester *int64 `json:"price_semester" binding:"omitempty,min=0,max=1000000000"`
	PriceYearly   *int64 `json:"price_yearly" binding:"omitempty,min=0,max=1000000000"`
	// Harga coret untuk harga bulanan (opsional); harus lebih mahal dari price_monthly.
	StrikePriceMonthly *int64           `json:"strike_price_monthly" binding:"omitempty,min=0,max=1000000000"`
	Features           *FeaturesRequest `json:"features" binding:"required"`
	SortOrder          *int             `json:"sort_order" binding:"omitempty,min=0,max=10000"`   // default 0
	Status             string           `json:"status" binding:"omitempty,oneof=active inactive"` // default active
}

// FeaturesRequest = akses fitur paket. Field yang tidak dikirim = false.
type FeaturesRequest struct {
	Module    bool `json:"module"`
	DailyQuiz bool `json:"daily_quiz"`
	UjiKom    bool `json:"uji_kom"`
	Tryout    bool `json:"tryout"`
	Rumath    bool `json:"rumath"`
}

// ListPlansQuery adalah query string GET /api/v1/subscription-plans (admin). Siswa: diabaikan,
// selalu paket aktif untuk jenjangnya.
type ListPlansQuery struct {
	Status           string `form:"status" binding:"omitempty,oneof=active inactive"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"` // paket jenjang ini + semua jenjang
	Search           string `form:"search" binding:"omitempty,max=100"`
}

// DeletePlansRequest adalah body DELETE /api/v1/subscription-plans: hapus 1-100 paket (soft delete).
type DeletePlansRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeletePlansResponse berisi ID paket yang dihapus.
type DeletePlansResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *subscriptionmodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// PricesResponse = harga paket per periode (rupiah). null = periode tidak dijual.
type PricesResponse struct {
	Monthly       *int64 `json:"monthly"`
	Semester      *int64 `json:"semester"`
	Yearly        *int64 `json:"yearly"`
	StrikeMonthly *int64 `json:"strike_monthly"` // harga coret bulanan
}

// StudentPlanResponse adalah paket untuk siswa (halaman Paket & Coin): tanpa data admin.
type StudentPlanResponse struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    *string         `json:"description"`
	Highlights     []string        `json:"highlights"`
	EducationLevel *RefResponse    `json:"education_level"` // null = semua jenjang
	MonthlyCoin    int             `json:"monthly_coin"`
	IsFeatured     bool            `json:"is_featured"`
	Prices         PricesResponse  `json:"prices"`
	Features       map[string]bool `json:"features"`       // {"module": true, "daily_quiz": true, ...}
	FeatureLabels  []string        `json:"feature_labels"` // fitur terbuka, contoh ["Modul", "Daily Quiz"]
}

// PlanResponse adalah 1 paket untuk admin (list & detail).
type PlanResponse struct {
	StudentPlanResponse
	SortOrder      int          `json:"sort_order"`
	Status         string       `json:"status"`
	ActiveStudents int          `json:"active_students"` // kolom "Pengguna": siswa yang paketnya aktif hari ini
	ModuleCount    int          `json:"module_count"`    // modul pembelajaran yang tersedia di paket ini
	CreatedBy      *RefResponse `json:"created_by"`      // null = data awal
	UpdatedBy      *RefResponse `json:"updated_by"`      // null = belum pernah diubah
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// ToStudentPlanResponse mengubah 1 paket menjadi response siswa.
func ToStudentPlanResponse(p subscriptionmodel.Plan) StudentPlanResponse {
	features := p.Features.Map()
	return StudentPlanResponse{
		ID: p.ID, Name: p.Name, Description: p.Description, Highlights: p.Highlights,
		EducationLevel: toRef(p.EducationLevel), MonthlyCoin: p.MonthlyCoin, IsFeatured: p.IsFeatured,
		Prices: PricesResponse{
			Monthly: p.PriceMonthly, Semester: p.PriceSemester, Yearly: p.PriceYearly, StrikeMonthly: p.StrikePriceMonthly,
		},
		Features: features, FeatureLabels: subscriptionmodel.EnabledLabels(features),
	}
}

// ToPlanResponse mengubah 1 paket menjadi response admin.
func ToPlanResponse(p subscriptionmodel.Plan) PlanResponse {
	return PlanResponse{
		StudentPlanResponse: ToStudentPlanResponse(p),
		SortOrder:           p.SortOrder, Status: p.Status,
		ActiveStudents: p.ActiveStudents, ModuleCount: p.ModuleCount,
		CreatedBy: toRef(p.CreatedBy), UpdatedBy: toRef(p.UpdatedBy),
		CreatedAt: p.CreatedAt.In(clock.Jakarta), UpdatedAt: p.UpdatedAt.In(clock.Jakarta),
	}
}

// ToPlanResponses mengubah daftar paket menjadi response admin. Kosong = [].
func ToPlanResponses(items []subscriptionmodel.Plan) []PlanResponse {
	result := make([]PlanResponse, 0, len(items))
	for _, p := range items {
		result = append(result, ToPlanResponse(p))
	}
	return result
}

// StudentPlansResponse adalah halaman Paket & Coin siswa: akses hari ini + paket yang dijual.
type StudentPlansResponse struct {
	Current AccessResponse        `json:"current"`
	Plans   []StudentPlanResponse `json:"plans"`
}

// ToStudentPlansResponse mengubah akses siswa + daftar paket menjadi response siswa.
func ToStudentPlansResponse(access subscriptionmodel.Access, plans []subscriptionmodel.Plan) StudentPlansResponse {
	r := StudentPlansResponse{Current: ToAccessResponse(access), Plans: make([]StudentPlanResponse, 0, len(plans))}
	for _, p := range plans {
		r.Plans = append(r.Plans, ToStudentPlanResponse(p))
	}
	return r
}

// ---------------------------------------------------------------------------
// Akses siswa (menu User / Coin & Akses)
// ---------------------------------------------------------------------------

// AccessResponse adalah akses 1 siswa hari ini (dipakai di list & detail user, dan halaman siswa).
type AccessResponse struct {
	Plan           *RefResponse    `json:"plan"`            // paket aktif hari ini; null = tidak ada
	SubscriptionID *string         `json:"subscription_id"` // id paket siswa yang aktif
	StartDate      *string         `json:"start_date"`      // "2026-11-01"
	EndDate        *string         `json:"end_date"`        // kolom "Berlaku s/d"
	DaysLeft       *int            `json:"days_left"`       // sisa hari termasuk hari ini
	Features       map[string]bool `json:"features"`        // akses akhir (paket + akses khusus)
	FeatureLabels  []string        `json:"feature_labels"`  // kolom "Akses", contoh ["Modul", "Daily Quiz"]
	Overrides      map[string]bool `json:"overrides"`       // akses khusus dari admin; {} = tidak ada
}

// ToAccessResponse mengubah akses siswa menjadi response.
func ToAccessResponse(a subscriptionmodel.Access) AccessResponse {
	features := a.Features
	if features == nil {
		features = subscriptionmodel.ResolveFeatures(a.Subscription, a.Overrides)
	}
	r := AccessResponse{
		Features: features, FeatureLabels: subscriptionmodel.EnabledLabels(features),
		Overrides: map[string]bool{},
	}
	for _, o := range a.Overrides {
		r.Overrides[o.Feature] = o.IsEnabled
	}
	if s := a.Subscription; s != nil {
		start, end := s.StartDate.Format(DateLayout), s.EndDate.Format(DateLayout)
		r.Plan = &RefResponse{ID: s.PlanID, Name: s.PlanName}
		r.SubscriptionID = &s.SubscriptionID
		r.StartDate, r.EndDate = &start, &end
		days := daysLeft(s.EndDate, time.Now())
		r.DaysLeft = &days
	}
	return r
}

// daysLeft = sisa hari paket termasuk hari ini (WIB): berakhir hari ini = 1.
func daysLeft(endDate, now time.Time) int {
	today, _ := time.Parse(DateLayout, now.In(clock.Jakarta).Format(DateLayout))
	end, _ := time.Parse(DateLayout, endDate.Format(DateLayout))
	return int(end.Sub(today).Hours()/24) + 1
}

// FeatureDetailResponse = 1 fitur di form edit akses: dari paket, akses khusus, dan hasil akhirnya.
type FeatureDetailResponse struct {
	Feature  string `json:"feature"`   // module, daily_quiz, uji_kom, tryout, rumath
	Label    string `json:"label"`     // "Modul", "Daily Quiz", ...
	FromPlan bool   `json:"from_plan"` // terbuka dari paket aktif
	Override *bool  `json:"override"`  // akses khusus admin; null = ikut paket
	Enabled  bool   `json:"enabled"`   // akses akhir
}

// StudentSubscriptionResponse adalah 1 periode paket siswa (riwayat paket).
type StudentSubscriptionResponse struct {
	ID          string       `json:"id"`
	Plan        RefResponse  `json:"plan"`
	Channel     string       `json:"channel"` // offline (dicatat admin) / online (beli di aplikasi)
	Period      string       `json:"period"`  // monthly, semester, yearly
	PricePaid   int64        `json:"price_paid"`
	StartDate   string       `json:"start_date"`
	EndDate     string       `json:"end_date"`
	Status      string       `json:"status"` // active, upcoming, expired, cancelled
	CancelledAt *time.Time   `json:"cancelled_at"`
	CancelledBy *RefResponse `json:"cancelled_by"`
	Notes       *string      `json:"notes"`
	CreatedBy   *RefResponse `json:"created_by"` // admin pencatat; null = beli sendiri / data awal
	CreatedAt   time.Time    `json:"created_at"`
}

// ToStudentSubscriptionResponse mengubah 1 paket siswa menjadi response.
func ToStudentSubscriptionResponse(s subscriptionmodel.StudentSubscription, today string) StudentSubscriptionResponse {
	r := StudentSubscriptionResponse{
		ID: s.ID, Plan: RefResponse{ID: s.Plan.ID, Name: s.Plan.Name}, Channel: s.Channel, Period: s.Period,
		PricePaid: s.PricePaid, StartDate: s.StartDate.Format(DateLayout), EndDate: s.EndDate.Format(DateLayout),
		Status: s.Status(today), CancelledBy: toRef(s.CancelledBy), Notes: s.Notes,
		CreatedBy: toRef(s.CreatedBy), CreatedAt: s.CreatedAt.In(clock.Jakarta),
	}
	if s.CancelledAt != nil {
		t := s.CancelledAt.In(clock.Jakarta)
		r.CancelledAt = &t
	}
	return r
}

// StudentAccessResponse adalah GET /api/v1/students/:id/access: akses hari ini, rincian per fitur
// (untuk form edit akses), dan riwayat paket.
type StudentAccessResponse struct {
	Student       AccessStudentResponse         `json:"student"`
	Access        AccessResponse                `json:"access"`
	Features      []FeatureDetailResponse       `json:"features"`
	Subscriptions []StudentSubscriptionResponse `json:"subscriptions"` // terbaru dulu
}

// AccessStudentResponse = data singkat siswa.
type AccessStudentResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StudentCode string `json:"student_code"`
	CoinBalance int    `json:"coin_balance"`
}

// ToStudentAccessResponse menyusun response akses 1 siswa.
func ToStudentAccessResponse(st subscriptionmodel.AccessStudent, a subscriptionmodel.Access, subs []subscriptionmodel.StudentSubscription, today string) StudentAccessResponse {
	r := StudentAccessResponse{
		Student: AccessStudentResponse{ID: st.ID, Name: st.Name, StudentCode: st.StudentCode, CoinBalance: st.CoinBalance},
		Access:  ToAccessResponse(a), Features: make([]FeatureDetailResponse, 0, len(subscriptionmodel.AllFeatures)),
		Subscriptions: make([]StudentSubscriptionResponse, 0, len(subs)),
	}
	fromPlan := subscriptionmodel.ResolveFeatures(a.Subscription, nil)
	for _, f := range subscriptionmodel.AllFeatures {
		d := FeatureDetailResponse{Feature: f, Label: subscriptionmodel.FeatureLabels[f], FromPlan: fromPlan[f], Enabled: r.Access.Features[f]}
		if v, ok := r.Access.Overrides[f]; ok {
			d.Override = &v
		}
		r.Features = append(r.Features, d)
	}
	for _, s := range subs {
		r.Subscriptions = append(r.Subscriptions, ToStudentSubscriptionResponse(s, today))
	}
	return r
}

// AssignRequest adalah body POST /api/v1/student-subscriptions: beri 1 paket ke 1-100 siswa sekaligus
// (pembayaran offline dicatat admin). Contoh:
//
//	{"plan_id": "...", "student_ids": ["...", "..."], "period": "monthly", "start_date": null,
//	 "price_paid": null, "notes": "Bayar tunai", "request_id": "..."}
type AssignRequest struct {
	PlanID     string   `json:"plan_id" binding:"required,uuid"`
	StudentIDs []string `json:"student_ids" binding:"required,min=1,max=100,unique,dive,uuid"`
	Period     string   `json:"period" binding:"required,oneof=monthly semester yearly"`
	// Tanggal mulai (opsional). null = otomatis: hari ini, atau sehari setelah paket terakhir siswa
	// berakhir (perpanjangan). Tanggal berakhir = mulai + 1/6/12 bulan - 1 hari.
	StartDate *string `json:"start_date" binding:"omitempty,datetime=2006-01-02"`
	// Harga yang dibayar (opsional, contoh diskon). null = harga paket untuk periode itu.
	PricePaid *int64  `json:"price_paid" binding:"omitempty,min=0,max=1000000000"`
	Notes     *string `json:"notes" binding:"omitempty,max=150"`
	// Opsional tapi disarankan: UUID baru dari FE setiap kali tombol simpan ditekan. Request yang sama
	// terkirim 2x → yang kedua ditolak 409, paket tidak dobel.
	RequestID *string `json:"request_id" binding:"omitempty,uuid"`
}

// AssignResponse adalah hasil beri paket: paket + hasil setiap siswa (urutan sama dengan request).
type AssignResponse struct {
	Plan       RefResponse             `json:"plan"`
	Period     string                  `json:"period"`
	TotalPrice int64                   `json:"total_price"` // price_paid x jumlah siswa
	Students   []AssignStudentResponse `json:"students"`
}

// AssignStudentResponse adalah hasil beri paket ke 1 siswa.
type AssignStudentResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	StudentCode    string `json:"student_code"`
	SubscriptionID string `json:"subscription_id"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	Status         string `json:"status"` // active, atau upcoming kalau perpanjangan / mulai nanti
	PricePaid      int64  `json:"price_paid"`
	CoinAdded      int    `json:"coin_added"`   // bonus coin bulan pertama
	CoinBalance    int    `json:"coin_balance"` // saldo coin setelahnya
}

// ToAssignResponse mengubah hasil beri paket menjadi response.
func ToAssignResponse(plan subscriptionmodel.Plan, period string, results []subscriptionmodel.AssignResult, today string) AssignResponse {
	r := AssignResponse{
		Plan: RefResponse{ID: plan.ID, Name: plan.Name}, Period: period,
		Students: make([]AssignStudentResponse, 0, len(results)),
	}
	for _, res := range results {
		s := res.Subscription
		r.TotalPrice += s.PricePaid
		r.Students = append(r.Students, AssignStudentResponse{
			ID: s.StudentID, Name: res.StudentName, StudentCode: res.StudentCode, SubscriptionID: s.ID,
			StartDate: s.StartDate.Format(DateLayout), EndDate: s.EndDate.Format(DateLayout),
			Status: s.Status(today), PricePaid: s.PricePaid, CoinAdded: res.CoinAdded, CoinBalance: res.CoinBalance,
		})
	}
	return r
}

// CancelRequest adalah body POST /api/v1/student-subscriptions/:id/cancel (opsional).
type CancelRequest struct {
	Reason *string `json:"reason" binding:"omitempty,max=150"`
}

// OverridesRequest adalah body PUT /api/v1/student-access-overrides: atur akses khusus 1-100 siswa
// sekaligus. true = dibuka walau tidak ada di paket, false = ditutup walau ada di paket, null = hapus
// akses khusus (kembali ikut paket). Fitur yang tidak dikirim tidak berubah. Contoh:
//
//	{"student_ids": ["..."], "features": {"rumath": true, "module": null}}
type OverridesRequest struct {
	StudentIDs []string         `json:"student_ids" binding:"required,min=1,max=100,unique,dive,uuid"`
	Features   map[string]*bool `json:"features" binding:"required,min=1,dive,keys,oneof=module daily_quiz uji_kom tryout rumath,endkeys"`
}

// StudentAccessItemResponse = akses 1 siswa setelah diubah.
type StudentAccessItemResponse struct {
	Student AccessStudentResponse `json:"student"`
	Access  AccessResponse        `json:"access"`
}
