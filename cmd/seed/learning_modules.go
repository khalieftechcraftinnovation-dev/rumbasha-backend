package main

// Seed modul pembelajaran: setiap bidang materi Matematika (SD & SMP) mendapat 3 modul untuk bab
// pertamanya, dengan file PDF dummy yang disimpan di STORAGE_DIR mengikuti hierarki mapel
// (sama persis dengan upload lewat API):
//
//	Ringkasan Materi    → paket Basic, Pro, Ultra   (2 halaman)
//	Latihan Soal        → paket Pro, Ultra          (3 halaman)
//	Pembahasan Lengkap  → paket Ultra               (4 halaman)
//
// Aman dijalankan berkali-kali: modul dengan nama & bidang materi yang sama dilewati.

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learningmodule"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
)

// moduleTier = 1 tingkat modul dummy dan paket yang membukanya.
type moduleTier struct {
	Title string
	Plans []string // nama paket
	Pages int
}

var moduleTiers = []moduleTier{
	{"Ringkasan Materi", []string{"Basic", "Pro", "Ultra"}, 2},
	{"Latihan Soal", []string{"Pro", "Ultra"}, 3},
	{"Pembahasan Lengkap", []string{"Ultra"}, 4},
}

// seedLearningModules membuat modul dummy + file PDF-nya. Mengembalikan jumlah modul baru.
func seedLearningModules(db *gorm.DB, store *storage.Local) (int, error) {
	var plans []struct{ ID, Name string }
	if err := db.Raw(`SELECT id, name FROM subscription_plans WHERE name IN ('Basic', 'Pro', 'Ultra') AND deleted_at IS NULL`).Scan(&plans).Error; err != nil {
		return 0, err
	}
	planIDs := map[string]string{}
	for _, p := range plans {
		planIDs[p.Name] = p.ID
	}
	if len(planIDs) != 3 {
		return 0, fmt.Errorf("paket Basic/Pro/Ultra belum ada (dummy_access_plans.sql)")
	}

	// Bidang materi Matematika + bab pertamanya (urut kurikulum).
	var fields []struct {
		LevelID, LevelName, GradeID, GradeName string
		SubjectID, SubjectName, FieldID, Field string
		ChapterID, Chapter                     string
	}
	err := db.Raw(`
		SELECT l.id AS level_id, l.name AS level_name, g.id AS grade_id, g.name AS grade_name,
		       s.id AS subject_id, s.name AS subject_name, f.id AS field_id, f.name AS field,
		       c.id AS chapter_id, c.name AS chapter
		FROM master_data_items f
		JOIN master_data_items s ON s.id = f.subject_id AND s.code = 'MTK'
		JOIN master_data_items g ON g.id = f.grade_id
		JOIN master_data_items l ON l.id = f.education_level_id
		JOIN LATERAL (
			SELECT c.id, c.name FROM master_data_items c
			WHERE c.type = 'chapter' AND c.parent_id = f.id AND c.deleted_at IS NULL
			ORDER BY c.sort_order, c.name LIMIT 1
		) c ON TRUE
		WHERE f.type = 'subject_field' AND f.deleted_at IS NULL
		ORDER BY g.sort_order, f.sort_order`).Scan(&fields).Error
	if err != nil {
		return 0, err
	}
	var adminID string
	db.Raw(`SELECT id FROM users WHERE username = 'admin.sbytimur'`).Scan(&adminID)

	created := 0
	for _, f := range fields {
		for _, tier := range moduleTiers {
			name := fmt.Sprintf("%s %s - %s", f.Chapter, f.GradeName, tier.Title)
			var exists int
			db.Raw(`SELECT COUNT(*) FROM learning_modules WHERE name = ? AND subject_field_id = ? AND deleted_at IS NULL`, name, f.FieldID).Scan(&exists)
			if exists > 0 {
				continue
			}

			id := uuid.NewString()
			path := learningmodulemodel.FilePath(learningmodulemodel.PathNames{
				Subject: f.SubjectName, Level: f.LevelName, Grade: f.GradeName, SubjectField: f.Field, Chapter: f.Chapter,
			}, id)
			pdf := dummyPDF(name, fmt.Sprintf("%s %s - %s - %s", f.SubjectName, f.LevelName, f.GradeName, f.Field), tier)
			if err := store.Save(path, pdf); err != nil {
				return created, err
			}

			err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(`
					INSERT INTO learning_modules (id, name, description,
						education_level_id, grade_id, subject_id, subject_field_id, chapter_id,
						file_name, storage_path, mime_type, size_bytes, page_count, status, created_by)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'application/pdf', ?, ?, 'published', ?)`,
					id, name, fmt.Sprintf("%s untuk %s %s (data dummy).", tier.Title, f.Chapter, f.GradeName),
					f.LevelID, f.GradeID, f.SubjectID, f.FieldID, f.ChapterID,
					fileName(name), path, len(pdf), tier.Pages, adminID).Error; err != nil {
					return err
				}
				for _, planName := range tier.Plans {
					if err := tx.Exec(`INSERT INTO learning_module_plans (module_id, plan_id) VALUES (?, ?)`, id, planIDs[planName]).Error; err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				_ = store.Remove(path)
				return created, err
			}
			created++
		}
	}
	return created, nil
}

