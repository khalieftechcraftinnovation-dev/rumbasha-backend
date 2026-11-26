package image

import (
	"context"
	"strings"

	"github.com/google/uuid"

	imagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/image"
	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/image"
)

// DefaultPerPage = baris per halaman galeri (contoh desain: 15).
const DefaultPerPage = 15

// filterFrom menyusun filter galeri dari query (default halaman 1, 15 per halaman, terbaru dulu).
func filterFrom(q imagedto.ListQuery) imagemodel.ListFilter {
	f := imagemodel.ListFilter{
		Category: q.Category, Search: strings.TrimSpace(q.Search),
		SubjectID: q.SubjectID, SubjectFieldID: q.SubjectFieldID, ChapterID: q.ChapterID,
		SubchapterID: q.SubchapterID, TopicID: q.TopicID,
		Sort: q.Sort, Page: q.Page, PerPage: q.PerPage,
	}
	if f.Sort == "" {
		f.Sort = imagemodel.SortNewest
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = DefaultPerPage
	}
	return f
}

// List mengambil 1 halaman galeri + soal yang memakai setiap gambar (2 query untuk 1 halaman).
func (s *ImageService) List(ctx context.Context, q imagedto.ListQuery) ([]imagemodel.ListItem, imagemodel.ListFilter, int, error) {
	f := filterFrom(q)
	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, f, 0, err
	}
	return items, f, total, s.attachUsedBy(ctx, items)
}

// CategoryCounts menghitung gambar per kategori (filter search & hierarki ikut, kategori diabaikan).
func (s *ImageService) CategoryCounts(ctx context.Context, q imagedto.ListQuery) (imagemodel.CategoryCounts, error) {
	return s.repo.CountByCategory(ctx, filterFrom(q))
}

// Get mengambil 1 gambar + soal yang memakainya. Bukan UUID / tidak ada → ErrImageNotFound.
func (s *ImageService) Get(ctx context.Context, id string) (imagemodel.ListItem, error) {
	if uuid.Validate(id) != nil {
		return imagemodel.ListItem{}, imagemodel.ErrImageNotFound
	}
	items, err := s.repo.FindByIDs(ctx, []string{id})
	if err != nil {
		return imagemodel.ListItem{}, err
	}
	if len(items) == 0 {
		return imagemodel.ListItem{}, imagemodel.ErrImageNotFound
	}
	if err := s.attachUsedBy(ctx, items); err != nil {
		return imagemodel.ListItem{}, err
	}
	return items[0], nil
}

func (s *ImageService) attachUsedBy(ctx context.Context, items []imagemodel.ListItem) error {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	used, err := s.repo.UsedBy(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		if codes, ok := used[items[i].ID]; ok {
			items[i].UsedBy = codes
		}
	}
	return nil
}
