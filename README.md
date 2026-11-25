# Rumbasha Backend

Backend API Rumbasha E-Learning dengan Go, framework [Gin](https://gin-gonic.com/), dan PostgreSQL.

## Persiapan (sekali saja)

1. **Go 1.27.** Kalau `go version` masih versi lama, jalankan: `go env -w GOTOOLCHAIN=go1.27.4`
2. **File `.env`.** Salin dari contoh, lalu isi `DB_PASSWORD` dan `JWT_SECRET` (buat dengan `openssl rand -hex 32`):

   ```bash
   cp .env.example .env
   ```

3. **Buat database** `rumbasha`: buka [scripts/db/create_database.sql](scripts/db/create_database.sql) di DBeaver (koneksi `postgres`), lalu jalankan sebagai script dengan Alt+X.

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
| `make test`            | Menjalankan semua test di database `rumbasha` — **mengosongkan semua tabel data**, jalankan `make seed` setelahnya |
| `make vet`             | Mengecek kode yang mencurigakan                |
| `make fmt`             | Merapikan format kode                          |
| `make build`           | Build ke folder `bin/`                         |
| `make check`           | `vet` + `test` (sebelum push)                  |
| `make migrate-up`      | Menjalankan migration yang belum               |
| `make migrate-down`    | Membatalkan 1 migration terakhir               |
| `make migrate-version` | Melihat versi migration saat ini               |
| `make seed`            | Mengisi data dummy (sekali, aman diulang)      |

## Akun dummy

Setelah `make seed`, semua akun memakai password **`password123`**. Total 158 user: 150 siswa, 4 tutor, 3 admin, 1 super admin. Ditambah ±730 soal Matematika SD di Bank Soal.

- [cmd/seed/dummy_data.sql](cmd/seed/dummy_data.sql): cabang, akun utama di bawah, paket, dan coin (dijalankan sekali).
- [cmd/seed/dummy_subjects.sql](cmd/seed/dummy_subjects.sql): 5 mapel (Matematika, IPA, B. Indonesia, B. Inggris, IPS) + hierarki lengkap **Matematika SD Kelas 1-6**: 15 bidang materi, 18 bab, 24 sub bab, 47 topik (aman dijalankan ulang).
- [cmd/seed/dummy_users.sql](cmd/seed/dummy_users.sql): 2 admin, 3 tutor, dan 43 siswa tambahan yang tersebar di semua cabang dan jenjang (aman dijalankan ulang). Siswa tambahan memakai username `siswa.<nama depan>`, contoh `siswa.adit`. Lalu 103 siswa lagi (total 150 siswa) dengan username `siswa.<nama depan>.<nama keluarga>`, contoh `siswa.arif.wijaya`, tersebar di semua kelas (SD & SMP lebih banyak). Siswa tambahan belum punya paket.
- [cmd/seed/dummy_questions.sql](cmd/seed/dummy_questions.sql): ±730 soal **Matematika SD Kelas 1-6** di semua 47 topik, 4 tipe bergiliran (single choice, multiple choice, benar/salah, isian angka), 3 level (243 soal per level). Setiap bidang materi punya **minimal 15 soal per level**, jadi Daily Quiz 5/10/15 soal selalu bisa di-generate. Peruntukan Daily Quiz & Tryout (Kelas 6 juga Uji Kom). Aman dijalankan ulang (soal dengan topik & teks sama dilewati).
- [cmd/seed/dummy_quiz_reports.sql](cmd/seed/dummy_quiz_reports.sql): 1 event **"Tryout Demo Matematika Kelas 6"** (publish, 10 soal, 7 hari lalu 08:00-10:00 WIB) yang sudah dikerjakan semua siswa aktif Kelas 6, beserta nilai, point, dan leaderboard-nya, untuk mencoba report & leaderboard quiz. Aman dijalankan ulang.
- [cmd/seed/dummy_daily_quizzes.sql](cmd/seed/dummy_daily_quizzes.sql): 2-6 **Daily Quiz bulan berjalan** per siswa aktif SD (soal dari bidang materi kelasnya, level bergiliran), lengkap dengan jawaban, nilai, point, dan leaderboard bulanan, untuk mencoba report & leaderboard Daily Quiz. Aman dijalankan ulang (ID dari hash siswa + bulan).

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
    quiz/                   quiz_routes.go   (dan folder lain: satu folder per fitur)
    *_test.go               Test endpoint dari ujung ke ujung (semua fitur).
  handler/                HANDLER     Baca request, panggil service, kirim response.
    login/  quiz/  dailyquiz/  user/  coin/  ...
  service/                SERVICE     Aturan bisnis.
    login/  quiz/  dailyquiz/  user/  coin/  subscription/  ...
  repository/             REPOSITORY  Query database (GORM atau native SQL).
    login/  quiz/  dailyquiz/  user/  coin/  subscription/  ...
  model/                  Struct yang mewakili tabel + daftar error.
    login/  quiz/  dailyquiz/  user/  coin/  subscription/  ...
  dto/                    Bentuk request & response JSON.
    pagination.go           Dipakai bersama semua fitur.
    login/  quiz/  dailyquiz/  user/  coin/  ...

  middleware/             Log API (APILog), cek login (RequireAuth) dan role (RequireRole).
  apilog/                 Menulis log API ke file .txt harian + hapus file lama.
  mailer/                 Kirim email lewat SMTP (dipakai fitur laporan bug).
  clock/                  Aturan waktu WIB (awal bulan, tanggal hari ini).
  token/                  Membuat & membaca access token (JWT) dan refresh token.
  response/               Helper format JSON (sukses, error, validasi).
  database/               Koneksi PostgreSQL + runner migration.
  config/                 Membaca konfigurasi dari .env.
  testutil/               Helper untuk test (database DB_NAME, tabel data dikosongkan).
```

Setiap layer dikelompokkan per fitur: `<layer>/<fitur>/<fitur>_<layer>.go`, misalnya `service/login/login_service.go`.
File di root folder layer (misalnya `dto/pagination.go`) dipakai bersama oleh semua fitur.

Contoh fitur: **login** (`/auth/...`), **quiz** (`/quizzes/...`), dan **subscription** (paket akses siswa; belum punya endpoint sendiri, dipakai oleh `/auth/me`). Fitur yang tidak punya endpoint sendiri tidak perlu folder di `handler/` dan `router/`.

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
- **Akun nonaktif atau lewat `end_date`** (tanggal berakhir akun, WIB, inklusif) ditolak saat login maupun refresh (403).
- Di database hanya disimpan **hash** refresh token dan **hash bcrypt** password.

Grup `public` (tanpa login) dan `private` (wajib login) dibuat di [router.go](internal/router/router.go), lalu diteruskan ke route tiap fitur. Contoh di [quiz_routes.go](internal/router/quiz/quiz_routes.go):

```go
private.GET("/student/quiz-announcements", studentOnly, h.Announcements) // wajib login + role siswa
private.POST("/quizzes", adminOnly, h.Create)                           // wajib login + role admin/super_admin
```

Di handler, ID dan role user yang login dibaca dengan `middleware.GetUserID(c)` dan `middleware.GetRole(c)`.

## Alur request

```text
GET /api/v1/quizzes/:id
  → router      router/quiz/quiz_routes.go          mencari handler untuk URL ini
  → handler     handler/quiz/quiz_handler.go        QuizHandler.GetDetail
  → service     service/quiz/quiz_service.go        QuizService.GetDetail
  → repository  repository/quiz/quiz_repository.go  QuizRepository.FindDetailByID → PostgreSQL
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
db.Raw(`SELECT * FROM questions WHERE code = ?`, code)   // ✅ aman
db.Raw(`SELECT * FROM questions WHERE code = '` + code + `'`) // ❌ SQL injection
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
| Coin      | `coin_packages` (paket isi coin), `coin_transactions` (riwayat coin, termasuk pembelian & hadiah `reward`), `quiz_prize_awards` & `daily_quiz_prize_awards` (penerima hadiah juara quiz & Daily Quiz bulanan), `daily_quiz_settings` (pengaturan global Daily Quiz) |
| Lain-lain | `schema_migrations` (catatan versi migration, jangan diubah manual)       |

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

Tutor adalah user dengan `role = 'tutor'` dan boleh mengajar siswa mana pun. Jatah gratis & biaya coin Daily Quiz berlaku global untuk semua siswa dan diatur admin di pengaturan Daily Quiz (tabel `daily_quiz_settings`, `PUT /api/v1/daily-quiz-settings`).

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
4. **Satu fitur = satu subfolder per layer**, misalnya `handler/quiz/`, `service/quiz/`, `repository/quiz/`.
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
| POST   | `/api/v1/log-reports`  | Admin        | Laporan bug: kirim log hari ini ke email dev    |
| POST   | `/api/v1/questions`    | Admin        | Tambah 1-100 soal sekaligus (1 tipe)            |
| GET    | `/api/v1/questions`    | Admin        | List Bank Soal: filter, search, pagination      |
| GET    | `/api/v1/questions/type-counts` | Admin | Jumlah soal per tipe (angka di tab)     |
| GET    | `/api/v1/dropdowns/:type` | Login (semua role) | Pilihan dropdown dari master data  |
| POST   | `/api/v1/quizzes`      | Admin        | Buat quiz Uji Kom / Tryout + soal yang dipilih  |
| GET    | `/api/v1/quizzes`      | Admin        | List quiz + status jadwal (akan/berjalan/lewat) |
| GET    | `/api/v1/quizzes/schedule-counts` | Admin | Jumlah quiz per status jadwal (angka tab) |
| GET    | `/api/v1/quizzes/:id`  | Admin        | Detail quiz + jadwal, kode akses, daftar soal   |
| PUT    | `/api/v1/quizzes/:id`  | Admin        | Ubah 1 quiz (ganti penuh), hanya yang belum mulai |
| DELETE | `/api/v1/quizzes`      | Admin        | Hapus 1-100 quiz (soft delete), kecuali yang berjalan |
| GET    | `/api/v1/student/quiz-announcements` | Siswa | Pengumuman quiz publish untuk kelas siswa |
| GET    | `/api/v1/daily-quizzes/grade-options` | Siswa | Dropdown kelas Daily Quiz (sesuai jenjang/kelompok) |
| POST   | `/api/v1/daily-quizzes` | Siswa       | Generate Daily Quiz dari soal acak              |
| POST   | `/api/v1/daily-quizzes/:id/open` | Siswa | Buka Daily Quiz (jatah gratis / potong coin) + kunci jawaban |
| POST   | `/api/v1/quizzes/:id/open` | Siswa    | Buka Uji Kom / Tryout dengan kode akses + kunci jawaban |
| GET    | `/api/v1/quizzes/:id/access-code` | Tutor, Admin | Kode akses quiz yang berlaku sekarang |
| POST   | `/api/v1/daily-quizzes/:id/submit` | Siswa | Kumpulkan jawaban Daily Quiz → nilai + review |
| GET    | `/api/v1/daily-quizzes/:id/review` | Siswa | Hasil & review Daily Quiz |
| GET    | `/api/v1/daily-quiz-settings` | Login | Pengaturan global Daily Quiz (point per level, hadiah juara bulanan) |
| PUT    | `/api/v1/daily-quiz-settings` | Admin, Super Admin | Ubah pengaturan global Daily Quiz |
| GET    | `/api/v1/daily-quiz-prizes` | Admin, Super Admin | Penerima hadiah juara Daily Quiz 1 bulan (per kelas) |
| POST   | `/api/v1/quizzes/:id/submit` | Siswa  | Kumpulkan jawaban Uji Kom / Tryout → nilai + review |
| GET    | `/api/v1/quizzes/:id/review` | Siswa  | Hasil & review Uji Kom / Tryout |
| GET    | `/api/v1/leaderboards/daily-quiz` | Login | Leaderboard Daily Quiz 1 bulan (filter jenjang/kelas) |
| GET    | `/api/v1/leaderboards/quizzes/:id` | Login | Leaderboard 1 quiz Uji Kom / Tryout |
| GET    | `/api/v1/coin-packages` | Admin, Super Admin, Siswa | Paket isi coin (siswa: hanya aktif) |
| GET    | `/api/v1/coin-packages/:id` | Admin, Super Admin | Detail paket coin |
| POST   | `/api/v1/coin-packages` | Admin, Super Admin | Buat paket coin |
| PUT    | `/api/v1/coin-packages/:id` | Admin, Super Admin | Ubah paket coin (ganti penuh) |
| DELETE | `/api/v1/coin-packages` | Admin, Super Admin | Hapus banyak paket coin (soft delete) |
| POST   | `/api/v1/coin-topups` | Admin, Super Admin | Tambah coin banyak siswa sekaligus sesuai 1 paket |
| GET    | `/api/v1/student/coin-transactions` | Siswa | Saldo, jatah Daily Quiz hari ini, riwayat coin sendiri |
| GET    | `/api/v1/students/:id/coin-transactions` | Admin, Super Admin | Riwayat coin 1 siswa |
| GET    | `/api/v1/users` | Admin, Super Admin | List user (filter role, jenjang, kelas, cabang, status; cari nama/username/email) |
| GET    | `/api/v1/users/role-counts` | Admin, Super Admin | Jumlah user per role (angka tab) |
| POST   | `/api/v1/users` | Admin, Super Admin | Tambah 1-100 user sekaligus (admin: hanya tutor & siswa) |
| POST   | `/api/v1/users/import/check` | Admin, Super Admin | Cek file Excel tambah user (tidak menyimpan) |
| GET    | `/api/v1/quiz-reports` | Admin, Super Admin, Tutor | Report Uji Kom & Tryout per event (peserta, nilai rata-rata / tertinggi / terendah) |
| GET    | `/api/v1/quiz-reports/:id` | Admin, Super Admin, Tutor | Detail report: peserta urut peringkat |
| GET    | `/api/v1/quiz-reports/:id/questions` | Admin, Super Admin, Tutor | Soal urut dari yang paling sering salah |
| GET    | `/api/v1/daily-quiz-reports` | Admin, Super Admin, Tutor | Report Daily Quiz per siswa (peringkat + rekap, filter periode/mapel/kelas) |
| GET    | `/api/v1/daily-quiz-reports/students/:id` | Admin, Super Admin, Tutor | Report Daily Quiz 1 siswa: total + rincian per mapel |
| GET    | `/api/v1/users/:id` | Admin, Super Admin | Detail user |
| PUT    | `/api/v1/users/:id` | Admin, Super Admin | Ubah user (admin: hanya tutor & siswa) |
| DELETE | `/api/v1/users` | Admin, Super Admin | Hapus banyak user (soft delete) |
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

Tabel (migration 000016 - 000018):

| Tabel | Isi |
| ----- | --- |
| `quizzes` | Info quiz: nama, tipe, letak (jenjang, kelas, mapel wajib; bidang materi s/d topik opsional), jendela event `start_at`..`end_at` (boleh beberapa hari), `duration_minutes` (lama mengerjakan 1 kali), biaya coin, `total_questions`, status (`draft`/`published`), `access_code` (kode akses) |
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
  "end_at": "2026-12-03T17:00",
  "duration_minutes": 60,
  "coin_cost": 0,
  "total_questions": 12,
  "bundle_quiz_ids": ["..."],
  "question_ids": ["...", "..."],
  "points": { "easy": 10, "medium": 15, "hard": 20 },
  "prize_coins": { "rank_1": 500, "rank_2": 300, "rank_3": 100 }
}
```

| Field | Aturan |
| ----- | ------ |
| `name` | Wajib, maks 200 karakter (spasi di awal/akhir dibuang) |
| `status` | **Wajib**, tombol simpan: `draft` (Simpan draf) atau `published` (Publish: quiz tampil di pengumuman dashboard siswa kelas tersebut) |
| `type` | `uji_kom` / `tryout` (Daily Quiz & RUMATH dibuat otomatis, bukan lewat API ini) |
| `placement` | Jenjang, kelas, mapel **wajib**. Bidang materi, bab, sub bab, topik opsional tapi berurutan (bab butuh bidang materi, dst). Kelas harus milik jenjang; bidang materi harus milik mapel & kelas; semua harus aktif. **Uji Kom hanya untuk Kelas 6, 9, dan 12** (Tryout bebas) |
| `start_at` | **Wajib**, mulai event dalam **WIB**, harus setelah sekarang. `"2026-12-01T08:00"` / `"2026-12-01 08:00"` (tanpa zona = WIB, sama dengan nilai input HTML `datetime-local`), atau RFC3339 dengan zona (`"2026-12-01T01:00:00Z"` = 08:00 WIB) |
| `end_at` | **Wajib**, selesai event (WIB, format sama), **boleh beda hari**. Harus setelah `start_at` dan minimal `start_at + duration_minutes` |
| `duration_minutes` | Wajib, 1-600 menit: lama mengerjakan 1 kali, di dalam jendela event |
| `coin_cost` | Opsional, default 0 (gratis) |
| `bundle_quiz_ids` | Opsional, maks 20. **Semua** soal dari quiz-quiz ini (bundle) ikut masuk |
| `question_ids` | Opsional, soal satuan. Minimal salah satu dari `bundle_quiz_ids` / `question_ids` diisi |
| `total_questions` | Wajib, harus **sama** dengan jumlah soal akhir (soal bundle + soal satuan, setelah soal kembar dibuang). Maks 200 |
| `points` | Opsional. Point leaderboard 1 soal **benar** per level soal: `easy`, `medium`, `hard` (0-1000). Level yang tidak diisi = default **10 / 15 / 20**. Salah / tidak dijawab tetap 0 |
| `prize_coins` | Opsional. Hadiah coin juara 1, 2, 3 leaderboard quiz: `rank_1`, `rank_2`, `rank_3` (0-1.000.000). Tidak diisi = 0 (tanpa hadiah). Dibagikan otomatis 23:59 WIB setelah event selesai (lihat **Hadiah juara**) |

Urutan soal: soal bundle (urut bundle di `bundle_quiz_ids`, lalu nomor soal di bundle itu), lalu `question_ids`. Soal yang muncul lebih dari sekali (di 2 bundle, atau di bundle + satuan) hanya diambil sekali, di posisi pertamanya.

Syarat setiap soal (error di field asalnya, `bundle_quiz_ids[i]` atau `question_ids[i]`, dengan kode soal, contoh `"Q-12: soal tidak aktif"`):

- ada & belum dihapus (`"Q-12: soal sudah dihapus"` untuk soal lama di bundle), status `active`;
- `used_for` berisi tipe quiz (quiz Uji Kom hanya boleh memakai soal bertanda Uji Kom);
- mapel & kelas sama dengan quiz; kalau quiz mengisi bidang materi/bab/sub bab/topik, soal juga harus di tingkat yang sama.

**Jadwal tidak boleh beririsan:** quiz (belum dihapus, tipe & status apa pun) dengan **mapel & kelas yang sama** tidak boleh punya jendela event yang beririsan. Quiz yang mulai tepat saat quiz lain selesai (10:00 → 10:00) boleh. Error di `start_at`, contoh `jadwal beririsan dengan quiz "Uji Kom A" (01-12-2026 08:00 WIB - 03-12-2026 17:00 WIB) untuk mapel & kelas yang sama`. Aturan ini juga dijaga database (exclusion constraint `quizzes_no_overlap`, extension `btree_gist`), jadi 2 request bersamaan pun tidak bisa lolos.

Respons 201 = detail quiz (`status` sesuai tombol), letak dengan nama, `start_at` & `end_at` dalam WIB (`"2026-12-01T08:00:00+07:00"`), `schedule_status`, `access_code`, dan `questions: [{no, id, code, type, question, difficulty}]` urut nomor soal. Semua kesalahan dikembalikan sekaligus (422); kalau letak quiz salah, soal tidak dibandingkan ke letak itu (supaya error tidak berlipat).

Performa: soal bundle 1 query, cek letak 1 query, cek bentrok jadwal 1 query (index GiST), cek semua soal 1 query, simpan 1 transaksi (1 INSERT quiz + 1 INSERT banyak baris untuk soal), ambil detail 2 query.

### Ubah quiz (`PUT /api/v1/quizzes/:id`)

Body **sama persis** dengan buat quiz (ganti penuh: nama, tipe, letak, jadwal, durasi, biaya, soal, point, hadiah; `points` / `prize_coins` yang tidak dikirim kembali ke default). Aturan validasi sama, dan quiz itu sendiri tidak dianggap bentrok dengan jadwal lamanya. Respons 200 = detail terbaru (`updated_by` terisi).

- **Hanya quiz yang belum dimulai** (`start_at` > sekarang). Yang sudah dimulai / selesai → **409** `Quiz sudah dimulai, tidak bisa diubah`.
- `access_code` dan `status` tidak berubah. Soal lama diganti seluruhnya dengan daftar soal baru.
- ID tidak ada / bukan UUID / sudah dihapus → 404.

### Detail quiz (`GET /api/v1/quizzes/:id`)

Bentuk sama dengan respons buat/ubah quiz: letak dengan nama, `start_at`/`end_at` (WIB), `schedule_status`, `duration_minutes`, `access_code`, `coin_cost`, `total_questions`, `points` (`{easy, medium, hard}`), `prize_coins` (`{rank_1, rank_2, rank_3}`), `prizes_distributed_at` (WIB, null = belum dibagikan), `status`, `created_by`/`updated_by`, `questions: [{no, id, code, type, question, difficulty}]` urut nomor soal (tanpa kunci jawaban; kunci ada di detail soal), dan `prize_awards: [{rank, student: {id, name}, points, duration_seconds, coin_amount, awarded_at}]` (penerima hadiah; `[]` sebelum dibagikan). ID tidak ada / bukan UUID / sudah dihapus → 404. 2 query: quiz + nama (JOIN lewat primary key) dan daftar soal; +1 query penerima hadiah hanya kalau hadiah sudah dibagikan.

### Hapus quiz (`DELETE /api/v1/quizzes`)

Body `{"ids": ["...", "..."]}`, 1-100 quiz, **soft delete** (`deleted_at` & `deleted_by` diisi, baris tetap ada).

- **Semua atau tidak sama sekali**: ID tidak ada / sudah dihapus, atau quiz yang **sedang berjalan** → 422 dengan error per `ids[i]` (contoh `quiz "Tryout A" sedang berjalan (sampai 03-12-2026 17:00 WIB), tidak bisa dihapus`) dan tidak ada yang terhapus. Quiz yang belum mulai maupun yang sudah lewat boleh dihapus.
- Quiz yang dihapus tidak tampil di list quiz, list bundle, dan penanda "dipakai quiz" di Bank Soal; jadwal & kode aksesnya tidak lagi menghalangi quiz lain.
- 1 transaksi: quiz dikunci (`FOR UPDATE`) saat dicek, lalu 1 UPDATE.

### List quiz (`GET /api/v1/quizzes`)

Terbaru dibuat dulu. Semua query opsional dan bisa digabung:

| Query | Isi |
| ----- | --- |
| `page`, `per_page` | Default 1 & 20, `per_page` maks 100 |
| `search` | Nama quiz (sebagian kata, tidak peka huruf besar/kecil) |
| `schedule_status` | `upcoming` (akan dimulai), `ongoing` (sedang berjalan), `finished` (sudah lewat) |
| `type` | `daily_quiz` / `uji_kom` / `tryout` / `rumath` |
| `status` | `draft` / `published` |
| `education_level_id`, `grade_id`, `subject_id` | Filter letak |

Setiap baris = data quiz seperti detail (letak dengan nama, `start_at`, `end_at`, `duration_minutes`, `access_code`, `coin_cost`, `total_questions`, `status`, `created_by`/`updated_by`) + **`schedule_status`**, **tanpa** daftar soal. `schedule_status` dihitung dari waktu sekarang: sekarang < `start_at` → `upcoming`; `start_at` ≤ sekarang < `end_at` → `ongoing`; sesudahnya → `finished`.

**Angka tab jadwal:** `GET /api/v1/quizzes/schedule-counts` → `{"all": 12, "upcoming": 4, "ongoing": 1, "finished": 7}`. Query string sama dengan list (filter lain ikut berlaku; `schedule_status` & pagination diabaikan). 1 query `COUNT(*) FILTER (...)`.

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
- **Scheduler** di dalam proses API ([internal/scheduler](internal/scheduler/scheduler.go), library `robfig/cron`) mengganti kode akses **setiap jam tepat (WIB)** untuk quiz yang **belum selesai** (`end_at > sekarang`, termasuk yang belum mulai & yang sedang berjalan). Quiz yang sudah selesai atau dihapus tidak diubah.
- Jadwal diatur `ACCESS_CODE_REFRESH_CRON` di `.env` (format cron 5 kolom, default `0 * * * *`; `off` = mati).
- **Aman untuk beberapa server**: setiap jalan memakai advisory lock PostgreSQL (`pg_try_advisory_xact_lock`), jadi hanya 1 server yang mengganti kode; server lain mencatat "dilewati".
- Performa: 1 transaksi, quiz dicari lewat index `end_at` (`quizzes_access_code_end_at_idx`), kode baru dicek sekaligus ke database, lalu UPDATE banyak baris per 1.000 quiz.
- Hasil setiap jalan dicetak di terminal server, contoh `scheduler: kode akses 12 quiz diganti (35ms)`.

### Hadiah juara (scheduler 23:59 WIB)

Admin mengisi `prize_coins` saat buat/ubah quiz. Hadiah dibagikan **otomatis** oleh scheduler setiap hari **23:59 WIB** (`QUIZ_PRIZE_CRON` di `.env`, default `59 23 * * *`; `off` = mati) untuk quiz yang `published`, belum dihapus, event **sudah selesai** (`end_at` <= jam 23:59 itu), dan belum dibagikan (`prizes_distributed_at` null).

Per quiz (1 transaksi per quiz; gagal → dicoba lagi besok 23:59):

1. Juara diambil dari `leaderboard_entries` quiz itu dengan urutan yang **sama** dengan API leaderboard (point tertinggi, lalu waktu tersingkat). Point & waktu sama persis → peringkat sama → **hadiah sama** (contoh peringkat 1, 2, 2, 4: dua juara 2 dapat hadiah juara 2, peringkat 4 tidak dapat).
2. Juara yang hadiahnya > 0: `students.coin_balance` ditambah, dicatat di `coin_transactions` (`type = reward`, `reference_type = quiz_prize`, `reference_id` = quiz, `idempotency_key = quiz_prize:<quiz>:<siswa>`) dan `quiz_prize_awards` (peringkat, point, waktu, coin, transaksi coin).
3. `quizzes.prizes_distributed_at` diisi, juga untuk quiz tanpa hadiah / tanpa peserta, supaya tidak diproses lagi.

Aman dijalankan di beberapa server: baris quiz dikunci (`FOR UPDATE`) lalu dicek ulang, ditambah `idempotency_key` unik, jadi coin tidak bisa dobel. Quiz yang dihapus sebelum 23:59 tidak dibagikan hadiahnya. Hasil dicetak di terminal, contoh `scheduler: hadiah quiz dibagikan: 3 quiz, 7 siswa, 2100 coin, 0 quiz gagal`. Pencarian quiz yang menunggu memakai partial index `quizzes_prize_pending_idx` (migration 000025).

### Kode akses untuk tutor (`GET /api/v1/quizzes/:id/access-code`)

Role **tutor, admin, super admin**. Data: `quiz_id`, `name`, `type`, `status`, `start_at`/`end_at`, `schedule_status`, `access_code` (yang berlaku sekarang), `access_code_updated_at`. Tutor membagikan kode ini ke siswa saat quiz dimulai; kode berganti setiap jam selama event belum selesai. Quiz tidak ada / dihapus → 404.

### Buka Uji Kom / Tryout (`POST /api/v1/quizzes/:id/open`)

Khusus siswa. Body `{"access_code": "K7Q2ZD"}`.

- Quiz harus **publish**, untuk **kelas siswa**, dan **sedang berjalan**. Draf / kelas lain / tidak ada → 404 (siswa tidak boleh tahu quiz itu ada). Belum mulai → 409 `Quiz belum dimulai. Mulai 01-12-2026 08:00 WIB`; sudah selesai → 409 `Quiz sudah selesai pada ...`.
- **Pertama kali dibuka wajib kode akses** yang berlaku sekarang (huruf besar/kecil sama saja). Kosong → 422 `access_code: wajib diisi`; salah / sudah berganti → 422 `kode akses salah atau sudah berganti, minta kode terbaru ke tutor`.
- Saat berhasil, dicatat di tabel `quiz_attempts` (1 baris per siswa per quiz, `started_at`). **Membuka ulang** (contoh refresh halaman setelah kode berganti) **tidak perlu kode lagi** (boleh tanpa body) dan `started_at` tetap.
- Respons 200: info quiz (tanpa kode akses), `started_at`, `deadline_at` = **min(`started_at` + durasi, `end_at`)**, `total_questions`, `coin_cost`, dan `questions` lengkap dengan kunci jawaban (bentuk soal di bagian Daily Quiz).
- Biaya coin quiz (`coin_cost`) belum dipotong saat dibuka.

### Submit jawaban & review (Uji Kom, Tryout, Daily Quiz)

| Endpoint | Untuk |
| -------- | ----- |
| `POST /api/v1/quizzes/:id/submit`, `GET /api/v1/quizzes/:id/review` | Uji Kom / Tryout |
| `POST /api/v1/daily-quizzes/:id/submit`, `GET /api/v1/daily-quizzes/:id/review` | Daily Quiz |

Body submit (khusus siswa):

```json
{
  "answers": [
    { "question_id": "...", "answer": "B", "is_correct": true },
    { "question_id": "...", "answer": ["A", "C"], "is_correct": false },
    { "question_id": "...", "answer": "0.75", "is_correct": true }
  ]
}
```

- **Penilaian di FE**: FE menilai benar/salah tiap soal memakai kunci jawaban dari response buka quiz, lalu mengirim `is_correct` (wajib per item). `answer` = jawaban siswa apa adanya (bebas bentuknya, disimpan untuk review). Soal yang tidak dikirim = tidak dijawab (salah, `student_answer: null`).
- Waktu per soal (opsional): `answered_at` & `time_spent_seconds` (lihat Waktu pengerjaan di bawah).
- **Nilai total 100**: nilai per soal = 100 / total soal; nilai = jumlah benar x nilai per soal (dihitung server, 2 angka di belakang koma; semua benar selalu tepat 100). Contoh 3 soal, 2 benar → 66.67.
- **Point (bukan nilai)**, untuk leaderboard: soal benar easy = 10, medium = 15, hard = 20 point (level soal saat submit); salah / tidak dijawab = 0. `result.points` = total point; per soal: `score` (nilai) & `points` (point). Berlaku untuk semua quiz.
- **Hanya sekali.** Submit ke-2 → 409 `Jawaban sudah dikumpulkan pada ...`. Harus sudah dibuka (belum → 409). Uji Kom/Tryout: lewat `deadline_at` + toleransi 2 menit → 409 `Waktu mengerjakan sudah habis pada ...`. Daily Quiz tanpa batas waktu.
- Soal yang bukan bagian quiz / dikirim 2x → 422 di `answers[i].question_id`.
- Disimpan: `submitted_at`, `score`, `correct_count`, dan `answers` (JSONB, urut nomor soal: `{question_id, answer, is_correct, points}`) di `quiz_attempts` (Uji Kom/Tryout) atau `daily_quizzes` (Daily Quiz). Submit = 1 UPDATE (`... AND submitted_at IS NULL`, aman dari submit dobel); review = baca 1 baris + isi soal.

Response submit = response review (200):

```json
{
  "id": "...", "name": "Uji Kom Matematika", "type": "uji_kom", "placement": { ... }, "started_at": "...",
  "state": "submitted",
  "result": { "score": 66.67, "score_per_question": 33.33, "points": 25, "correct_count": 2, "total_questions": 3, "submitted_at": "2026-12-01T09:05:00+07:00", "duration_seconds": 1520 },
  "questions": [
    {
      "no": 1, "id": "...", "code": "Q-12", "type": "single_choice", "question": "...", "formula": null,
      "image_url": null, "table": null, "display_order": ["question"], "difficulty": "easy",
      "answer": { "options": [ ... ], "correct_key": "B" },
      "explanation": "Pembahasan ...",
      "student_answer": "B", "is_correct": true, "score": 33.33, "points": 10,
      "answered_at": "2026-12-01T08:05:30+07:00", "time_spent_seconds": 42
    }
  ]
}
```

Urutan `questions` sama dengan saat dikerjakan, lengkap dengan **jawaban yang benar** (`answer`) dan **pembahasan** (`explanation`). Daily Quiz: tanpa `name`/`type`/`started_at`, ada `difficulty` & `opened_at`.

**Waktu pengerjaan:**

- Per soal (opsional, dari FE) di setiap item `answers`: `answered_at` (RFC3339, jam soal terakhir dijawab, contoh `"2026-12-01T08:05:30+07:00"`) dan `time_spent_seconds` (lama mengerjakan soal itu, 0-86400). Disimpan di JSONB `answers` dan tampil lagi di review (`answered_at` dalam WIB; `null` kalau tidak dikirim).
- Total: `result.duration_seconds` = submit - mulai (Uji Kom/Tryout: `started_at` saat pertama dibuka; Daily Quiz: `opened_at`), dihitung server dan disimpan di kolom `duration_seconds` (migration 000022).

**Sudah pernah dikerjakan → langsung review:** buka quiz (`POST /quizzes/:id/open` atau `POST /daily-quizzes/:id/open`) untuk quiz yang sudah di-submit mengembalikan **data review** (bentuk sama dengan response submit) dengan `"state": "submitted"` — tanpa kode akses, tidak memotong jatah/coin, dan untuk Uji Kom/Tryout tetap bisa walaupun event sudah selesai. Response buka quiz biasa berisi `"state": "in_progress"`. FE cukup cek `data.state`: `in_progress` → halaman soal, `submitted` → halaman review.

### Pengumuman quiz di dashboard siswa (`GET /api/v1/student/quiz-announcements`)

Khusus siswa. Berisi quiz Uji Kom & Tryout yang **sudah publish**, **belum selesai** (`end_at` > sekarang), dan untuk **kelas siswa** (kelas akun siswa); urut yang paling dekat dimulai. Pagination `page`/`per_page` (default 20).

Setiap item: `id`, `name`, `type`, `placement` (nama letak), `start_at`/`end_at` (WIB), `schedule_status` (`upcoming`/`ongoing`), `duration_minutes`, `total_questions`, `coin_cost`, `points` (point per level), `prize_coins` (hadiah juara 1-3). **Tanpa kode akses** (dibagikan tutor saat quiz dimulai) dan tanpa soal. Siswa yang kelasnya belum diisi → `data: []`. Index `quizzes_announcement_idx (grade_id, start_at)` khusus quiz publish.

## Daily Quiz (siswa)

Quiz latihan yang **di-generate siswa** dari soal acak sesuai pilihannya. **Tanpa kode akses dan tanpa timer.** Disimpan di tabel terpisah `daily_quizzes` + `daily_quiz_questions` (migration 000019), bukan di `quizzes`, karena milik 1 siswa dan tidak ikut jadwal event / aturan bentrok / list quiz admin.

### Pilihan kelas (`GET /api/v1/daily-quizzes/grade-options`)

Dropdown kelas untuk siswa yang login (format sama dengan dropdown lain + `is_default`):

- Hanya kelas di **jenjang siswa**. Khusus SD dibagi 2 kelompok: **Kelas 1-3** dan **Kelas 4-6** (siswa Kelas 5 → Kelas 4, 5, 6).
- `is_default: true` = kelas siswa sendiri (dipakai kalau `grade_id` tidak dikirim).
- Kelas/jenjang siswa belum diisi → 422 `Kelas kamu belum diisi, hubungi admin`.

Bidang materi untuk kelas yang dipilih: `GET /api/v1/dropdowns/subject_field?subject_id=...&grade_id=...`.

### Generate (`POST /api/v1/daily-quizzes`)

```json
{
  "placement": {
    "grade_id": "...", "subject_id": "...", "subject_field_id": "...",
    "chapter_id": "...", "subchapter_id": null, "topic_id": null
  },
  "difficulty": "easy",
  "total_questions": 10
}
```

| Field | Aturan |
| ----- | ------ |
| `placement.grade_id` | Opsional (kosong = kelas siswa); harus dari pilihan kelas di atas |
| `placement.subject_id`, `placement.subject_field_id` | **Wajib**. Bidang materi harus milik mapel & kelas yang dipilih |
| `placement.chapter_id`, `subchapter_id`, `topic_id` | Opsional, berurutan (sub bab butuh bab, topik butuh sub bab) |
| `difficulty` | Level: `easy` / `medium` / `hard` = parameter `difficulty` soal. **Tipe soal boleh campur** (pilihan ganda, PG kompleks, benar/salah, isian) |
| `total_questions` | `5`, `10`, atau `15` |

- Soal diambil **acak** (`ORDER BY random()`) dari soal yang **aktif**, bertanda **Daily Quiz** (`used_for`), level sama, dan di letak yang dipilih. Kandidat sudah dipersempit index bidang materi, jadi tetap cepat.
- Soal kurang → 422 di `total_questions`: `soal yang tersedia hanya 8, pilih total soal yang lebih kecil atau ubah pilihan` (0 soal: `belum ada soal untuk pilihan ini, ...`).
- Respons 201: `id`, `placement` (nama), `difficulty`, `total_questions`, `created_at`, dan `questions: [{no, id, type, question, formula, image_url, table, display_order, options, input_type}]`. **Tanpa kunci jawaban & pembahasan.** `options` (`key`, `content`, `image_url`) untuk pilihan ganda / PG kompleks / benar-salah; `input_type` (`number`/`text`) untuk isian. Gambar Image Library sudah diubah menjadi link.
- Generate gratis; jatah & coin dihitung saat Daily Quiz **dibuka** (di bawah).

### Buka Daily Quiz (`POST /api/v1/daily-quizzes/:id/open`)

Siswa membuka Daily Quiz miliknya untuk dikerjakan. Tanpa body, tanpa kode akses, tanpa timer.

- **Jatah gratis per hari (WIB)**: `free_per_day` (pengaturan global, nilai awal 2) Daily Quiz pertama yang dibuka hari itu gratis. Berikutnya memotong `coin_cost` coin (nilai awal 10; 0 = selalu gratis). Keduanya diubah admin lewat `PUT /api/v1/daily-quiz-settings` dan langsung berlaku untuk Daily Quiz yang dibuka setelahnya (tanpa restart server).
- Coin dipotong dari `students.coin_balance` dan dicatat di `coin_transactions` (`type = usage`, `reference_type = daily_quiz`, `reference_id` = id Daily Quiz, `idempotency_key = daily_quiz_open:<id>`).
- Coin tidak cukup → **402** `Coin tidak cukup: jatah gratis hari ini sudah habis, butuh 10 coin, saldo 5 coin`; tidak ada yang berubah.
- **Membuka ulang** Daily Quiz yang sama tidak dihitung lagi dan tidak memotong coin (`usage.already_opened = true`).
- Aman dari 2 request bersamaan: 1 transaksi, saldo siswa dikunci (`FOR UPDATE`) selama jatah dihitung & coin dipotong.
- Daily Quiz siswa lain / tidak ada → 404.

Respons 200: info Daily Quiz, `opened_at`, `usage: {free_per_day, opened_today, coin_charged, coin_balance, already_opened}`, dan `questions` lengkap dengan kunci jawaban (lihat bentuk soal di bawah).

### Pengaturan global Daily Quiz (`GET` / `PUT /api/v1/daily-quiz-settings`)

Berlaku untuk **seluruh** Daily Quiz (tabel `daily_quiz_settings`, selalu 1 baris, migration 000026 & 000027): point per level, hadiah juara bulanan, jatah gratis per hari, dan biaya coin. Lihat: semua role yang login. Ubah: admin & super admin.

```json
{
  "points": { "easy": 10, "medium": 15, "hard": 20 },
  "prize_coins": { "rank_1": 300, "rank_2": 200, "rank_3": 100 },
  "free_per_day": 2,
  "coin_cost": 10
}
```

| Field | Aturan |
| ----- | ------ |
| `points` | Wajib (ketiganya), 0-1000: point 1 soal **benar** per level. Dipakai untuk Daily Quiz yang di-submit **setelah** diubah; point yang sudah tercatat tidak berubah. Nilai awal 10 / 15 / 20 |
| `prize_coins` | Wajib (ketiganya), 0-1.000.000: hadiah coin juara 1, 2, 3 bulanan **per kelas** (0 = tanpa hadiah). Nilai awal 0 |
| `free_per_day` | Wajib, 0-100: Daily Quiz gratis per siswa per hari (WIB). Nilai awal 2 (migration 000027, sebelumnya `DAILY_QUIZ_FREE_QUOTA` di `.env`) |
| `coin_cost` | Wajib, 0-1.000.000: coin per Daily Quiz setelah jatah gratis habis (0 = selalu gratis). Nilai awal 10 (sebelumnya `DAILY_QUIZ_COIN_COST`) |

Respons: `points`, `prize_coins`, `free_per_day`, `coin_cost`, `updated_by` (`{id, name}`, null = belum pernah diubah), `updated_at` (WIB). Jatah gratis & biaya coin langsung berlaku untuk Daily Quiz yang dibuka setelah diubah.

### Hadiah juara bulanan Daily Quiz (per kelas)

Dibagikan otomatis oleh scheduler hadiah (`QUIZ_PRIZE_CRON`, jalan setiap hari 23:59 WIB) pada **hari terakhir bulan jam 23:59 WIB**. Bulan lalu yang terlewat (contoh server mati) ikut dibagikan di jadwal berikutnya.

1. Bulan itu dicatat di `daily_quiz_prize_periods` beserta **snapshot** hadiah dari pengaturan saat itu (mengubah pengaturan setelahnya tidak mengubah bulan yang sudah dibagikan). Primary key `period` mencegah 1 bulan dibagikan 2x (juga kalau beberapa server jalan bersamaan).
2. Juara 1-3 **per kelas** dari `leaderboard_entries` bulan itu (kelas = snapshot kelas saat point masuk), urutan sama dengan leaderboard yang difilter kelas: point tertinggi, lalu waktu tersingkat; sama persis → peringkat & hadiah sama.
3. Setiap juara yang hadiahnya > 0: saldo coin bertambah + `coin_transactions` (`type = reward`, `reference_type = daily_quiz_prize`, `idempotency_key = daily_quiz_prize:<bulan>:<siswa>`) + `daily_quiz_prize_awards`.

Penerima: `GET /api/v1/daily-quiz-prizes?month=2026-11&grade_id=...` (admin & super admin) → `month`, `distributed_at` (null = belum), `prize_coins` (snapshot), `awards: [{grade, rank, student: {id, name, student_code}, points, duration_seconds, coin_amount, awarded_at}]` urut kelas lalu peringkat.

### Bentuk soal saat quiz dibuka (Daily Quiz, Uji Kom, Tryout)

```json
{
  "no": 1, "id": "...", "code": "Q-12", "type": "single_choice",
  "question": "KPK dari 4 dan 6 adalah ...", "formula": null,
  "image_url": "http://.../storage/images/question/2026/11/a.png",
  "table": null, "display_order": ["question", "image"], "difficulty": "easy",
  "answer": {
    "options": [
      { "key": "A", "content": "10", "image_id": null },
      { "key": "B", "content": "", "image_id": "...", "image_url": "http://.../b.png" }
    ],
    "correct_key": "B"
  }
}
```

- `answer` = opsi + **kunci jawaban**, bentuk sama dengan create soal (`correct_key` / `correct_keys` / `accepted_answers` + `input_type`, dst.).
- **Pembahasan (`explanation`) tidak diberikan.**
- Gambar Image Library (soal & opsi) sudah diubah menjadi link `image_url`. 2 query untuk semua soal (soal + link gambar opsi).

## Leaderboard

**Nilai ≠ point.** Nilai (0-100) hanya untuk hasil quiz. Leaderboard memakai **point**: setiap soal **benar** mendapat point sesuai level soal — easy **10**, medium **15**, hard **20**; salah / tidak dijawab **0**. Berlaku untuk semua quiz (Daily Quiz, Uji Kom, Tryout). Point dihitung server saat submit dan disimpan di kolom `points` (migration 000023). Besar point per level bisa diubah admin: **Uji Kom & Tryout** per quiz (`points` saat buat/ubah quiz, migration 000025), **Daily Quiz** global untuk seluruh Daily Quiz (`PUT /api/v1/daily-quiz-settings`, migration 000026).

**Tabel leaderboard (migration 000024)** — diisi di **transaksi yang sama** dengan submit (submit & leaderboard berhasil/gagal bersama):

| Tabel | Isi |
| ----- | --- |
| `leaderboard_entries` | 1 baris per siswa per papan: Daily Quiz per bulan (`board = daily_quiz`, `period = 2026-11`) atau 1 quiz Uji Kom/Tryout (`board = uji_kom/tryout`, `quiz_id`). Berisi total `points`, total `duration_seconds`, `quiz_count`, `score` (Uji Kom/Tryout), snapshot `education_level_id` & `grade_id`, `last_submitted_at`. Unik per (board, period, quiz_id, siswa) |
| `leaderboard_point_logs` | Riwayat setiap point masuk: `entry_id`, `source_type` (`daily_quiz` / `quiz_attempt`) + `source_id` (pengerjaan asalnya), `points`, `duration_seconds`, `created_at`. 1 pengerjaan hanya bisa tercatat 1x (unik `source_type + source_id`) |

Contoh melacak point 1 siswa: `SELECT * FROM leaderboard_point_logs WHERE student_id = ... ORDER BY created_at DESC`. Hasil quiz yang sudah di-submit sebelum tabel ini ada ikut diisi oleh migration.

**Urutan peringkat:** point tertinggi; kalau point sama, **total waktu pengerjaan paling singkat** (`duration_seconds`) di atas; kalau point & waktu sama persis → peringkat sama (`RANK`, contoh 1, 2, 2, 4).

Semua role yang login boleh melihat. Kalau yang melihat siswa, posisinya sendiri ada di `data.me` (null kalau belum ikut).

### Daily Quiz (`GET /api/v1/leaderboards/daily-quiz`)

Total point **semua Daily Quiz yang di-submit dalam 1 bulan** (WIB) per siswa.

| Query | Isi |
| ----- | --- |
| `month` | `YYYY-MM`, default bulan ini (WIB). Rentang = tanggal 1 00:00 WIB s/d tanggal 1 bulan berikutnya |
| `education_level_id` | Filter jenjang (jenjang akun siswa) |
| `grade_id` | Filter kelas (kelas akun siswa) |
| `page`, `per_page` | Default 1 & 20, maks 100 |

Peringkat dihitung di dalam filter (contoh filter Kelas 5: juara 1 = siswa Kelas 5 dengan point tertinggi). Filter memakai **snapshot** jenjang & kelas siswa saat point masuk, jadi peringkat bulan lalu tidak berubah walaupun siswa naik kelas.

```json
{
  "data": {
    "month": "2026-11", "from": "2026-11-01T00:00:00+07:00", "to": "2026-12-01T00:00:00+07:00",
    "entries": [{
      "rank": 1,
      "student": { "id": "...", "name": "Budi Santoso", "student_code": "STU000001", "school_origin": "SDN 1" },
      "education_level": { "id": "...", "name": "SD" }, "grade": { "id": "...", "name": "Kelas 5" },
      "points": 150, "duration_seconds": 1830, "quiz_count": 6
    }],
    "me": { "rank": 12, "...": "..." }
  },
  "meta": { "page": 1, "per_page": 20, "total": 87 }
}
```

`duration_seconds` & `quiz_count` = total lama mengerjakan & jumlah Daily Quiz bulan itu. Dibaca langsung dari `leaderboard_entries` (`board = daily_quiz`, `period = YYYY-MM`), index `leaderboard_entries_daily_idx` — tidak menjumlah ulang hasil quiz.

Hadiah bulanan: `data.prize_coins` (`{rank_1, rank_2, rank_3}`, hadiah juara per kelas: snapshot kalau bulan itu sudah dibagikan, selain itu pengaturan saat ini) dan `data.prizes_distributed_at` (WIB, null = belum). Kalau difilter `grade_id`, setiap entry (dan `me`) punya `prize_coins` = hadiah untuk peringkatnya di kelas itu (0 di luar juara 1-3); tanpa filter kelas field ini tidak ada, karena juara dihitung per kelas.

### Uji Kom / Tryout (`GET /api/v1/leaderboards/quizzes/:id`)

**1 quiz = 1 leaderboard** (Uji Kom & Tryout masing-masing, tidak digabung). Peserta = siswa yang sudah submit quiz itu; `points` & `duration_seconds` dari pengerjaannya, `score` = nilainya (info). `data.quiz` = `{id, name, type, start_at, end_at, prize_coins: {rank_1, rank_2, rank_3}, prizes_distributed_at}`; setiap entry punya `prize_coins` = hadiah untuk peringkatnya (0 di luar juara 1-3; sebelum hadiah dibagikan masih sementara). Pagination `page`/`per_page`.

Siswa hanya boleh melihat quiz yang sudah publish untuk kelasnya (selain itu 404); tutor & admin boleh semua quiz. Dibaca dari `leaderboard_entries` (`quiz_id`), index `leaderboard_entries_quiz_idx`.

## Report quiz (Uji Kom & Tryout)

Report **per event** (1 quiz = 1 event) untuk quiz Uji Kom & Tryout yang sudah **publish** (draf & terhapus tidak ikut). Akses: super admin, admin, tutor (hanya membaca).

- **Peserta** = siswa yang sudah **submit** (yang baru membuka belum dihitung); akun yang sudah dihapus tidak dihitung, sama dengan leaderboard.
- **Peringkat** = aturan leaderboard & hadiah: point tertinggi, lalu waktu tersingkat; sama persis → peringkat sama.

### List per event (`GET /api/v1/quiz-reports`)

Event terbaru dulu (tanggal mulai). Query opsional: `type` (`uji_kom`/`tryout`), `search` (nama quiz), `education_level_id`, `grade_id`, `subject_id`, `date_from` & `date_to` (tanggal mulai event `YYYY-MM-DD` WIB, inklusif), `page`, `per_page` (maks 100).

```json
{ "quiz_id": "...", "name": "Tryout Demo Matematika Kelas 6", "type": "tryout",
  "education_level": {...}, "grade": {...}, "subject": {...},
  "start_at": "2026-11-18T08:00:00+07:00", "end_at": "2026-11-18T10:00:00+07:00", "total_questions": 10,
  "participants": 13, "average_score": 61.54, "highest_score": 80, "lowest_score": 40 }
