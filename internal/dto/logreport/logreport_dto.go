// Package logreport (dto) berisi bentuk request dan response JSON untuk fitur kirim log ke email.
//
// Di package lain, import dengan alias logreportdto:
//
//	logreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/logreport"
package logreport

// SendRequest adalah body JSON untuk POST /api/v1/log-reports.
type SendRequest struct {
	// Penjelasan bug dari admin: apa yang dilakukan, apa yang terjadi, kira-kira jam berapa.
	Description string `json:"description" binding:"required,max=2000"`
	// Opsional: nilai header X-Request-ID dari response yang error, supaya mudah dicari di log.
	RequestID string `json:"request_id" binding:"max=100"`
}

// SendResponse adalah info file log yang dikirim.
type SendResponse struct {
	FileName   string `json:"file_name"`  // contoh api-2026-11-23.txt (atau .txt.gz kalau dikompres)
	FileSize   int    `json:"file_size"`  // ukuran lampiran dalam byte
	Compressed bool   `json:"compressed"` // true kalau file besar sehingga dikompres (gzip)
}
