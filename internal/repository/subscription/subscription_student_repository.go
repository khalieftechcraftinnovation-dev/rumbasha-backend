package subscription

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// FindAccessStudents mengambil siswa (role student, belum dihapus) berdasarkan ID + hari terakhir
// paketnya yang belum lewat, 1 query. Hasilnya map id → siswa; ID yang tidak ada = bukan siswa.
// [Native SQL, primary key + index student_subscriptions_student_id_end_date_idx]
func (r *SubscriptionRepository) FindAccessStudents(ctx context.Context, ids []string, today string) (map[string]subscriptionmodel.AccessStudent, error) {
	var rows []subscriptionmodel.AccessStudent
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.real_name AS name, s.student_code, u.education_level_id, s.coin_balance,
		       (SELECT MAX(ss.end_date) FROM student_subscriptions ss
		        WHERE ss.student_id = u.id AND ss.cancelled_at IS NULL AND ss.end_date >= ?::date) AS last_end_date
		FROM users u
		JOIN students s ON s.user_id = u.id
		WHERE u.id IN ? AND u.role = 'student' AND u.deleted_at IS NULL`, today, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]subscriptionmodel.AccessStudent, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

// assignedRow = 1 baris paket siswa yang baru disimpan.
type assignedRow struct {
	ID, StudentID      string
	StartDate, EndDate time.Time
	CreatedAt          time.Time
}

// AssignMany menyimpan paket untuk banyak siswa sekaligus dalam 1 transaksi (semua atau tidak sama
// sekali), dicatat sebagai pembayaran offline oleh admin. Kalau paket punya monthly_coin, siswa
// langsung mendapat bonus coin bulan pertama (coin_transactions type plan_grant). [Native SQL, 2 query]
//
//	INSERT INTO student_subscriptions (...) VALUES (...), (...) RETURNING ...
//	WITH upd AS (UPDATE students ... RETURNING ...) INSERT INTO coin_transactions ... SELECT ... FROM upd
//
// Kunci unik kembar (request_id sama) → ErrAssignDuplicate; tanggal bertumpuk dengan paket lain
// (diubah orang lain di saat yang sama) → ErrSubscriptionOverlap.
func (r *SubscriptionRepository) AssignMany(ctx context.Context, in subscriptionmodel.AssignInput, subs []subscriptionmodel.NewSubscription) (map[string]subscriptionmodel.StudentSubscription, map[string]int, error) {
	assigned := map[string]subscriptionmodel.StudentSubscription{}
	balances := map[string]int{} // siswa → saldo coin setelah bonus (hanya yang mendapat bonus)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		values := make([]string, 0, len(subs))
		args := make([]any, 0, len(subs)*11)
		for _, s := range subs {
			values = append(values, "(?, ?, 'offline', ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args, s.StudentID, in.Plan.ID, in.Period, in.PricePaid,
				s.StartDate.Format("2006-01-02"), s.EndDate.Format("2006-01-02"),
				in.Notes, in.AdminID, s.IdempotencyKey, in.Now, in.Now)
		}
		var rows []assignedRow
		err := tx.Raw(`
			INSERT INTO student_subscriptions
				(student_id, plan_id, channel, period, price_paid, start_date, end_date,
				 notes, created_by, idempotency_key, created_at, updated_at)
			VALUES `+strings.Join(values, ", ")+`
			RETURNING id, student_id, start_date, end_date, created_at`, args...).
			Scan(&rows).Error
		switch {
		case isUniqueViolation(err):
			return subscriptionmodel.ErrAssignDuplicate
		case isExclusionViolation(err):
			return subscriptionmodel.ErrSubscriptionOverlap
		case err != nil:
			return err
		}
		subIDs := make([]string, 0, len(rows))
		for _, row := range rows {
			assigned[row.StudentID] = subscriptionmodel.StudentSubscription{
				ID: row.ID, StudentID: row.StudentID, StartDate: row.StartDate, EndDate: row.EndDate, CreatedAt: row.CreatedAt,
			}
			subIDs = append(subIDs, row.ID)
		}

		if in.Plan.MonthlyCoin <= 0 {
			return nil
		}
		var granted []struct {
			StudentID    string
			BalanceAfter int
		}
		err = tx.Raw(`
			WITH upd AS (
				UPDATE students s SET coin_balance = s.coin_balance + ?, updated_at = NOW()
				FROM student_subscriptions ss
				WHERE ss.id IN ? AND s.user_id = ss.student_id
				RETURNING s.user_id, s.coin_balance, ss.id AS subscription_id
			)
			INSERT INTO coin_transactions
				(student_id, type, amount, balance_after, reference_type, reference_id,
				 idempotency_key, description, created_by, created_at)
			SELECT upd.user_id, 'plan_grant', ?, upd.coin_balance, 'student_subscription', upd.subscription_id,
			       'plan_grant:' || upd.subscription_id || ':1', ?, ?, ?
			FROM upd
			RETURNING student_id, balance_after`,
			in.Plan.MonthlyCoin, subIDs,
			in.Plan.MonthlyCoin, "Bonus coin bulan pertama paket "+in.Plan.Name, in.AdminID, in.Now).
			Scan(&granted).Error
		if err != nil {
			return err
		}
		for _, g := range granted {
			balances[g.StudentID] = g.BalanceAfter
		}
		return nil
	})
	return assigned, balances, err
}

// subscriptionSelect = kolom paket siswa + nama paket, pembatal, dan pencatat.
const subscriptionSelect = `
	SELECT ss.id, ss.student_id, ss.plan_id, p.name AS plan_name, ss.channel, ss.period, ss.price_paid,
	       ss.start_date, ss.end_date, ss.cancelled_at,
	       ss.cancelled_by AS cancelled_by_id, cu.real_name AS cancelled_by_name, ss.notes,
	       ss.created_by AS created_by_id, au.real_name AS created_by_name, ss.created_at
	FROM student_subscriptions ss
	JOIN subscription_plans p ON p.id = ss.plan_id
	LEFT JOIN users cu ON cu.id = ss.cancelled_by
	LEFT JOIN users au ON au.id = ss.created_by`

type subscriptionRow struct {
	ID, StudentID, PlanID, PlanName string
	Channel, Period                 string
	PricePaid                       int64
	StartDate, EndDate              time.Time
	CancelledAt                     *time.Time
	CancelledByID, CancelledByName  *string
	Notes                           *string
	CreatedByID, CreatedByName      *string
	CreatedAt                       time.Time
}

func (row subscriptionRow) toModel() subscriptionmodel.StudentSubscription {
	return subscriptionmodel.StudentSubscription{
		ID: row.ID, StudentID: row.StudentID, Plan: subscriptionmodel.Ref{ID: row.PlanID, Name: row.PlanName},
		Channel: row.Channel, Period: row.Period, PricePaid: row.PricePaid,
		StartDate: row.StartDate, EndDate: row.EndDate, CancelledAt: row.CancelledAt,
		CancelledBy: ref(row.CancelledByID, row.CancelledByName), Notes: row.Notes,
		CreatedBy: ref(row.CreatedByID, row.CreatedByName), CreatedAt: row.CreatedAt,
	}
}

// ListStudentSubscriptions mengambil semua paket 1 siswa (riwayat), terbaru dulu. 1 siswa hanya
// punya sedikit baris (1 per periode), jadi tanpa pagination.
// [Native SQL, index student_subscriptions_student_id_end_date_idx]
func (r *SubscriptionRepository) ListStudentSubscriptions(ctx context.Context, studentID string) ([]subscriptionmodel.StudentSubscription, error) {
	var rows []subscriptionRow
	err := r.db.WithContext(ctx).
		Raw(subscriptionSelect+` WHERE ss.student_id = ? ORDER BY ss.end_date DESC, ss.created_at DESC`, studentID).
		Scan(&rows).Error
	items := make([]subscriptionmodel.StudentSubscription, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toModel())
	}
	return items, err
}

// FindSubscription mengambil 1 paket siswa (siswanya belum dihapus). Tidak ada →
// ErrSubscriptionNotFound. [Native SQL, primary key]
func (r *SubscriptionRepository) FindSubscription(ctx context.Context, id string) (subscriptionmodel.StudentSubscription, error) {
	var rows []subscriptionRow
	err := r.db.WithContext(ctx).Raw(subscriptionSelect+`
		JOIN users su ON su.id = ss.student_id AND su.deleted_at IS NULL
		WHERE ss.id = ?`, id).Scan(&rows).Error
	if err != nil {
		return subscriptionmodel.StudentSubscription{}, err
	}
	if len(rows) == 0 {
		return subscriptionmodel.StudentSubscription{}, subscriptionmodel.ErrSubscriptionNotFound
	}
	return rows[0].toModel(), nil
}

// CancelSubscription membatalkan 1 paket siswa (cancelled_at diisi). Alasan (opsional) ditambahkan
// ke catatan. Sudah dibatalkan → ErrSubscriptionCancelled. Coin bonus yang sudah diberikan tidak
// ditarik. [Native SQL, primary key]
func (r *SubscriptionRepository) CancelSubscription(ctx context.Context, id string, reason *string, cancelledBy string) error {
	note := "Dibatalkan admin"
	if reason != nil {
		note += ": " + *reason
	}
	result := r.db.WithContext(ctx).Exec(`
		UPDATE student_subscriptions SET
			cancelled_at = NOW(), cancelled_by = ?,
			notes = CASE WHEN notes IS NULL OR notes = '' THEN ? ELSE notes || E'\n' || ? END,
			updated_at = NOW()
		WHERE id = ? AND cancelled_at IS NULL`, cancelledBy, note, note, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return subscriptionmodel.ErrSubscriptionCancelled
	}
	return nil
}

// SetOverrides mengatur akses khusus banyak siswa sekaligus dalam 1 transaksi:
// nilai true/false = disimpan (UPSERT), nil = dihapus (kembali ikut paket). [Native SQL, ≤ 2 query]
func (r *SubscriptionRepository) SetOverrides(ctx context.Context, studentIDs []string, features map[string]*bool, updatedBy string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var values []string
		var args []any
		var removed []string
		for feature, enabled := range features {
			if enabled == nil {
				removed = append(removed, feature)
				continue
			}
			for _, id := range studentIDs {
				values = append(values, "(?, ?, ?, ?, NOW(), NOW())")
				args = append(args, id, feature, *enabled, updatedBy)
			}
		}
		if len(removed) > 0 {
			err := tx.Exec(`DELETE FROM student_feature_overrides WHERE student_id IN ? AND feature IN ?`, studentIDs, removed).Error
			if err != nil {
				return err
			}
		}
		if len(values) == 0 {
			return nil
		}
		return tx.Exec(`
			INSERT INTO student_feature_overrides (student_id, feature, is_enabled, updated_by, created_at, updated_at)
			VALUES `+strings.Join(values, ", ")+`
			ON CONFLICT (student_id, feature) DO UPDATE SET
				is_enabled = EXCLUDED.is_enabled, updated_by = EXCLUDED.updated_by, updated_at = NOW()`, args...).Error
	})
}

// FindActiveMany mengambil paket aktif (tanggal date) banyak siswa sekaligus, 1 query.
// Hasilnya map siswa → paket aktif; siswa tanpa paket aktif tidak ada di map.
// [Native SQL, index student_subscriptions_student_id_end_date_idx]
func (r *SubscriptionRepository) FindActiveMany(ctx context.Context, studentIDs []string, date string) (map[string]subscriptionmodel.ActiveSubscription, error) {
	var rows []struct {
		StudentID string
		Sub       subscriptionmodel.ActiveSubscription `gorm:"embedded"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT ss.student_id,
		       ss.id AS subscription_id, p.id AS plan_id, p.name AS plan_name, ss.channel,
		       ss.start_date, ss.end_date,
		       p.has_module, p.has_daily_quiz, p.has_uji_kom, p.has_tryout, p.has_rumath
		FROM student_subscriptions ss
		JOIN subscription_plans p ON p.id = ss.plan_id
		WHERE ss.student_id IN ? AND ss.cancelled_at IS NULL
		  AND ?::date BETWEEN ss.start_date AND ss.end_date`, studentIDs, date).
		Scan(&rows).Error
	result := make(map[string]subscriptionmodel.ActiveSubscription, len(rows))
	for _, row := range rows {
		result[row.StudentID] = row.Sub
	}
	return result, err
}

// FindOverridesMany mengambil akses khusus banyak siswa sekaligus, 1 query (primary key).
func (r *SubscriptionRepository) FindOverridesMany(ctx context.Context, studentIDs []string) (map[string][]subscriptionmodel.FeatureOverride, error) {
	var rows []subscriptionmodel.FeatureOverride
	err := r.db.WithContext(ctx).Where("student_id IN ?", studentIDs).Find(&rows).Error
	result := map[string][]subscriptionmodel.FeatureOverride{}
	for _, row := range rows {
		result[row.StudentID] = append(result[row.StudentID], row)
	}
	return result, err
}
