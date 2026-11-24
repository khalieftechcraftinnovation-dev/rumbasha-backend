// Package image (repository) mengakses tabel media_assets (Image Library).
//
// Di package lain, import dengan alias imagerepo:
//
//	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/image"
package image

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/image"
)

// ImageRepository mengakses tabel media_assets.
type ImageRepository struct {
	db *gorm.DB
}

// NewImageRepository membuat ImageRepository. Dipanggil sekali di main.go.
func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// FindHierarchyByIDs mengambil item hierarki mapel (mapel s/d topik) untuk dicek. [Native SQL, 1 tabel]
// Hasilnya map id → item; item yang dihapus (soft delete) tidak ikut.
func (r *ImageRepository) FindHierarchyByIDs(ctx context.Context, ids []string) (map[string]imagemodel.HierarchyItem, error) {
	var items []imagemodel.HierarchyItem
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, type, parent_id, subject_id, status
			FROM master_data_items
			WHERE id IN ? AND deleted_at IS NULL`, ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]imagemodel.HierarchyItem, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

// CreateMany menyimpan semua gambar dengan 1 query INSERT banyak baris. [Native SQL]
// ID sudah dibuat di service. CreatedAt setiap gambar diisi dari database.
func (r *ImageRepository) CreateMany(ctx context.Context, images []imagemodel.Image) error {
	rows := make([]string, 0, len(images))
	args := make([]any, 0, len(images)*16)
	indexByID := make(map[string]int, len(images))

	for i, img := range images {
		indexByID[img.ID] = i
		rows = append(rows, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			img.ID, img.Category, img.Name, img.FileName, img.StoragePath, img.URL, img.MimeType, img.SizeBytes,
			img.Width, img.Height, img.SubjectID, img.SubjectFieldID, img.ChapterID, img.SubchapterID, img.TopicID,
			img.UploadedBy,
		)
	}

	var saved []struct {
		ID        string
		CreatedAt time.Time
	}
	err := r.db.WithContext(ctx).
		Raw(`
			INSERT INTO media_assets (
				id, category, name, file_name, storage_path, url, mime_type, size_bytes,
				width, height, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
				uploaded_by
			) VALUES `+strings.Join(rows, ",\n")+`
			RETURNING id, created_at`, args...).
		Scan(&saved).Error
	if err != nil {
		return err
	}
	if len(saved) != len(images) {
		return fmt.Errorf("jumlah gambar tersimpan %d, seharusnya %d", len(saved), len(images))
	}
	for _, row := range saved {
		images[indexByID[row.ID]].CreatedAt = row.CreatedAt
	}
	return nil
}

// imageUsageQuery mencari soal AKTIF yang memakai gambar: sebagai gambar soal (image_id)
// atau gambar opsi jawaban (answer.options[].image_id). Hasil: id gambar → kode soal.
//
// Catatan performa: bagian opsi membaca JSON answer semua soal aktif bertipe pilihan.
// Masih ringan untuk ribuan soal; kalau nanti lambat, tambahkan index GIN di answer.
const imageUsageQuery = `
	SELECT u.image_id, string_agg(DISTINCT u.code, ', ' ORDER BY u.code) AS codes
	FROM (
		SELECT q.image_id::text AS image_id, q.code
		FROM questions q
		WHERE q.deleted_at IS NULL AND q.image_id IN ?

		UNION ALL

		SELECT opt->>'image_id', q.code
		FROM questions q
		CROSS JOIN LATERAL jsonb_array_elements(q.answer->'options') AS opt
		WHERE q.deleted_at IS NULL AND q.type <> 'free_text' AND opt->>'image_id' IN ?
	) u
	GROUP BY u.image_id`

// DeleteMany menghapus permanen (hard delete) banyak gambar di dalam 1 transaksi.
//
// Semua atau tidak sama sekali: transaksi dibatalkan kalau ada ID yang tidak ada, atau ada
// gambar yang masih dipakai soal aktif. Hasilnya menjelaskan alasannya (lihat DeleteResult).
// File di disk TIDAK dihapus di sini; service menghapusnya setelah database berhasil.
func (r *ImageRepository) DeleteMany(ctx context.Context, ids []string) (imagemodel.DeleteResult, error) {
	result := imagemodel.DeleteResult{FoundIDs: map[string]bool{}, UsedBy: map[string]string{}}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return result, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	// 1. Kunci baris gambar (FOR UPDATE) supaya tidak dipakai/dihapus request lain di tengah proses.
	var found []struct {
		ID          string
		StoragePath string
	}
	err := tx.Raw(`SELECT id, storage_path FROM media_assets WHERE id IN ? FOR UPDATE`, ids).Scan(&found).Error
	if err != nil {
		return result, err
	}
	for _, f := range found {
		result.FoundIDs[f.ID] = true
		result.StoragePaths = append(result.StoragePaths, f.StoragePath)
	}

	// 2. Cek dipakai soal aktif.
	var usage []struct {
		ImageID string
		Codes   string
	}
	if err := tx.Raw(imageUsageQuery, ids, ids).Scan(&usage).Error; err != nil {
		return result, err
	}
	for _, u := range usage {
		result.UsedBy[u.ImageID] = u.Codes
	}

	if len(found) != len(ids) || len(usage) > 0 {
		result.StoragePaths = nil
		return result, nil // ROLLBACK lewat defer
	}

	// 3. Hapus. Soal yang sudah di-soft delete otomatis dilepas (image_id → NULL, ON DELETE SET NULL).
	if err := tx.Exec(`DELETE FROM media_assets WHERE id IN ?`, ids).Error; err != nil {
		return result, err
	}
	if err := tx.Commit().Error; err != nil {
		return result, err
	}
	result.Deleted = true
	return result, nil
}
