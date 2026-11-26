package subscription

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subscription"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// todayAccess = paket aktif hari ini + akses khusus + akses akhir 1 siswa (2 query).
func (s *SubscriptionService) todayAccess(ctx context.Context, studentID string) (subscriptionmodel.Access, error) {
	subscription, err := s.repo.FindActive(ctx, studentID, today())
	if err != nil {
		return subscriptionmodel.Access{}, err
	}
	overrides, err := s.repo.FindOverrides(ctx, studentID)
	if err != nil {
		return subscriptionmodel.Access{}, err
	}
	return subscriptionmodel.Access{
		Subscription: subscription, Overrides: overrides,
		Features: subscriptionmodel.ResolveFeatures(subscription, overrides),
	}, nil
}

// AccessMany mengambil akses hari ini banyak siswa sekaligus (2 query, bukan per siswa). Dipakai list
// & detail menu User. Hasilnya map siswa → akses (semua ID ada di map).
func (s *SubscriptionService) AccessMany(ctx context.Context, studentIDs []string) (map[string]subscriptionmodel.Access, error) {
	result := make(map[string]subscriptionmodel.Access, len(studentIDs))
	if len(studentIDs) == 0 {
		return result, nil
	}
	active, err := s.repo.FindActiveMany(ctx, studentIDs, today())
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.FindOverridesMany(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range studentIDs {
		var sub *subscriptionmodel.ActiveSubscription
		if a, ok := active[id]; ok {
			sub = &a
		}
		result[id] = subscriptionmodel.Access{
			Subscription: sub, Overrides: overrides[id],
			Features: subscriptionmodel.ResolveFeatures(sub, overrides[id]),
		}
	}
	return result, nil
}

// StudentAccess mengambil data akses 1 siswa untuk admin: siswa, akses hari ini, dan riwayat paket.
// Bukan UUID / bukan siswa → ErrStudentNotFound.
func (s *SubscriptionService) StudentAccess(ctx context.Context, studentID string) (subscriptionmodel.AccessStudent, subscriptionmodel.Access, []subscriptionmodel.StudentSubscription, error) {
	if uuid.Validate(studentID) != nil {
		return subscriptionmodel.AccessStudent{}, subscriptionmodel.Access{}, nil, subscriptionmodel.ErrStudentNotFound
	}
	students, err := s.repo.FindAccessStudents(ctx, []string{studentID}, today())
	if err != nil {
		return subscriptionmodel.AccessStudent{}, subscriptionmodel.Access{}, nil, err
	}
	student, ok := students[studentID]
	if !ok {
		return subscriptionmodel.AccessStudent{}, subscriptionmodel.Access{}, nil, subscriptionmodel.ErrStudentNotFound
	}
	access, err := s.todayAccess(ctx, studentID)
	if err != nil {
		return subscriptionmodel.AccessStudent{}, subscriptionmodel.Access{}, nil, err
	}
	subs, err := s.repo.ListStudentSubscriptions(ctx, studentID)
	if err != nil {
		return subscriptionmodel.AccessStudent{}, subscriptionmodel.Access{}, nil, err
	}
	return student, access, subs, nil
}

// Assign memberi 1 paket aktif ke 1-100 siswa sekaligus (pembayaran offline dicatat admin). Semua
// atau tidak sama sekali: kesalahan dikumpulkan per field, tidak ada paket yang tersimpan.
//
// Tanggal mulai otomatis (start_date kosong): hari ini, atau sehari setelah paket terakhir siswa
// berakhir (perpanjangan, FR-SUB-011). Tanggal berakhir = mulai + 1/6/12 bulan - 1 hari.
// Bonus coin bulan pertama (monthly_coin paket) langsung masuk ke saldo siswa.
func (s *SubscriptionService) Assign(ctx context.Context, req subscriptiondto.AssignRequest, adminID string) (subscriptionmodel.Plan, []subscriptionmodel.AssignResult, error) {
	errs := []subscriptionmodel.FieldError{}

	plan, err := s.repo.FindPlan(ctx, req.PlanID, today())
	if err != nil && err != subscriptionmodel.ErrPlanNotFound {
		return subscriptionmodel.Plan{}, nil, err
	}
	price, planErr := assignPrice(plan, err, req)
	if planErr != nil {
		errs = append(errs, *planErr)
	}

	students, err := s.repo.FindAccessStudents(ctx, req.StudentIDs, today())
	if err != nil {
		return subscriptionmodel.Plan{}, nil, err
	}
	todayDate, _ := time.Parse(subscriptiondto.DateLayout, today())
	subs := make([]subscriptionmodel.NewSubscription, 0, len(req.StudentIDs))
	for i, id := range req.StudentIDs {
		st, ok := students[id]
		message := "siswa tidak ditemukan"
		var sub subscriptionmodel.NewSubscription
		if ok {
			sub, message = newSubscription(st, plan, req, todayDate)
		}
		if message != "" {
			errs = append(errs, subscriptionmodel.FieldError{Field: fmt.Sprintf("student_ids[%d]", i), Message: message})
			continue
		}
		subs = append(subs, sub)
	}
	if len(errs) > 0 {
		return subscriptionmodel.Plan{}, nil, &subscriptionmodel.ValidationError{Fields: errs}
	}

	var notes *string
	if req.Notes != nil {
		if n := strings.TrimSpace(*req.Notes); n != "" {
			notes = &n
		}
	}
	in := subscriptionmodel.AssignInput{Plan: plan, Period: req.Period, PricePaid: price, Notes: notes, AdminID: adminID, Now: time.Now()}
	assigned, balances, err := s.repo.AssignMany(ctx, in, subs)
	if err != nil {
		return subscriptionmodel.Plan{}, nil, err
	}

	// Hasil urut sesuai student_ids di request.
	results := make([]subscriptionmodel.AssignResult, 0, len(req.StudentIDs))
	for _, id := range req.StudentIDs {
		st := students[id]
		sub := assigned[id]
		sub.Plan = subscriptionmodel.Ref{ID: plan.ID, Name: plan.Name}
		sub.Channel, sub.Period, sub.PricePaid, sub.Notes = "offline", req.Period, price, notes
		res := subscriptionmodel.AssignResult{Subscription: sub, StudentName: st.Name, StudentCode: st.StudentCode, CoinBalance: st.CoinBalance}
		if balance, ok := balances[id]; ok {
			res.CoinAdded, res.CoinBalance = plan.MonthlyCoin, balance
		}
		results = append(results, res)
	}
	return plan, results, nil
}

// addMonths menambah n bulan dengan tanggal dijepit ke akhir bulan: 31 Jan + 1 bulan = 28/29 Feb
// (bawaan Go AddDate akan melompat ke Maret).
func addMonths(t time.Time, n int) time.Time {
	firstOfTarget := time.Date(t.Year(), t.Month()+time.Month(n), 1, 0, 0, 0, 0, t.Location())
	lastDay := firstOfTarget.AddDate(0, 1, -1).Day()
	day := min(t.Day(), lastDay)
	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), day, 0, 0, 0, 0, t.Location())
}

