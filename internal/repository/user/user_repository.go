// Package user (repository) mengakses tabel users (+ students, branches, master_data_items) untuk
// menu User di panel admin.
//
// Di package lain, import dengan alias userrepo:
//
//	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user"
package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
)

// UserRepository mengakses data user untuk menu User.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat UserRepository. Dipanggil sekali di main.go.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// listWhere menyusun kondisi WHERE list user. withRole = false untuk jumlah per role (tab).
// Hanya filter yang diisi yang dipakai; nilai selalu lewat placeholder ?.
func listWhere(f usermodel.ListFilter, withRole bool) (string, []any) {
	conds := []string{"u.deleted_at IS NULL"}
	args := []any{}
	equals := []struct{ column, value string }{
		{"u.education_level_id::text", f.EducationLevelID},
		{"u.grade_id::text", f.GradeID},
		{"u.branch_id::text", f.BranchID},
		{"u.status", f.Status},
	}
	if withRole {
		equals = append(equals, struct{ column, value string }{"u.role", f.Role})
	}
	for _, e := range equals {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		conds = append(conds, "(u.real_name ILIKE ? OR u.username ILIKE ? OR u.email ILIKE ?)")
		args = append(args, like, like, like)
	}

	// Filter paket akses: EXISTS ke paket siswa yang aktif hari ini
	// (index student_subscriptions_student_id_end_date_idx), hanya akun siswa.
	activeSub := `SELECT 1 FROM student_subscriptions ss
		WHERE ss.student_id = u.id AND ss.cancelled_at IS NULL AND ?::date BETWEEN ss.start_date AND ss.end_date`
	if f.PlanID != "" {
		conds = append(conds, "u.role = 'student' AND EXISTS ("+activeSub+" AND ss.plan_id = ?)")
		args = append(args, f.Today, f.PlanID)
	}
	switch f.Subscription {
	case usermodel.SubscriptionActive:
		conds = append(conds, "u.role = 'student' AND EXISTS ("+activeSub+")")
		args = append(args, f.Today)
	case usermodel.SubscriptionExpiring:
		conds = append(conds, "u.role = 'student' AND EXISTS ("+activeSub+" AND ss.end_date <= ?::date)")
		args = append(args, f.Today, f.ExpiringUntil)
	case usermodel.SubscriptionNone:
		conds = append(conds, "u.role = 'student' AND NOT EXISTS ("+activeSub+")")
		args = append(args, f.Today)
	}
	return strings.Join(conds, " AND "), args
}

// userRow menampung 1 baris user + nama cabang/jenjang/kelas + data siswa (flat, untuk Scan).
type userRow struct {
	ID, Username, RealName string
	Email, Phone           *string
	Role                   string
	BranchID, BranchName   *string
	LevelID, LevelName     *string
	GradeID, GradeName     *string
	StudentCode            *string
	CoinBalance            *int
	Status                 string
	JoinDate               time.Time
	EndDate                *time.Time
	LastLoginAt            *time.Time
	CreatedAt              time.Time

	// Hanya untuk detail.
	ParentPhone, SchoolOrigin  *string
	DateOfBirth                *time.Time
	TargetSchool               *string
	UpdatedByID, UpdatedByName *string
	UpdatedAt                  time.Time
}

func ref(id, name *string) *usermodel.Ref {
	if id == nil || name == nil {
		return nil
	}
	return &usermodel.Ref{ID: *id, Name: *name}
}

func (row userRow) toListItem() usermodel.ListItem {
	return usermodel.ListItem{
		ID: row.ID, Username: row.Username, RealName: row.RealName, Email: row.Email, Phone: row.Phone,
		Role: row.Role, Branch: ref(row.BranchID, row.BranchName),
		EducationLevel: ref(row.LevelID, row.LevelName), Grade: ref(row.GradeID, row.GradeName),
		StudentCode: row.StudentCode, CoinBalance: row.CoinBalance,
		Status: row.Status, JoinDate: row.JoinDate, EndDate: row.EndDate, LastLoginAt: row.LastLoginAt, CreatedAt: row.CreatedAt,
	}
}

