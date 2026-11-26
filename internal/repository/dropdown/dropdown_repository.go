// Package dropdown (repository) membaca pilihan dropdown dari tabel master_data_items.
//
// Di package lain, import dengan alias dropdownrepo:
//
//	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dropdown"
package dropdown

import (
	"context"
	"strings"

	"gorm.io/gorm"

	dropdownmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dropdown"
)

// DropdownRepository mengakses tabel master_data_items untuk dropdown.
type DropdownRepository struct {
	db *gorm.DB
}

// NewDropdownRepository membuat DropdownRepository. Dipanggil sekali di main.go.
func NewDropdownRepository(db *gorm.DB) *DropdownRepository {
	return &DropdownRepository{db: db}
}

// FindOptions mengambil item aktif 1 type, urut sort_order lalu nama. [Native SQL, 1 tabel]
// Memakai index master_data_items_dropdown_idx (type, parent_id, sort_order). Contoh:
//
//	SELECT id, code, name, parent_id FROM master_data_items
//	WHERE type = 'chapter' AND deleted_at IS NULL AND status = 'active' AND parent_id = ?
//	ORDER BY sort_order, name
func (r *DropdownRepository) FindOptions(ctx context.Context, f dropdownmodel.Filter) ([]dropdownmodel.Option, error) {
	conds := []string{"type = ?", "deleted_at IS NULL", "status = 'active'"}
	args := []any{f.Type}

	if f.Root {
		conds = append(conds, "parent_id IS NULL")
	}
	if f.ParentID != "" {
		conds = append(conds, "parent_id = ?")
		args = append(args, f.ParentID)
	}
	if f.GradeID != "" {
		conds = append(conds, "grade_id = ?")
		args = append(args, f.GradeID)
	}

	options := []dropdownmodel.Option{}
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, code, name, parent_id FROM master_data_items
			WHERE `+strings.Join(conds, " AND ")+`
			ORDER BY sort_order, name`, args...).
		Scan(&options).Error
	return options, err
}

// FindPlanOptions mengambil paket akses aktif (subscription_plans, belum dihapus), urut sort_order
// lalu nama. parent_id = jenjang sasaran paket (NULL = semua jenjang). [Native SQL, 1 tabel]
func (r *DropdownRepository) FindPlanOptions(ctx context.Context) ([]dropdownmodel.Option, error) {
	options := []dropdownmodel.Option{}
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, NULL AS code, name, education_level_id AS parent_id FROM subscription_plans
			WHERE deleted_at IS NULL AND status = 'active'
			ORDER BY sort_order, name`).
		Scan(&options).Error
	return options, err
}

// FindBranchOptions mengambil cabang aktif (tabel branches, belum dihapus), urut nama. [Native SQL, 1 tabel]
func (r *DropdownRepository) FindBranchOptions(ctx context.Context) ([]dropdownmodel.Option, error) {
	options := []dropdownmodel.Option{}
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, NULL AS code, name, NULL AS parent_id FROM branches
			WHERE deleted_at IS NULL AND status = 'active'
			ORDER BY name`).
		Scan(&options).Error
	return options, err
}
