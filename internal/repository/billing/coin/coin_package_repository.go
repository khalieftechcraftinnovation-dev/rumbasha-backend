package coin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/coin"
)

// CoinRepository mengakses coin_packages, coin_transactions, dan saldo students.
type CoinRepository struct {
	db *gorm.DB
}

// NewCoinRepository membuat CoinRepository. Dipanggil sekali di main.go.
func NewCoinRepository(db *gorm.DB) *CoinRepository {
	return &CoinRepository{db: db}
}

// packageSelect = kolom 1 paket + nama pembuat & pengubah terakhir.
const packageSelect = `
	SELECT p.id, p.name, p.coin_amount, p.price, p.description, p.sort_order, p.status,
	       p.created_by, cu.real_name AS created_by_name, p.updated_by, uu.real_name AS updated_by_name,
	       p.created_at, p.updated_at
	FROM coin_packages p
	LEFT JOIN users cu ON cu.id = p.created_by
	LEFT JOIN users uu ON uu.id = p.updated_by`

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ListPackages mengambil semua paket yang belum dihapus, urut tampil (sort_order, lalu jumlah coin).
// Paket coin sedikit (puluhan), jadi tanpa pagination. [Native SQL, index coin_packages_sort_idx]
func (r *CoinRepository) ListPackages(ctx context.Context, f coinmodel.PackageFilter) ([]coinmodel.Package, error) {
	conds := []string{"p.deleted_at IS NULL"}
	args := []any{}
	if f.Status != "" {
		conds = append(conds, "p.status = ?")
		args = append(args, f.Status)
	}
	if f.Search != "" {
		conds = append(conds, "p.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	packages := []coinmodel.Package{}
	err := r.db.WithContext(ctx).
		Raw(packageSelect+` WHERE `+strings.Join(conds, " AND ")+` ORDER BY p.sort_order, p.coin_amount, p.name`, args...).
		Scan(&packages).Error
	return packages, err
}

// FindPackage mengambil 1 paket yang belum dihapus. Tidak ada → ErrPackageNotFound. [Native SQL, primary key]
func (r *CoinRepository) FindPackage(ctx context.Context, id string) (coinmodel.Package, error) {
	var rows []coinmodel.Package
	if err := r.db.WithContext(ctx).Raw(packageSelect+` WHERE p.id = ? AND p.deleted_at IS NULL`, id).Scan(&rows).Error; err != nil {
		return coinmodel.Package{}, err
	}
	if len(rows) == 0 {
		return coinmodel.Package{}, coinmodel.ErrPackageNotFound
	}
	return rows[0], nil
}

// CreatePackage menyimpan paket baru, mengembalikan ID-nya. Nama kembar → ErrPackageNameTaken.
// [Native SQL]
func (r *CoinRepository) CreatePackage(ctx context.Context, p coinmodel.Package, createdBy string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO coin_packages (name, coin_amount, price, description, sort_order, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		p.Name, p.CoinAmount, p.Price, p.Description, p.SortOrder, p.Status, createdBy).
		Scan(&id).Error
	if isUniqueViolation(err) {
		return "", coinmodel.ErrPackageNameTaken
	}
	return id, err
}

// UpdatePackage mengganti isi 1 paket (ganti penuh). Tidak ada → ErrPackageNotFound; nama kembar →
// ErrPackageNameTaken. Riwayat coin lama tidak berubah (price_paid tersimpan di transaksinya).
// [Native SQL, primary key]
func (r *CoinRepository) UpdatePackage(ctx context.Context, p coinmodel.Package, updatedBy string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE coin_packages SET
			name = ?, coin_amount = ?, price = ?, description = ?, sort_order = ?, status = ?,
			updated_by = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`,
		p.Name, p.CoinAmount, p.Price, p.Description, p.SortOrder, p.Status, updatedBy, p.ID)
	if isUniqueViolation(result.Error) {
		return coinmodel.ErrPackageNameTaken
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return coinmodel.ErrPackageNotFound
	}
	return nil
}

// SoftDeletePackages menghapus banyak paket (soft delete) dalam 1 transaksi: semua ID harus ada
// (belum dihapus), kalau tidak → tidak ada yang terhapus dan missing berisi ID yang tidak ada.
// Riwayat pembelian dengan paket ini tetap ada. [Native SQL, 2 query]
func (r *CoinRepository) SoftDeletePackages(ctx context.Context, ids []string, deletedBy string) (missing []string, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var found []string
	if err := tx.Raw(`SELECT id FROM coin_packages WHERE id IN ? AND deleted_at IS NULL FOR UPDATE`, ids).Scan(&found).Error; err != nil {
		return nil, err
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
		return missing, nil
	}

	err = tx.Exec(`UPDATE coin_packages SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW() WHERE id IN ?`, deletedBy, ids).Error
	if err != nil {
		return nil, err
	}
	return nil, tx.Commit().Error
}

// FindStudents mengambil siswa (akun role student yang belum dihapus) berdasarkan ID, 1 query.
// Hasilnya map id → siswa; ID yang tidak ada di map berarti bukan siswa / tidak ada.
// [Native SQL, primary key]
func (r *CoinRepository) FindStudents(ctx context.Context, ids []string) (map[string]coinmodel.Student, error) {
	var rows []coinmodel.Student
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.real_name AS name, s.student_code, u.status, s.coin_balance
		FROM users u
		JOIN students s ON s.user_id = u.id
		WHERE u.id IN ? AND u.role = 'student' AND u.deleted_at IS NULL`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]coinmodel.Student, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

// TopUp menambah coin banyak siswa sekaligus sesuai 1 paket, dalam 1 transaksi (semua atau tidak
// sama sekali). Dicatat sebagai pembelian tunai oleh admin: coin_transactions type 'purchase',
// reference coin_package, price_paid = harga paket, created_by = admin. [Native SQL, 1 query]
//
//	WITH upd AS (UPDATE students ... RETURNING user_id, coin_balance)
//	INSERT INTO coin_transactions (...) SELECT ... FROM upd RETURNING ...
//
// requestID (opsional) → idempotency_key "coin_topup:<requestID>:<siswa>": request yang sama dikirim
// ulang (klik 2x) ditolak ErrTopUpDuplicate, coin tidak bertambah 2x.
// Jumlah baris yang ter-update harus sama dengan jumlah siswa; kalau tidak (siswa dihapus di
// tengah jalan) → ErrStudentNotFound dan semua dibatalkan.
func (r *CoinRepository) TopUp(ctx context.Context, p coinmodel.Package, studentIDs []string, requestID *string, description, adminID string, now time.Time) ([]coinmodel.TopUpResult, error) {
	var results []coinmodel.TopUpResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`
			WITH upd AS (
				UPDATE students s SET coin_balance = s.coin_balance + ?, updated_at = NOW()
				FROM users u
				WHERE s.user_id = u.id AND u.id IN ? AND u.role = 'student' AND u.deleted_at IS NULL
				RETURNING s.user_id, s.coin_balance, s.student_code, u.real_name
			),
			ins AS (
				INSERT INTO coin_transactions
					(student_id, type, amount, balance_after, price_paid, reference_type, reference_id,
					 idempotency_key, description, created_by, created_at)
				SELECT upd.user_id, 'purchase', ?, upd.coin_balance, ?, 'coin_package', ?,
				       CASE WHEN ?::text IS NULL THEN NULL ELSE 'coin_topup:' || ?::text || ':' || upd.user_id END,
				       ?, ?, ?
				FROM upd
				RETURNING id, student_id, balance_after
			)
			SELECT ins.student_id, upd.real_name AS student_name, upd.student_code,
			       ?::int AS coin_added, ins.balance_after, ins.id AS transaction_id
			FROM ins JOIN upd ON upd.user_id = ins.student_id`,
			p.CoinAmount, studentIDs,
			p.CoinAmount, p.Price, p.ID, requestID, requestID, description, adminID, now,
			p.CoinAmount).
			Scan(&results).Error
		if isUniqueViolation(err) {
			return coinmodel.ErrTopUpDuplicate
		}
		if err != nil {
			return err
		}
		if len(results) != len(studentIDs) {
			return coinmodel.ErrStudentNotFound
		}
		return nil
	})
	return results, err
}

// isUniqueViolation = true kalau database menolak karena data unik kembar (kode PostgreSQL 23505),
// contoh nama paket kembar atau idempotency_key top-up yang sama.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
