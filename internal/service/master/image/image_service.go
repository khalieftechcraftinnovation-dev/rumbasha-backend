// Package image (service) berisi aturan bisnis Image Library: upload & hapus gambar.
//
// Upload (semua atau tidak sama sekali):
//  1. Cek field form, jumlah file, ukuran, dan isi file (harus benar-benar gambar).
//  2. Cek hierarki mapel ke database (1 query).
//  3. Simpan semua file ke disk, lalu simpan data dengan 1 query INSERT.
//     Kalau langkah 3 gagal, file yang sudah tersimpan dihapus lagi.
//
// Hapus (hard delete, semua atau tidak sama sekali): ditolak kalau ada ID yang tidak ada
// atau gambar masih dipakai soal aktif. File di disk dihapus setelah database berhasil.
//
// Di package lain, import dengan alias imageservice:
//
//	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/image"
package image

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	_ "image/gif"  // daftarkan decoder GIF untuk image.DecodeConfig
	_ "image/jpeg" // daftarkan decoder JPEG
	_ "image/png"  // daftarkan decoder PNG
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // daftarkan decoder WebP

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	imagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/image"
	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/image"
	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/image"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
)

// Batas upload. Ubah di sini kalau perlu.
const (
	MaxFilesPerUpload = 10      // jumlah file per request
	MaxFileSize       = 5 << 20 // 5 MB per file
)

// allowedTypes adalah jenis gambar yang diterima → ekstensi file yang disimpan.
// Jenis dibaca dari ISI file (bukan dari nama file), jadi file .exe yang diganti namanya
// menjadi .png tetap ditolak. SVG sengaja tidak diterima (bisa berisi script).
var allowedTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadFile adalah 1 file dari form. Data nil kalau file terlalu besar (tidak dibaca handler).
type UploadFile struct {
	FileName string
	Size     int64
	Data     []byte
}

// ImageService berisi logika bisnis Image Library.
type ImageService struct {
	repo    *imagerepo.ImageRepository
	storage *storage.Local
}

// NewImageService membuat ImageService. Dipanggil sekali di main.go.
func NewImageService(repo *imagerepo.ImageRepository, store *storage.Local) *ImageService {
	return &ImageService{repo: repo, storage: store}
}

// UploadMany menyimpan 1 sampai MaxFilesPerUpload gambar dengan kategori & hierarki yang sama.
// Hasilnya urut sesuai urutan file di request.
func (s *ImageService) UploadMany(ctx context.Context, req imagedto.UploadImagesRequest, files []UploadFile, uploadedBy string) ([]imagemodel.ListItem, error) {
	errs := &fieldErrors{}

	// ---- 1. Cek tanpa database --------------------------------------------------
	name := strings.TrimSpace(req.Name)
	if name == "" {
		errs.add("name", "wajib diisi")
	}
	checkHierarchyOrder(req, errs)

	switch {
	case len(files) == 0:
		errs.add("files", "wajib diisi minimal 1 gambar")
	case len(files) > MaxFilesPerUpload:
		errs.add("files", fmt.Sprintf("maksimal %d gambar per upload", MaxFilesPerUpload))
	}

	now := time.Now().In(clock.Jakarta)
	images := make([]imagemodel.Image, 0, len(files))
	for i, f := range files {
		img, problem := inspectFile(f)
		if problem != "" {
			errs.add(fmt.Sprintf("files[%d]", i), fmt.Sprintf("%s: %s", f.FileName, problem))
			continue
		}

		img.ID = uuid.NewString()
		img.Category = req.Category
		img.Name = name
		if len(files) > 1 {
			img.Name = fmt.Sprintf("%s %d", name, i+1) // nomor urut di belakang nama
		}
		img.StoragePath = fmt.Sprintf("images/%s/%04d/%02d/%s%s", req.Category, now.Year(), now.Month(), img.ID, allowedTypes[img.MimeType])
		img.URL = s.storage.URL(img.StoragePath)
		img.SubjectID, img.SubjectFieldID, img.ChapterID = req.SubjectID, req.SubjectFieldID, req.ChapterID
		img.SubchapterID, img.TopicID = req.SubchapterID, req.TopicID
		img.UploadedBy = uploadedBy
		images = append(images, img)
	}
	if errs.any() {
		return nil, errs.toError()
	}

	// ---- 2. Cek hierarki mapel ke database -------------------------------------
	if err := s.checkHierarchy(ctx, req, errs); err != nil {
		return nil, err
	}
	if errs.any() {
		return nil, errs.toError()
	}

	// ---- 3. Simpan file, lalu data ----------------------------------------------
	saved := make([]string, 0, len(images))
	for i, img := range images {
		if err := s.storage.Save(img.StoragePath, files[i].Data); err != nil {
			s.removeFiles(saved)
			return nil, fmt.Errorf("gagal menyimpan file %s: %w", files[i].FileName, err)
		}
		saved = append(saved, img.StoragePath)
	}
	if err := s.repo.CreateMany(ctx, images); err != nil {
		s.removeFiles(saved)
		return nil, fmt.Errorf("gagal menyimpan data %d gambar: %w", len(images), err)
	}
	// Baca ulang supaya respons sama dengan galeri (nama hierarki & pengunggah).
	ids := make([]string, 0, len(images))
	for _, img := range images {
		ids = append(ids, img.ID)
	}
	return s.repo.FindByIDs(ctx, ids)
}

