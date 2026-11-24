// Package storage menyimpan file upload di disk server (folder STORAGE_DIR).
//
// File disajikan publik oleh router di URL: <APP_URL>/storage/<path relatif>.
// Kalau nanti pindah ke object storage (S3, dsb.), cukup ganti package ini.
package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PublicPrefix adalah awalan URL untuk file di storage, contoh /storage/images/question/...
const PublicPrefix = "/storage"

// Local menyimpan file di folder Dir.
type Local struct {
	Dir     string // folder dasar, contoh "storage"
	BaseURL string // alamat publik server tanpa "/" di akhir, contoh "http://localhost:8080"
}

// NewLocal membuat Local. Folder dibuat otomatis saat file pertama disimpan.
func NewLocal(dir, baseURL string) *Local {
	return &Local{Dir: dir, BaseURL: strings.TrimRight(baseURL, "/")}
}

// Save menyimpan data ke relPath (contoh "images/question/2026/11/abc.png").
// relPath selalu memakai "/" (bukan "\"), dibuat oleh aplikasi, bukan dari input user.
func (s *Local) Save(relPath string, data []byte) error {
	full, err := s.fullPath(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("gagal membuat folder %s: %w", filepath.Dir(full), err)
	}
	// O_EXCL: gagal kalau file sudah ada (nama memakai UUID, jadi seharusnya tidak pernah terjadi).
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("gagal membuat file %s: %w", relPath, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(full)
		return fmt.Errorf("gagal menulis file %s: %w", relPath, err)
	}
	return f.Close()
}

// Remove menghapus file. File yang sudah tidak ada dianggap berhasil.
func (s *Local) Remove(relPath string) error {
	full, err := s.fullPath(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("gagal menghapus file %s: %w", relPath, err)
	}
	return nil
}

// URL mengembalikan link publik file, contoh http://localhost:8080/storage/images/question/2026/11/abc.png.
func (s *Local) URL(relPath string) string {
	return s.BaseURL + PublicPrefix + "/" + relPath
}

// fullPath mengubah relPath menjadi lokasi di disk, dan menolak path yang keluar dari Dir
// (contoh "../../etc/passwd"), sebagai pengaman tambahan.
func (s *Local) fullPath(relPath string) (string, error) {
	clean := path.Clean("/" + relPath)[1:]
	if clean == "" || clean != relPath {
		return "", fmt.Errorf("path file tidak valid: %q", relPath)
	}
	return filepath.Join(s.Dir, filepath.FromSlash(clean)), nil
}
