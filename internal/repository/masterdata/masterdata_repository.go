// Package masterdata (repository) mengakses tabel master_data_items dan branches untuk menu Master
// Data di panel admin.
//
// Di package lain, import dengan alias masterdatarepo:
//
//	masterdatarepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/masterdata"
package masterdata

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/masterdata"
)

// MasterDataRepository mengakses data master & cabang.
type MasterDataRepository struct {
	db *gorm.DB
}

// NewMasterDataRepository membuat MasterDataRepository. Dipanggil sekali di main.go.
func NewMasterDataRepository(db *gorm.DB) *MasterDataRepository {
	return &MasterDataRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// itemSelect = kolom 1 item + nama induk, letak (mapel, jenjang, kelas), pembuat & pengubah.
// Semua JOIN lewat primary key.
const itemSelect = `
	SELECT x.id, x.type, x.code, x.name,
	       x.parent_id, p.name AS parent_name,
	       x.subject_id, s.name AS subject_name,
	       x.education_level_id AS level_id, l.name AS level_name,
	       x.grade_id, g.name AS grade_name,
	       x.sort_order, x.status, x.is_system,
	       x.created_by AS created_by_id, cu.real_name AS created_by_name,
	       x.updated_by AS updated_by_id, uu.real_name AS updated_by_name,
	       x.created_at, x.updated_at
	FROM master_data_items x
	LEFT JOIN master_data_items p ON p.id = x.parent_id
	LEFT JOIN master_data_items s ON s.id = x.subject_id
	LEFT JOIN master_data_items l ON l.id = x.education_level_id
	LEFT JOIN master_data_items g ON g.id = x.grade_id
	LEFT JOIN users cu ON cu.id = x.created_by
	LEFT JOIN users uu ON uu.id = x.updated_by`

// itemRow menampung 1 baris itemSelect (flat, untuk Scan).
type itemRow struct {
	ID, Type                   string
	Code                       *string
	Name                       string
	ParentID, ParentName       *string
	SubjectID, SubjectName     *string
	LevelID, LevelName         *string
	GradeID, GradeName         *string
	SortOrder                  int
	Status                     string
	IsSystem                   bool
	CreatedByID, CreatedByName *string
	UpdatedByID, UpdatedByName *string
	CreatedAt, UpdatedAt       time.Time
}

func ref(id, name *string) *masterdatamodel.Ref {
	if id == nil {
		return nil
	}
	r := &masterdatamodel.Ref{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return r
}

func (row itemRow) toItem() masterdatamodel.Item {
	item := masterdatamodel.Item{
		ID: row.ID, Type: row.Type, Code: row.Code, Name: row.Name,
		Parent:  ref(row.ParentID, row.ParentName),
		Subject: ref(row.SubjectID, row.SubjectName), EducationLevel: ref(row.LevelID, row.LevelName),
		Grade:     ref(row.GradeID, row.GradeName),
		SortOrder: row.SortOrder, Status: row.Status, IsSystem: row.IsSystem,
		CreatedBy: ref(row.CreatedByID, row.CreatedByName), UpdatedBy: ref(row.UpdatedByID, row.UpdatedByName),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.Type == masterdatamodel.TypeGrade {
		item.EducationLevel = item.Parent // jenjang kelas = induknya
	}
	return item
}

// typeRankSQL = CASE urutan type di tab "Semua" (mapel, bidang materi, ..., sekolah, lalu type sistem).
func typeRankSQL() string {
	types := make([]string, 0, len(masterdatamodel.Types))
	for t := range masterdatamodel.Types {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return masterdatamodel.Types[types[i]].Rank < masterdatamodel.Types[types[j]].Rank
	})
	var b strings.Builder
	b.WriteString("CASE x.type")
	for _, t := range types {
		fmt.Fprintf(&b, " WHEN '%s' THEN %d", t, masterdatamodel.Types[t].Rank)
	}
	b.WriteString(" END")
	return b.String()
}

// listWhere menyusun WHERE list / jumlah per tab. withTypes = false untuk jumlah per tab (type diabaikan).
func listWhere(f masterdatamodel.ListFilter, withTypes bool) (string, []any) {
	conds := []string{"x.deleted_at IS NULL"}
	args := []any{}
	if withTypes {
		conds = append(conds, "x.type IN ?")
		args = append(args, f.Types)
	}
	for _, e := range []struct{ column, value string }{
		{"x.status", f.Status},
		{"x.parent_id::text", f.ParentID},
		{"x.subject_id::text", f.SubjectID},
		{"x.grade_id::text", f.GradeID},
	} {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.EducationLevelID != "" {
		// Bidang materi s/d topik: kolom education_level_id; kelas: induknya.
		conds = append(conds, "(x.education_level_id = ? OR (x.type = 'grade' AND x.parent_id = ?))")
		args = append(args, f.EducationLevelID, f.EducationLevelID)
	}
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		conds = append(conds, "(x.name ILIKE ? OR x.code ILIKE ?)")
		args = append(args, like, like)
	}
	return strings.Join(conds, " AND "), args
}

// List mengambil 1 halaman data master + total. Urut type (tab), kelas, urutan, nama. [Native SQL]
//
// Halaman diambil dulu (CTE), baru di-JOIN ke induk/letak/pembuat untuk ≤ per_page baris.
// Cari nama memakai index trigram master_data_items_name_trgm_idx. COUNT dilewati di halaman
// terakhir yang tidak penuh. Pemakaian (kolom Isi) dihitung terpisah lewat Usage.
func (r *MasterDataRepository) List(ctx context.Context, f masterdatamodel.ListFilter) ([]masterdatamodel.Item, int, error) {
	where, args := listWhere(f, true)
	order := typeRankSQL() + ", g.sort_order NULLS FIRST, x.sort_order, x.name, x.id"

	var rows []itemRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT x.id, ROW_NUMBER() OVER (ORDER BY `+order+`) AS n
			FROM master_data_items x
			LEFT JOIN master_data_items g ON g.id = x.grade_id
			WHERE `+where+`
			ORDER BY `+order+`
			LIMIT ? OFFSET ?
		)`+strings.Replace(itemSelect, "FROM master_data_items x", "FROM page pg JOIN master_data_items x ON x.id = pg.id", 1)+`
		ORDER BY pg.n`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]masterdatamodel.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toItem())
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM master_data_items x WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// CountByType menghitung data per type (angka tab) dengan filter yang sama seperti list (type
// diabaikan), 1 query GROUP BY. [Native SQL]
func (r *MasterDataRepository) CountByType(ctx context.Context, f masterdatamodel.ListFilter) (masterdatamodel.Counts, error) {
	where, args := listWhere(f, false)
	var rows []struct {
		Type  string
		Total int
	}
	err := r.db.WithContext(ctx).Raw(`SELECT x.type, COUNT(*)::int AS total FROM master_data_items x WHERE `+where+` GROUP BY x.type`, args...).
		Scan(&rows).Error
	counts := masterdatamodel.Counts{}
	for _, row := range rows {
		counts[row.Type] = row.Total
	}
	return counts, err
}

// FindItem mengambil 1 data master yang belum dihapus. Tidak ada → ErrNotFound. [Native SQL, primary key]
func (r *MasterDataRepository) FindItem(ctx context.Context, id string) (masterdatamodel.Item, error) {
	var rows []itemRow
	if err := r.db.WithContext(ctx).Raw(itemSelect+` WHERE x.id = ? AND x.deleted_at IS NULL`, id).Scan(&rows).Error; err != nil {
		return masterdatamodel.Item{}, err
	}
	if len(rows) == 0 {
		return masterdatamodel.Item{}, masterdatamodel.ErrNotFound
	}
	return rows[0].toItem(), nil
}

// FindParent mengambil item induk (belum dihapus). Tidak ada → ErrNotFound. [Native SQL, primary key]
func (r *MasterDataRepository) FindParent(ctx context.Context, id string) (masterdatamodel.Parent, error) {
	var rows []masterdatamodel.Parent
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, type, name, status, parent_id, subject_id, education_level_id, grade_id
		FROM master_data_items WHERE id = ? AND deleted_at IS NULL`, id).Scan(&rows).Error
	if err != nil {
		return masterdatamodel.Parent{}, err
	}
	if len(rows) == 0 {
		return masterdatamodel.Parent{}, masterdatamodel.ErrNotFound
	}
	return rows[0], nil
}