// fileName = nama file asli dummy, contoh "bilangan-cacah-kelas-1-ringkasan-materi.pdf".
func fileName(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-") + ".pdf"
}

// dummyPDF membuat PDF sederhana yang valid (bisa dibuka di browser): 1 judul + beberapa halaman.
func dummyPDF(title, subtitle string, tier moduleTier) []byte {
	escape := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	type object struct{ body string }
	var objs []object
	add := func(body string) int { objs = append(objs, object{body}); return len(objs) }

	catalog := add("") // diisi setelah pages diketahui
	pagesObj := add("")
	font := add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	kids := []string{}
	for page := 1; page <= tier.Pages; page++ {
		text := fmt.Sprintf("BT /F1 20 Tf 60 770 Td (%s) Tj 0 -28 Td /F1 12 Tf (%s) Tj 0 -40 Td (Halaman %d dari %d) Tj 0 -24 Td (Paket: %s) Tj 0 -24 Td (Isi materi dummy untuk mencoba fitur modul pembelajaran Rumbasha.) Tj ET",
			escape.Replace(title), escape.Replace(subtitle), page, tier.Pages, strings.Join(tier.Plans, ", "))
		content := add(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(text), text))
		pageObj := add(fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 595 842] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R >> >> >>", pagesObj, content, font))
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObj))
	}
	objs[catalog-1].body = fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObj)
	objs[pagesObj-1].body = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), tier.Pages)

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o.body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, catalog, xref)
	return b.Bytes()
}

// logModuleSeed mencetak hasil seed modul.
func logModuleSeed(n int, err error) {
	if err != nil {
		log.Fatalf("gagal mengisi modul pembelajaran: %v", err)
	}
	log.Printf("modul pembelajaran: %d modul baru (PDF di STORAGE_DIR/modules/...)", n)
}

// removeOrphanModuleFiles menghapus file PDF di STORAGE_DIR/modules yang tidak lagi dipakai baris
// learning_modules mana pun (contoh setelah database dikosongkan oleh test lalu di-seed ulang).
// Hanya dijalankan oleh seed (development); production tidak memakai seed.
func removeOrphanModuleFiles(db *gorm.DB, storageDir string) (int, error) {
	var paths []string
	if err := db.Raw(`SELECT storage_path FROM learning_modules`).Scan(&paths).Error; err != nil {
		return 0, err
	}
	used := make(map[string]bool, len(paths))
	for _, p := range paths {
		used[filepath.ToSlash(p)] = true
	}
	removed := 0
	root := filepath.Join(storageDir, "modules")
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // folder belum ada / tidak bisa dibaca: lewati
		}
		rel, err := filepath.Rel(storageDir, p)
		if err != nil {
			return nil
		}
		if !used[filepath.ToSlash(rel)] {
			if err := os.Remove(p); err == nil {
				removed++
			}
		}
		return nil
	})
	return removed, err
}