// DeleteMany menghapus permanen 1 sampai 100 gambar (data + file).
func (s *ImageService) DeleteMany(ctx context.Context, ids []string) ([]string, error) {
	result, err := s.repo.DeleteMany(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("gagal menghapus gambar: %w", err)
	}

	if !result.Deleted {
		errs := &fieldErrors{}
		for i, id := range ids {
			field := fmt.Sprintf("ids[%d]", i)
			switch {
			case !result.FoundIDs[id]:
				errs.add(field, "gambar tidak ditemukan")
			case result.UsedBy[id] != "":
				errs.add(field, "masih dipakai soal "+result.UsedBy[id]+", ganti/hapus gambarnya dari soal dulu")
			}
		}
		return nil, errs.toError()
	}

	// Data sudah terhapus. File yang gagal dihapus hanya dicatat di log (tidak membatalkan hapus),
	// supaya tidak ada data yang menunjuk ke file yang sudah hilang.
	s.removeFiles(result.StoragePaths)
	return ids, nil
}

// removeFiles menghapus file dari disk; kegagalan hanya dicatat di log server.
func (s *ImageService) removeFiles(paths []string) {
	for _, p := range paths {
		if err := s.storage.Remove(p); err != nil {
			log.Printf("peringatan: %v", err)
		}
	}
}

// inspectFile mengecek ukuran & isi 1 file, lalu mengisi jenis, ukuran, lebar, dan tinggi gambar.
// problem berisi alasan penolakan (kosong = file valid).
func inspectFile(f UploadFile) (imagemodel.Image, string) {
	img := imagemodel.Image{FileName: f.FileName, SizeBytes: f.Size}
	switch {
	case f.Size > MaxFileSize || f.Data == nil:
		return img, fmt.Sprintf("ukuran maksimal %d MB", MaxFileSize>>20)
	case len(f.Data) == 0:
		return img, "file kosong"
	}

	img.MimeType = http.DetectContentType(f.Data) // dibaca dari isi file (maks 512 byte pertama)
	if _, ok := allowedTypes[img.MimeType]; !ok {
		return img, "harus gambar PNG, JPG, GIF, atau WebP"
	}

	config, _, err := stdimage.DecodeConfig(bytes.NewReader(f.Data))
	if err != nil {
		return img, "file gambar rusak atau tidak bisa dibaca"
	}
	img.Width, img.Height = config.Width, config.Height
	return img, ""
}

// checkHierarchyOrder: hierarki hanya untuk kategori question, dan harus diisi berurutan
// dari atas (bidang materi butuh mapel, bab butuh bidang materi, dst.).
func checkHierarchyOrder(req imagedto.UploadImagesRequest, errs *fieldErrors) {
	levels := []struct {
		field string
		value *string
	}{
		{"subject_id", req.SubjectID}, {"subject_field_id", req.SubjectFieldID}, {"chapter_id", req.ChapterID},
		{"subchapter_id", req.SubchapterID}, {"topic_id", req.TopicID},
	}

	for i, lv := range levels {
		if lv.value == nil {
			continue
		}
		if req.Category != imagemodel.CategoryQuestion {
			errs.add(lv.field, "hanya untuk kategori question")
			continue
		}
		if i > 0 && levels[i-1].value == nil {
			errs.add(levels[i-1].field, "wajib diisi jika "+lv.field+" diisi")
		}
	}
}

// checkHierarchy mengecek ID hierarki ada, tipenya benar, aktif, dan saling berhubungan,
// dengan 1 query ke master_data_items.
func (s *ImageService) checkHierarchy(ctx context.Context, req imagedto.UploadImagesRequest, errs *fieldErrors) error {
	if req.SubjectID == nil {
		return nil // tanpa hierarki (checkHierarchyOrder memastikan yang lain juga kosong)
	}

	ids := []string{*req.SubjectID}
	for _, id := range []*string{req.SubjectFieldID, req.ChapterID, req.SubchapterID, req.TopicID} {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	items, err := s.repo.FindHierarchyByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("gagal membaca master data: %w", err)
	}

	// check mengecek 1 tingkat: ada, tipenya sesuai, aktif, dan cocok dengan induknya.
	check := func(field, label, itemType string, id *string, parentOK func(imagemodel.HierarchyItem) bool, parentLabel string) bool {
		if id == nil {
			return false
		}
		item, ok := items[*id]
		switch {
		case !ok || item.Type != itemType:
			errs.add(field, label+" tidak ditemukan")
		case item.Status != "active":
			errs.add(field, label+" tidak aktif")
		case parentOK != nil && !parentOK(item):
			errs.add(field, label+" bukan bagian dari "+parentLabel+" yang dipilih")
		default:
			return true
		}
		return false
	}
	isChildOf := func(parent *string) func(imagemodel.HierarchyItem) bool {
		return func(item imagemodel.HierarchyItem) bool {
			return item.ParentID != nil && parent != nil && *item.ParentID == *parent
		}
	}

	_ = check("subject_id", "mapel", "subject", req.SubjectID, nil, "") &&
		check("subject_field_id", "bidang materi", "subject_field", req.SubjectFieldID, isChildOf(req.SubjectID), "mapel") &&
		check("chapter_id", "bab", "chapter", req.ChapterID, isChildOf(req.SubjectFieldID), "bidang materi") &&
		check("subchapter_id", "sub bab", "subchapter", req.SubchapterID, isChildOf(req.ChapterID), "bab") &&
		check("topic_id", "topik", "topic", req.TopicID, isChildOf(req.SubchapterID), "sub bab")
	return nil
}

// fieldErrors mengumpulkan kesalahan per field.
type fieldErrors struct {
	list []imagemodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, imagemodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool { return len(e.list) > 0 }

func (e *fieldErrors) toError() error { return &imagemodel.ValidationError{Fields: e.list} }
