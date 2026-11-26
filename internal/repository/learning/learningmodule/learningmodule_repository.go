// Package learningmodule (repository) mengakses tabel learning_modules & learning_module_plans.
//
// Di package lain, import dengan alias learningmodulerepo:
//
//	learningmodulerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/learning/learningmodule"
package learningmodule

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learning/learningmodule"
)

// LearningModuleRepository mengakses data modul pembelajaran.
type LearningModuleRepository struct {
	db *gorm.DB
}

// NewLearningModuleRepository membuat LearningModuleRepository. Dipanggil sekali di main.go.
func NewLearningModuleRepository(db *gorm.DB) *LearningModuleRepository {
	return &LearningModuleRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// FindHierarchy mengambil item master data letak modul (belum dihapus), 1 query. Hasil map id → item.
func (r *LearningModuleRepository) FindHierarchy(ctx context.Context, ids []string) (map[string]learningmodulemodel.HierarchyItem, error) {
	var rows []learningmodulemodel.HierarchyItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT i.id, i.type, i.status, i.name, i.parent_id, i.education_level_id, i.grade_id,
		       l.name AS level_name, g.name AS grade_name
		FROM master_data_items i
		LEFT JOIN master_data_items l ON l.id = i.education_level_id
		LEFT JOIN master_data_items g ON g.id = i.grade_id
		WHERE i.id IN ? AND i.deleted_at IS NULL`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]learningmodulemodel.HierarchyItem, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

// ActivePlans mengambil paket akses aktif (belum dihapus) dari daftar ID, 1 query. Hasil set ID.
func (r *LearningModuleRepository) ActivePlans(ctx context.Context, ids []string) (map[string]bool, error) {
	var found []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT id FROM subscription_plans WHERE id IN ? AND deleted_at IS NULL AND status = 'active'`, ids).
		Scan(&found).Error
	result := make(map[string]bool, len(found))
	for _, id := range found {
		result[id] = true
	}
	return result, err
}

// Create menyimpan modul + paket aksesnya dalam 1 transaksi, mengembalikan ID-nya. [Native SQL, 2 query]
func (r *LearningModuleRepository) Create(ctx context.Context, m learningmodulemodel.Save, createdBy string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`
			INSERT INTO learning_modules (name, description,
				education_level_id, grade_id, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
				file_name, storage_path, mime_type, size_bytes, page_count, status, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id`,
			m.Name, m.Description,
			m.EducationLevelID, m.GradeID, m.SubjectID, m.SubjectFieldID, m.ChapterID, m.SubchapterID, m.TopicID,
			m.File.FileName, m.File.StoragePath, m.File.MimeType, m.File.SizeBytes, m.PageCount, m.Status, createdBy).
			Scan(&id).Error
		if err != nil {
			return err
		}
		return insertPlans(tx, id, m.PlanIDs)
	})
	return id, err
}

