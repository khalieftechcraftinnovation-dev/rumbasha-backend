package coin

import (
	"context"
	"time"

	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/coin"
)

// FindStudentCoin mengambil saldo coin 1 siswa (akun role student yang belum dihapus).
// Tidak ada → ErrStudentNotFound. [Native SQL, primary key]
func (r *CoinRepository) FindStudentCoin(ctx context.Context, studentID string) (coinmodel.StudentCoin, error) {
	var rows []coinmodel.StudentCoin
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS student_id, u.real_name AS name, s.student_code, s.coin_balance
		FROM users u
		JOIN students s ON s.user_id = u.id
		WHERE u.id = ? AND u.role = 'student' AND u.deleted_at IS NULL`, studentID).
		Scan(&rows).Error
	if err != nil {
		return coinmodel.StudentCoin{}, err
	}
	if len(rows) == 0 {
		return coinmodel.StudentCoin{}, coinmodel.ErrStudentNotFound
	}
	return rows[0], nil
}

// ListTransactions mengambil 1 halaman riwayat coin 1 siswa, terbaru dulu, + total baris.
// Nama referensi diisi untuk paket coin (nama paket), hadiah quiz (nama quiz), dan paket
// langganan (nama plan); lainnya null. [Native SQL, index coin_transactions_student_id_created_at_idx]
//
// Halaman diambil dulu (CTE), baru di-JOIN ke tabel referensi, supaya JOIN hanya untuk ≤ per_page baris.
func (r *CoinRepository) ListTransactions(ctx context.Context, studentID string, f coinmodel.TransactionFilter) ([]coinmodel.Transaction, int, error) {
	where := `t.student_id = ?`
	args := []any{studentID}
	if f.Type != "" {
		where += ` AND t.type = ?`
		args = append(args, f.Type)
	}

	items := []coinmodel.Transaction{}
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT t.* FROM coin_transactions t
			WHERE `+where+`
			ORDER BY t.created_at DESC, t.id DESC
			LIMIT ? OFFSET ?
		)
		SELECT t.id, t.type, t.amount, t.balance_after, t.price_paid, t.reference_type, t.reference_id,
		       COALESCE(cp.name, z.name, sp.name) AS reference_name,
		       t.description, t.created_by, u.real_name AS created_by_name, t.created_at
		FROM page t
		LEFT JOIN coin_packages         cp ON t.reference_type = 'coin_package' AND cp.id = t.reference_id
		LEFT JOIN quizzes               z  ON t.reference_type = 'quiz_prize'   AND z.id  = t.reference_id
		LEFT JOIN student_subscriptions ss ON t.reference_type = 'student_subscription' AND ss.id = t.reference_id
		LEFT JOIN subscription_plans    sp ON sp.id = ss.plan_id
		LEFT JOIN users                 u  ON u.id = t.created_by
		ORDER BY t.created_at DESC, t.id DESC`, pageArgs...).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	// Halaman terakhir yang tidak penuh: total sudah pasti, COUNT tidak perlu.
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM coin_transactions t WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// FindDailyQuota mengambil jatah Daily Quiz siswa hari ini: pengaturan global (jatah & biaya) +
// jumlah Daily Quiz yang sudah dibuka sejak dayStart (00:00 WIB). [Native SQL, 2 query kecil]
func (r *CoinRepository) FindDailyQuota(ctx context.Context, studentID string, dayStart time.Time) (coinmodel.DailyQuota, error) {
	var q coinmodel.DailyQuota
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.free_per_day, s.coin_cost,
		       (SELECT COUNT(*) FROM daily_quizzes d WHERE d.student_id = ? AND d.opened_at >= ?)::int AS used_today
		FROM daily_quiz_settings s WHERE s.id = 1`, studentID, dayStart).
		Scan(&q).Error
	return q, err
}