// CancelSubscription membatalkan 1 paket siswa (aktif atau belum mulai). Akses siswa langsung
// mengikuti (paket batal tidak dihitung lagi). Paket yang sudah lewat tidak bisa dibatalkan.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, id string, req subscriptiondto.CancelRequest, adminID string) (subscriptionmodel.StudentSubscription, error) {
	if uuid.Validate(id) != nil {
		return subscriptionmodel.StudentSubscription{}, subscriptionmodel.ErrSubscriptionNotFound
	}
	sub, err := s.repo.FindSubscription(ctx, id)
	if err != nil {
		return subscriptionmodel.StudentSubscription{}, err
	}
	switch sub.Status(today()) {
	case subscriptionmodel.SubscriptionCancelled:
		return subscriptionmodel.StudentSubscription{}, subscriptionmodel.ErrSubscriptionCancelled
	case subscriptionmodel.SubscriptionExpired:
		return subscriptionmodel.StudentSubscription{}, &subscriptionmodel.ValidationError{Fields: []subscriptionmodel.FieldError{{
			Field: "id", Message: "paket sudah berakhir, tidak bisa dibatalkan",
		}}}
	}
	var reason *string
	if req.Reason != nil {
		if r := strings.TrimSpace(*req.Reason); r != "" {
			reason = &r
		}
	}
	if err := s.repo.CancelSubscription(ctx, id, reason, adminID); err != nil {
		return subscriptionmodel.StudentSubscription{}, err
	}
	return s.repo.FindSubscription(ctx, id)
}