// Update mengganti isi modul (ganti penuh) + paket aksesnya dalam 1 transaksi. Kalau m.File diisi,
// data file ikut diganti dan oldPath = lokasi file lama (dihapus service setelah berhasil).
// Tidak ada / sudah dihapus → ErrModuleNotFound. [Native SQL, 4 query]
func (r *LearningModuleRepository) Update(ctx context.Context, m learningmodulemodel.Save, updatedBy string) (oldPath string, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var paths []string
		if err := tx.Raw(`SELECT storage_path FROM learning_modules WHERE id = ? AND deleted_at IS NULL FOR UPDATE`, m.ID).
			Scan(&paths).Error; err != nil {
			return err
		}
		if len(paths) == 0 {
			return learningmodulemodel.ErrModuleNotFound
		}

		sets := `name = ?, description = ?,
			education_level_id = ?, grade_id = ?, subject_id = ?, subject_field_id = ?,
			chapter_id = ?, subchapter_id = ?, topic_id = ?, status = ?, updated_by = ?, updated_at = NOW()`
		args := []any{m.Name, m.Description, m.EducationLevelID, m.GradeID, m.SubjectID, m.SubjectFieldID,
			m.ChapterID, m.SubchapterID, m.TopicID, m.Status, updatedBy}
		// page_count: diisi → diganti; kosong + PDF baru → hasil hitung PDF (bisa NULL); kosong tanpa PDF → tetap.
		if m.PageCount != nil || m.File != nil {
			sets += `, page_count = ?`
			args = append(args, m.PageCount)
		}
		if m.File != nil {
			sets += `, file_name = ?, storage_path = ?, mime_type = ?, size_bytes = ?`
			args = append(args, m.File.FileName, m.File.StoragePath, m.File.MimeType, m.File.SizeBytes)
			oldPath = paths[0]
		} else if m.RelocatedPath != "" {
			sets += `, storage_path = ?` // letak berubah: file sudah dipindah service
			args = append(args, m.RelocatedPath)
		}
		args = append(args, m.ID)
		if err := tx.Exec(`UPDATE learning_modules SET `+sets+` WHERE id = ?`, args...).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM learning_module_plans WHERE module_id = ?`, m.ID).Error; err != nil {
			return err
		}
		return insertPlans(tx, m.ID, m.PlanIDs)
	})
	if err != nil {
		return "", err
	}
	return oldPath, nil
}

// insertPlans menyimpan semua paket akses modul dalam 1 INSERT banyak baris.
func insertPlans(tx *gorm.DB, moduleID string, planIDs []string) error {
	rows := make([]string, 0, len(planIDs))
	args := make([]any, 0, len(planIDs)*2)
	for _, planID := range planIDs {
		rows = append(rows, "(?, ?)")
		args = append(args, moduleID, planID)
	}
	return tx.Exec(`INSERT INTO learning_module_plans (module_id, plan_id) VALUES `+strings.Join(rows, ", "), args...).Error
}

// moduleRow menampung 1 modul + nama letak & user (flat, untuk Scan).
type moduleRow struct {
	ID, Name                 string
	Description              *string
	LevelID, LevelName       string
	GradeID, GradeName       string
	SubjectID, SubjectName   string
	FieldID, FieldName       string
	ChapterID, ChapterName   *string
	SubID, SubName           *string
	TopicID, TopicName       *string
	FileName, StoragePath    string
	MimeType                 string
	SizeBytes                int64
	PageCount                *int
	Status                   string
	CreatedByID, CreatedName *string
	UpdatedByID, UpdatedName *string
	CreatedAt, UpdatedAt     time.Time
}

func optRef(id, name *string) *learningmodulemodel.Ref {
	if id == nil || name == nil {
		return nil
	}
	return &learningmodulemodel.Ref{ID: *id, Name: *name}
}

func (row moduleRow) toModule() learningmodulemodel.Module {
	return learningmodulemodel.Module{
		ID: row.ID, Name: row.Name, Description: row.Description,
		EducationLevel: learningmodulemodel.Ref{ID: row.LevelID, Name: row.LevelName},
		Grade:          learningmodulemodel.Ref{ID: row.GradeID, Name: row.GradeName},
		Subject:        learningmodulemodel.Ref{ID: row.SubjectID, Name: row.SubjectName},
		SubjectField:   learningmodulemodel.Ref{ID: row.FieldID, Name: row.FieldName},
		Chapter:        optRef(row.ChapterID, row.ChapterName),
		Subchapter:     optRef(row.SubID, row.SubName),
		Topic:          optRef(row.TopicID, row.TopicName),
		FileName:       row.FileName, StoragePath: row.StoragePath, MimeType: row.MimeType, SizeBytes: row.SizeBytes, PageCount: row.PageCount,
		Status:    row.Status,
		CreatedBy: optRef(row.CreatedByID, row.CreatedName), UpdatedBy: optRef(row.UpdatedByID, row.UpdatedName),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Plans: []learningmodulemodel.Ref{},
	}
}

// moduleSelect = kolom 1 modul (alias m) + nama letak & user. Semua JOIN lewat primary key.
const moduleSelect = `
	SELECT m.id, m.name, m.description,
	       m.education_level_id AS level_id, l.name AS level_name,
	       m.grade_id, g.name AS grade_name,
	       m.subject_id, s.name AS subject_name,
	       m.subject_field_id AS field_id, f.name AS field_name,
	       m.chapter_id, c.name AS chapter_name,
	       m.subchapter_id AS sub_id, sc.name AS sub_name,
	       m.topic_id, t.name AS topic_name,
	       m.file_name, m.storage_path, m.mime_type, m.size_bytes, m.page_count, m.status,
	       m.created_by AS created_by_id, cu.real_name AS created_name,
	       m.updated_by AS updated_by_id, uu.real_name AS updated_name,
	       m.created_at, m.updated_at`

const moduleJoins = `
	JOIN      master_data_items l  ON l.id  = m.education_level_id
	JOIN      master_data_items g  ON g.id  = m.grade_id
	JOIN      master_data_items s  ON s.id  = m.subject_id
	JOIN      master_data_items f  ON f.id  = m.subject_field_id
	LEFT JOIN master_data_items c  ON c.id  = m.chapter_id
	LEFT JOIN master_data_items sc ON sc.id = m.subchapter_id
	LEFT JOIN master_data_items t  ON t.id  = m.topic_id
	LEFT JOIN users cu ON cu.id = m.created_by
	LEFT JOIN users uu ON uu.id = m.updated_by`

// FindByID mengambil 1 modul (belum dihapus) + paket aksesnya. Tidak ada → ErrModuleNotFound. [2 query]
func (r *LearningModuleRepository) FindByID(ctx context.Context, id string) (learningmodulemodel.Module, error) {
	var rows []moduleRow
	err := r.db.WithContext(ctx).Raw(moduleSelect+` FROM learning_modules m`+moduleJoins+`
		WHERE m.id = ? AND m.deleted_at IS NULL`, id).Scan(&rows).Error
	if err != nil {
		return learningmodulemodel.Module{}, err
	}
	if len(rows) == 0 {
		return learningmodulemodel.Module{}, learningmodulemodel.ErrModuleNotFound
	}
	modules := []learningmodulemodel.Module{rows[0].toModule()}
	if err := r.attachPlans(ctx, modules); err != nil {
		return learningmodulemodel.Module{}, err
	}
	return modules[0], nil
}

// List mengambil 1 halaman modul (terbaru dulu) + total. [Native SQL, 3 query]
// Halaman diambil dulu (CTE), baru di-JOIN ke nama letak; paket akses diambil 1 query untuk
// semua modul di halaman itu. COUNT dilewati di halaman terakhir yang tidak penuh.
func (r *LearningModuleRepository) List(ctx context.Context, f learningmodulemodel.ListFilter) ([]learningmodulemodel.Module, int, error) {
	conds := []string{"m.deleted_at IS NULL"}
	args := []any{}
	for _, e := range []struct{ column, value string }{
		{"m.education_level_id::text", f.EducationLevelID},
		{"m.grade_id::text", f.GradeID},
		{"m.subject_id::text", f.SubjectID},
		{"m.subject_field_id::text", f.SubjectFieldID},
		{"m.chapter_id::text", f.ChapterID},
		{"m.status", f.Status},
	} {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.PlanID != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM learning_module_plans mp WHERE mp.module_id = m.id AND mp.plan_id = ?)")
		args = append(args, f.PlanID)
	}
	if f.Search != "" {
		conds = append(conds, "m.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	where := strings.Join(conds, " AND ")

	var rows []moduleRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT m.id FROM learning_modules m
			WHERE `+where+`
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT ? OFFSET ?
		)`+moduleSelect+`
		FROM page p
		JOIN learning_modules m ON m.id = p.id`+moduleJoins+`
		ORDER BY m.created_at DESC, m.id DESC`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	modules := make([]learningmodulemodel.Module, 0, len(rows))
	for _, row := range rows {
		modules = append(modules, row.toModule())
	}
	if err := r.attachPlans(ctx, modules); err != nil {
		return nil, 0, err
	}

	if len(modules) < f.PerPage && (len(modules) > 0 || f.Page == 1) {
		return modules, (f.Page-1)*f.PerPage + len(modules), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM learning_modules m WHERE `+where, args...).Scan(&total).Error
	return modules, total, err
}

// attachPlans mengisi paket akses semua modul dengan 1 query (urut sort_order paket).
func (r *LearningModuleRepository) attachPlans(ctx context.Context, modules []learningmodulemodel.Module) error {
	if len(modules) == 0 {
		return nil
	}
	ids := make([]string, 0, len(modules))
	index := make(map[string]int, len(modules))
	for i, m := range modules {
		ids = append(ids, m.ID)
		index[m.ID] = i
	}
	var rows []struct{ ModuleID, PlanID, PlanName string }
	err := r.db.WithContext(ctx).Raw(`
		SELECT mp.module_id, sp.id AS plan_id, sp.name AS plan_name
		FROM learning_module_plans mp
		JOIN subscription_plans sp ON sp.id = mp.plan_id
		WHERE mp.module_id IN ?
		ORDER BY sp.sort_order, sp.name`, ids).Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		i := index[row.ModuleID]
		modules[i].Plans = append(modules[i].Plans, learningmodulemodel.Ref{ID: row.PlanID, Name: row.PlanName})
	}
	return nil
}

// SoftDeleteMany menghapus banyak modul (soft delete) dalam 1 transaksi: semua ID harus ada (belum
// dihapus), kalau tidak → tidak ada yang terhapus dan missing berisi ID yang tidak ada. File PDF
// tetap disimpan (data bisa dipulihkan). [Native SQL, 2 query]
func (r *LearningModuleRepository) SoftDeleteMany(ctx context.Context, ids []string, deletedBy string) (missing []string, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found []string
		if err := tx.Raw(`SELECT id FROM learning_modules WHERE id IN ? AND deleted_at IS NULL FOR UPDATE`, ids).Scan(&found).Error; err != nil {
			return err
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
			return nil
		}
		return tx.Exec(`UPDATE learning_modules SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW() WHERE id IN ?`, deletedBy, ids).Error
	})
	return missing, err
}
