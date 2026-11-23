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

Setelah `make seed`, semua akun memakai password **`password123`**. Total 55 user, mengikuti contoh desain menu User: 47 siswa, 4 tutor, 3 admin, 1 super admin.

- [cmd/seed/dummy_data.sql](cmd/seed/dummy_data.sql): cabang, akun utama di bawah, paket, dan coin (dijalankan sekali).
- [cmd/seed/dummy_users.sql](cmd/seed/dummy_users.sql): 2 admin, 3 tutor, dan 43 siswa tambahan yang tersebar di semua cabang dan jenjang (aman dijalankan ulang). Siswa tambahan memakai username `siswa.<nama depan>`, contoh `siswa.adit`. Siswa tambahan belum punya paket.

Akun utama untuk dicoba:

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

  middleware/             Log API (APILog), cek login (RequireAuth) dan role (RequireRole).
  apilog/                 Menulis log API ke file .txt harian + hapus file lama.
  mailer/                 Kirim email lewat SMTP (dipakai fitur laporan bug).
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

| Kebutuhan                                      | Pakai                                          |
|------------------------------------------------|------------------------------------------------|
| Insert, update, delete, get by ID              | GORM: `Create`, `Updates`, `Delete`, `First`   |
| List dengan filter, JOIN 1–2 tabel             | GORM atau native SQL, mana yang lebih jelas    |
| **JOIN lebih dari 2 tabel**, agregasi, laporan | **Wajib native SQL**: `db.Raw(...).Scan(...)`  |

Tabel yang di-JOIN lebih dari sekali tetap dihitung per JOIN. Contoh: `users` + `master_data_items` (jenjang) + `master_data_items` (kelas) = 3 tabel, jadi wajib native SQL (lihat [user_repository.go](internal/repository/login/user_repository.go)).

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

**Prioritas saat menulis kode** (kalau bertentangan, nomor lebih kecil menang):

1. **Performa sistem.** Hindari query berulang (N+1), ambil data yang dibutuhkan dalam satu query, dan jangan membuat request menunggu pekerjaan yang bisa dikerjakan di background (contoh: penulisan log).
2. **Mudah di-debug.** Kode lurus dan mudah diikuti (handler → service → repository), error selalu dicatat ke log API lewat `c.Error(err)`, dan nama fungsi atau variabel jelas.
3. **Logging API.** Setiap request tercatat di `logs/api-YYYY-MM-DD.txt` (lihat [Log API](#log-api)).

**Aturan struktur:**

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
| POST   | `/api/v1/log-reports`  | Admin        | Laporan bug: kirim log hari ini ke email dev    |

"Admin" = role `admin` atau `super_admin`.

### Postman

Collection lengkap ada di [docs/postman/Rumbasha.postman_collection.json](docs/postman/Rumbasha.postman_collection.json). Import ke Postman (File → Import), lalu jalankan salah satu request **Login**: token otomatis tersimpan dan dipakai request lain. Folder **Login per role (akun dummy)** berisi satu request untuk setiap akun dari `make seed`.

Uji semua endpoint sekaligus dengan Postman CLI (server harus sudah jalan):

```bash
postman collection run docs/postman/Rumbasha.postman_collection.json -i "Health" -i "Auth" -i "Login per role (akun dummy)"
```

Setiap ada endpoint baru, tambahkan juga request-nya ke collection ini.

## Format response

```json
{ "status_code": 200, "message": "...", "data": {}, "meta": { "page": 1, "per_page": 20, "total": 1 } }
{ "status_code": 422, "message": "Validasi gagal", "data": null, "errors": [{ "field": "code", "message": "wajib diisi" }] }
```

`status_code` selalu sama dengan HTTP status code. `data` selalu ada (`null` kalau kosong). `meta` hanya muncul di endpoint list, `errors` hanya muncul saat validasi gagal.

Status: 401 belum login / sesi habis, 403 tidak punya akses / akun nonaktif, 422 validasi gagal, 400 JSON rusak, 404 data tidak ada, 409 data kembar, 500 error server (detail hanya di log).

## Log API

Setiap request dicatat ke `logs/api-YYYY-MM-DD.txt` (satu file per hari, tanggal WIB). File yang lebih lama dari 15 hari dihapus otomatis. Folder dan batas harinya diatur lewat `LOG_DIR` dan `LOG_RETENTION_DAYS` di `.env`.

Satu baris = satu request (JSON). Contoh (dirapikan supaya mudah dibaca):

```json
{
  "time": "2026-11-23 17:18:16.595",
  "request_id": "fc8b1118-9208-4481-8081-85483d0864a7",
  "method": "POST",
  "endpoint": "/api/v1/auth/login",
  "route": "/api/v1/auth/login",
  "status": 422,
  "duration_ms": 0.8,
  "client_ip": "::1",
  "user_agent": "PostmanRuntime/7.43.0",
  "user_id": "",
  "payload": { "username": "siswa.budi" },
  "process": ["Key: 'LoginRequest.password' Error:Field validation for 'password' failed on the 'required' tag"],
  "response": { "status_code": 422, "message": "Validasi gagal", "data": null, "errors": [ ... ] }
}
```

- **`request_id`** juga dikirim di header response `X-Request-ID`. Kalau frontend melaporkan error, minta nilai header ini, lalu cari di file log (Ctrl+F).
- **`process`** berisi catatan proses inti: error validasi, error dari service, alasan token ditolak, panic. Untuk menambah catatan dari handler, panggil `_ = c.Error(err)`. Fungsi `handleError` di setiap handler sudah melakukannya.
- **Password dan token selalu disamarkan** (`"***"`). Daftar field rahasia ada di `sensitiveKeys` di [internal/apilog/entry.go](internal/apilog/entry.go). Tambahkan di sana kalau ada field rahasia baru.
- Payload atau response di atas 10 KB hanya dicatat ukurannya. Upload file tidak dicatat isinya. `/health` tidak dicatat.
- Log ditulis di background, jadi request tidak menunggu disk. Hentikan server dengan **Ctrl+C** (bukan menutup terminal) supaya sisa log sempat ditulis.

### Laporan bug: kirim log ke email

Admin dan super admin bisa mengirim deskripsi bug beserta file log hari ini ke email developer:

```text
POST /api/v1/log-reports   {"description": "Siswa tidak bisa login jam 10 pagi", "request_id": "fc8b1118-..."}
  → email ke LOG_REPORT_EMAIL, lampiran logs/api-YYYY-MM-DD.txt (di atas 5 MB dikirim sebagai .txt.gz)
```

Setting di `.env` (contoh untuk Gmail):

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=akun.pengirim@gmail.com
SMTP_PASSWORD=abcdefghijklmnop        # App Password 16 huruf, BUKAN password Gmail
SMTP_FROM=akun.pengirim@gmail.com
LOG_REPORT_EMAIL=khalieftechcraftinnovation@gmail.com
```

App Password dibuat di <https://myaccount.google.com/apppasswords>. Akun Gmail pengirim harus sudah memakai verifikasi 2 langkah. Kalau `SMTP_HOST` atau `LOG_REPORT_EMAIL` kosong, server tetap jalan dan endpoint ini membalas 503.

| Status | Arti                                                        |
|--------|-------------------------------------------------------------|
| 200    | Email terkirim                                              |
| 404    | Belum ada log hari ini                                      |
| 429    | Baru saja mengirim; tunggu 1 menit                          |
| 502    | Server email menolak / tidak bisa dihubungi (detail di log) |
| 503    | SMTP belum diisi di `.env`                                  |
