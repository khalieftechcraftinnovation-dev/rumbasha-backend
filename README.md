# Rumbasha Backend

Backend API Rumbasha E-Learning dengan Go dan framework [Gin](https://gin-gonic.com/).

## Menjalankan

Butuh Go 1.27. Kalau `go version` masih versi lama, jalankan sekali:
`go env -w GOTOOLCHAIN=go1.27.4`

```bash
cp .env.example .env
make run                              # atau: go run ./cmd/api
curl http://localhost:8080/health
```

| Perintah     | Fungsi                              |
|--------------|-------------------------------------|
| `make run`   | Menjalankan API                     |
| `make test`  | Menjalankan semua test              |
| `make vet`   | Mengecek kode yang mencurigakan     |
| `make fmt`   | Merapikan format kode               |
| `make build` | Build ke folder `bin/`              |
| `make check` | `vet` + `test` (sebelum push)       |

## Struktur folder

```text
cmd/api/main.go           Titik awal aplikasi: menyusun semua layer, lalu menjalankan server.
internal/
  router/router.go        ROUTE       Semua URL → handler ada di satu file ini.
  handler/                HANDLER     Baca request, panggil service, kirim response.
  service/                SERVICE     Aturan bisnis.
  repository/             REPOSITORY  Akses database.
  model/                  Struct data utama + daftar error.
  dto/                    Bentuk request & response JSON.
  response/               Helper format JSON (sukses, error, validasi).
  config/                 Membaca konfigurasi dari .env.
```

## Alur request

```text
GET /api/v1/subjects
  → router      (router.go)              mencari handler untuk URL ini
  → handler     (subject_handler.go)     SubjectHandler.List
  → service     (subject_service.go)     SubjectService.List
  → repository  (subject_repository.go)  SubjectMemoryRepository.FindAll
```

Saat debug, ikuti alur di atas: Ctrl+klik nama fungsinya di VS Code untuk langsung melompat ke layer berikutnya.

## Aturan

1. **Arah panggilan hanya ke bawah:** handler → service → repository. Handler tidak boleh memanggil repository langsung.
2. **Aturan bisnis hanya di service.** Handler cukup baca request dan kirim response.
3. **Semua layer disusun di `main.go`.** Setiap layer menerima layer di bawahnya lewat parameter `New...()`.
4. **Satu domain = satu file per layer**, misalnya `subject_handler.go`, `subject_service.go`, `subject_repository.go`.
5. **Service domain A tidak boleh memakai repository domain B.** Kalau butuh data domain lain, panggil service-nya. Ini yang membuat project mudah dipecah jadi microservice nanti.

## Menambah domain baru (contoh: `material`)

1. `internal/model/material.go`: struct `Material` + error-nya.
2. `internal/dto/material.go`: struct request & response.
3. `internal/repository/material_repository.go`: interface + implementasi.
4. `internal/service/material_service.go`: logika bisnis (+ file `_test.go`).
5. `internal/handler/material_handler.go`: handler + `handleError`.
6. `internal/router/router.go`: tambah field di `Handlers` dan daftarkan route-nya.
7. `cmd/api/main.go`: susun repository → service → handler.

## Format response

```json
{ "success": true,  "message": "...", "data": {}, "meta": { "page": 1, "per_page": 20, "total": 1 } }
{ "success": false, "message": "Validasi gagal", "errors": [{ "field": "code", "message": "wajib diisi" }] }
```

Status: 422 validasi gagal, 400 JSON rusak, 404 data tidak ada, 409 data kembar, 500 error server (detail hanya di log).