```

`average_score` / `highest_score` / `lowest_score` = nilai 0-100 (`null` kalau belum ada peserta). Performa: halaman quiz diambil dulu (CTE, index `quizzes_report_idx`), statistik dihitung hanya untuk quiz di halaman itu lewat index `quiz_attempts_report_idx` (migration 000032).

### Detail per event (`GET /api/v1/quiz-reports/:id`)

`quiz` = ringkasan event (bentuk sama dengan list) + `participants[]` urut peringkat: `rank`, `student {id, name, student_code}`, `correct_count`, `total_questions`, `correct_label` (contoh `"9/10"`), `score` (nilai), `points`, `duration_seconds` (waktu penyelesaian), `submitted_at`. Query: `search` (nama / kode siswa; peringkat tetap peringkat asli), `page`, `per_page`. Quiz tidak ada / draf / dihapus / bukan Uji Kom-Tryout → 404.

### Soal paling sering salah (`GET /api/v1/quiz-reports/:id/questions`)

Semua soal quiz, urut dari **persentase benar terendah** (paling sering salah) ke tertinggi; sama → salah lebih banyak dulu, lalu nomor soal. Setiap soal: `no` (nomor di quiz), `id`, `code`, `type`, `question`, `difficulty`, `participants`, `correct_count`, `wrong_count` (menjawab tapi salah), `unanswered_count` (tidak dijawab), `correct_percent` (0-100, contoh `13.33`), `correct_label` (contoh `"13%"`). Dihitung dari jawaban yang tersimpan saat submit (kolom JSONB `answers`), 1 query.

## Report Daily Quiz (per siswa)

Daily Quiz dibuat siswa sendiri (soal acak, mapel bebas), jadi **tidak ada event**: report langsung **per siswa**. Akses: super admin, admin, tutor (hanya membaca). Yang dihitung: Daily Quiz yang sudah **submit** di periode; akun yang sudah dihapus tidak ikut.

**Periode** = tanggal submit (WIB): `date_from` & `date_to` (`YYYY-MM-DD`, inklusif, maks 366 hari). Kosong = **bulan berjalan** (sama dengan leaderboard Daily Quiz bulanan); hanya `date_from` = sampai hari ini; hanya `date_to` = mulai tanggal 1 bulan itu.

**Mapel:** karena siswa bebas memilih mapel, report punya 2 cara melihatnya:

- **Filter `subject_id`** di list → semua angka & peringkat hanya dari Daily Quiz mapel itu (peringkat per mapel). Tanpa filter = semua mapel digabung.
- **Detail per siswa** → rekap total + **rincian per mapel**.

### List (`GET /api/v1/daily-quiz-reports`)

Query opsional: `date_from`, `date_to`, `subject_id`, `education_level_id` & `grade_id` (jenjang/kelas **akun** siswa), `search` (nama / kode siswa), `page`, `per_page` (maks 100).

```json
{
  "data": {
    "period": { "date_from": "2026-11-01", "date_to": "2026-11-30" },
    "students": [{
      "rank": 1,
      "student": { "id": "...", "name": "Hafiz Pratama", "student_code": "STU000035", "education_level": {...}, "grade": {...} },
      "total_quizzes": 6, "quizzes_by_difficulty": { "easy": 2, "medium": 2, "hard": 2 },
      "average_score": 63.33, "average_duration_seconds": 467,
      "total_questions": 55, "correct_count": 35, "wrong_count": 20, "correct_percent": 63.64,
      "total_points": 505, "total_duration_seconds": 2804
    }]
  },
  "meta": { "page": 1, "per_page": 20, "total": 66 }
}
```

| Kolom | Isi |
| ----- | --- |
| `rank` | Peringkat di dalam filter: **total point tertinggi**, lalu **total waktu tersingkat** (aturan leaderboard); sama persis → peringkat sama. Pencarian nama tidak mengubah peringkat |
| `total_quizzes`, `quizzes_by_difficulty` | Jumlah Daily Quiz, dan per level (level dipilih siswa saat generate) |
| `average_score`, `average_duration_seconds` | Rata-rata nilai (0-100) & rata-rata waktu pengerjaan per Daily Quiz |
| `total_questions` | Soal yang dikerjakan (semua soal di Daily Quiz yang di-submit) |
| `correct_count`, `wrong_count`, `correct_percent` | Benar, tidak benar (salah + tidak dijawab), persentase benar |
| `total_points`, `total_duration_seconds` | Total point & total waktu |

### Detail 1 siswa (`GET /api/v1/daily-quiz-reports/students/:id`)

Query: `date_from`, `date_to`. Respons: `period`, `student`, `total` (rekap gabungan semua mapel, kolom sama dengan list tanpa `rank`), `subjects[]` = `{subject: {id, name}, ...kolom rekap}` per mapel, urut jumlah Daily Quiz terbanyak. Bukan siswa / tidak ada → 404.

Performa: rekap dihitung dari kolom ringkasan `daily_quizzes` (bukan jawaban per soal) lewat index `daily_quizzes_report_idx` & `daily_quizzes_student_report_idx` yang berisi semua kolom yang dijumlahkan (index-only scan, migration 000034).

## User (menu admin)

Semua endpoint `/api/v1/users` khusus **admin & super admin**. Melihat: semua akun. **Menambah, mengubah & menghapus**: super admin semua akun; **admin hanya tutor & siswa** (akun admin / super admin → 403). Akun sendiri tidak bisa dihapus atau dinonaktifkan. Role tidak bisa diubah.

### List (`GET /api/v1/users`) & jumlah per role (`GET /api/v1/users/role-counts`)

| Query | Isi |
| ----- | --- |
| `role` | `super_admin`, `admin`, `tutor`, `student` |
| `education_level_id` | Jenjang (siswa) |
| `grade_id` | Kelas (siswa) |
| `branch_id` | Cabang |
| `status` | `active` / `inactive` |
| `search` | Nama, username, atau email (tidak peka huruf besar/kecil) |
| `page`, `per_page` | Default 1 & 20, maks 100 |

List urut terbaru dulu. Setiap baris: `id, username, name, email, phone, role, branch` (null untuk super admin = semua cabang), `education_level, grade, student_code, coin_balance` (siswa), `status, join_date, end_date, ended` (true = end_date sudah lewat, tidak bisa login), `last_login_at, created_at`. User yang sudah dihapus tidak tampil.

`role-counts` = angka tab `{all, super_admin, admin, tutor, student}` dengan filter yang sama (role diabaikan), 1 query `GROUP BY`.

Performa: halaman diambil dulu (CTE, index `users_list_idx`) lalu di-JOIN ke cabang/jenjang/kelas/siswa; pencarian memakai index trigram `users_real_name_trgm_idx`, `users_username_trgm_idx`, `users_email_trgm_idx` (migration 000029); COUNT dilewati di halaman terakhir yang tidak penuh.

### Detail (`GET /api/v1/users/:id`) & ubah (`PUT /api/v1/users/:id`)

Detail = data list + `parent_phone, school_origin, date_of_birth, target_school` (siswa), `updated_by, updated_at`. Tidak ada / dihapus → 404.

```json
{
  "username": "siswa.budi", "name": "Budi Santoso", "email": null, "phone": "0812...",
  "parent_phone": "0813...", "school_origin": "SMPN 1 Surabaya",
  "branch_id": "...", "education_level_id": "...", "grade_id": "...", "status": "active",
  "join_date": "2025-07-01", "end_date": "2026-06-30",
  "password": "passwordBaru123", "date_of_birth": "2012-05-17", "target_school": "SMAN 5 Surabaya"
}
```

| Field | Aturan |
| ----- | ------ |
| `username` | Wajib, 3-30 karakter huruf kecil/angka/titik/garis bawah (otomatis dikecilkan), unik termasuk akun yang sudah dihapus |
| `name`, `status` | Wajib |
| `email` | Opsional, format email |
| `branch_id` | Wajib untuk siswa, tutor, admin (cabang aktif); harus kosong untuk super admin |
| `education_level_id`, `grade_id` | Wajib untuk siswa, kelas harus milik jenjang; harus kosong untuk role lain |
| `password` | Opsional 8-72 karakter: diisi = reset password |
| `join_date` | **Wajib**, tanggal bergabung `YYYY-MM-DD` (migration 000031) |
| `end_date` | Opsional, tanggal berakhir `YYYY-MM-DD`, tidak boleh sebelum `join_date`. **Lewat tanggal ini (WIB) akun tidak bisa login** (403 "Masa aktif akun Anda sudah berakhir"); di tanggal itu sendiri masih bisa. Kosong = tanpa batas |
| `date_of_birth`, `target_school` | Khusus siswa (`YYYY-MM-DD`, tidak boleh di masa depan) |

Ganti penuh: field opsional yang tidak dikirim menjadi kosong. Semua kesalahan dikembalikan sekaligus (422). **Dinonaktifkan, password diganti, atau `end_date` sudah lewat → semua sesi login user itu diakhiri** (refresh token dicabut; access token yang sudah terbit tetap berlaku sampai habis, maks `JWT_ACCESS_TTL`). Kolom `updated_by` mencatat admin pengubah.

### Tambah (`POST /api/v1/users`, 1-100 akun sekaligus)

```json
{
  "users": [
    { "role": "student", "username": "siswa.andi", "name": "Andi Wijaya", "password": "password123",
      "branch_id": "...", "education_level_id": "...", "grade_id": "...",
      "parent_phone": "0813...", "school_origin": "SDN 1 Surabaya", "date_of_birth": "2014-02-01", "target_school": "SMPN 1 Surabaya" },
    { "role": "tutor", "username": "tutor.sari", "name": "Sari Dewi", "password": "password123", "branch_id": "..." }
  ]
}
```

- Field per akun sama dengan **ubah** (tabel di atas, termasuk `join_date` wajib & `end_date`), ditambah `role` wajib (`super_admin`, `admin`, `tutor`, `student`), `password` wajib (8-72), `status` opsional (default `active`).
- Admin hanya boleh menambah tutor & siswa (`users[i].role` → 422); super admin semua role.
- **Semua atau tidak sama sekali**: semua kesalahan dikembalikan sekaligus per `users[i].<field>`, contoh username kembar di dalam request (`sama dengan users[0]`) atau sudah dipakai akun lain (termasuk akun yang sudah dihapus). Lebih dari 100 akun → 422 field `users`.
- Respons 201: akun yang dibuat, urut sesuai request (bentuk sama dengan baris list). Kode siswa (`STU000123`) dibuat otomatis urut sesuai request. Kolom `users.created_by` mencatat admin pembuat (migration 000030).
- Performa: data pendukung (cabang, jenjang/kelas, username terpakai) masing-masing **1 query untuk semua akun**; hash password bcrypt **paralel** (worker = jumlah CPU; 100 akun ±2 detik, berurutan bisa ±7 detik); simpan 2 query (`INSERT` banyak baris ke `users`, lalu `students`) dalam 1 transaksi.

### Cek file Excel tambah user (`POST /api/v1/users/import/check`)

Upload Excel **hanya mengecek**, tidak menyimpan (alur sama dengan import soal). Template: [docs/Template Upload/template_user.xlsx](docs/Template%20Upload/template_user.xlsx), contoh terisi: `contoh_terisi_user.xlsx` (sheet `Template` + `Petunjuk`).

Request: `multipart/form-data`, field `file` (.xlsx, maks 5 MB, **maks 100 user**).

| Kolom | Isi |
| ----- | --- |
| `role` | Wajib. `student`/`siswa`, `tutor`, `admin`, `super_admin` (tidak peka huruf besar/kecil) |
| `username`, `name`, `password` | Wajib |
| `email`, `phone` (no HP), `parent_phone` (no HP ortu) | Opsional. No HP yang terbaca sebagai angka (0 di depan hilang, contoh `812...`) otomatis ditambah `0` |
| `branch_name` | **Nama** cabang (tidak peka huruf besar/kecil) → `branch_id` |
| `education_level_name`, `grade_name` | **Nama** jenjang & kelas → ID. Jenjang boleh kosong kalau nama kelasnya hanya ada di 1 jenjang (contoh `Kelas 5` → SD) |
| `school_origin`, `target_school` | Opsional |
| `join_date` | **Wajib**. `2026-01-15`, `15/01/2026`, `15-01-2026`, atau sel berformat tanggal Excel |
| `end_date`, `date_of_birth` | Opsional, format tanggal sama |
| `status` | `active`/`aktif` atau `inactive`/`nonaktif`; kosong = active |

- Kolom wajib (`role, username, name, password, join_date`) harus ada di baris 1; kolom tidak dikenal ditolak. Baris kosong dilewati.
- Nama cabang (1 query) dan jenjang/kelas (1 query) dicocokkan sekali untuk seluruh file; setiap baris dicek dengan aturan yang **sama persis** dengan tambah user (format, role yang boleh dibuat, cabang/kelas per role, tanggal, username kembar di file maupun di database).
- 1 sel = 1 error (error pertama, contoh `cabang "X" tidak ditemukan`, tidak ditumpuk "wajib diisi").

Valid (200) → `data.users` bisa langsung dikirim FE sebagai `{"users": data.users}` ke `POST /api/v1/users` (field `row` & `*_name` diabaikan):

```json
{ "total": 2, "users": [{ "row": 2, "role": "student", "username": "siswa.andi", "password": "password123",
  "branch_id": "...", "branch_name": "Surabaya Timur", "education_level_id": "...", "education_level_name": null,
  "grade_id": "...", "grade_name": "Kelas 5", "join_date": "2026-01-15", "end_date": "2026-12-31", "status": "active", "...": "..." }] }
