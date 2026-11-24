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
- [cmd/seed/dummy_subjects.sql](cmd/seed/dummy_subjects.sql): 5 mapel (Matematika, IPA, B. Indonesia, B. Inggris, IPS) + hierarki lengkap **Matematika SD Kelas 1-6**: 15 bidang materi, 18 bab, 24 sub bab, 47 topik (aman dijalankan ulang).
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
| Master    | `branches` (cabang), `master_data_items` (data acuan generik: jenjang, kelas, hierarki mapel)       |
| Akun      | `users` (semua akun + role), `refresh_tokens` (sesi login)                                          |
| Siswa     | `students` (kode siswa, tanggal lahir, target sekolah, saldo coin)                                  |
| Paket     | `subscription_plans` (paket + harga + akses fitur), `student_subscriptions` (paket dibeli/riwayat), |
|           | `student_feature_overrides` (akses khusus per siswa)                                                |
| Coin      | `coin_packages` (paket isi coin), `coin_transactions` (riwayat coin, termasuk pembelian)            |
| Lain-lain | `subjects` (contoh awal), `schema_migrations` (catatan versi migration, jangan diubah manual)       |

`master_data_items` dibedakan lewat kolom `type` dan membentuk pohon lewat `parent_id`:

```text
education_level  Jenjang   SD                        (data awal migration)
└─ grade         Kelas     Kelas 5                   (data awal migration)

subject          Mapel     Matematika                ┐ sama untuk semua jenjang, tanpa induk
└─ subject_field  Bidang Materi  Bilangan  [SD, Kelas 5] │ ← mulai di sini terhubung ke jenjang & kelas
   └─ chapter       Bab        Pecahan                │ diisi admin lewat Master Data
      └─ subchapter   Sub Bab    Operasi Pecahan      │ (contoh: make seed → dummy_subjects.sql)
         └─ topic       Topik      Penjumlahan pecahan ┘

question_type    Tipe soal  single_choice = Pilihan ganda    ┐ data awal migration 000015, tanpa induk.
quiz_type        Tipe quiz  daily_quiz = Daily Quiz          │ code = nilai yang DISIMPAN di tabel lain
difficulty       Level      easy = Mudah                     ┘ (questions.type / used_for / difficulty)
```

- Mapel hanya satu (Matematika dipakai SD, SMP, SMA). Yang wajib punya induk hanya kelas, bidang materi, bab, sub bab, dan topik.
- Tipe soal, tipe quiz, dan level **tidak** direlasikan lewat id: tabel lain menyimpan `code`-nya sebagai teks (contoh `questions.difficulty = 'easy'`). Karena itu `code` ketiganya wajib diisi dan **tidak boleh diubah**; yang boleh diubah hanya `name` & `sort_order`.
- Bidang materi s/d topik wajib mengisi `subject_id`, `education_level_id`, `grade_id`. Kolom ini **disalin dari induk** saat data dibuat, supaya daftar bab/topik per mapel & kelas cukup query 1 tabel tanpa JOIN berlapis (index `master_data_items_curriculum_idx`).
- Database menolak kelas yang bukan milik jenjangnya (Kelas 4 + SMP) lewat foreign key `(grade_id, education_level_id)`.
- Nama bidang materi boleh sama di kelas berbeda ("Bilangan" Kelas 1 dan Kelas 4), tapi tidak boleh kembar di kelas yang sama.

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
| POST   | `/api/v1/questions`    | Admin        | Tambah 1-100 soal sekaligus (1 tipe)            |
| GET    | `/api/v1/questions`    | Admin        | List Bank Soal: filter, search, pagination      |
| GET    | `/api/v1/questions/type-counts` | Admin | Jumlah soal per tipe (angka di tab)     |
| GET    | `/api/v1/dropdowns/:type` | Login (semua role) | Pilihan dropdown dari master data  |
| POST   | `/api/v1/quizzes`      | Admin        | Buat quiz Uji Kom / Tryout + soal yang dipilih  |
| GET    | `/api/v1/quiz-bundles` | Admin        | Daftar bundle (quiz lama) untuk dipakai ulang   |
| GET    | `/api/v1/questions/:id`| Admin        | Detail soal + kunci jawaban & pembahasan        |
| PUT    | `/api/v1/questions/:id`| Admin        | Ubah 1 soal (ganti penuh)                       |
| DELETE | `/api/v1/questions`    | Admin        | Hapus 1-100 soal sekaligus (soft delete)        |
| POST   | `/api/v1/questions/import/check` | Admin | Cek file Excel soal (tidak menyimpan)  |
| POST   | `/api/v1/images`       | Admin        | Upload 1-10 gambar (multipart)                  |
| DELETE | `/api/v1/images`       | Admin        | Hapus permanen 1-100 gambar                     |
| GET    | `/storage/images/...`  | Publik       | File gambar (link dari field `url`)             |

