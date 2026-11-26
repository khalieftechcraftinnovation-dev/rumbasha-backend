package subscription

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subscription"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// today = tanggal hari ini (WIB), contoh "2026-11-26".
func today() string {
	return clock.TodayInJakarta(time.Now())
}

// ListPlans mengambil paket akses untuk admin (filter status, jenjang, nama).
func (s *SubscriptionService) ListPlans(ctx context.Context, q subscriptiondto.ListPlansQuery) ([]subscriptionmodel.Plan, error) {
	return s.repo.ListPlans(ctx, subscriptionmodel.PlanFilter{
		Status: q.Status, EducationLevelID: q.EducationLevelID, Search: strings.TrimSpace(q.Search), Today: today(),
	})
}

// StudentPlans mengambil akses siswa hari ini + paket aktif yang dijual untuk jenjangnya (paket
// jenjang itu + paket semua jenjang). Halaman Paket & Coin.
func (s *SubscriptionService) StudentPlans(ctx context.Context, studentID string) (subscriptionmodel.Access, []subscriptionmodel.Plan, error) {
	students, err := s.repo.FindAccessStudents(ctx, []string{studentID}, today())
	if err != nil {
		return subscriptionmodel.Access{}, nil, err
	}
	student, ok := students[studentID]
	if !ok {
		return subscriptionmodel.Access{}, nil, subscriptionmodel.ErrStudentNotFound
	}
	access, err := s.todayAccess(ctx, studentID)
	if err != nil {
		return subscriptionmodel.Access{}, nil, err
	}
	f := subscriptionmodel.PlanFilter{Status: subscriptionmodel.PlanActive, Today: today()}
	if student.EducationLevelID != nil {
		f.EducationLevelID = *student.EducationLevelID
	}
	plans, err := s.repo.ListPlans(ctx, f)
	if err != nil {
		return subscriptionmodel.Access{}, nil, err
	}
	if student.EducationLevelID == nil {
		// Jenjang siswa belum diisi: hanya paket untuk semua jenjang.
		all := plans[:0]
		for _, p := range plans {
			if p.EducationLevel == nil {
				all = append(all, p)
			}
		}
		plans = all
	}
	return access, plans, nil
}

// GetPlan mengambil 1 paket. Bukan UUID / tidak ada / sudah dihapus → ErrPlanNotFound.
func (s *SubscriptionService) GetPlan(ctx context.Context, id string) (subscriptionmodel.Plan, error) {
	if uuid.Validate(id) != nil {
		return subscriptionmodel.Plan{}, subscriptionmodel.ErrPlanNotFound
	}
	return s.repo.FindPlan(ctx, id, today())
}

// CreatePlan membuat paket baru lalu mengembalikan datanya.
func (s *SubscriptionService) CreatePlan(ctx context.Context, req subscriptiondto.PlanRequest, adminID string) (subscriptionmodel.Plan, error) {
	p, err := s.buildPlan(ctx, req)
	if err != nil {
		return subscriptionmodel.Plan{}, err
	}
	id, err := s.repo.CreatePlan(ctx, p, adminID)
	if err != nil {
		return subscriptionmodel.Plan{}, planNameError(err)
	}
	return s.repo.FindPlan(ctx, id, today())
}

// UpdatePlan mengganti isi 1 paket (ganti penuh). Fitur baru langsung berlaku untuk siswa yang
// paketnya aktif; harga baru hanya untuk pemberian / pembelian berikutnya.
//
// Sasaran jenjang tidak boleh diubah ke jenjang lain kalau masih ada siswa jenjang berbeda yang
// paketnya aktif / belum mulai dengan paket ini.
func (s *SubscriptionService) UpdatePlan(ctx context.Context, id string, req subscriptiondto.PlanRequest, adminID string) (subscriptionmodel.Plan, error) {
	if uuid.Validate(id) != nil {
		return subscriptionmodel.Plan{}, subscriptionmodel.ErrPlanNotFound
	}
	current, err := s.repo.FindPlan(ctx, id, today())
	if err != nil {
		return subscriptionmodel.Plan{}, err
	}
	p, err := s.buildPlan(ctx, req)
	if err != nil {
		return subscriptionmodel.Plan{}, err
	}
	p.ID = id

	levelChanged := p.EducationLevelID != nil && (current.EducationLevel == nil || current.EducationLevel.ID != *p.EducationLevelID)
	if levelChanged {
		n, err := s.repo.CountOtherLevelStudents(ctx, id, *p.EducationLevelID, today())
		if err != nil {
			return subscriptionmodel.Plan{}, err
		}
		if n > 0 {
			return subscriptionmodel.Plan{}, &subscriptionmodel.ValidationError{Fields: []subscriptionmodel.FieldError{{
				Field:   "education_level_id",
				Message: fmt.Sprintf("masih dipakai %d siswa jenjang lain (paket aktif / belum mulai)", n),
			}}}
		}
	}

	if err := s.repo.UpdatePlan(ctx, p, adminID); err != nil {
		return subscriptionmodel.Plan{}, planNameError(err)
	}
	return s.repo.FindPlan(ctx, id, today())
}

