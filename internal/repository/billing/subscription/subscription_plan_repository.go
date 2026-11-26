package subscription

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/subscription"
)

// planSelect = kolom 1 paket + nama jenjang, pembuat, pengubah, dan jumlah pemakaiannya.
// Placeholder ? pertama = tanggal hari ini (WIB) untuk menghitung siswa yang paketnya aktif.
// Jumlah pengguna memakai index student_subscriptions_plan_id_idx, jumlah modul memakai
// learning_module_plans_plan_id_idx.
const planSelect = `
	SELECT p.id, p.name, p.description, p.highlights,
	       p.education_level_id AS level_id, l.name AS level_name,
	       p.monthly_coin, p.is_featured,
	       p.price_monthly, p.price_semester, p.price_yearly, p.strike_price_monthly,
	       p.has_module, p.has_daily_quiz, p.has_uji_kom, p.has_tryout, p.has_rumath,
	       p.sort_order, p.status,
	       (SELECT COUNT(*) FROM student_subscriptions ss
	        WHERE ss.plan_id = p.id AND ss.cancelled_at IS NULL
	          AND ?::date BETWEEN ss.start_date AND ss.end_date) AS active_students,
	       (SELECT COUNT(*) FROM learning_module_plans mp
	        JOIN learning_modules m ON m.id = mp.module_id AND m.deleted_at IS NULL
	        WHERE mp.plan_id = p.id) AS module_count,
	       p.created_by AS created_by_id, cu.real_name AS created_by_name,
	       p.updated_by AS updated_by_id, uu.real_name AS updated_by_name,
	       p.created_at, p.updated_at
	FROM subscription_plans p
	LEFT JOIN master_data_items l ON l.id = p.education_level_id
	LEFT JOIN users cu ON cu.id = p.created_by
	LEFT JOIN users uu ON uu.id = p.updated_by`

// planRow menampung 1 baris planSelect (flat, untuk Scan).
type planRow struct {
	ID, Name                                      string
	Description, Highlights                       *string
	LevelID, LevelName                            *string
	MonthlyCoin                                   int
	IsFeatured                                    bool
	PriceMonthly, PriceSemester, PriceYearly      *int64
	StrikePriceMonthly                            *int64
	HasModule, HasDailyQuiz, HasUjiKom, HasTryout bool
	HasRumath                                     bool
	SortOrder                                     int
	Status                                        string
	ActiveStudents, ModuleCount                   int
	CreatedByID, CreatedByName                    *string
	UpdatedByID, UpdatedByName                    *string
	CreatedAt, UpdatedAt                          time.Time
}