"Admin" = role `admin` atau `super_admin`.

### Tambah soal (`POST /api/v1/questions`)

Membuat 1 sampai 100 soal sekaligus, **semuanya 1 tipe** (`type` ditulis sekali di level atas). Bagian umum soal sama untuk semua tipe; hanya isi `answer` yang berbeda. Contoh lengkap ada di Postman (folder **Bank soal**).

```json
{
  "type": "single_choice",
  "questions": [
    {
      "question": "Hasil dari $\\frac{1}{2} + \\frac{1}{4}$ adalah ...",
      "formula": null,
      "image_id": null,
      "table": { "headers": ["Nama", "Nilai"], "rows": [["Andi", "80"]] },
      "explanation": "Samakan penyebut ...",
      "display_order": ["question", "table"],
      "placement": {
        "education_level_id": "...", "grade_id": "...", "subject_id": "...",
        "subject_field_id": "...", "chapter_id": "...",
        "subchapter_id": null, "topic_id": null
      },
      "difficulty": "easy",
      "used_for": ["daily_quiz", "uji_kom"],
      "status": "active",
      "answer": { "options": [{ "key": "A", "content": "...", "image_id": null }], "correct_key": "A" }
    },
    { "question": "Soal kedua ...", "...": "..." }
  ]
}
```

Response 201: `"message": "2 soal berhasil dibuat"`, `data` = daftar soal (dengan `id` & `code`) urut sesuai request.

- **Tidak boleh bercampur tipe.** `type` per soal tidak perlu diisi; kalau diisi dan berbeda dari `type` atas → 422 di `questions[i].type`.
- **Semua atau tidak sama sekali.** 1 soal salah → tidak ada yang tersimpan. Field error diawali nomor soal (mulai 0), contoh `questions[2].answer.correct_key`.
- **Hemat query**, berapa pun jumlah soalnya: 1 query cek master data, 1 query cek gambar, 1 query `INSERT` banyak baris.

| `type`            | Isi `answer`                                                                                              |
|-------------------|-----------------------------------------------------------------------------------------------------------|
| `single_choice`   | `options` (2-5, key urut A, B, ...) + `correct_key`                                                       |
| `multiple_choice` | `options` (2-5) + `correct_keys` (min 1) + `scoring` (`all_or_nothing` default / `partial`)               |
| `true_false`      | `options` (1-10 pernyataan) + `correct_keys: [{"key": "A", "correct": true}, ...]` untuk SETIAP opsi       |
| `free_text`       | `input_type` (`number` / `text`) + `accepted_answers` (1-20) + `tolerance` (number) / `case_sensitive` (text)   |

Aturan penting:

- `question` dan `explanation` (pembahasan) **wajib** untuk semua tipe; isi spasi saja ditolak.
- `formula`, gambar, dan `table` opsional dan berlaku untuk **semua tipe** (ada di level soal, bukan di `answer`): boleh salah satu, beberapa, atau ketiganya sekaligus.
- Placement wajib sampai `chapter_id`; `subchapter_id` & `topic_id` opsional (topik butuh sub bab). Jenjang, kelas, mapel, dan bidang materi harus cocok dengan bab.
- `display_order` opsional. Kalau diisi, harus berisi `question` + PERSIS bagian yang diisi (`formula`, `image`, `table`).
- Field yang tidak dikenal di `answer` ditolak (contoh `correct_key` di soal `multiple_choice`), supaya salah ketik langsung ketahuan.
- Free text `text`: jawaban siswa benar jika **mengandung salah satu** kata kunci di `accepted_answers`.
- Gambar soal & opsi: `image_id` (harus ada di tabel `media_assets`, API upload gambar belum dibuat) **atau** `image_url` (link luar http/https, maks 2000 karakter). Tidak boleh dua-duanya. Opsi boleh tanpa teks kalau punya gambar.
- Semua kesalahan dikembalikan sekaligus (422) dengan `field` lengkap, contoh `questions[0].answer.options[1].content`, `questions[1].placement.grade_id`.
- `answer` disimpan dalam bentuk rapi di kolom JSONB `questions.answer` (spasi dibuang, default diisi, `correct_keys` true/false diurutkan sesuai opsi).