// SetOverrides mengatur akses khusus 1-100 siswa sekaligus lalu mengembalikan akses terbarunya
// (urut sesuai request). Siswa tidak ditemukan → 422, tidak ada yang berubah.
func (s *SubscriptionService) SetOverrides(ctx context.Context, req subscriptiondto.OverridesRequest, adminID string) ([]subscriptionmodel.AccessStudent, map[string]subscriptionmodel.Access, error) {
	students, err := s.repo.FindAccessStudents(ctx, req.StudentIDs, today())
	if err != nil {
		return nil, nil, err
	}
	errs := []subscriptionmodel.FieldError{}
	for i, id := range req.StudentIDs {
		if _, ok := students[id]; !ok {
			errs = append(errs, subscriptionmodel.FieldError{Field: fmt.Sprintf("student_ids[%d]", i), Message: "siswa tidak ditemukan"})
		}
	}
	if len(errs) > 0 {
		return nil, nil, &subscriptionmodel.ValidationError{Fields: errs}
	}
	if err := s.repo.SetOverrides(ctx, req.StudentIDs, req.Features, adminID); err != nil {
		return nil, nil, err
	}
	access, err := s.AccessMany(ctx, req.StudentIDs)
	if err != nil {
		return nil, nil, err
	}
	ordered := make([]subscriptionmodel.AccessStudent, 0, len(req.StudentIDs))
	for _, id := range req.StudentIDs {
		ordered = append(ordered, students[id])
	}
	return ordered, access, nil
}

// assignPrice mengecek paket yang akan diberikan dan menentukan harga yang dibayar.
// findErr = hasil FindPlan (nil atau ErrPlanNotFound). Kesalahan dikembalikan sebagai FieldError.
func assignPrice(plan subscriptionmodel.Plan, findErr error, req subscriptiondto.AssignRequest) (int64, *subscriptionmodel.FieldError) {
	if findErr != nil {
		return 0, &subscriptionmodel.FieldError{Field: "plan_id", Message: "paket akses tidak ditemukan atau sudah dihapus"}
	}
	if plan.Status != subscriptionmodel.PlanActive {
		return 0, &subscriptionmodel.FieldError{Field: "plan_id", Message: fmt.Sprintf("paket %q tidak aktif", plan.Name)}
	}
	listPrice := plan.Price(req.Period)
	if listPrice == nil {
		return 0, &subscriptionmodel.FieldError{Field: "period", Message: fmt.Sprintf("paket %q tidak dijual untuk periode ini", plan.Name)}
	}
	if req.PricePaid != nil {
		return *req.PricePaid, nil // harga khusus (contoh diskon)
	}
	return *listPrice, nil
}

// newSubscription menyusun periode paket 1 siswa. message != "" = siswa tidak bisa diberi paket ini.
//
// Tanggal mulai: start_date dari request, atau otomatis hari ini / sehari setelah paket terakhir
// siswa berakhir (perpanjangan). Tanggal berakhir = mulai + 1/6/12 bulan - 1 hari.
func newSubscription(st subscriptionmodel.AccessStudent, plan subscriptionmodel.Plan, req subscriptiondto.AssignRequest, todayDate time.Time) (subscriptionmodel.NewSubscription, string) {
	if plan.EducationLevel != nil && (st.EducationLevelID == nil || *st.EducationLevelID != plan.EducationLevel.ID) {
		return subscriptionmodel.NewSubscription{}, fmt.Sprintf("%s bukan siswa jenjang %s (sasaran paket %q)", st.Name, plan.EducationLevel.Name, plan.Name)
	}

	start := todayDate
	if st.LastEndDate != nil {
		start = st.LastEndDate.AddDate(0, 0, 1) // perpanjangan: sehari setelah paket terakhir
	}
	if req.StartDate != nil {
		fixed, _ := time.Parse(subscriptiondto.DateLayout, *req.StartDate) // format sudah dicek binding
		if st.LastEndDate != nil && !fixed.After(*st.LastEndDate) {
			return subscriptionmodel.NewSubscription{}, fmt.Sprintf(
				"%s masih punya paket sampai %s; kosongkan start_date untuk perpanjangan otomatis atau batalkan paket itu dulu",
				st.Name, st.LastEndDate.Format(subscriptiondto.DateLayout))
		}
		start = fixed
	}

	months := subscriptionmodel.PeriodMonths[req.Period]
	sub := subscriptionmodel.NewSubscription{StudentID: st.ID, StartDate: start, EndDate: addMonths(start, months).AddDate(0, 0, -1)}
	if req.RequestID != nil {
		key := "assign:" + *req.RequestID + ":" + st.ID
		sub.IdempotencyKey = &key
	}
	return sub, ""
}