func ref(id, name *string) *subscriptionmodel.Ref {
	if id == nil {
		return nil
	}
	r := &subscriptionmodel.Ref{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return r
}

func (row planRow) toPlan() subscriptionmodel.Plan {
	highlights := []string{}
	if row.Highlights != nil {
		for _, line := range strings.Split(*row.Highlights, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				highlights = append(highlights, line)
			}
		}
	}
	return subscriptionmodel.Plan{
		ID: row.ID, Name: row.Name, Description: row.Description, Highlights: highlights,
		EducationLevel: ref(row.LevelID, row.LevelName),
		MonthlyCoin:    row.MonthlyCoin, IsFeatured: row.IsFeatured,
		PriceMonthly: row.PriceMonthly, PriceSemester: row.PriceSemester, PriceYearly: row.PriceYearly,
		StrikePriceMonthly: row.StrikePriceMonthly,
		Features: subscriptionmodel.Features{
			Module: row.HasModule, DailyQuiz: row.HasDailyQuiz, UjiKom: row.HasUjiKom,
			Tryout: row.HasTryout, Rumath: row.HasRumath,
		},
		SortOrder: row.SortOrder, Status: row.Status,
		ActiveStudents: row.ActiveStudents, ModuleCount: row.ModuleCount,
		CreatedBy: ref(row.CreatedByID, row.CreatedByName), UpdatedBy: ref(row.UpdatedByID, row.UpdatedByName),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ListPlans mengambil semua paket yang belum dihapus, urut tampil (sort_order, nama). Paket akses
// sedikit (puluhan), jadi tanpa pagination. [Native SQL, index subscription_plans_list_idx]
func (r *SubscriptionRepository) ListPlans(ctx context.Context, f subscriptionmodel.PlanFilter) ([]subscriptionmodel.Plan, error) {
	conds := []string{"p.deleted_at IS NULL"}
	args := []any{f.Today}
	if f.Status != "" {
		conds = append(conds, "p.status = ?")
		args = append(args, f.Status)
	}
	if f.EducationLevelID != "" {
		// Paket untuk jenjang ini + paket semua jenjang (education_level_id NULL).
		conds = append(conds, "(p.education_level_id = ? OR p.education_level_id IS NULL)")
		args = append(args, f.EducationLevelID)
	}
	if f.Search != "" {
		conds = append(conds, "p.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	var rows []planRow
	err := r.db.WithContext(ctx).
		Raw(planSelect+` WHERE `+strings.Join(conds, " AND ")+` ORDER BY p.sort_order, p.name`, args...).
		Scan(&rows).Error
	plans := make([]subscriptionmodel.Plan, 0, len(rows))
	for _, row := range rows {
		plans = append(plans, row.toPlan())
	}
	return plans, err
}

// FindPlan mengambil 1 paket yang belum dihapus. Tidak ada → ErrPlanNotFound. [Native SQL, primary key]
func (r *SubscriptionRepository) FindPlan(ctx context.Context, id, today string) (subscriptionmodel.Plan, error) {
	var rows []planRow
	if err := r.db.WithContext(ctx).Raw(planSelect+` WHERE p.id = ? AND p.deleted_at IS NULL`, today, id).Scan(&rows).Error; err != nil {
		return subscriptionmodel.Plan{}, err
	}
	if len(rows) == 0 {
		return subscriptionmodel.Plan{}, subscriptionmodel.ErrPlanNotFound
	}
	return rows[0].toPlan(), nil
}

// FindEducationLevel mengambil nama jenjang aktif (master data). Tidak ada → "" tanpa error.
func (r *SubscriptionRepository) FindEducationLevel(ctx context.Context, id string) (string, error) {
	var names []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT name FROM master_data_items
		WHERE id = ? AND type = 'education_level' AND status = 'active' AND deleted_at IS NULL`, id).
		Scan(&names).Error
	if err != nil || len(names) == 0 {
		return "", err
	}
	return names[0], nil
}

// CreatePlan menyimpan paket baru, mengembalikan ID-nya. Nama kembar → ErrPlanNameTaken. [Native SQL]
func (r *SubscriptionRepository) CreatePlan(ctx context.Context, p subscriptionmodel.PlanSave, createdBy string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO subscription_plans
			(name, description, highlights, education_level_id, monthly_coin, is_featured,
			 price_monthly, price_semester, price_yearly, strike_price_monthly,
			 has_module, has_daily_quiz, has_uji_kom, has_tryout, has_rumath,
			 sort_order, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		p.Name, p.Description, p.Highlights, p.EducationLevelID, p.MonthlyCoin, p.IsFeatured,
		p.PriceMonthly, p.PriceSemester, p.PriceYearly, p.StrikePriceMonthly,
		p.Features.Module, p.Features.DailyQuiz, p.Features.UjiKom, p.Features.Tryout, p.Features.Rumath,
		p.SortOrder, p.Status, createdBy).
		Scan(&id).Error
	if isUniqueViolation(err) {
		return "", subscriptionmodel.ErrPlanNameTaken
	}
	return id, err
}

// UpdatePlan mengganti isi 1 paket (ganti penuh). Tidak ada → ErrPlanNotFound; nama kembar →
// ErrPlanNameTaken. Perubahan fitur langsung berlaku untuk siswa yang paketnya aktif; harga hanya
// untuk pembelian berikutnya (price_paid tersimpan di paket siswa). [Native SQL, primary key]
func (r *SubscriptionRepository) UpdatePlan(ctx context.Context, p subscriptionmodel.PlanSave, updatedBy string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE subscription_plans SET
			name = ?, description = ?, highlights = ?, education_level_id = ?, monthly_coin = ?, is_featured = ?,
			price_monthly = ?, price_semester = ?, price_yearly = ?, strike_price_monthly = ?,
			has_module = ?, has_daily_quiz = ?, has_uji_kom = ?, has_tryout = ?, has_rumath = ?,
			sort_order = ?, status = ?, updated_by = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`,
		p.Name, p.Description, p.Highlights, p.EducationLevelID, p.MonthlyCoin, p.IsFeatured,
		p.PriceMonthly, p.PriceSemester, p.PriceYearly, p.StrikePriceMonthly,
		p.Features.Module, p.Features.DailyQuiz, p.Features.UjiKom, p.Features.Tryout, p.Features.Rumath,
		p.SortOrder, p.Status, updatedBy, p.ID)
	if isUniqueViolation(result.Error) {
		return subscriptionmodel.ErrPlanNameTaken
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return subscriptionmodel.ErrPlanNotFound
	}
	return nil
}

// CountOtherLevelStudents menghitung siswa yang paketnya (aktif / belum mulai) memakai paket ini
// tetapi jenjangnya bukan levelID. Dipakai sebelum sasaran jenjang paket diubah. [Native SQL]
func (r *SubscriptionRepository) CountOtherLevelStudents(ctx context.Context, planID, levelID, today string) (int, error) {
	var total int
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT ss.student_id)
		FROM student_subscriptions ss
		JOIN users u ON u.id = ss.student_id
		WHERE ss.plan_id = ? AND ss.cancelled_at IS NULL AND ss.end_date >= ?::date
		  AND u.education_level_id IS DISTINCT FROM ?::uuid`, planID, today, levelID).
		Scan(&total).Error
	return total, err
}

// SoftDeletePlans menghapus banyak paket (soft delete) dalam 1 transaksi. Semua atau tidak sama sekali:
//   - missing = ID yang tidak ada / sudah dihapus,
//   - blocked = paket yang masih dipakai siswa (aktif / belum mulai) atau menjadi satu-satunya paket
//     sebuah modul (dihitung bersama paket lain yang ikut dihapus).
//
// Kalau lolos: hubungan paket ke modul (learning_module_plans) dihapus, lalu paket ditandai terhapus.
// Riwayat paket siswa tetap utuh (baris student_subscriptions tidak diubah). [Native SQL, 4 query]
func (r *SubscriptionRepository) SoftDeletePlans(ctx context.Context, ids []string, deletedBy, today string) (missing []string, blocked []subscriptionmodel.PlanUsage, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, nil, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var found []string
	if err := tx.Raw(`SELECT id FROM subscription_plans WHERE id IN ? AND deleted_at IS NULL FOR UPDATE`, ids).Scan(&found).Error; err != nil {
		return nil, nil, err
	}
	exists := make(map[string]bool, len(found))
	for _, id := range found {
		exists[id] = true
	}
	for _, id := range ids {
		if !exists[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return missing, nil, nil
	}

	err = tx.Raw(`
		SELECT p.id AS plan_id,
		       (SELECT COUNT(DISTINCT ss.student_id) FROM student_subscriptions ss
		        WHERE ss.plan_id = p.id AND ss.cancelled_at IS NULL AND ss.end_date >= ?::date) AS current_students,
		       (SELECT COUNT(*) FROM learning_module_plans mp
		        JOIN learning_modules m ON m.id = mp.module_id AND m.deleted_at IS NULL
		        WHERE mp.plan_id = p.id
		          AND NOT EXISTS (SELECT 1 FROM learning_module_plans o
		                          WHERE o.module_id = mp.module_id AND o.plan_id NOT IN ?)) AS only_plan_modules
		FROM subscription_plans p
		WHERE p.id IN ?`, today, ids, ids).
		Scan(&blocked).Error
	if err != nil {
		return nil, nil, err
	}
	stillUsed := blocked[:0]
	for _, u := range blocked {
		if u.CurrentStudents > 0 || u.OnlyPlanModules > 0 {
			stillUsed = append(stillUsed, u)
		}
	}
	if len(stillUsed) > 0 {
		return nil, stillUsed, nil
	}

	if err := tx.Exec(`DELETE FROM learning_module_plans WHERE plan_id IN ?`, ids).Error; err != nil {
		return nil, nil, err
	}
	err = tx.Exec(`UPDATE subscription_plans SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW() WHERE id IN ?`, deletedBy, ids).Error
	if err != nil {
		return nil, nil, err
	}
	return nil, nil, tx.Commit().Error
}

// isUniqueViolation = true kalau database menolak karena data unik kembar (kode PostgreSQL 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isExclusionViolation = true kalau database menolak karena tanggal paket siswa bertumpuk
// (constraint student_subscriptions_no_overlap, kode PostgreSQL 23P01).
func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}