### List soal / Bank Soal (`GET /api/v1/questions`)

Tabel Bank Soal, **urut terbaru dulu** (`created_at DESC`). Semua query string opsional dan bisa digabung (AND):

| Query | Isi |
| ----- | ---- |
| `page`, `per_page` | Default 1 & 20, `per_page` maks 100 |
| `search` | Cari di **kode soal** dan **teks soal**, bagian mana saja, tidak peka huruf besar/kecil (maks 100 karakter). `%` dan `_` dicari apa adanya |
| `type` | `single_choice` / `multiple_choice` / `true_false` / `free_text` |
| `difficulty` | Kolom "Level": `easy` / `medium` / `hard` |
| `used_for` | Peruntukan: `daily_quiz` / `uji_kom` / `tryout` / `rumath` (soal yang `used_for`-nya berisi nilai ini) |
| `status` | `active` / `inactive` |
| `quiz_id` | Isi 1 bundle: hanya soal yang ada di quiz ini (lihat [Bundle soal](#bundle-soal-pakai-ulang-soal-quiz-lama)) |
| `education_level_id`, `grade_id`, `subject_id`, `subject_field_id`, `chapter_id`, `subchapter_id`, `topic_id` | Filter per tingkat hierarki (UUID) |

```http
GET /api/v1/questions?page=1&per_page=20&subject_id=...&difficulty=easy&search=garis
```

```json
{
  "status_code": 200, "message": "Data soal berhasil diambil",
  "data": [{
    "id": "...", "code": "Q-1264", "type": "free_text",
    "question": "Perhatikan garis bilangan. Hasil dari $-3 + 5$ adalah ...",
    "media": { "formula": false, "image": true, "table": false },
    "placement": {
      "education_level": { "id": "...", "name": "SD" },
      "grade":           { "id": "...", "name": "Kelas 4" },
      "subject":         { "id": "...", "name": "Matematika" },
      "subject_field":   { "id": "...", "name": "Bilangan" },
      "chapter":         { "id": "...", "name": "Bilangan Bulat" },
      "subchapter": null, "topic": null
    },
    "difficulty": "easy", "used_for": ["uji_kom"], "status": "active",
    "created_by": { "id": "...", "name": "Admin Surabaya Timur" }, "created_at": "...",
    "updated_by": null, "updated_at": "...",
    "quizzes": [{ "id": "...", "name": "Uji Kompetensi Matematika - Bilangan Bulat", "type": "uji_kom" }]
  }],
  "meta": { "page": 1, "per_page": 20, "total": 2 }
}
```

- Kolom tabel: Kode = `code`, Pertanyaan = `question`, Tipe = `type` + `media` (label "+N media" = jumlah `true`), Hirarki = `placement`, Level = `difficulty`, Dipakai di = `used_for`, Create by/on = `created_by`/`created_at`, Last updated by/on = `updated_by`/`updated_at`.
- `media` hanya untuk isi **soal** (rumus, gambar Image Library atau link luar, tabel), bukan gambar di opsi jawaban.
- `updated_by: null` = soal belum pernah diubah (`updated_at` = waktu dibuat).
- **Tanpa** kunci jawaban & pembahasan supaya response kecil; ambil lewat detail kalau perlu.
- Soal yang dihapus (soft delete) tidak pernah tampil. Hasil kosong → `data: []`.
- `quizzes` = penanda **"sudah dipakai di quiz"**: quiz (belum dihapus) yang memakai soal ini, terbaru dulu; `[]` = belum pernah dipakai. Diambil dari tabel `quiz_questions` lewat index `question_id`, hanya untuk soal di halaman ini (bukan seluruh bank soal).
- Untuk layar **pilih soal quiz**: `?used_for=uji_kom&status=active&subject_id=...&grade_id=...` → hanya soal yang boleh dipilih ke quiz tersebut.
- Performa: halaman dipotong dulu di tabel `questions` (CTE), baru JOIN nama untuk ≤ `per_page` soal itu saja. Pencarian memakai index trigram `pg_trgm` (migration 000013). Uji 20.000 soal: ~1 ms dengan index vs ~350 ms tanpa index. Total (`meta.total`) dihitung dengan `COUNT(*)` terpisah, kecuali sudah pasti dari halaman terakhir yang tidak penuh.
- Soal dari 1 request bulk create punya `created_at` yang sama; urutannya tetap sesuai request (id soal memakai UUID v7 yang berurutan).
- Tab tipe dipilih (`type=...`): index `questions_type_created_at_idx` (migration 000014) langsung memberi 20 soal terbaru tipe itu tanpa sorting (uji 100.000 soal: < 0,1 ms).

### Jumlah soal per tipe / angka di tab (`GET /api/v1/questions/type-counts`)

Untuk tab **Semua · Pilihan ganda · PG kompleks · Benar / Salah · Isian**. Query string **sama persis** dengan list (`search`, `difficulty`, hierarki), jadi angka di tab selalu cocok dengan isi tabel. `type`, `page`, `per_page` diabaikan (boleh ikut dikirim), karena semua tipe selalu dihitung.

```http
GET /api/v1/questions/type-counts?subject_id=...&search=garis
```

```json
{
  "status_code": 200, "message": "Jumlah soal per tipe",
  "data": { "all": 264, "single_choice": 128, "multiple_choice": 2, "true_false": 2, "free_text": 132 }
}
```

- Semua tipe selalu ada (0 kalau kosong). `all` = jumlah semuanya.
- FE: panggil endpoint ini saat search/filter berubah; saat hanya pindah tab atau halaman, cukup panggil list (angka tab tidak berubah).
- 1 query `GROUP BY type` (bukan 1 query per tipe). Uji 100.000 soal: ±55 ms tanpa filter, ±6 ms dengan search.

### Detail soal (`GET /api/v1/questions/:id`)

Khusus admin & super admin karena berisi **kunci jawaban** (`answer`) dan **pembahasan** (`explanation`). Bentuknya sama dengan data create, ditambah nama placement & pembuat:

```json
{
  "id": "...", "code": "Q-12", "type": "single_choice",
  "question": "KPK dari 4 dan 6 adalah ...", "formula": null, "image_id": null, "table": null,
  "explanation": "Kelipatan persekutuan terkecil dari 4 dan 6 adalah 12",
  "display_order": ["question"],
  "placement": {
    "education_level": { "id": "...", "name": "SD" },
    "grade":           { "id": "...", "name": "Kelas 4" },
    "subject":         { "id": "...", "name": "Matematika" },
    "subject_field":   { "id": "...", "name": "Bilangan" },
    "chapter":         { "id": "...", "name": "KPK dan FPB" },
    "subchapter": null,
    "topic": null
  },
  "difficulty": "easy", "used_for": ["daily_quiz"], "status": "active",
  "answer": { "options": [ ... ], "correct_key": "B" },
  "created_by": { "id": "...", "name": "Admin Surabaya Timur" },
  "created_at": "...", "updated_at": "..."
}
```

- 1 query native SQL (`questions` + 7× `master_data_items` + `users`, semua JOIN lewat primary key). SQL-nya ada di `questionDetailQuery` ([question_repository.go](internal/repository/question/question_repository.go)), bisa dicoba di DBeaver.
- ID tidak ada, bukan UUID, atau soal sudah dihapus → 404 `Soal tidak ditemukan`.

### Ubah soal (`PUT /api/v1/questions/:id`)

Hanya **1 soal per request**. Body = 1 soal (bentuk sama dengan item `questions[]` di create) + `type` **wajib**:

```json
{ "type": "multiple_choice", "question": "...", "explanation": "...", "placement": { ... }, "difficulty": "medium", "used_for": ["daily_quiz"], "answer": { ... } }
```

- **Ganti penuh**: field opsional yang tidak dikirim (`formula`, `image_id`, `table`, `subchapter_id`, `topic_id`) menjadi kosong. Kirim ulang semua field dari detail soal, lalu ubah yang perlu.
- Tipe soal boleh diganti; `answer` dicek sesuai tipe baru. Semua aturan create berlaku, field error tanpa awalan (contoh `answer.correct_key`).
- `id`, `code`, `created_by` tidak berubah. `updated_by` & `updated_at` diisi otomatis.
- Response 200 = detail terbaru (sama dengan GET detail). Soal tidak ada / sudah dihapus → 404.

### Hapus soal (`DELETE /api/v1/questions`)

**Soft delete** 1 sampai 100 soal sekaligus: `deleted_at` & `deleted_by` diisi, baris tetap ada di database (bisa dipulihkan manual / untuk laporan).

```json
{ "ids": ["uuid-soal-1", "uuid-soal-2"] }
```

- **Semua atau tidak sama sekali**: kalau ada ID yang tidak ada atau sudah dihapus → 422 dengan error di `ids[i]`, dan tidak ada soal yang terhapus.
- ID harus UUID huruf kecil dan tidak boleh kembar.
- Response 200: `"message": "2 soal berhasil dihapus"`, `data.deleted_ids`.
- Soal yang sudah dihapus tidak muncul lagi di detail, update, maupun hapus (404 / 422).

### Cek file Excel soal (`POST /api/v1/questions/import/check`)

Upload Excel **hanya mengecek**, tidak menyimpan. Template ada di [docs/Template Upload/](docs/Template%20Upload/) (`template_single_choice.xlsx`, `template_multiple_choice.xlsx`, `template_true_false.xlsx`, `template_free_text.xlsx`); 1 file = 1 tipe soal.

Request: `multipart/form-data`, field `file` (.xlsx, maks 5 MB, maks 100 soal).

- **Tipe soal dideteksi dari header** (`correct_key` → single choice, `correct_keys` → multiple choice, `statement_a_content` → true/false, `input_type` → free text). Nama kolom tidak boleh diubah.
- **Placement diisi nama**, bukan ID. Nama dicocokkan ke master data mengikuti hierarki (tidak peka huruf besar/kecil): kelas dicari di jenjangnya, bidang materi di mapel + kelasnya, dst. Semua nama di file diambil dengan **1 query**.
- Setiap baris dicek dengan aturan yang **sama persis** dengan create, ditambah aturan template: `display_order` wajib, opsi wajib (single choice A-D, multiple choice A-C, true/false A-B), `scoring` wajib, tabel butuh `table_headers` + `table_rows`.
- `image_link` → `image_url`. Baris kosong dilewati.

Valid (200) → `data` berbentuk body create + `row` & nama placement. FE bisa langsung mengirim `{"type": data.type, "questions": data.questions}` ke `POST /api/v1/questions` (field tambahan diabaikan):

```json
{
  "type": "single_choice", "total": 1,
  "questions": [{
    "row": 2, "question": "KPK dari 4 dan 6 adalah ...", "image_url": null, "table": { ... },
    "placement": { "education_level_id": "...", "education_level_name": "SD", "grade_id": "...", "grade_name": "Kelas 4", "...": "..." },
    "answer": { "options": [ ... ], "correct_key": "B" }, "...": "..."
  }]
}
```

Ada error (422) → **semua data ditolak**, semua error dikembalikan sekaligus, urut per baris lalu kolom:

```json
{
  "status_code": 422,
  "message": "File Excel berisi 2 kesalahan, semua data ditolak",
  "data": null,
  "errors": [
    { "field": "grade_name", "message": "kelas \"Kelas 9\" tidak ada di jenjang SD", "row": 3, "column": "I" },
    { "field": "used_for", "message": "nilai ke-2: harus salah satu dari: daily_quiz, uji_kom, tryout, rumath", "row": 5, "column": "P" }
  ]
}
```

Masalah file (bukan .xlsx, header diubah, tanpa data, > 100 soal) → 422 dengan `field: "file"`.

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

## Quiz (Uji Kom & Tryout)

Tabel (migration 000016 & 000017):

| Tabel | Isi |
| ----- | --- |
| `quizzes` | Info quiz: nama, tipe, letak (jenjang, kelas, mapel wajib; bidang materi s/d topik opsional), `start_at` (waktu mulai), durasi, biaya coin, `total_questions`, status (`draft`/`published`), `access_code` (kode akses) |
| `quiz_questions` | Soal yang dipilih: `(quiz_id, question_id, sort_order)`. 1 soal tidak bisa dipilih 2x di quiz yang sama; boleh dipakai di banyak quiz |

`total_questions` disimpan di `quizzes` (selalu = jumlah baris `quiz_questions`) supaya list quiz tidak perlu `COUNT`. Penanda "soal sudah dipakai quiz" di list Bank Soal juga dibaca dari `quiz_questions` (index `quiz_questions_question_idx`), jadi tidak ada kolom tambahan di `questions` yang harus selalu disinkronkan.

### Buat quiz (`POST /api/v1/quizzes`)

```json
{
  "name": "Uji Kompetensi Matematika - KPK dan FPB",
  "type": "uji_kom",
  "placement": {
    "education_level_id": "...", "grade_id": "...", "subject_id": "...",
    "subject_field_id": "...", "chapter_id": "..."
  },
  "start_at": "2026-12-01T08:00",
  "duration_minutes": 60,
  "coin_cost": 0,
  "total_questions": 12,
  "bundle_quiz_ids": ["..."],
  "question_ids": ["...", "..."]
}
```

| Field | Aturan |
| ----- | ------ |
| `name` | Wajib, maks 200 karakter (spasi di awal/akhir dibuang) |
| `type` | `uji_kom` / `tryout` (Daily Quiz & RUMATH dibuat otomatis, bukan lewat API ini) |
| `placement` | Jenjang, kelas, mapel **wajib**. Bidang materi, bab, sub bab, topik opsional tapi berurutan (bab butuh bidang materi, dst). Kelas harus milik jenjang; bidang materi harus milik mapel & kelas; semua harus aktif. **Uji Kom hanya untuk Kelas 6, 9, dan 12** (Tryout bebas) |
| `start_at` | **Wajib**, waktu mulai dalam **WIB**, harus setelah sekarang. `"2026-12-01T08:00"` / `"2026-12-01 08:00"` (tanpa zona = WIB, sama dengan nilai input HTML `datetime-local`), atau RFC3339 dengan zona (`"2026-12-01T01:00:00Z"` = 08:00 WIB) |
| `duration_minutes` | Wajib, 1-600 menit |
| `coin_cost` | Opsional, default 0 (gratis) |
| `bundle_quiz_ids` | Opsional, maks 20. **Semua** soal dari quiz-quiz ini (bundle) ikut masuk |
| `question_ids` | Opsional, soal satuan. Minimal salah satu dari `bundle_quiz_ids` / `question_ids` diisi |
| `total_questions` | Wajib, harus **sama** dengan jumlah soal akhir (soal bundle + soal satuan, setelah soal kembar dibuang). Maks 200 |

Urutan soal: soal bundle (urut bundle di `bundle_quiz_ids`, lalu nomor soal di bundle itu), lalu `question_ids`. Soal yang muncul lebih dari sekali (di 2 bundle, atau di bundle + satuan) hanya diambil sekali, di posisi pertamanya.

Syarat setiap soal (error di field asalnya, `bundle_quiz_ids[i]` atau `question_ids[i]`, dengan kode soal, contoh `"Q-12: soal tidak aktif"`):

- ada & belum dihapus (`"Q-12: soal sudah dihapus"` untuk soal lama di bundle), status `active`;
- `used_for` berisi tipe quiz (quiz Uji Kom hanya boleh memakai soal bertanda Uji Kom);
- mapel & kelas sama dengan quiz; kalau quiz mengisi bidang materi/bab/sub bab/topik, soal juga harus di tingkat yang sama.

Respons 201 = detail quiz (`status: "draft"`), letak dengan nama, `start_at` & `end_at` (= mulai + durasi) dalam WIB (`"2026-12-01T08:00:00+07:00"`), `access_code`, dan `questions: [{no, id, code, type, question, difficulty}]` urut nomor soal. Semua kesalahan dikembalikan sekaligus (422); kalau letak quiz salah, soal tidak dibandingkan ke letak itu (supaya error tidak berlipat).

Performa: soal bundle 1 query, cek letak 1 query, cek semua soal 1 query, simpan 1 transaksi (1 INSERT quiz + 1 INSERT banyak baris untuk soal), ambil detail 2 query.

### Bundle soal (pakai ulang soal quiz lama)

Bundle = quiz yang sudah dibuat (belum dihapus). Saat membuat quiz baru, admin bisa:

1. **Ambil bundle utuh**: kirim ID-nya di `bundle_quiz_ids`.
2. **Pilih sebagian soal dari bundle**: buka isinya lewat `GET /api/v1/questions?quiz_id=<id>` (semua kolom, filter, dan pagination list soal tetap berlaku, contoh `&used_for=uji_kom&status=active`), centang soal yang mau dipakai, kirim sebagai `question_ids`.

Daftar bundle: `GET /api/v1/quiz-bundles` (admin), terbaru dulu.

| Query | Isi |
| ----- | --- |
| `page`, `per_page` | Default 1 & 20, `per_page` maks 100 |
| `search` | Nama quiz (sebagian kata, tidak peka huruf besar/kecil) |
| `type` | `uji_kom` / `tryout` |
| `education_level_id`, `grade_id`, `subject_id` | Filter letak |

```json
{
  "data": [{
    "id": "...", "name": "Uji Kompetensi Matematika - KPK dan FPB", "type": "uji_kom",
    "education_level": { "id": "...", "name": "SD" }, "grade": { "id": "...", "name": "Kelas 6" },
    "subject": { "id": "...", "name": "Matematika" }, "total_questions": 12, "created_at": "..."
  }],
  "meta": { "page": 1, "per_page": 20, "total": 1 }
}
```

### Kode akses & pergantian per jam

- Setiap quiz Uji Kom & Tryout mendapat `access_code` acak saat dibuat: **6 karakter A-Z dan 0-9** (contoh `K7Q2ZD`), dibuat dengan `crypto/rand` (tidak bisa ditebak). Kode **unik** di antara quiz yang belum dihapus (index unik `quizzes_access_code_unique`); kalau kebetulan bentrok, kode dibuat ulang otomatis.
- **Scheduler** di dalam proses API ([internal/scheduler](internal/scheduler/scheduler.go), library `robfig/cron`) mengganti kode akses **setiap jam tepat (WIB)** untuk quiz yang **belum selesai** (`start_at + durasi > sekarang`, termasuk yang belum mulai & yang sedang berjalan). Quiz yang sudah selesai atau dihapus tidak diubah.
- Jadwal diatur `ACCESS_CODE_REFRESH_CRON` di `.env` (format cron 5 kolom, default `0 * * * *`; `off` = mati).
- **Aman untuk beberapa server**: setiap jalan memakai advisory lock PostgreSQL (`pg_try_advisory_xact_lock`), jadi hanya 1 server yang mengganti kode; server lain mencatat "dilewati".
- Performa: 1 transaksi, quiz dicari lewat index `start_at` (durasi maks 600 menit → quiz yang belum selesai pasti mulai < 10 jam lalu), kode baru dicek sekaligus ke database, lalu UPDATE banyak baris per 1.000 quiz.
- Hasil setiap jalan dicetak di terminal server, contoh `scheduler: kode akses 12 quiz diganti (35ms)`.

## Dropdown (`GET /api/v1/dropdowns/:type`)

Pilihan dropdown dari `master_data_items`. Semua role yang login boleh (siswa juga butuh, contoh memilih mapel & level Daily Quiz). Hanya item **aktif**, urut `sort_order` lalu nama.

| `:type` | Isi | Filter (query string) |
| ------- | --- | --------------------- |
| `education_level` | Jenjang | - |
| `grade` | Kelas | `education_level_id` (opsional) |
| `subject` | Mapel | - |
| `subject_field` | Bidang materi | `subject_id` **wajib**, `grade_id` (opsional) |
| `chapter` | Bab | `subject_field_id` **wajib** |
| `subchapter` | Sub bab | `chapter_id` **wajib** |
| `topic` | Topik | `subchapter_id` **wajib** |
| `question_type` | Tipe soal | - |
| `quiz_type` | Tipe quiz (Dipakai untuk) | - |
| `difficulty` | Level | - |

```http
GET /api/v1/dropdowns/difficulty
GET /api/v1/dropdowns/subject_field?subject_id=...&grade_id=...
```

```json
{
  "status_code": 200, "message": "Data dropdown",
  "data": [
    { "value": "easy",   "id": "...", "code": "easy",   "name": "Mudah",  "parent_id": null },
    { "value": "medium", "id": "...", "code": "medium", "name": "Sedang", "parent_id": null },
    { "value": "hard",   "id": "...", "code": "hard",   "name": "Sulit",  "parent_id": null }
  ]
}
```

- **FE cukup memakai `value` (dikirim ke API lain) dan `name` (ditampilkan).** `value` = `code` untuk `question_type`, `quiz_type`, `difficulty` (contoh `"easy"`, `"daily_quiz"`), dan = `id` untuk jenjang s/d topik. Jadi `value` selalu bisa langsung dipakai di body create soal (`type`, `used_for`, `difficulty`, `placement.*_id`) dan di filter list soal.
- Filter yang tidak berlaku untuk type tersebut diabaikan. Induk wajib belum diisi → 422 (contoh `subject_id: wajib diisi`); format bukan UUID → 422; type tidak dikenal → 404 berisi daftar type.
- 1 query 1 tabel, memakai index `master_data_items_dropdown_idx (type, parent_id, sort_order)`.
- Kelas tanpa filter → 14 kelas (urut Nol Kecil s/d Kelas 12), `parent_id` = id jenjangnya (bisa dipakai FE untuk mengelompokkan).

## Image Library (gambar)

Data gambar ada di tabel `media_assets`, filenya di disk:

```text
storage/                          ← STORAGE_DIR di .env (tidak di-commit, ada di .gitignore)
└─ images/
   ├─ question/2026/11/<uuid>.png  ← kategori / tahun / bulan / id gambar
   ├─ banner/...
   └─ landing/...
```

Link gambar (`url`) disimpan lengkap, dibentuk dari `APP_URL` di `.env`, contoh `http://localhost:8080/storage/images/question/2026/11/<uuid>.png`, dan bisa dibuka **tanpa login**. Di production isi `APP_URL` dengan domain asli **sebelum** upload gambar pertama. Request ke `/storage/...` tidak dicatat di log API.

### Upload gambar (`POST /api/v1/images`)

`multipart/form-data`:

| Field | Keterangan |
|---|---|
| `files` | 1-10 file, masing-masing maks 5 MB. PNG, JPG, GIF, WebP (dicek dari **isi** file, bukan nama). SVG ditolak (bisa berisi script). |
| `category` | Wajib: `question` / `banner` / `landing` |
| `name` | Wajib, maks 150 karakter. Upload lebih dari 1 file → nama diberi nomor urut: `Segitiga 1`, `Segitiga 2`, ... |
| `subject_id`, `subject_field_id`, `chapter_id`, `subchapter_id`, `topic_id` | Opsional, **hanya untuk `question`**, diisi berurutan dari atas (bab butuh bidang materi, dst.) dan harus saling cocok. |

- Semua file dalam 1 upload memakai kategori & hierarki yang sama.
- **Semua atau tidak sama sekali**: 1 file salah → tidak ada yang tersimpan, error di `files[i]`.
- Response 201: daftar gambar (urut sesuai file) dengan `id`, `name`, `url`, `mime_type`, `size_bytes`, `width`, `height`, hierarki. `id` dipakai sebagai `image_id` di soal.

### Hapus gambar (`DELETE /api/v1/images`)

```json
{ "ids": ["uuid-gambar-1", "uuid-gambar-2"] }
```

- **Hapus permanen** (data + file), 1-100 gambar.
- **Semua atau tidak sama sekali**: ditolak (422, error di `ids[i]`) kalau ada ID yang tidak ada, atau gambar **masih dipakai soal aktif** (sebagai `image_id` soal atau gambar opsi jawaban), contoh `"masih dipakai soal Q-12, Q-15"`.
- Soal yang sudah dihapus (soft delete) tidak menghalangi; `image_id`-nya otomatis jadi `NULL`.

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