// userColumns & userJoins = kolom & JOIN 1 user lengkap (alias u). Semua JOIN lewat primary key.
const userColumns = `
	u.id, u.username, u.real_name, u.email, u.phone, u.role,
	u.branch_id, b.name AS branch_name,
	u.education_level_id AS level_id, l.name AS level_name,
	u.grade_id, g.name AS grade_name,
	s.student_code, s.coin_balance, u.status, u.join_date, u.end_date, u.last_login_at, u.created_at`

const userJoins = `
	LEFT JOIN branches          b ON b.id = u.branch_id
	LEFT JOIN master_data_items l ON l.id = u.education_level_id
	LEFT JOIN master_data_items g ON g.id = u.grade_id
	LEFT JOIN students          s ON s.user_id = u.id`

// List mengambil 1 halaman user (terbaru dulu) + total. [Native SQL]
//
// Halaman diambil dulu (CTE, index users_list_idx / index filter), baru di-JOIN ke cabang, jenjang,
// kelas, dan data siswa, supaya JOIN hanya untuk ≤ per_page baris. Cari nama/username/email memakai
// index trigram. COUNT dilewati di halaman terakhir yang tidak penuh.
func (r *UserRepository) List(ctx context.Context, f usermodel.ListFilter) ([]usermodel.ListItem, int, error) {
	where, args := listWhere(f, true)

	var rows []userRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT u.id FROM users u
			WHERE `+where+`
			ORDER BY u.created_at DESC, u.id DESC
			LIMIT ? OFFSET ?
		)
		SELECT `+userColumns+`
		FROM page p
		JOIN users u ON u.id = p.id`+userJoins+`
		ORDER BY u.created_at DESC, u.id DESC`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]usermodel.ListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toListItem())
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM users u WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// CountByRole menghitung user per role dengan filter yang sama seperti list (kecuali role), untuk
// angka di tab. 1 query GROUP BY. [Native SQL]
func (r *UserRepository) CountByRole(ctx context.Context, f usermodel.ListFilter) (usermodel.RoleCounts, error) {
	where, args := listWhere(f, false)
	var rows []struct {
		Role  string
		Total int
	}
	err := r.db.WithContext(ctx).Raw(`SELECT u.role, COUNT(*) AS total FROM users u WHERE `+where+` GROUP BY u.role`, args...).
		Scan(&rows).Error
	var c usermodel.RoleCounts
	for _, row := range rows {
		switch row.Role {
		case "super_admin":
			c.SuperAdmin = row.Total
		case "admin":
			c.Admin = row.Total
		case "tutor":
			c.Tutor = row.Total
		case "student":
			c.Student = row.Total
		}
	}
	return c, err
}

// FindDetail mengambil 1 user lengkap (belum dihapus). Tidak ada → ErrUserNotFound. [Native SQL, primary key]
func (r *UserRepository) FindDetail(ctx context.Context, id string) (usermodel.Detail, error) {
	var rows []userRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT `+userColumns+`,
		       u.parent_phone, u.school_origin, s.date_of_birth, s.target_school,
		       u.updated_by AS updated_by_id, uu.real_name AS updated_by_name, u.updated_at
		FROM users u`+userJoins+`
		LEFT JOIN users uu ON uu.id = u.updated_by
		WHERE u.id = ? AND u.deleted_at IS NULL`, id).
		Scan(&rows).Error
	if err != nil {
		return usermodel.Detail{}, err
	}
	if len(rows) == 0 {
		return usermodel.Detail{}, usermodel.ErrUserNotFound
	}
	row := rows[0]
	return usermodel.Detail{
		ListItem: row.toListItem(), ParentPhone: row.ParentPhone, SchoolOrigin: row.SchoolOrigin,
		DateOfBirth: row.DateOfBirth, TargetSchool: row.TargetSchool,
		UpdatedBy: ref(row.UpdatedByID, row.UpdatedByName), UpdatedAt: row.UpdatedAt,
	}, nil
}

