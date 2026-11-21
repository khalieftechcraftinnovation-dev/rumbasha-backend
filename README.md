# Rumbasha Backend

Backend API Rumbasha E-Learning dengan Go, framework [Gin](https://gin-gonic.com/), dan PostgreSQL.

## Persiapan (sekali saja)

1. **Go 1.27.** Kalau `go version` masih versi lama, jalankan: `go env -w GOTOOLCHAIN=go1.27.4`
2. **File `.env`.** Salin dari contoh, lalu isi `DB_PASSWORD` dan `JWT_SECRET` (buat dengan `openssl rand -hex 32`):

   ```bash
   cp .env.example .env
   ```

3. **Buat database** `rumbasha` dan `rumbasha_test`: buka [scripts/db/create_database.sql](scripts/db/create_database.sql) di DBeaver (koneksi `postgres`), lalu jalankan sebagai script dengan Alt+X.

4. **Buat tabel** dengan migration:

   ```bash
   make migrate-up
   ```

5. **Isi data dummy** (akun tiap role, paket, coin). Hanya untuk development, ditolak kalau `APP_ENV=production`:

   ```bash
   make seed
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
| `make seed`            | Mengisi data dummy (sekali, aman diulang)      |

## Akun dummy

Setelah `make seed`, semua akun memakai password **`password123`**. Isinya ada di [cmd/seed/dummy_data.sql](cmd/seed/dummy_data.sql).

| Username         | Role        | Keterangan                                                              |
|------------------|-------------|-------------------------------------------------------------------------|
| `superadmin`     | super_admin | Semua cabang                                                            |
| `admin.sbytimur` | admin       | Cabang Surabaya Timur                                                   |
| `tutor.rina`     | tutor       | Cabang Surabaya Timur                                                   |
| `siswa.budi`     | student     | SMP Kelas 7, paket **offline** Reguler SMP bulan ini, 100 coin          |
| `siswa.citra`    | student     | SD Kelas 5, paket **online** Reguler SD aktif + akses khusus RUMATH     |
| `siswa.dimas`    | student     | SMA Kelas 10, paket online **sudah expired** kemarin                    |
| `siswa.eka`      | student     | PraSD Nol Besar, akun **nonaktif** (login ditolak)                      |

## Struktur folder

```text
cmd/
  api/main.go             Titik awal aplikasi: konek DB, susun layer, jalankan server.
  migrate/main.go         Program untuk menjalankan migration.
  seed/                   Program + file SQL data dummy.
migrations/               File SQL pembuat tabel (*.up.sql & *.down.sql).
internal/
  router/                 ROUTE       Daftar URL → handler.
    router.go               Pintu utama: Gin, middleware global, /health, memanggil route tiap fitur.
    login/                  login_routes.go
    subject/                subject_routes.go
    *_test.go               Test endpoint dari ujung ke ujung (semua fitur).
  handler/                HANDLER     Baca request, panggil service, kirim response.
    login/  subject/
  service/                SERVICE     Aturan bisnis.
    login/  subject/  subscription/
  repository/             REPOSITORY  Query database (GORM atau native SQL).
    login/  subject/  subscription/
  model/                  Struct yang mewakili tabel + daftar error.
    login/  subject/  subscription/
  dto/                    Bentuk request & response JSON.
    pagination.go           Dipakai bersama semua fitur.
    login/  subject/

  middleware/             Cek login (RequireAuth) dan role (RequireRole) sebelum handler.
  clock/                  Aturan waktu WIB (awal bulan, tanggal hari ini).
  token/                  Membuat & membaca access token (JWT) dan refresh token.
  response/               Helper format JSON (sukses, error, validasi).
  database/               Koneksi PostgreSQL + runner migration.
  config/                 Membaca konfigurasi dari .env.
  testutil/               Helper untuk test (database test).
```

Setiap layer dikelompokkan per fitur: `<layer>/<fitur>/<fitur>_<layer>.go`, misalnya `service/login/login_service.go`.
File di root folder layer (misalnya `dto/pagination.go`) dipakai bersama oleh semua fitur.

Fitur saat ini: **login** (`/auth/...`), **subject** (`/subjects/...`), dan **subscription** (paket akses siswa; belum punya endpoint sendiri, dipakai oleh `/auth/me`). Fitur yang tidak punya endpoint sendiri tidak perlu folder di `handler/` dan `router/`.

### Import dengan alias

Di Go, nama package = nama folder terakhir. Karena `service/login`, `model/login`, `dto/login`, dan seterusnya semuanya bernama `login`, setiap import diberi alias `<fitur><layer>` supaya tidak bentrok dan langsung jelas asalnya:

```go
import (
    loginmodel   ".../internal/model/login"
    logindto     ".../internal/dto/login"
    loginrepo    ".../internal/repository/login"
    loginservice ".../internal/service/login"
    loginhandler ".../internal/handler/login"
    loginrouter  ".../internal/router/login"
)

loginService := loginservice.NewLoginService(...)   // jelas: service milik fitur login
if errors.Is(err, loginmodel.ErrUserInactive) {}    // jelas: error milik model fitur login
```

Alias yang dipakai: `<fitur>model`, `<fitur>dto`, `<fitur>repo`, `<fitur>service`, `<fitur>handler`, `<fitur>router`. Komentar di baris pertama setiap package juga menuliskan alias-nya.

## Login & sesi

```text
POST /auth/login   {username, password}
  ← access_token  (JWT, 15 menit)          → kirim di header: Authorization: Bearer <access_token>
  ← refresh_token (sampai tgl 1 bulan depan, 00:00 WIB)

access_token habis? → POST /auth/refresh {refresh_token} → dapat pasangan token baru
keluar             → POST /auth/logout  {refresh_token}
```

- **Semua user wajib login ulang setiap tanggal 1, jam 00:00 WIB.** Refresh tidak bisa memperpanjang sesi melewati batas itu. Ini berkaitan dengan paket offline yang dibayar manual setiap bulan di bimbel.
- **Paket dicek setiap hari** di `student_subscriptions` memakai tanggal hari ini (WIB), untuk paket offline maupun online. Paket yang habis tanggal 15 otomatis tidak berlaku mulai tanggal 16, tanpa perlu login ulang. Lihat `GET /auth/me`.
- **Refresh token sekali pakai.** Setiap refresh menukar token lama dengan yang baru; token lama langsung tidak berlaku.
- Di database hanya disimpan **hash** refresh token dan **hash bcrypt** password.

Grup `public` (tanpa login) dan `private` (wajib login) dibuat di [router.go](internal/router/router.go), lalu diteruskan ke route tiap fitur. Contoh di [subject_routes.go](internal/router/subject/subject_routes.go):

```go
private.GET("/subjects", h.List)                 // wajib login
private.POST("/subjects", adminOnly, h.Create)   // wajib login + role admin/super_admin
```

Di handler, ID dan role user yang login dibaca dengan `middleware.GetUserID(c)` dan `middleware.GetRole(c)`.

## Alur request

```text
GET /api/v1/subjects
  → router      router/subject/subject_routes.go          mencari handler untuk URL ini
  → handler     handler/subject/subject_handler.go        SubjectHandler.List
  → service     service/subject/subject_service.go        SubjectService.List
  → repository  repository/subject/subject_repository.go  SubjectRepository.FindAll → PostgreSQL
```

Saat debug, ikuti alur di atas: Ctrl+klik nama fungsinya di VS Code untuk langsung melompat ke layer berikutnya.
Dengan `DB_DEBUG=true` di `.env`, setiap query SQL ikut tercetak di terminal.

## Database

### Kapan pakai GORM, kapan native SQL

Contoh keduanya ada di [subject_repository.go](internal/repository/subject/subject_repository.go).

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

| Domain    | Tabel                                                                                               |
|-----------|-----------------------------------------------------------------------------------------------------|
| Master    | `branches` (cabang), `master_data_items` (data acuan generik: jenjang & kelas)                      |
| Akun      | `users` (semua akun + role), `refresh_tokens` (sesi login)                                          |
| Siswa     | `students` (kode siswa, tanggal lahir, target sekolah, saldo coin)                                  |
| Paket     | `subscription_plans` (paket + harga + akses fitur), `student_subscriptions` (paket dibeli/riwayat), |
|           | `student_feature_overrides` (akses khusus per siswa)                                                |
| Coin      | `coin_packages` (paket isi coin), `coin_transactions` (riwayat coin, termasuk pembelian)            |
| Lain-lain | `subjects` (contoh awal), `schema_migrations` (catatan versi migration, jangan diubah manual)       |

Tutor adalah user dengan `role = 'tutor'` dan boleh mengajar siswa mana pun. Jatah gratis Daily Quiz berlaku global untuk semua siswa dan diatur di `.env` (`DAILY_QUIZ_FREE_QUOTA`, `DAILY_QUIZ_COIN_COST`).

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
4. **Satu fitur = satu subfolder per layer**, misalnya `handler/subject/`, `service/subject/`, `repository/subject/`.
5. **Service fitur A tidak boleh memakai repository fitur B.** Kalau butuh data fitur lain, panggil service-nya (contoh: `LoginHandler` memakai `SubscriptionService` untuk `/auth/me`). Ini yang membuat project mudah dipecah jadi microservice nanti.

## Menambah fitur baru (contoh: `material`)

1. `migrations/00000N_create_materials_table.up.sql` + `.down.sql`, lalu `make migrate-up`.
2. `internal/model/material/material_model.go`: struct `Material` + `TableName()` + error-nya.
3. `internal/dto/material/material_dto.go`: struct request & response.
4. `internal/repository/material/material_repository.go`: query database.
5. `internal/service/material/material_service.go`: logika bisnis.
6. `internal/handler/material/material_handler.go`: handler + `handleError`.
7. `internal/router/material/material_routes.go`: fungsi `Register(...)` berisi route-nya.
8. `internal/router/router.go`: tambah field di `Handlers` dan panggil `materialrouter.Register(...)`.
9. `cmd/api/main.go`: susun repository → service → handler.

Setiap file baru diawali `package material`, dan diimport dengan alias (`materialmodel`, `materialservice`, ...).

## Endpoint

| Method | URL                    | Akses        | Fungsi                                          |
|--------|------------------------|--------------|-------------------------------------------------|
| GET    | `/health`              | Publik       | Cek server hidup                                |
| POST   | `/api/v1/auth/login`   | Publik       | Login dengan username + password                |
| POST   | `/api/v1/auth/refresh` | Publik       | Tukar refresh token dengan token baru           |
| POST   | `/api/v1/auth/logout`  | Publik       | Tutup sesi                                      |
| GET    | `/api/v1/auth/me`      | Login        | Data user; untuk siswa + paket & fitur hari ini |
| GET    | `/api/v1/subjects`     | Login        | Daftar subject (pagination)                     |
| GET    | `/api/v1/subjects/:id` | Login        | Detail subject                                  |
| POST   | `/api/v1/subjects`     | Admin        | Tambah subject                                  |
| PUT    | `/api/v1/subjects/:id` | Admin        | Ubah subject                                    |
| DELETE | `/api/v1/subjects/:id` | Admin        | Hapus subject                                   |

"Admin" = role `admin` atau `super_admin`.

## Format response

```json
{ "success": true,  "message": "...", "data": {}, "meta": { "page": 1, "per_page": 20, "total": 1 } }
{ "success": false, "message": "Validasi gagal", "errors": [{ "field": "code", "message": "wajib diisi" }] }
```

Status: 401 belum login / sesi habis, 403 tidak punya akses / akun nonaktif, 422 validasi gagal, 400 JSON rusak, 404 data tidak ada, 409 data kembar, 500 error server (detail hanya di log).