// usageSQL = pemakaian item per sumber. Setiap baris = 1 kolom tabel lain yang menunjuk ke item
// (soal / modul / quiz / gambar yang belum dihapus, akun aktif, item anak). Setiap cabang query
// memakai "kolom IN (id halaman)" sehingga bisa memakai index per kolom.
var usageSQL = func() string {
	sources := []struct{ table, alias, deleted string }{
		{"questions", "question", "deleted_at IS NULL"},
		{"learning_modules", "module", "deleted_at IS NULL"},
		{"quizzes", "quiz", "deleted_at IS NULL"},
	}
	var parts []string
	for _, s := range sources {
		for _, col := range []string{"grade_id", "subject_id", "subject_field_id", "chapter_id", "subchapter_id", "topic_id"} {
			parts = append(parts, fmt.Sprintf("SELECT %s AS ref_id, '%s' AS src FROM %s WHERE %s AND %s IN @ids", col, s.alias, s.table, s.deleted, col))
		}
	}
	for _, col := range []string{"subject_id", "subject_field_id", "chapter_id", "subchapter_id", "topic_id"} {
		parts = append(parts, fmt.Sprintf("SELECT %s, 'image' FROM media_assets WHERE deleted_at IS NULL AND %s IN @ids", col, col))
	}
	parts = append(parts,
		"SELECT grade_id, 'user' FROM users WHERE deleted_at IS NULL AND grade_id IN @ids",
		"SELECT education_level_id, 'user' FROM users WHERE deleted_at IS NULL AND education_level_id IN @ids",
		"SELECT parent_id, 'child' FROM master_data_items WHERE deleted_at IS NULL AND parent_id IN @ids",
		// Kelas dipakai bidang materi lewat kolom grade_id (bukan induk).
		"SELECT grade_id, 'child' FROM master_data_items WHERE deleted_at IS NULL AND type = 'subject_field' AND grade_id IN @ids",
	)
	return "SELECT ref_id::text AS ref_id, src, COUNT(*)::int AS total FROM (\n" + strings.Join(parts, "\nUNION ALL\n") + "\n) u GROUP BY 1, 2"
}()

