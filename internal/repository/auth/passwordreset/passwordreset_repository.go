// Package passwordreset (repository) mengakses tabel password_reset_requests (+ users untuk mereset
// kata sandi).
//
// Di package lain, import dengan alias passwordresetrepo:
//
//	passwordresetrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/passwordreset"
package passwordreset

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	passwordresetmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/passwordreset"
)

// PasswordResetRepository mengakses permintaan lupa kata sandi.
type PasswordResetRepository struct {
	db *gorm.DB
}

// NewPasswordResetRepository membuat PasswordResetRepository. Dipanggil sekali di main.go.
func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// phoneDigits = ekspresi SQL no HP tanpa selain angka, awalan 62 → 0 (0812-3456 = +62 812 3456).
const phoneDigits = `regexp_replace(regexp_replace(%s, '\D', '', 'g'), '^62', '0')`

// FindUserIDs mencari akun (belum dihapus) yang cocok dengan isian form: username, email, atau no HP
// (phone akun). Maksimal MaxMatches akun. phone = angka saja (kosong = tidak dicari per no HP).
// [Native SQL; username memakai index unik, email/no HP membaca tabel users]
func (r *PasswordResetRepository) FindUserIDs(ctx context.Context, username, email, phone string) ([]string, error) {
	conds := []string{"u.username = ?", "LOWER(u.email) = ?"}
	args := []any{username, email}
	if phone != "" {
		conds = append(conds, strings.Replace(phoneDigits, "%s", "u.phone", 1)+" = ?")
		args = append(args, phone)
	}
	args = append(args, passwordresetmodel.MaxMatches)
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id FROM users u
		WHERE u.deleted_at IS NULL AND (`+strings.Join(conds, " OR ")+`)
		ORDER BY u.created_at
		LIMIT ?`, args...).Scan(&ids).Error
	return ids, err
}

// UpsertPending mencatat permintaan untuk setiap akun, 1 query. Akun yang sudah punya permintaan
// menunggu → permintaan itu diperbarui (waktu terakhir, jumlah +1, kontak/pesan baru kalau diisi),
// bukan baris baru (index unik password_reset_requests_pending_unique). [Native SQL]
func (r *PasswordResetRepository) UpsertPending(ctx context.Context, userIDs []string, n passwordresetmodel.NewRequest, now time.Time) error {
	values := make([]string, 0, len(userIDs))
	args := make([]any, 0, len(userIDs)*8)
	for _, id := range userIDs {
		values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args, id, n.Identifier, n.Contact, n.Note, n.IPAddress, n.UserAgent, now, now)
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO password_reset_requests
			(user_id, identifier, contact, note, ip_address, user_agent, requested_at, created_at)
		VALUES `+strings.Join(values, ", ")+`
		ON CONFLICT (user_id) WHERE status = 'pending' DO UPDATE SET
			identifier    = EXCLUDED.identifier,
			contact       = COALESCE(EXCLUDED.contact, password_reset_requests.contact),
			note          = COALESCE(EXCLUDED.note, password_reset_requests.note),
			ip_address    = EXCLUDED.ip_address,
			user_agent    = EXCLUDED.user_agent,
			request_count = password_reset_requests.request_count + 1,
			requested_at  = EXCLUDED.requested_at`, args...).Error
}

// requestSelect = kolom 1 permintaan + akun, cabang, dan admin yang memproses.
const requestSelect = `
	SELECT pr.id, pr.identifier, pr.contact, pr.note, pr.request_count, pr.status, pr.ip_address,
	       pr.requested_at, pr.resolved_at, pr.resolution_note, pr.created_at,
	       u.id AS user_id, u.username, u.real_name, u.role, u.status AS user_status,
	       u.email, u.phone, u.parent_phone, b.id AS branch_id, b.name AS branch_name,
	       pr.resolved_by AS resolved_by_id, ru.real_name AS resolved_by_name
	FROM password_reset_requests pr
	JOIN users u          ON u.id = pr.user_id
	LEFT JOIN branches b  ON b.id = u.branch_id
	LEFT JOIN users ru    ON ru.id = pr.resolved_by`

type requestRow struct {
	ID, Identifier               string
	Contact, Note                *string
	RequestCount                 int
	Status                       string
	IPAddress                    *string
	RequestedAt                  time.Time
	ResolvedAt                   *time.Time
	ResolutionNote               *string
	CreatedAt                    time.Time
	UserID, Username, RealName   string
	Role, UserStatus             string
	Email, Phone, ParentPhone    *string
	BranchID, BranchName         *string
	ResolvedByID, ResolvedByName *string
}

