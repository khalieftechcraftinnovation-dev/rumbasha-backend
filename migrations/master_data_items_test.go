package migrations_test

import (
	"testing"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/testutil"
)

// Mengecek aturan tabel master_data_items dari migration 000007 (hierarki mapel).
// Setiap kasus dijalankan di dalam transaksi lalu di-ROLLBACK, jadi data awal tidak berubah.
func TestMasterDataItemsHierarchyRules(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Data bantu yang dibuat ulang di setiap kasus (di dalam transaksi).
	const setup = `
		INSERT INTO master_data_items (type, code, name) VALUES ('subject', 'TEST_MTK', 'Test Matematika');
		INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
		SELECT 'subject_field', 'Bilangan', s.id, s.id, g.parent_id, g.id
		FROM master_data_items s, master_data_items g
		WHERE s.code = 'TEST_MTK' AND g.type = 'grade' AND g.code = 'GRADE_4';`

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "mapel tidak boleh punya induk",
			sql:     `INSERT INTO master_data_items (type, name, parent_id) SELECT 'subject', 'IPA Test', id FROM master_data_items WHERE code = 'SD'`,
			wantErr: true,
		},
		{
			name: "bidang materi di kelas lain dengan nama sama boleh",
			sql: `INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
				SELECT 'subject_field', 'Bilangan', s.id, s.id, g.parent_id, g.id
				FROM master_data_items s, master_data_items g
				WHERE s.code = 'TEST_MTK' AND g.type = 'grade' AND g.code = 'GRADE_5'`,
			wantErr: false,
		},
		{
			name: "bidang materi kembar di kelas yang sama ditolak",
			sql: `INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
				SELECT 'subject_field', 'bilangan', s.id, s.id, g.parent_id, g.id
				FROM master_data_items s, master_data_items g
				WHERE s.code = 'TEST_MTK' AND g.type = 'grade' AND g.code = 'GRADE_4'`,
			wantErr: true,
		},
		{
			name: "kelas harus milik jenjangnya (Kelas 4 + SMP ditolak)",
			sql: `INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
				SELECT 'subject_field', 'Geometri', s.id, s.id, l.id, g.id
				FROM master_data_items s, master_data_items g, master_data_items l
				WHERE s.code = 'TEST_MTK' AND g.type = 'grade' AND g.code = 'GRADE_4'
				  AND l.type = 'education_level' AND l.code = 'SMP'`,
			wantErr: true,
		},
		{
			name: "bab tanpa kelas ditolak",
			sql: `INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id)
				SELECT 'chapter', 'Pecahan', f.id, f.subject_id, f.education_level_id
				FROM master_data_items f WHERE f.type = 'subject_field' AND f.name = 'Bilangan'`,
			wantErr: true,
		},
		{
			name: "bab dengan data lengkap diterima",
			sql: `INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id)
				SELECT 'chapter', 'Pecahan', f.id, f.subject_id, f.education_level_id, f.grade_id
				FROM master_data_items f WHERE f.type = 'subject_field' AND f.name = 'Bilangan'`,
			wantErr: false,
		},
		{
			name:    "topik tanpa induk ditolak",
			sql:     `INSERT INTO master_data_items (type, name) VALUES ('topic', 'Topik yatim')`,
			wantErr: true,
		},
		{
			name:    "kelas tidak boleh mengisi kolom mapel",
			sql:     `UPDATE master_data_items SET subject_id = (SELECT id FROM master_data_items WHERE code = 'TEST_MTK') WHERE code = 'GRADE_1'`,
			wantErr: true,
		},
		{
			name:    "level tidak boleh punya induk",
			sql:     `INSERT INTO master_data_items (type, code, name, parent_id) SELECT 'difficulty', 'very_hard', 'Sangat sulit', id FROM master_data_items WHERE code = 'SD'`,
			wantErr: true,
		},
		{
			name:    "tipe quiz wajib punya code (code itulah yang disimpan di soal)",
			sql:     `INSERT INTO master_data_items (type, name) VALUES ('quiz_type', 'Olimpiade')`,
			wantErr: true,
		},
		{
			name:    "code tipe soal tidak boleh kembar",
			sql:     `INSERT INTO master_data_items (type, code, name) VALUES ('question_type', 'free_text', 'Isian lain')`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := db.Begin()
			defer tx.Rollback()

			if err := tx.Exec(setup).Error; err != nil {
				t.Fatalf("setup gagal: %v", err)
			}

			err := tx.Exec(tt.sql).Error
			if tt.wantErr && err == nil {
				t.Fatal("seharusnya ditolak database, tapi diterima")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("seharusnya diterima, tapi error: %v", err)
			}
		})
	}
}