// Usage menghitung pemakaian banyak item sekaligus (kolom "Isi" dan syarat hapus), 1 query.
// Hasilnya map id → pemakaian (item tanpa pemakaian tidak ada di map = semua 0). [Native SQL]
func (r *MasterDataRepository) Usage(ctx context.Context, ids []string) (map[string]masterdatamodel.Usage, error) {
	return usage(ctx, r.db, ids)
}

func usage(ctx context.Context, db *gorm.DB, ids []string) (map[string]masterdatamodel.Usage, error) {
	result := map[string]masterdatamodel.Usage{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		RefID, Src string
		Total      int
	}
	if err := db.WithContext(ctx).Raw(usageSQL, map[string]any{"ids": ids}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		u := result[row.RefID]
		switch row.Src {
		case "question":
			u.Questions += row.Total
		case "module":
			u.Modules += row.Total
		case "quiz":
			u.Quizzes += row.Total
		case "image":
			u.Images += row.Total
		case "user":
			u.Users += row.Total
		case "child":
			u.Children += row.Total
		}
		result[row.RefID] = u
	}
	return result, nil
}

// CreateItem menyimpan data baru, mengembalikan ID-nya. sort_order kosong = urutan terakhir + 1 di
// antara saudaranya (type, induk, dan kelas yang sama). Nama / kode kembar → ErrNameTaken /
// ErrCodeTaken. [Native SQL]
func (r *MasterDataRepository) CreateItem(ctx context.Context, n masterdatamodel.NewItem, createdBy string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO master_data_items
			(type, code, name, parent_id, subject_id, education_level_id, grade_id, sort_order, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?,
			COALESCE(?::int, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM master_data_items
			                  WHERE type = ? AND parent_id IS NOT DISTINCT FROM ?::uuid
			                    AND grade_id IS NOT DISTINCT FROM ?::uuid AND deleted_at IS NULL)),
			?, ?)
		RETURNING id`,
		n.Type, n.Code, n.Name, n.ParentID, n.SubjectID, n.EducationLevelID, n.GradeID,
		n.SortOrder, n.Type, n.ParentID, n.GradeID, n.Status, createdBy).
		Scan(&id).Error
	return id, uniqueError(err)
}

// UpdateItem mengganti kode, nama, urutan, dan status 1 data. Tidak ada → ErrNotFound. [Native SQL]
func (r *MasterDataRepository) UpdateItem(ctx context.Context, u masterdatamodel.ItemUpdate, updatedBy string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE master_data_items SET code = ?, name = ?, sort_order = ?, status = ?, updated_by = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`,
		u.Code, u.Name, u.SortOrder, u.Status, updatedBy, u.ID)
	if result.Error != nil {
		return uniqueError(result.Error)
	}
	if result.RowsAffected == 0 {
		return masterdatamodel.ErrNotFound
	}
	return nil
}