func ref(id, name *string) *passwordresetmodel.Ref {
	if id == nil {
		return nil
	}
	r := &passwordresetmodel.Ref{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return r
}

func (row requestRow) toRequest() passwordresetmodel.Request {
	return passwordresetmodel.Request{
		ID: row.ID, Identifier: row.Identifier, Contact: row.Contact, Note: row.Note, RequestCount: row.RequestCount,
		Status: row.Status, IPAddress: row.IPAddress, RequestedAt: row.RequestedAt, ResolvedAt: row.ResolvedAt,
		ResolvedBy: ref(row.ResolvedByID, row.ResolvedByName), ResolutionNote: row.ResolutionNote, CreatedAt: row.CreatedAt,
		User: passwordresetmodel.User{
			ID: row.UserID, Username: row.Username, RealName: row.RealName, Role: row.Role, Status: row.UserStatus,
			Email: row.Email, Phone: row.Phone, ParentPhone: row.ParentPhone, Branch: ref(row.BranchID, row.BranchName),
		},
	}
}

// listWhere menyusun WHERE daftar. withStatus = false untuk jumlah per status.
func listWhere(f passwordresetmodel.ListFilter, withStatus bool) (string, []any) {
	conds := []string{"u.deleted_at IS NULL", "u.role IN ?"}
	args := []any{f.Roles}
	if withStatus && f.Status != "" {
		conds = append(conds, "pr.status = ?")
		args = append(args, f.Status)
	}
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		conds = append(conds, "(u.real_name ILIKE ? OR u.username ILIKE ? OR u.email ILIKE ? OR u.phone ILIKE ? OR pr.identifier ILIKE ?)")
		args = append(args, like, like, like, like, like)
	}
	return strings.Join(conds, " AND "), args
}

// List mengambil 1 halaman permintaan (terbaru dulu) + total. Akun yang sudah dihapus tidak tampil.
// [Native SQL, index password_reset_requests_list_idx]
func (r *PasswordResetRepository) List(ctx context.Context, f passwordresetmodel.ListFilter) ([]passwordresetmodel.Request, int, error) {
	where, args := listWhere(f, true)
	var rows []requestRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(requestSelect+` WHERE `+where+`
		ORDER BY pr.requested_at DESC, pr.id DESC
		LIMIT ? OFFSET ?`, pageArgs...).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]passwordresetmodel.Request, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toRequest())
	}
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM password_reset_requests pr JOIN users u ON u.id = pr.user_id WHERE `+where, args...).
		Scan(&total).Error
	return items, total, err
}

// CountByStatus menghitung permintaan per status (angka tab & badge), 1 query GROUP BY.
func (r *PasswordResetRepository) CountByStatus(ctx context.Context, f passwordresetmodel.ListFilter) (passwordresetmodel.Counts, error) {
	where, args := listWhere(f, false)
	var rows []struct {
		Status string
		Total  int
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT pr.status, COUNT(*)::int AS total
		FROM password_reset_requests pr JOIN users u ON u.id = pr.user_id
		WHERE `+where+` GROUP BY pr.status`, args...).Scan(&rows).Error
	var c passwordresetmodel.Counts
	for _, row := range rows {
		switch row.Status {
		case passwordresetmodel.StatusPending:
			c.Pending = row.Total
		case passwordresetmodel.StatusResolved:
			c.Resolved = row.Total
		case passwordresetmodel.StatusRejected:
			c.Rejected = row.Total
		}
	}
	return c, err
}

// Find mengambil 1 permintaan (akun belum dihapus). Tidak ada → ErrNotFound. [Native SQL, primary key]
func (r *PasswordResetRepository) Find(ctx context.Context, id string) (passwordresetmodel.Request, error) {
	var rows []requestRow
	if err := r.db.WithContext(ctx).Raw(requestSelect+` WHERE pr.id = ? AND u.deleted_at IS NULL`, id).Scan(&rows).Error; err != nil {
		return passwordresetmodel.Request{}, err
	}
	if len(rows) == 0 {
		return passwordresetmodel.Request{}, passwordresetmodel.ErrNotFound
	}
	return rows[0].toRequest(), nil
}

// Resolve mereset kata sandi pemilik permintaan dalam 1 transaksi: permintaan dikunci & harus masih
// menunggu (ErrNotPending), kata sandi diganti, SEMUA sesi user dicabut (wajib login dengan kata
// sandi baru), lalu permintaan ditandai selesai. [Native SQL, 4 query]
func (r *PasswordResetRepository) Resolve(ctx context.Context, id, userID, passwordHash string, note *string, adminID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var status string
		if err := tx.Raw(`SELECT status FROM password_reset_requests WHERE id = ? FOR UPDATE`, id).Scan(&status).Error; err != nil {
			return err
		}
		if status != passwordresetmodel.StatusPending {
			return passwordresetmodel.ErrNotPending
		}
		result := tx.Exec(`UPDATE users SET password_hash = ?, updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			passwordHash, adminID, now, userID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return passwordresetmodel.ErrNotFound
		}
		if err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, userID).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE password_reset_requests SET status = 'resolved', resolved_at = ?, resolved_by = ?, resolution_note = ?
			WHERE id = ?`, now, adminID, note, id).Error
	})
}

// Reject menolak permintaan yang masih menunggu (kata sandi tidak berubah). Sudah diproses → ErrNotPending.
func (r *PasswordResetRepository) Reject(ctx context.Context, id string, note *string, adminID string, now time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE password_reset_requests SET status = 'rejected', resolved_at = ?, resolved_by = ?, resolution_note = ?
		WHERE id = ? AND status = 'pending'`, now, adminID, note, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return passwordresetmodel.ErrNotPending
	}
	return nil
}