// FindTargets mengambil role & nama user yang akan diubah / dihapus (belum dihapus), 1 query.
// Hasilnya map id → user; ID yang tidak ada di map berarti tidak ada / sudah dihapus.
func (r *UserRepository) FindTargets(ctx context.Context, ids []string) (map[string]usermodel.Target, error) {
	var rows []usermodel.Target
	err := r.db.WithContext(ctx).Raw(`SELECT id, role, real_name FROM users WHERE id IN ? AND deleted_at IS NULL`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]usermodel.Target, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

// FindMasterData mengambil jenjang / kelas yang dipilih (belum dihapus). Hasilnya map id → item.
func (r *UserRepository) FindMasterData(ctx context.Context, ids []string) (map[string]usermodel.MasterDataItem, error) {
	var rows []usermodel.MasterDataItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, type, name, parent_id, status FROM master_data_items
		WHERE id IN ? AND deleted_at IS NULL`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]usermodel.MasterDataItem, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

// Update mengganti data 1 user (+ data siswa) dalam 1 transaksi. revokeSessions = true → semua sesi
// login user itu diakhiri (refresh token dicabut), contoh saat dinonaktifkan atau password diganti.
// Username kembar → ErrUsernameTaken. [Native SQL, 2-3 query]
func (r *UserRepository) Update(ctx context.Context, u usermodel.Update, isStudent, revokeSessions bool, updatedBy string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			UPDATE users SET
				username = ?, real_name = ?, email = ?, phone = ?, parent_phone = ?, school_origin = ?,
				branch_id = ?, education_level_id = ?, grade_id = ?, status = ?,
				join_date = ?, end_date = ?,
				password_hash = COALESCE(?, password_hash),
				updated_by = ?, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL`,
			u.Username, u.RealName, u.Email, u.Phone, u.ParentPhone, u.SchoolOrigin,
			u.BranchID, u.EducationLevelID, u.GradeID, u.Status, u.JoinDate, u.EndDate, u.PasswordHash,
			updatedBy, now, u.ID)
		if isUniqueViolation(result.Error, "users_username_unique") {
			return usermodel.ErrUsernameTaken
		}
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return usermodel.ErrUserNotFound
		}

		if isStudent {
			err := tx.Exec(`UPDATE students SET date_of_birth = ?, target_school = ?, updated_at = ? WHERE user_id = ?`,
				u.DateOfBirth, u.TargetSchool, now, u.ID).Error
			if err != nil {
				return err
			}
		}
		if revokeSessions {
			return revokeSessionsOf(tx, []string{u.ID}, now)
		}
		return nil
	})
}

// SoftDelete menghapus banyak user (deleted_at & deleted_by diisi) dan mengakhiri semua sesi login
// mereka, dalam 1 transaksi. Username tetap terpakai (unik selamanya), riwayat tetap ada. [Native SQL, 2 query]
func (r *UserRepository) SoftDelete(ctx context.Context, ids []string, deletedBy string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`UPDATE users SET deleted_at = ?, deleted_by = ?, updated_at = ? WHERE id IN ? AND deleted_at IS NULL`,
			now, deletedBy, now, ids)
		if result.Error != nil {
			return result.Error
		}
		if int(result.RowsAffected) != len(ids) {
			return usermodel.ErrUserNotFound // ada yang dihapus request lain di saat bersamaan
		}
		return revokeSessionsOf(tx, ids, now)
	})
}

// revokeSessionsOf mencabut semua refresh token aktif milik user (sesi berakhir; access token yang
// sudah terbit tetap berlaku sampai habis, maks JWT_ACCESS_TTL).
func revokeSessionsOf(tx *gorm.DB, userIDs []string, now time.Time) error {
	return tx.Exec(`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id IN ? AND revoked_at IS NULL`, now, userIDs).Error
}

// isUniqueViolation = true kalau database menolak karena data unik kembar (kode PostgreSQL 23505)
// pada constraint tertentu.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
