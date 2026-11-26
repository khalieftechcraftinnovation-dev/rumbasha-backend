// Package landingpage (repository) berisi query database fitur Landing Page.
//
// Di package lain, import dengan alias landingpagerepo:
//
//	landingpagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/site/landingpage"
package landingpage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	landingpagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/site/landingpage"
)

// LandingPageRepository mengakses tabel landing_pages, landing_sections, landing_changes,
// dan landing_publish_logs.
type LandingPageRepository struct {
	db *gorm.DB
}

// NewLandingPageRepository membuat LandingPageRepository. Dipanggil sekali di main.go.
func NewLandingPageRepository(db *gorm.DB) *LandingPageRepository {
	return &LandingPageRepository{db: db}
}

// pageSelect = kolom halaman + nama pelaku publish terakhir + jumlah perubahan belum dipublish.
const pageSelect = `
	SELECT p.code, p.name, p.published_at, p.published_by::text AS published_by,
	       u.real_name AS published_by_name,
	       (SELECT COUNT(*)::int FROM landing_changes c
	        WHERE c.page_code = p.code AND c.published_at IS NULL) AS pending_count
	FROM landing_pages p
	LEFT JOIN users u ON u.id = p.published_by`

// ListPages mengambil semua halaman (tab) urut tampil. [Native SQL]
func (r *LandingPageRepository) ListPages(ctx context.Context) ([]landingpagemodel.Page, error) {
	var pages []landingpagemodel.Page
	err := r.db.WithContext(ctx).Raw(pageSelect + ` ORDER BY p.sort_order, p.code`).Scan(&pages).Error
	return pages, err
}

// FindPage mengambil 1 halaman. Tidak ada → ErrPageNotFound. [Native SQL, primary key]
func (r *LandingPageRepository) FindPage(ctx context.Context, code string) (landingpagemodel.Page, error) {
	var pages []landingpagemodel.Page
	if err := r.db.WithContext(ctx).Raw(pageSelect+` WHERE p.code = ?`, code).Scan(&pages).Error; err != nil {
		return landingpagemodel.Page{}, err
	}
	if len(pages) == 0 {
		return landingpagemodel.Page{}, landingpagemodel.ErrPageNotFound
	}
	return pages[0], nil
}

// ListSections mengambil semua section DRAFT 1 halaman, urut tampil. [Native SQL]
func (r *LandingPageRepository) ListSections(ctx context.Context, pageCode string) ([]landingpagemodel.Section, error) {
	var rows []struct {
		Key, Name, Content       string
		IsVisible                bool
		SortOrder                int
		UpdatedBy, UpdatedByName *string
		UpdatedAt                time.Time
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.key, s.name, s.content::text AS content, s.is_visible, s.sort_order,
		       s.updated_by::text AS updated_by, u.real_name AS updated_by_name, s.updated_at
		FROM landing_sections s
		LEFT JOIN users u ON u.id = s.updated_by
		WHERE s.page_code = ?
		ORDER BY s.sort_order, s.key`, pageCode).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	sections := make([]landingpagemodel.Section, 0, len(rows))
	for _, row := range rows {
		sections = append(sections, landingpagemodel.Section{
			Key: row.Key, Name: row.Name, Content: json.RawMessage(row.Content),
			IsVisible: row.IsVisible, SortOrder: row.SortOrder,
			UpdatedBy: row.UpdatedBy, UpdatedByName: row.UpdatedByName, UpdatedAt: row.UpdatedAt,
		})
	}
	return sections, nil
}

// ListPendingChanges mengambil perubahan yang belum dipublish, urut waktu. [Native SQL]
func (r *LandingPageRepository) ListPendingChanges(ctx context.Context, pageCode string) ([]landingpagemodel.Change, error) {
	var changes []landingpagemodel.Change
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.section_key, c.description, c.changed_by::text AS changed_by,
		       u.real_name AS changed_by_name, c.changed_at
		FROM landing_changes c
		LEFT JOIN users u ON u.id = c.changed_by
		WHERE c.page_code = ? AND c.published_at IS NULL
		ORDER BY c.changed_at, c.id`, pageCode).Scan(&changes).Error
	return changes, err
}

