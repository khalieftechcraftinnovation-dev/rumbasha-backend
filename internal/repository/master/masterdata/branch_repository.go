package masterdata

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/masterdata"
)

// branchSelect = kolom 1 cabang + jumlah akun aktif (index users_branch_id_idx) + pembuat & pengubah.
const branchSelect = `
	SELECT b.id, b.name, b.status,
	       (SELECT COUNT(*) FROM users u WHERE u.branch_id = b.id AND u.deleted_at IS NULL)::int AS users,
	       b.created_by AS created_by_id, cu.real_name AS created_by_name,
	       b.updated_by AS updated_by_id, uu.real_name AS updated_by_name,
	       b.created_at, b.updated_at
	FROM branches b
	LEFT JOIN users cu ON cu.id = b.created_by
	LEFT JOIN users uu ON uu.id = b.updated_by`

type branchRow struct {
	ID, Name, Status           string
	Users                      int
	CreatedByID, CreatedByName *string
	UpdatedByID, UpdatedByName *string
	CreatedAt, UpdatedAt       time.Time
}

func (row branchRow) toBranch() masterdatamodel.Branch {
	return masterdatamodel.Branch{
		ID: row.ID, Name: row.Name, Status: row.Status, Users: row.Users,
		CreatedBy: ref(row.CreatedByID, row.CreatedByName), UpdatedBy: ref(row.UpdatedByID, row.UpdatedByName),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// ListBranches mengambil semua cabang yang belum dihapus, urut nama. Cabang sedikit, tanpa
// pagination. [Native SQL]
func (r *MasterDataRepository) ListBranches(ctx context.Context, f masterdatamodel.BranchFilter) ([]masterdatamodel.Branch, error) {
	conds := []string{"b.deleted_at IS NULL"}
	args := []any{}
	if f.Status != "" {
		conds = append(conds, "b.status = ?")
		args = append(args, f.Status)
	}
	if f.Search != "" {
		conds = append(conds, "b.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	var rows []branchRow
	err := r.db.WithContext(ctx).Raw(branchSelect+` WHERE `+strings.Join(conds, " AND ")+` ORDER BY b.name`, args...).Scan(&rows).Error
	items := make([]masterdatamodel.Branch, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toBranch())
	}
	return items, err
}

// CountBranches menghitung cabang (angka tab Cabang) dengan filter yang sama seperti list.
func (r *MasterDataRepository) CountBranches(ctx context.Context, f masterdatamodel.BranchFilter) (int, error) {
	conds := []string{"deleted_at IS NULL"}
	args := []any{}
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, f.Status)
	}
	if f.Search != "" {
		conds = append(conds, "name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	var total int
	err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM branches WHERE `+strings.Join(conds, " AND "), args...).Scan(&total).Error
	return total, err
}

// FindBranch mengambil 1 cabang yang belum dihapus. Tidak ada → ErrBranchNotFound. [Native SQL]
func (r *MasterDataRepository) FindBranch(ctx context.Context, id string) (masterdatamodel.Branch, error) {
	var rows []branchRow
	if err := r.db.WithContext(ctx).Raw(branchSelect+` WHERE b.id = ? AND b.deleted_at IS NULL`, id).Scan(&rows).Error; err != nil {
		return masterdatamodel.Branch{}, err
	}
	if len(rows) == 0 {
		return masterdatamodel.Branch{}, masterdatamodel.ErrBranchNotFound
	}
	return rows[0].toBranch(), nil
}

// CreateBranch menyimpan cabang baru. Nama kembar (tanpa peduli huruf besar/kecil) → ErrNameTaken.
func (r *MasterDataRepository) CreateBranch(ctx context.Context, name, status, createdBy string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Raw(`INSERT INTO branches (name, status, created_by) VALUES (?, ?, ?) RETURNING id`, name, status, createdBy).
		Scan(&id).Error
	return id, uniqueError(err)
}

// UpdateBranch mengganti nama & status 1 cabang. Tidak ada → ErrBranchNotFound.
func (r *MasterDataRepository) UpdateBranch(ctx context.Context, id, name, status, updatedBy string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE branches SET name = ?, status = ?, updated_by = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`, name, status, updatedBy, id)
	if result.Error != nil {
		return uniqueError(result.Error)
	}
	if result.RowsAffected == 0 {
		return masterdatamodel.ErrBranchNotFound
	}
	return nil
}

// SoftDeleteBranches menghapus banyak cabang (soft delete) dalam 1 transaksi, semua atau tidak sama
// sekali. missing = ID tidak ada; used = cabang → jumlah akun yang masih memakainya. [Native SQL]
func (r *MasterDataRepository) SoftDeleteBranches(ctx context.Context, ids []string, deletedBy string) (missing []string, used map[string]int, err error) {
	used = map[string]int{}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found []struct {
			ID    string
			Users int
		}
		if err := tx.Raw(`
			SELECT b.id, (SELECT COUNT(*) FROM users u WHERE u.branch_id = b.id AND u.deleted_at IS NULL)::int AS users
			FROM branches b WHERE b.id IN ? AND b.deleted_at IS NULL FOR UPDATE`, ids).Scan(&found).Error; err != nil {
			return err
		}
		exists := map[string]bool{}
		for _, f := range found {
			exists[f.ID] = true
			if f.Users > 0 {
				used[f.ID] = f.Users
			}
		}
		for _, id := range ids {
			if !exists[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 || len(used) > 0 {
			return nil
		}
		return tx.Exec(`UPDATE branches SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW() WHERE id IN ?`, deletedBy, ids).Error
	})
	return missing, used, err
}