// DeleteCheck = hasil pengecekan hapus 1 item.
type DeleteCheck struct {
	ID       string
	Name     string
	IsSystem bool
	Usage    masterdatamodel.Usage `gorm:"-"` // dihitung terpisah, bukan kolom
}

// SoftDeleteItems menghapus banyak data (soft delete) dalam 1 transaksi, semua atau tidak sama sekali.
// Baris dikunci dulu (FOR UPDATE) lalu pemakaiannya dihitung di transaksi yang sama.
// missing = ID yang tidak ada / sudah dihapus; blocked = data sistem atau masih dipakai. [Native SQL]
func (r *MasterDataRepository) SoftDeleteItems(ctx context.Context, ids []string, deletedBy string) (missing []string, blocked []DeleteCheck, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found []DeleteCheck
		if err := tx.Raw(`SELECT id, name, is_system FROM master_data_items WHERE id IN ? AND deleted_at IS NULL FOR UPDATE`, ids).
			Scan(&found).Error; err != nil {
			return err
		}
		byID := make(map[string]DeleteCheck, len(found))
		for _, f := range found {
			byID[f.ID] = f
		}
		foundIDs := make([]string, 0, len(found))
		for _, id := range ids {
			if _, ok := byID[id]; ok {
				foundIDs = append(foundIDs, id)
			} else {
				missing = append(missing, id)
			}
		}

		// Pemakaian tetap dihitung walau ada ID yang tidak ada, supaya semua kesalahan tampil sekaligus.
		used, err := usage(ctx, tx, foundIDs)
		if err != nil {
			return err
		}
		for _, id := range foundIDs {
			c := byID[id]
			c.Usage = used[id]
			// Anak yang ikut dihapus di request yang sama tetap dihitung: hapus dari bawah dulu.
			if c.IsSystem || c.Usage.Total() > 0 {
				blocked = append(blocked, c)
			}
		}
		if len(missing) > 0 || len(blocked) > 0 {
			return nil
		}
		return tx.Exec(`UPDATE master_data_items SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW() WHERE id IN ?`, deletedBy, ids).Error
	})
	return missing, blocked, err
}

// uniqueError mengubah error unik PostgreSQL (23505) menjadi ErrNameTaken / ErrCodeTaken sesuai
// nama index-nya.
func uniqueError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if strings.Contains(pgErr.ConstraintName, "code") {
		return masterdatamodel.ErrCodeTaken
	}
	return masterdatamodel.ErrNameTaken
}