// UpdateSectionContent mengganti isi 1 section dan mencatat perubahannya.
// changed = false kalau isinya sama persis (tidak dicatat sebagai perubahan).
// Section tidak ada → ErrSectionNotFound. [Native SQL, 1 transaksi]
func (r *LandingPageRepository) UpdateSectionContent(ctx context.Context, pageCode, key string, content json.RawMessage, userID string) (changed bool, err error) {
	return r.updateSection(ctx, pageCode, key, userID,
		`content = ?::jsonb`, `content IS DISTINCT FROM ?::jsonb`, string(content),
		func(name string) string { return name + " · isi diubah" })
}

// SetSectionVisible mengubah saklar "Tampil" 1 section dan mencatat perubahannya.
// changed = false kalau nilainya sudah sama. [Native SQL, 1 transaksi]
func (r *LandingPageRepository) SetSectionVisible(ctx context.Context, pageCode, key string, visible bool, userID string) (changed bool, err error) {
	return r.updateSection(ctx, pageCode, key, userID,
		`is_visible = ?`, `is_visible IS DISTINCT FROM ?`, visible,
		func(name string) string {
			if visible {
				return name + " · ditampilkan"
			}
			return name + " · disembunyikan"
		})
}

// updateSection = UPDATE 1 kolom section (hanya kalau nilainya berbeda) + catat ke landing_changes.
func (r *LandingPageRepository) updateSection(ctx context.Context, pageCode, key, userID, setSQL, diffSQL string, value any, describe func(name string) string) (bool, error) {
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var names []string
		err := tx.Raw(`
			UPDATE landing_sections
			SET `+setSQL+`, updated_by = ?, updated_at = NOW()
			WHERE page_code = ? AND key = ? AND `+diffSQL+`
			RETURNING name`, value, userID, pageCode, key, value).Scan(&names).Error
		if err != nil {
			return err
		}

		if len(names) == 0 {
			// Tidak ada yang berubah: section tidak ada, atau nilainya memang sama.
			var exists bool
			if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM landing_sections WHERE page_code = ? AND key = ?)`, pageCode, key).
				Scan(&exists).Error; err != nil {
				return err
			}
			if !exists {
				return landingpagemodel.ErrSectionNotFound
			}
			return nil
		}

		changed = true
		return insertChange(tx, pageCode, &key, describe(names[0]), userID)
	})
	return changed, err
}

// ReorderSections mengatur ulang urutan section. keys = SEMUA key section halaman, urut baru
// (sudah dicek lengkap oleh service). changed = false kalau urutannya sama. [Native SQL, 1 transaksi]
func (r *LandingPageRepository) ReorderSections(ctx context.Context, pageCode string, keys []string, userID string) (changed bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Section 1 halaman hanya sedikit (≤ 10), jadi cukup 1 UPDATE per section.
		for i, key := range keys {
			res := tx.Exec(`
				UPDATE landing_sections SET sort_order = ?, updated_by = ?, updated_at = NOW()
				WHERE page_code = ? AND key = ? AND sort_order <> ?`, i+1, userID, pageCode, key, i+1)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				changed = true
			}
		}
		if !changed {
			return nil
		}
		return insertChange(tx, pageCode, nil, "Urutan section diubah", userID)
	})
	return changed, err
}

// insertChange mencatat 1 perubahan yang belum dipublish.
func insertChange(tx *gorm.DB, pageCode string, sectionKey *string, description, userID string) error {
	return tx.Exec(`
		INSERT INTO landing_changes (page_code, section_key, description, changed_by)
		VALUES (?, ?, ?, ?)`, pageCode, sectionKey, description, userID).Error
}

// Publish menyalin section yang tampil ke snapshot halaman publik, menandai perubahan sudah
// dipublish, dan menulis riwayat publish. Semua dalam 1 transaksi; baris halaman dikunci
// (FOR UPDATE) supaya 2 admin yang menekan Publish bersamaan tidak saling menimpa.
// Halaman yang sudah pernah dipublish tanpa perubahan baru → ErrNothingToPublish. [Native SQL]
func (r *LandingPageRepository) Publish(ctx context.Context, pageCode, userID string) (landingpagemodel.PublishResult, error) {
	var result landingpagemodel.PublishResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Kunci halaman.
		var pages []struct{ PublishedAt *time.Time }
		if err := tx.Raw(`SELECT published_at FROM landing_pages WHERE code = ? FOR UPDATE`, pageCode).
			Scan(&pages).Error; err != nil {
			return err
		}
		if len(pages) == 0 {
			return landingpagemodel.ErrPageNotFound
		}

		// 2. Perubahan yang ikut dipublish.
		var descriptions []string
		if err := tx.Raw(`
			UPDATE landing_changes SET published_at = NOW()
			WHERE page_code = ? AND published_at IS NULL
			RETURNING description`, pageCode).Scan(&descriptions).Error; err != nil {
			return err
		}
		if len(descriptions) == 0 && pages[0].PublishedAt != nil {
			return landingpagemodel.ErrNothingToPublish
		}

		// 3. Snapshot section yang tampil (urut tampil). Tidak ada yang tampil = array kosong.
		var publishedAt []time.Time
		if err := tx.Raw(`
			UPDATE landing_pages p
			SET published_content = COALESCE((
			        SELECT jsonb_agg(jsonb_build_object('key', s.key, 'name', s.name, 'content', s.content)
			                         ORDER BY s.sort_order, s.key)
			        FROM landing_sections s
			        WHERE s.page_code = p.code AND s.is_visible), '[]'::jsonb),
			    published_at = NOW(),
			    published_by = ?
			WHERE p.code = ?
			RETURNING p.published_at`, userID, pageCode).Scan(&publishedAt).Error; err != nil {
			return err
		}
		if len(publishedAt) != 1 {
			return fmt.Errorf("gagal menyimpan snapshot halaman %s", pageCode)
		}

		// 4. Riwayat publish.
		changesJSON, err := json.Marshal(descriptions)
		if err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO landing_publish_logs (page_code, published_by, published_at, changes)
			VALUES (?, ?, ?, ?::jsonb)`, pageCode, userID, publishedAt[0], string(changesJSON)).Error; err != nil {
			return err
		}

		result = landingpagemodel.PublishResult{PublishedAt: publishedAt[0], Changes: descriptions}
		return nil
	})
	return result, err
}

