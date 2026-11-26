// Package image (dto) berisi bentuk request dan response untuk fitur Image Library.
//
// Di package lain, import dengan alias imagedto:
//
//	imagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/image"
package image

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/image"
)

// UploadImagesRequest adalah field teks di form POST /api/v1/images (multipart/form-data).
// File gambarnya dikirim di field "files" (boleh lebih dari satu).
// Semua file dalam 1 request memakai kategori & hierarki yang sama; nama diberi nomor urut
// kalau file lebih dari satu ("Segitiga 1", "Segitiga 2", ...).
type UploadImagesRequest struct {
	Category       string  `form:"category" binding:"required,oneof=question banner landing"`
	Name           string  `form:"name" binding:"required,max=150"`
	SubjectID      *string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID *string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID      *string `form:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID   *string `form:"subchapter_id" binding:"omitempty,uuid"`
	TopicID        *string `form:"topic_id" binding:"omitempty,uuid"`
}

// DeleteImagesRequest adalah body DELETE /api/v1/images: hapus permanen 1 sampai 100 gambar.
type DeleteImagesRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteImagesResponse berisi ID gambar yang berhasil dihapus.
type DeleteImagesResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// ListQuery adalah query string GET /api/v1/images (galeri). Semua opsional.
//
//	?category=question&search=segitiga&chapter_id=...&sort=newest&page=1&per_page=15
type ListQuery struct {
	Category       string `form:"category" binding:"omitempty,oneof=question banner landing"`
	Search         string `form:"search" binding:"omitempty,max=150"` // nama gambar / nama file
	SubjectID      string `form:"subject_id" binding:"omitempty,uuid"`
	SubjectFieldID string `form:"subject_field_id" binding:"omitempty,uuid"`
	ChapterID      string `form:"chapter_id" binding:"omitempty,uuid"`
	SubchapterID   string `form:"subchapter_id" binding:"omitempty,uuid"`
	TopicID        string `form:"topic_id" binding:"omitempty,uuid"`
	Sort           string `form:"sort" binding:"omitempty,oneof=newest oldest name largest"` // default newest
	Page           int    `form:"page" binding:"omitempty,min=1"`
	PerPage        int    `form:"per_page" binding:"omitempty,min=1,max=100"` // default 15
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *imagemodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// HierarchyResponse = nama hierarki mapel gambar soal (null = tidak diisi).
type HierarchyResponse struct {
	Subject      *RefResponse `json:"subject"`
	SubjectField *RefResponse `json:"subject_field"`
	Chapter      *RefResponse `json:"chapter"`
	Subchapter   *RefResponse `json:"subchapter"`
	Topic        *RefResponse `json:"topic"`
}

// ImageResponse adalah data 1 gambar (galeri, detail, hasil upload).
type ImageResponse struct {
	ID             string            `json:"id"`
	Category       string            `json:"category"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	FileName       string            `json:"file_name"`
	MimeType       string            `json:"mime_type"`
	SizeBytes      int64             `json:"size_bytes"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	SubjectID      *string           `json:"subject_id"`
	SubjectFieldID *string           `json:"subject_field_id"`
	ChapterID      *string           `json:"chapter_id"`
	SubchapterID   *string           `json:"subchapter_id"`
	TopicID        *string           `json:"topic_id"`
	Hierarchy      HierarchyResponse `json:"hierarchy"` // nama hierarki (id sama dengan *_id di atas)
	UploadedBy     string            `json:"uploaded_by"`
	Uploader       *RefResponse      `json:"uploader"`   // id + nama pengunggah
	UsedBy         []string          `json:"used_by"`    // kode soal aktif yang memakai gambar, contoh ["Q-12"]
	CanDelete      bool              `json:"can_delete"` // tidak dipakai soal aktif
	CreatedAt      time.Time         `json:"created_at"` // WIB
}

// ToImageResponse mengubah 1 gambar menjadi response.
func ToImageResponse(img imagemodel.ListItem) ImageResponse {
	h := img.Hierarchy
	usedBy := img.UsedBy
	if usedBy == nil {
		usedBy = []string{}
	}
	return ImageResponse{
		ID: img.ID, Category: img.Category, Name: img.Name, URL: img.URL,
		FileName: img.FileName, MimeType: img.MimeType, SizeBytes: img.SizeBytes,
		Width: img.Width, Height: img.Height,
		SubjectID: img.SubjectID, SubjectFieldID: img.SubjectFieldID, ChapterID: img.ChapterID,
		SubchapterID: img.SubchapterID, TopicID: img.TopicID,
		Hierarchy: HierarchyResponse{
			Subject: toRef(h.Subject), SubjectField: toRef(h.SubjectField), Chapter: toRef(h.Chapter),
			Subchapter: toRef(h.Subchapter), Topic: toRef(h.Topic),
		},
		UploadedBy: img.UploadedBy, Uploader: toRef(img.Uploader),
		UsedBy: usedBy, CanDelete: len(usedBy) == 0,
		CreatedAt: img.CreatedAt.In(clock.Jakarta),
	}
}

// ToImageResponses mengubah daftar gambar menjadi response (urutan tetap). Kosong = [].
func ToImageResponses(images []imagemodel.ListItem) []ImageResponse {
	result := make([]ImageResponse, 0, len(images))
	for _, img := range images {
		result = append(result, ToImageResponse(img))
	}
	return result
}

// CategoryCountsResponse = jumlah gambar per kategori (filter search & hierarki ikut).
type CategoryCountsResponse struct {
	All      int `json:"all"`
	Question int `json:"question"`
	Banner   int `json:"banner"`
	Landing  int `json:"landing"`
}

// ToCategoryCountsResponse mengubah jumlah per kategori menjadi response.
func ToCategoryCountsResponse(c imagemodel.CategoryCounts) CategoryCountsResponse {
	return CategoryCountsResponse{All: c.Question + c.Banner + c.Landing, Question: c.Question, Banner: c.Banner, Landing: c.Landing}
}
