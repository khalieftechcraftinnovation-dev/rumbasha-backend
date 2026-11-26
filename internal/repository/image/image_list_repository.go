package image

import (
	"context"
	"strings"
	"time"

	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/image"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// listSelect = kolom 1 gambar + nama hierarki mapel & pengunggah (semua JOIN lewat primary key).
// %s = klausa FROM yang berisi alias m.
const listSelect = `
	SELECT m.id, m.category, m.name, m.file_name, m.storage_path, m.url, m.mime_type, m.size_bytes,
	       COALESCE(m.width, 0) AS width, COALESCE(m.height, 0) AS height,
	       m.subject_id, s.name AS subject_name, m.subject_field_id, f.name AS subject_field_name,
	       m.chapter_id, c.name AS chapter_name, m.subchapter_id, sc.name AS subchapter_name,
	       m.topic_id, t.name AS topic_name,
	       COALESCE(m.uploaded_by::text, '') AS uploaded_by, u.real_name AS uploader_name, m.created_at
	%s
	LEFT JOIN master_data_items s  ON s.id  = m.subject_id
	LEFT JOIN master_data_items f  ON f.id  = m.subject_field_id
	LEFT JOIN master_data_items c  ON c.id  = m.chapter_id
	LEFT JOIN master_data_items sc ON sc.id = m.subchapter_id
	LEFT JOIN master_data_items t  ON t.id  = m.topic_id
	LEFT JOIN users u              ON u.id  = m.uploaded_by`

type listRow struct {
	ID, Category, Name, FileName, StoragePath, URL, MimeType string
	SizeBytes                                                int64
	Width, Height                                            int
	SubjectID, SubjectName                                   *string
	SubjectFieldID, SubjectFieldName                         *string
	ChapterID, ChapterName                                   *string
	SubchapterID, SubchapterName                             *string
	TopicID, TopicName                                       *string
	UploadedBy                                               string
	UploaderName                                             *string
	CreatedAt                                                time.Time
}

func ref(id, name *string) *imagemodel.Ref {
	if id == nil {
		return nil
	}
	r := &imagemodel.Ref{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return r
}

func (row listRow) toItem() imagemodel.ListItem {
	item := imagemodel.ListItem{
		Image: imagemodel.Image{
			ID: row.ID, Category: row.Category, Name: row.Name, FileName: row.FileName, StoragePath: row.StoragePath,
			URL: row.URL, MimeType: row.MimeType, SizeBytes: row.SizeBytes, Width: row.Width, Height: row.Height,
			SubjectID: row.SubjectID, SubjectFieldID: row.SubjectFieldID, ChapterID: row.ChapterID,
			SubchapterID: row.SubchapterID, TopicID: row.TopicID, UploadedBy: row.UploadedBy, CreatedAt: row.CreatedAt,
		},
		Hierarchy: imagemodel.Hierarchy{
			Subject: ref(row.SubjectID, row.SubjectName), SubjectField: ref(row.SubjectFieldID, row.SubjectFieldName),
			Chapter: ref(row.ChapterID, row.ChapterName), Subchapter: ref(row.SubchapterID, row.SubchapterName),
			Topic: ref(row.TopicID, row.TopicName),
		},
		UsedBy: []string{},
	}
	if row.UploadedBy != "" && row.UploaderName != nil {
		item.Uploader = &imagemodel.Ref{ID: row.UploadedBy, Name: *row.UploaderName}
	}
	return item
}

// sortSQL = urutan galeri. Selalu diakhiri id supaya pagination stabil.
var sortSQL = map[string]string{
	imagemodel.SortNewest:  "m.created_at DESC, m.id DESC",
	imagemodel.SortOldest:  "m.created_at ASC, m.id ASC",
	imagemodel.SortName:    "LOWER(m.name) ASC, m.created_at DESC, m.id DESC",
	imagemodel.SortLargest: "m.size_bytes DESC, m.created_at DESC, m.id DESC",
}

// listWhere menyusun WHERE galeri. withCategory = false untuk jumlah per kategori.
func listWhere(f imagemodel.ListFilter, withCategory bool) (string, []any) {
	conds := []string{"m.deleted_at IS NULL"}
	args := []any{}
	filters := []struct{ column, value string }{
		{"m.subject_id::text", f.SubjectID},
		{"m.subject_field_id::text", f.SubjectFieldID},
		{"m.chapter_id::text", f.ChapterID},
		{"m.subchapter_id::text", f.SubchapterID},
		{"m.topic_id::text", f.TopicID},
	}
	if withCategory {
		filters = append(filters, struct{ column, value string }{"m.category", f.Category})
	}
	for _, e := range filters {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		conds = append(conds, "(m.name ILIKE ? OR m.file_name ILIKE ?)")
		args = append(args, like, like)
	}
	return strings.Join(conds, " AND "), args
}

// List mengambil 1 halaman galeri + total. [Native SQL]
//
// Halaman diambil dulu (CTE: index media_assets_list_idx, atau media_assets_category_created_idx
// kalau difilter kategori; cari memakai index trigram nama & nama file), baru di-JOIN ke nama
// hierarki & pengunggah untuk ≤ per_page baris. COUNT dilewati di halaman terakhir yang tidak penuh.
func (r *ImageRepository) List(ctx context.Context, f imagemodel.ListFilter) ([]imagemodel.ListItem, int, error) {
	where, args := listWhere(f, true)
	order := sortSQL[f.Sort]
	if order == "" {
		order = sortSQL[imagemodel.SortNewest]
	}

	var rows []listRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT m.id, ROW_NUMBER() OVER (ORDER BY `+order+`) AS n
			FROM media_assets m
			WHERE `+where+`
			ORDER BY `+order+`
			LIMIT ? OFFSET ?
		)`+strings.Replace(listSelect, "%s", "FROM page p JOIN media_assets m ON m.id = p.id", 1)+`
		ORDER BY p.n`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]imagemodel.ListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toItem())
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM media_assets m WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// FindByIDs mengambil gambar berdasarkan ID (urutan sesuai ids; ID yang tidak ada dilewati). [Native SQL]
func (r *ImageRepository) FindByIDs(ctx context.Context, ids []string) ([]imagemodel.ListItem, error) {
	var rows []listRow
	err := r.db.WithContext(ctx).
		Raw(strings.Replace(listSelect, "%s", "FROM media_assets m", 1)+` WHERE m.id IN ? AND m.deleted_at IS NULL`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[string]imagemodel.ListItem, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.toItem()
	}
	items := make([]imagemodel.ListItem, 0, len(rows))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// CountByCategory menghitung gambar per kategori dengan filter yang sama seperti galeri (kategori
// diabaikan), 1 query GROUP BY. [Native SQL]
func (r *ImageRepository) CountByCategory(ctx context.Context, f imagemodel.ListFilter) (imagemodel.CategoryCounts, error) {
	where, args := listWhere(f, false)
	var rows []struct {
		Category string
		Total    int
	}
	err := r.db.WithContext(ctx).Raw(`SELECT m.category, COUNT(*)::int AS total FROM media_assets m WHERE `+where+` GROUP BY m.category`, args...).
		Scan(&rows).Error
	var c imagemodel.CategoryCounts
	for _, row := range rows {
		switch row.Category {
		case imagemodel.CategoryQuestion:
			c.Question = row.Total
		case imagemodel.CategoryBanner:
			c.Banner = row.Total
		case imagemodel.CategoryLanding:
			c.Landing = row.Total
		}
	}
	return c, err
}

// UsedBy mencari kode soal aktif yang memakai setiap gambar (gambar soal / opsi jawaban), 1 query
// untuk semua ID (query sama dengan pengecekan hapus). Hasil: id gambar → kode soal urut.
func (r *ImageRepository) UsedBy(ctx context.Context, ids []string) (map[string][]string, error) {
	result := map[string][]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		ImageID string
		Codes   string
	}
	if err := r.db.WithContext(ctx).Raw(imageUsageQuery, ids, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ImageID] = strings.Split(row.Codes, ", ")
	}
	return result, nil
}