// DeletePlans menghapus 1-100 paket (soft delete). Semua atau tidak sama sekali; kesalahan per
// field "ids[i]": tidak ditemukan, masih dipakai siswa, atau menjadi satu-satunya paket modul.
func (s *SubscriptionService) DeletePlans(ctx context.Context, ids []string, adminID string) ([]string, error) {
	missing, blocked, err := s.repo.SoftDeletePlans(ctx, ids, adminID, today())
	if err != nil {
		return nil, err
	}
	if len(missing) == 0 && len(blocked) == 0 {
		return ids, nil
	}

	messages := map[string]string{}
	for _, id := range missing {
		messages[id] = "paket tidak ditemukan atau sudah dihapus"
	}
	for _, u := range blocked {
		var reasons []string
		if u.CurrentStudents > 0 {
			reasons = append(reasons, fmt.Sprintf("masih dipakai %d siswa (aktif / belum mulai)", u.CurrentStudents))
		}
		if u.OnlyPlanModules > 0 {
			reasons = append(reasons, fmt.Sprintf("menjadi satu-satunya paket di %d modul", u.OnlyPlanModules))
		}
		messages[u.PlanID] = strings.Join(reasons, " dan ") + "; nonaktifkan paket atau pindahkan dulu"
	}
	errs := []subscriptionmodel.FieldError{}
	for i, id := range ids {
		if m, ok := messages[id]; ok {
			errs = append(errs, subscriptionmodel.FieldError{Field: fmt.Sprintf("ids[%d]", i), Message: m})
		}
	}
	return nil, &subscriptionmodel.ValidationError{Fields: errs}
}

// buildPlan merapikan & memvalidasi isi request (semua kesalahan dikumpulkan sekaligus).
func (s *SubscriptionService) buildPlan(ctx context.Context, req subscriptiondto.PlanRequest) (subscriptionmodel.PlanSave, error) {
	errs := []subscriptionmodel.FieldError{}
	add := func(field, message string) {
		errs = append(errs, subscriptionmodel.FieldError{Field: field, Message: message})
	}

	p := subscriptionmodel.PlanSave{
		Name: strings.TrimSpace(req.Name), IsFeatured: req.IsFeatured,
		PriceMonthly: req.PriceMonthly, PriceSemester: req.PriceSemester, PriceYearly: req.PriceYearly,
		StrikePriceMonthly: req.StrikePriceMonthly, Status: req.Status,
		Features: subscriptionmodel.Features{
			Module: req.Features.Module, DailyQuiz: req.Features.DailyQuiz, UjiKom: req.Features.UjiKom,
			Tryout: req.Features.Tryout, Rumath: req.Features.Rumath,
		},
	}
	if p.Name == "" {
		add("name", "wajib diisi")
	}
	if req.Description != nil {
		if d := strings.TrimSpace(*req.Description); d != "" {
			p.Description = &d
		}
	}
	var lines []string
	for i, h := range req.Highlights {
		h = strings.TrimSpace(strings.ReplaceAll(h, "\n", " "))
		if h == "" {
			add(fmt.Sprintf("highlights[%d]", i), "tidak boleh kosong")
			continue
		}
		lines = append(lines, h)
	}
	if len(lines) > 0 {
		joined := strings.Join(lines, "\n")
		p.Highlights = &joined
	}
	if req.MonthlyCoin != nil {
		p.MonthlyCoin = *req.MonthlyCoin
	}
	if req.SortOrder != nil {
		p.SortOrder = *req.SortOrder
	}
	if p.Status == "" {
		p.Status = subscriptionmodel.PlanActive
	}

	// Sama dengan constraint subscription_plans_has_price & strike_price_valid di database.
	if p.PriceMonthly == nil && p.PriceSemester == nil && p.PriceYearly == nil {
		add("price_monthly", "minimal 1 harga periode (bulanan, semester, atau tahunan) wajib diisi")
	}
	if p.StrikePriceMonthly != nil {
		switch {
		case p.PriceMonthly == nil:
			add("strike_price_monthly", "harga coret hanya boleh diisi kalau ada harga bulanan")
		case *p.StrikePriceMonthly <= *p.PriceMonthly:
			add("strike_price_monthly", "harus lebih mahal dari harga bulanan")
		}
	}
	if f := p.Features; !f.Module && !f.DailyQuiz && !f.UjiKom && !f.Tryout && !f.Rumath {
		add("features", "minimal 1 fitur dibuka")
	}

	if req.EducationLevelID != nil {
		name, err := s.repo.FindEducationLevel(ctx, *req.EducationLevelID)
		if err != nil {
			return subscriptionmodel.PlanSave{}, err
		}
		if name == "" {
			add("education_level_id", "jenjang tidak ditemukan atau tidak aktif")
		}
		p.EducationLevelID = req.EducationLevelID
	}

	if len(errs) > 0 {
		return subscriptionmodel.PlanSave{}, &subscriptionmodel.ValidationError{Fields: errs}
	}
	return p, nil
}

// planNameError mengubah ErrPlanNameTaken menjadi pesan validasi field name.
func planNameError(err error) error {
	if err == subscriptionmodel.ErrPlanNameTaken {
		return &subscriptionmodel.ValidationError{Fields: []subscriptionmodel.FieldError{{Field: "name", Message: "sudah dipakai paket lain"}}}
	}
	return err
}
