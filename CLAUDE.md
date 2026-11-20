# Rumbasha Backend

Backend API untuk Rumbasha E-Learning Platform (klien: PT Rumbasha Menggapai Asa; dikerjakan oleh PT Khalief Techcraft Innovation).
Siswa SD dan SMP belajar lewat modul PDF, Daily Quiz, Uji Kom/Tryout, dan RUMATH. Admin mengelola konten, user, paket, dan laporan.
Frontend (React) dikerjakan terpisah dan memakai kontrak di `docs/openapi.yaml`.

## Sumber kebenaran
- `docs/BRD.md`: kebutuhan bisnis (BRD v1.2). Setiap fitur punya ID `FR-xxx-nnn`. Keputusan yang belum final ada di Bagian 15 sebagai `BD-nnn`.
- `docs/ROADMAP.md`: urutan dan jadwal pengerjaan per fitur (slice).
- `docs/STATUS.md`: status per fitur untuk laporan klien. **Perbarui setiap fitur selesai.**
- `docs/openapi.yaml`: kontrak API. Ubah kontrak lebih dulu, baru kode.

Jangan membaca seluruh BRD sekaligus. Cari bagian yang relevan dengan `grep -n "FR-LMS" docs/BRD.md` atau buka heading terkait, lalu baca hanya bagian itu beserta Bagian 4 (Business Rules), 8 (Database), dan 9 (API standards).

## Stack
Go (Gin), PostgreSQL, modular monolith, deploy Linux VPS (Hostinger).
Default yang dipakai (ubah hanya bila ada alasan kuat, catat di `docs/DECISIONS.md`):
- `pgx/v5` + `sqlc` untuk akses data, `golang-migrate` untuk migrasi (`migrations/`).
- JWT access token (singkat) + refresh token (disimpan ter-hash di tabel `refresh_tokens`). Password memakai bcrypt.
- Konfigurasi dari environment (`.env`, contoh di `.env.example`). Tidak ada kredensial di kode.
- File (PDF, gambar) lewat interface `storage.Storage`; implementasi awal disk lokal, bisa diganti S3-compatible nanti. DB hanya menyimpan metadata dan path.

## Struktur
Ikuti BRD Bagian 7: `cmd/api`, `internal/platform/*` (database, logger, config, middleware, storage, response), `internal/modules/<modul>/{handler,service,repository,model,dto}`.
Aturan: modul tidak mengimpor `repository` modul lain. Komunikasi antarmodul lewat interface `service`. Handler tipis, logika bisnis ada di `service`.

## Konvensi
- Base path `/api/v1`. Format response sukses dan error mengikuti BRD Bagian 9 (`success`, `message`, `data`, `meta` / `errors[]`).
- Primary key `uuid` (`gen_random_uuid()`). Kolom waktu `timestamptz` disimpan UTC; tampilan dan reset harian memakai `Asia/Jakarta`.
- Soft delete (`deleted_at`) untuk data yang dihapus admin. Hapus permanen hanya bila diminta eksplisit (lihat BD-026).
- Setiap tabel punya `created_at`, `updated_at`. Tabel data cabang punya `branch_id` (lihat BD-025).
- Otorisasi di middleware berdasarkan role (`super_admin`, `admin`, `tutor`, `student`). Jangan mengandalkan FE untuk pembatasan akses.
- Validasi semua request. Jangan mengembalikan password atau data sensitif. Pagination wajib pada endpoint list.
- Rumus matematika disimpan sebagai teks LaTeX (`$...$`); backend tidak merender.
- Nama dan komentar kode dalam bahasa Inggris; pesan ke pengguna dalam bahasa Indonesia.

## Asumsi sementara untuk keputusan yang belum final
Pakai asumsi ini sampai klien menjawab, dan tandai dengan komentar `// BD-nnn` di kode terkait:
- BD-025 (cabang): `users.branch_id` nullable; Super Admin tanpa cabang. Filter cabang disiapkan di repository tetapi belum dipaksakan.
- BD-026 (hapus): semua penghapusan = soft delete.
- BD-016 (poin): Daily Quiz poin = nilai ÷ 2 (sesuai desain admin Leaderboard). Rumus harus bisa diganti lewat konfigurasi.
- BD-031 (Subbab/Topik): tidak dibuat dulu. Hierarki: Jenjang, Kelas, Mapel, Bidang materi, Bab.

## Perintah
Target `Makefile` (buat pada slice 0):
`make run`, `make test`, `make lint`, `make migrate-up`, `make migrate-down`, `make sqlc`, `make seed`, `make docs`.
Database lokal lewat `docker compose up -d db`.

## Alur kerja per fitur
1. Baca bagian BRD yang relevan dan item `docs/ROADMAP.md` untuk slice tersebut.
2. Buat rencana singkat dan tunjukkan sebelum menulis kode (plan mode).
3. Perbarui `docs/openapi.yaml`.
4. Migrasi, query sqlc, repository, service, handler.
5. Test: unit untuk logika (penilaian, kuota, poin) dan integration untuk alur utama (auth, publish materi, submit quiz).
6. Seed data demo (`make seed`) dan perbarui `docs/STATUS.md`.

Definition of Done mengikuti BRD Bagian 16.

## Jangan
- Jangan menambah fitur di luar slice yang sedang dikerjakan.
- Jangan mengubah `docs/BRD.md` kecuali diminta; BRD dikelola terpisah (versi .docx dari klien).
- Jangan menaruh akun demo atau kredensial di kode produksi. Akun demo hanya lewat `make seed` di lingkungan non-production.
- Jangan membuat migrasi yang mengubah migrasi lama; selalu migrasi baru.