```

Ada error (422) → **semua data ditolak**, semua error dikembalikan sekaligus dengan `row` & `column` Excel, contoh `{"field": "branch_name", "message": "cabang \"Cabang X\" tidak ditemukan", "row": 3, "column": "H"}`.

### Hapus (`DELETE /api/v1/users`)

Body `{"ids": [...]}` (1-100). **Soft delete**: `deleted_at` & `deleted_by` diisi, baris & riwayat (nilai, coin, leaderboard) tetap ada, username tetap terpakai, sesi login diakhiri, tidak bisa login lagi. Semua atau tidak sama sekali: ID tidak ada, akun sendiri, atau akun yang tidak boleh dihapus → 422 per `ids[i]`.

## Coin

Saldo ada di `students.coin_balance` (tidak boleh negatif); setiap perubahan saldo selalu dicatat 1 baris `coin_transactions` di transaksi database yang sama (BR-010, BR-012).

### Paket isi coin (`/api/v1/coin-packages`)

| Method | Isi |
| ------ | --- |
| `GET /coin-packages` | Siswa: paket **aktif** saja (tab "Isi coin"): `id, name, coin_amount, price, description`. Admin & super admin: semua paket + `sort_order, status, created_by, updated_by, created_at, updated_at`; query `status` (active/inactive), `search` (nama). Urut `sort_order`, lalu jumlah coin. Tanpa pagination (paket sedikit) |
| `GET /coin-packages/:id` | Detail 1 paket (admin). Tidak ada / dihapus → 404 |
| `POST /coin-packages` | Buat paket (admin), respons 201 |
| `PUT /coin-packages/:id` | Ubah paket, ganti penuh (admin). Harga baru hanya untuk top-up berikutnya; riwayat lama menyimpan `price_paid` saat itu |
| `DELETE /coin-packages` | Hapus 1-100 paket, body `{"ids": [...]}` (soft delete). Ada ID yang tidak ada → 422 per `ids[i]`, tidak ada yang terhapus |

```json
{ "name": "300 coin", "coin_amount": 300, "price": 68000, "description": "Hemat 10%", "sort_order": 2, "status": "active" }
```

`name` wajib (maks 100, unik tanpa peduli huruf besar/kecil di antara paket yang belum dihapus: index `coin_packages_name_unique`), `coin_amount` 1-1.000.000, `price` wajib (rupiah, 0 boleh), `description` opsional (maks 150), `sort_order` default 0, `status` default `active`. Kolom audit `created_by`/`updated_by`/`deleted_by` (migration 000028).

### Top-up coin siswa oleh admin (`POST /api/v1/coin-topups`)

Admin memilih **1 paket aktif** lalu **1-100 siswa** sekaligus; setiap siswa mendapat coin sebanyak paket itu. Dicatat sebagai **pembelian tunai di cabang**: `coin_transactions` `type = purchase`, `reference_type = coin_package`, `price_paid` = harga paket, `created_by` = admin.

```json
{ "coin_package_id": "...", "student_ids": ["...", "..."], "note": "Bayar tunai di kasir", "request_id": "uuid-baru-setiap-klik" }
```

- **Semua atau tidak sama sekali**: paket tidak ada / nonaktif (`coin_package_id`) atau ID bukan siswa (`student_ids[i]`) → 422, tidak ada coin yang berubah.
- `note` opsional (maks 150), ikut di deskripsi transaksi: `Isi coin paket 300 coin (dicatat admin): Bayar tunai di kasir`.
- `request_id` opsional tapi disarankan (UUID baru dari FE setiap tombol simpan ditekan): request yang sama terkirim 2x → **409**, coin tidak bertambah 2x (`idempotency_key = coin_topup:<request_id>:<siswa>`).
- Admin & super admin boleh semua siswa (belum ada pembatasan cabang).
- Respons 201: `package`, `total_coins`, `total_price`, `students: [{id, name, student_code, coin_added, balance_after, transaction_id}]` urut sesuai `student_ids`.
- Performa: cek paket 1 query, cek siswa 1 query, simpan **1 query** (`UPDATE students ... RETURNING` + `INSERT coin_transactions ... SELECT` dalam 1 CTE) di 1 transaksi.

### Riwayat coin

- `GET /api/v1/student/coin-transactions` (siswa sendiri) dan `GET /api/v1/students/:id/coin-transactions` (admin & super admin; bukan siswa / tidak ada → 404).
- Query: `type` (`plan_grant`, `purchase`, `usage`, `admin_adjustment`, `refund`, `reward`), `page`, `per_page` (default 20, maks 100). Urut terbaru dulu.

```json
{
  "data": {
    "student": { "id": "...", "name": "Citra Ayu", "student_code": "STU000002", "coin_balance": 290 },
    "daily_quiz": { "free_per_day": 2, "used_today": 1, "remaining_free": 1, "coin_cost_after": 10 },
    "transactions": [{
      "id": "...", "type": "purchase", "type_label": "Isi coin", "amount": 300, "balance_after": 300,
      "price_paid": 68000, "reference": { "type": "coin_package", "id": "...", "name": "300 coin" },
      "description": "Isi coin paket 300 coin (dicatat admin)", "created_by": { "id": "...", "name": "Admin Surabaya Timur" },
      "created_at": "2026-11-25T10:00:00+07:00"
    }]
  },
  "meta": { "page": 1, "per_page": 20, "total": 12 }
}
```

- `daily_quiz` (jatah Daily Quiz gratis hari ini, FR-TOK-006) hanya di riwayat milik siswa sendiri; untuk admin `null`.
- `amount` positif = masuk, negatif = keluar. `created_by` null = sistem / siswa sendiri (contoh buka Daily Quiz, hadiah).
- `reference.name` diisi untuk paket coin (nama paket, juga paket yang sudah dihapus), hadiah quiz (nama quiz), dan paket langganan (nama plan).
- Performa: halaman diambil dulu lewat index `coin_transactions_student_id_created_at_idx`, baru di-JOIN ke tabel referensi; COUNT dilewati di halaman terakhir yang tidak penuh.

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
