package learningmodule

import (
	"context"
	"strings"
	"time"

	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learningmodule"
)

// FindStudentLevel mengambil jenjang akun siswa (nil kalau belum diisi). [Native SQL, primary key]
func (r *LearningModuleRepository) FindStudentLevel(ctx context.Context, studentID string) (*learningmodulemodel.Ref, error) {
	var rows []struct{ ID, Name string }
	err := r.db.WithContext(ctx).Raw(`
		SELECT l.id, l.name FROM users u
		JOIN master_data_items l ON l.id = u.education_level_id
		WHERE u.id = ? AND u.deleted_at IS NULL`, studentID).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &learningmodulemodel.Ref{ID: rows[0].ID, Name: rows[0].Name}, nil
}

// unlockedExpr = kolom boolean "modul bisa dibuka siswa" (alias m). Parameter: fitur modul terbuka,
// semua modul terbuka (akses khusus admin), ID paket aktif siswa ("" = tidak ada).
const unlockedExpr = `(?::boolean AND (?::boolean OR EXISTS (
	SELECT 1 FROM learning_module_plans mp WHERE mp.module_id = m.id AND mp.plan_id::text = ?)))`

// studentModuleRow = baris modul + status kunci + riwayat buka siswa (flat, untuk Scan).
type studentModuleRow struct {
	Base                        moduleRow `gorm:"embedded"`
	Unlocked                    bool
	FirstOpenedAt, LastOpenedAt *time.Time
	OpenCount                   int
}

func (row studentModuleRow) toStudentModule() learningmodulemodel.StudentModule {
	return learningmodulemodel.StudentModule{
		Module: row.Base.toModule(), Locked: !row.Unlocked,
		FirstOpenedAt: row.FirstOpenedAt, LastOpenedAt: row.LastOpenedAt, OpenCount: row.OpenCount,
	}
}

// studentColumns = kolom tambahan list / buka modul siswa: status kunci + riwayat buka (alias o).
const studentColumns = `, ` + unlockedExpr + ` AS unlocked,
	o.first_opened_at, o.last_opened_at, COALESCE(o.open_count, 0) AS open_count`

// StudentList mengambil 1 halaman modul untuk siswa + total. [Native SQL, 3 query]
//
// Isi: modul PUBLISH (belum dihapus) di JENJANG siswa, urut kurikulum (kelas → mapel → bidang
// materi → bab → nama). Modul di luar paket siswa tetap tampil dengan locked = true.
// Filter access: "unlocked" / "locked". Halaman diambil dulu (CTE, index learning_modules_student_idx),
// baru di-JOIN nama letak, paket akses (1 query), dan riwayat buka siswa.
func (r *LearningModuleRepository) StudentList(ctx context.Context, f learningmodulemodel.StudentFilter) ([]learningmodulemodel.StudentModule, int, error) {
	conds := []string{"m.deleted_at IS NULL", "m.status = 'published'", "m.education_level_id = ?"}
	args := []any{f.EducationLevelID}
	for _, e := range []struct{ column, value string }{
		{"m.grade_id::text", f.GradeID},
		{"m.subject_id::text", f.SubjectID},
		{"m.subject_field_id::text", f.SubjectFieldID},
		{"m.chapter_id::text", f.ChapterID},
	} {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.Search != "" {
		conds = append(conds, "m.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	switch f.Access {
	case "unlocked":
		conds = append(conds, unlockedExpr)
		args = append(args, f.ModuleFeature, f.AllModules, f.PlanID)
	case "locked":
		conds = append(conds, "NOT "+unlockedExpr)
		args = append(args, f.ModuleFeature, f.AllModules, f.PlanID)
	}
	where := strings.Join(conds, " AND ")

	// Urut kurikulum: kelas → mapel → bidang materi → bab (tanpa bab dulu) → nama.
	const order = `ORDER BY g.sort_order, s.sort_order, s.name, f.sort_order, c.sort_order NULLS FIRST, m.name, m.id`
	var rows []studentModuleRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	selectArgs := append([]any{f.ModuleFeature, f.AllModules, f.PlanID, f.StudentID}, pageArgs...)
	err := r.db.WithContext(ctx).Raw(moduleSelect+studentColumns+`
		FROM learning_modules m`+moduleJoins+`
		LEFT JOIN learning_module_opens o ON o.module_id = m.id AND o.student_id = ?
		WHERE m.id IN (
			SELECT m.id FROM learning_modules m
			JOIN master_data_items g ON g.id = m.grade_id
			JOIN master_data_items s ON s.id = m.subject_id
			JOIN master_data_items f ON f.id = m.subject_field_id
			LEFT JOIN master_data_items c ON c.id = m.chapter_id
			WHERE `+where+`
			`+order+`
			LIMIT ? OFFSET ?
		)
		`+order, selectArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]learningmodulemodel.StudentModule, 0, len(rows))
	modules := make([]learningmodulemodel.Module, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toStudentModule())
		modules = append(modules, items[len(items)-1].Module)
	}
	if err := r.attachPlans(ctx, modules); err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Plans = modules[i].Plans
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM learning_modules m WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// FindStudentModule mengambil 1 modul PUBLISH di jenjang siswa (+ status kunci, paket akses, riwayat
// buka). Tidak ada / draf / dihapus / jenjang lain → ErrModuleNotFound. [Native SQL, 2 query]
func (r *LearningModuleRepository) FindStudentModule(ctx context.Context, moduleID string, f learningmodulemodel.StudentFilter) (learningmodulemodel.StudentModule, error) {
	var rows []studentModuleRow
	err := r.db.WithContext(ctx).Raw(moduleSelect+studentColumns+`
		FROM learning_modules m`+moduleJoins+`
		LEFT JOIN learning_module_opens o ON o.module_id = m.id AND o.student_id = ?
		WHERE m.id = ? AND m.deleted_at IS NULL AND m.status = 'published' AND m.education_level_id = ?`,
		f.ModuleFeature, f.AllModules, f.PlanID, f.StudentID, moduleID, f.EducationLevelID).
		Scan(&rows).Error
	if err != nil {
		return learningmodulemodel.StudentModule{}, err
	}
	if len(rows) == 0 {
		return learningmodulemodel.StudentModule{}, learningmodulemodel.ErrModuleNotFound
	}
	item := rows[0].toStudentModule()
	modules := []learningmodulemodel.Module{item.Module}
	if err := r.attachPlans(ctx, modules); err != nil {
		return learningmodulemodel.StudentModule{}, err
	}
	item.Plans = modules[0].Plans
	return item, nil
}

// RecordOpen mencatat siswa membuka modul (baris baru, atau terakhir dibuka + jumlah buka
// diperbarui), lalu mengembalikan riwayatnya. [Native SQL, 1 query]
func (r *LearningModuleRepository) RecordOpen(ctx context.Context, studentID, moduleID string, now time.Time) (first, last time.Time, count int, err error) {
	var row struct {
		FirstOpenedAt, LastOpenedAt time.Time
		OpenCount                   int
	}
	err = r.db.WithContext(ctx).Raw(`
		INSERT INTO learning_module_opens (student_id, module_id, first_opened_at, last_opened_at, open_count)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT (student_id, module_id) DO UPDATE SET
			last_opened_at = EXCLUDED.last_opened_at,
			open_count     = learning_module_opens.open_count + 1
		RETURNING first_opened_at, last_opened_at, open_count`, studentID, moduleID, now, now).
		Scan(&row).Error
	return row.FirstOpenedAt, row.LastOpenedAt, row.OpenCount, err
}
