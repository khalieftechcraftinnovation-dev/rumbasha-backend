// Package image (dto) berisi bentuk request dan response untuk fitur Image Library.
//
// Di package lain, import dengan alias imagedto:
//
//	imagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/image"
package image

import (
	"time"

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

// ImageResponse adalah data 1 gambar yang dikirim ke client.
type ImageResponse struct {
	ID             string    `json:"id"`
	Category       string    `json:"category"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	FileName       string    `json:"file_name"`
	MimeType       string    `json:"mime_type"`
	SizeBytes      int64     `json:"size_bytes"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	SubjectID      *string   `json:"subject_id"`
	SubjectFieldID *string   `json:"subject_field_id"`
	ChapterID      *string   `json:"chapter_id"`
	SubchapterID   *string   `json:"subchapter_id"`
	TopicID        *string   `json:"topic_id"`
	UploadedBy     string    `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToImageResponses mengubah daftar gambar menjadi response (urutan tetap).
func ToImageResponses(images []imagemodel.Image) []ImageResponse {
	result := make([]ImageResponse, 0, len(images))
	for _, img := range images {
		result = append(result, ImageResponse{
			ID: img.ID, Category: img.Category, Name: img.Name, URL: img.URL,
			FileName: img.FileName, MimeType: img.MimeType, SizeBytes: img.SizeBytes,
			Width: img.Width, Height: img.Height,
			SubjectID: img.SubjectID, SubjectFieldID: img.SubjectFieldID, ChapterID: img.ChapterID,
			SubchapterID: img.SubchapterID, TopicID: img.TopicID,
			UploadedBy: img.UploadedBy, CreatedAt: img.CreatedAt,
		})
	}
	return result
}
