# Rumbasha Backend

Backend API Rumbasha E-Learning dengan Go, framework [Gin](https://gin-gonic.com/), dan PostgreSQL.

## Persiapan (sekali saja)

1. **Go 1.27.** Kalau `go version` masih versi lama, jalankan: `go env -w GOTOOLCHAIN=go1.27.4`
2. **File `.env`.** Salin dari contoh, lalu isi `DB_PASSWORD`:

   ```bash
   cp .env.example .env
   ```

3. **Buat database** `rumbasha` dan `rumbasha_test`: buka [scripts/db/create_database.sql](scripts/db/create_database.sql) di DBeaver (koneksi `postgres`), lalu jalankan sebagai script dengan Alt+X.

4. **Buat tabel** dengan migration:

   ```bash
   make migrate-up
   ```

## Menjalankan

```bash
make run                              # atau: go run ./cmd/api
curl http://localhost:8080/health
```

| Perintah               | Fungsi                                         |
|------------------------|------------------------------------------------|
| `make run`             | Menjalankan API                                |
| `make test`            | Menjalankan semua test (pakai `rumbasha_test`) |
| `make vet`             | Mengecek kode yang mencurigakan                |
| `make fmt`             | Merapikan format kode                          |
| `make build`           | Build ke folder `bin/`                         |
| `make check`           | `vet` + `test` (sebelum push)                  |
| `make migrate-up`      | Menjalankan migration yang belum               |
| `make migrate-down`    | Membatalkan 1 migration terakhir               |
| `make migrate-version` | Melihat versi migration saat ini               |

## Struktur folder

```text
cmd/
  api/main.go             Titik awal aplikasi: konek DB, susun layer, jalankan server.
  migrate/main.go         Program untuk menjalankan migration.
migrations/               File SQL pembuat tabel (*.up.sql & *.down.sql).
internal/
  router/router.go        ROUTE       Semua URL → handler ada di satu file ini.
  handler/                HANDLER     Baca request, panggil service, kirim response.
  service/                SERVICE     Aturan bisnis.
  repository/             REPOSITORY  Query database (GORM atau native SQL).
  model/                  Struct yang mewakili tabel + daftar error.
  dto/                    Bentuk request & response JSON.
  response/               Helper format JSON (sukses, error, validasi).
  database/               Koneksi PostgreSQL + runner migration.
  config/                 Membaca konfigurasi dari .env.
  testutil/               Helper untuk test (database test).
```

## Alur request

```text
GET /api/v1/subjects
  → router      (router.go)              mencari handler untuk URL ini
  → handler     (subject_handler.go)     SubjectHandler.List
  → service     (subject_service.go)     SubjectService.List
  → repository  (subject_repository.go)  SubjectRepository.FindAll → PostgreSQL
```

Saat debug, ikuti alur di atas: Ctrl+klik nama fungsinya di VS Code untuk langsung melompat ke layer berikutnya.
Dengan `DB_DEBUG=true` di `.env`, setiap query SQL ikut tercetak di terminal.

## Database

### Kapan pakai GORM, kapan native SQL

Contoh keduanya ada di [subject_repository.go](internal/repository/subject_repository.go).

| Kebutuhan                            | Pakai                                        |
|--------------------------------------|----------------------------------------------|
| Insert, update, delete, get by ID    | GORM: `Create`, `Updates`, `Delete`, `First` |
| List dengan filter, JOIN 1–2 tabel   | GORM atau native SQL, mana yang lebih jelas  |
| JOIN banyak tabel, agregasi, laporan | Native SQL: `db.Raw(...).Scan(...)`          |

Di native SQL, selalu pakai `?` untuk nilai dari user, jangan sambung string:

```go
db.Raw(`SELECT * FROM subjects WHERE code = ?`, code)   // ✅ aman
db.Raw(`SELECT * FROM subjects WHERE code = '` + code + `'`) // ❌ SQL injection
```

### Tabel

Semua tabel ada di satu schema (`public`), jadi cukup ditulis `SELECT ... FROM users` tanpa awalan schema.

| Domain     | Tabel                                                                                          |
|------------|------------------------------------------------------------------------------------------------|
| Master     | `branches` (cabang), `master_data_items` (data acuan generik: jenjang, kelas)                  |
| Auth/akun  | `users`, `roles`, `permissions`, `role_permissions`, `refresh_tokens`, `password_reset_tokens` |
| Siswa      | `students` (data tambahan siswa: kode siswa, tanggal lahir, target sekolah)                    |
| Tutor      | `tutors` (profil tutor), `tutor_students` (siswa bimbingan)                                    |
| Paket      | `features`, `subscription_plans`, `subscription_plan_prices`, `subscription_plan_features`,    |
|            | `student_subscriptions`, `subscription_histories`, `student_feature_overrides`                 |
| Coin       | `coin_packages`, `student_coin_balances`, `coin_transactions`, `coin_purchases`,               |
|            | `daily_quota_usages`                                                                           |
| Lain-lain  | `subjects` (contoh awal), `schema_migrations` (catatan versi migration, jangan diubah manual)  |

Supaya tetap mudah dipecah jadi microservice nanti: data milik satu domain hanya ditulis oleh repository domain itu sendiri.

### Mengubah struktur tabel

Struktur tabel hanya diubah lewat file migration, tidak lewat GORM AutoMigrate dan tidak lewat DBeaver secara manual.

1. Buat pasangan file dengan nomor berikutnya di `migrations/`:
   `000002_create_materials_table.up.sql` dan `000002_create_materials_table.down.sql`
2. Tulis SQL-nya (`.down.sql` berisi kebalikannya). Boleh dicoba dulu di DBeaver.
3. Jalankan `make migrate-up`.
4. Tambahkan nama tabel baru ke `TRUNCATE` di [internal/testutil/database.go](internal/testutil/database.go).

Jangan mengubah file migration yang sudah pernah dijalankan; buat file baru.
Kalau migration gagal di tengah jalan (status `dirty`), perbaiki tabelnya, lalu `go run ./cmd/migrate force <versi-terakhir-yang-benar>`.

## Aturan

1. **Arah panggilan hanya ke bawah:** handler → service → repository. Handler tidak boleh memanggil repository langsung.
2. **Aturan bisnis hanya di service.** Handler cukup baca request dan kirim response.
3. **Semua layer disusun di `main.go`.** Setiap layer menerima layer di bawahnya lewat parameter `New...()`.
4. **Satu domain = satu file per layer**, misalnya `subject_handler.go`, `subject_service.go`, `subject_repository.go`.
5. **Service domain A tidak boleh memakai repository domain B.** Kalau butuh data domain lain, panggil service-nya. Ini yang membuat project mudah dipecah jadi microservice nanti.

## Menambah domain baru (contoh: `material`)

1. `migrations/00000N_create_materials_table.up.sql` + `.down.sql`, lalu `make migrate-up`.
2. `internal/model/material.go`: struct `Material` + `TableName()` + error-nya.
3. `internal/dto/material.go`: struct request & response.
4. `internal/repository/material_repository.go`: query database.
5. `internal/service/material_service.go`: logika bisnis.
6. `internal/handler/material_handler.go`: handler + `handleError`.
7. `internal/router/router.go`: tambah field di `Handlers` dan daftarkan route-nya.
8. `cmd/api/main.go`: susun repository → service → handler.

## Endpoint

| Method | URL                      | Fungsi              |
|--------|--------------------------|---------------------|
| GET    | `/health`                | Cek server hidup    |
| GET    | `/api/v1/subjects`       | Daftar (pagination) |
| POST   | `/api/v1/subjects`       | Tambah              |
| GET    | `/api/v1/subjects/:id`   | Detail              |
| PUT    | `/api/v1/subjects/:id`   | Ubah                |
| DELETE | `/api/v1/subjects/:id`   | Hapus               |

## Format response

```json
{ "success": true,  "message": "...", "data": {}, "meta": { "page": 1, "per_page": 20, "total": 1 } }
{ "success": false, "message": "Validasi gagal", "errors": [{ "field": "code", "message": "wajib diisi" }] }
```

Status: 422 validasi gagal, 400 JSON rusak, 404 data tidak ada, 409 data kembar, 500 error server (detail hanya di log).