// FindPublished mengambil isi halaman publik (snapshot publish terakhir). 1 baris, primary key,
// tanpa JOIN. Halaman tidak ada → ErrPageNotFound. [Native SQL]
func (r *LandingPageRepository) FindPublished(ctx context.Context, pageCode string) (landingpagemodel.PublishedPage, error) {
	var rows []struct {
		Code, Name       string
		PublishedContent *string
		PublishedAt      *time.Time
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT code, name, published_content::text AS published_content, published_at
		FROM landing_pages WHERE code = ?`, pageCode).Scan(&rows).Error
	if err != nil {
		return landingpagemodel.PublishedPage{}, err
	}
	if len(rows) == 0 {
		return landingpagemodel.PublishedPage{}, landingpagemodel.ErrPageNotFound
	}

	row := rows[0]
	page := landingpagemodel.PublishedPage{
		Code: row.Code, Name: row.Name, PublishedAt: row.PublishedAt,
		Sections: []landingpagemodel.PublishedSection{},
	}
	if row.PublishedContent != nil {
		if err := json.Unmarshal([]byte(*row.PublishedContent), &page.Sections); err != nil {
			return landingpagemodel.PublishedPage{}, fmt.Errorf("published_content halaman %s rusak: %w", pageCode, err)
		}
	}
	return page, nil
}
