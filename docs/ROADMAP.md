# Roadmap Backend Rumbasha

Target laporan ke klien: **Minggu, 29 November 2026** (batas efektif kerja: Sabtu 28 November).
Prinsip: kerjakan per fitur sampai bisa didemo (vertical slice). Dua sampai tiga fitur yang matang lebih baik daripada banyak fitur setengah jadi.

## Slice 0: Fondasi (18–19 Nov)
- Repo Go (Gin), struktur modular (BRD Bagian 7), `docker-compose.yml` (PostgreSQL), `Makefile`, `.env.example`.
- Migrasi awal, `golang-migrate`, `sqlc`, logger, config, middleware (request id, recovery, CORS), helper response/error sesuai BRD Bagian 9.
- Endpoint `GET /health`, halaman dokumentasi API dari `docs/openapi.yaml`.
- Rujukan: BRD Bagian 5, 6, 7, 9, 16.
- Hasil: `make run` jalan, health check hijau, FE bisa membaca OpenAPI.

## Slice 1: Auth dan RBAC (19–20 Nov)
- FR: AUTH-001 sampai 008, 010, 013, 014 (AUTH-006 reset password: bisa menyusul).
- Tabel: `users`, `roles`/`user_roles`, `refresh_tokens`, `branches`.
- Login email atau nomor HP, refresh, logout, ganti kata sandi, middleware role.
- Seed: 1 Super Admin, 1 Admin, 1 Siswa (hanya lingkungan non-production).
- Demo: login tiga role dan diarahkan sesuai role.

## Slice 2: Cabang, Master Data, User admin (20–22 Nov)
- FR: MST-001 sampai 007, STU-001 sampai 006, STU-012 sampai 015, STU-017, TUT-007.
- Tabel: `master_data_items` (mapel, bidang materi, kelas, sekolah), `branches`, profil siswa dan tutor.
- Stretch: STU-016 (Import Excel user).
- Demo: admin membuat, mengubah, mencari, dan menonaktifkan user; mengelola master data.

## Slice 3: Materi (22–25 Nov)
- FR: LMS-001 sampai 003, 006 sampai 025, MAT-002, MAT-005 sampai 009; progres baca sebagian (LMS-010, 016, 017, 018).
- Upload PDF (storage), draft/publish, daftar admin dengan filter, Modul siswa (filter mapel, bab, daftar PDF), halaman terakhir dibaca, unduh.
- Rujukan: BRD 3.4, 3.10, BD-019, BD-031.
- Demo: admin mengunggah PDF dan publish, siswa melihat dan membaca di Modul.

## Slice 4: Bank Soal dan Daily Quiz (25–28 Nov)
- FR: QUIZ-001 sampai 006, 009 sampai 032 (subset: tipe pilihan ganda dan isian angka); TOK-008 (kuota gratis 2 kali per hari).
- Admin: Tambah soal, daftar, filter, Import Excel, rumus LaTeX. Siswa: buat Daily Quiz, kerjakan (timer, ragu-ragu), nilai, riwayat, poin.
- Stretch: potong coin setelah kuota habis (TOK-001, 002, 003, 005, 009).
- Rujukan: BRD 3.5, BD-016, BD-033.
- Demo: admin impor soal, siswa mengerjakan Daily Quiz dan melihat nilai.

## 28 Nov: Rilis demo
- Deploy staging di VPS, isi seed data demo, rekam video demo tiap fitur, cek ulang kontrak dengan FE.
- Lengkapi `docs/STATUS.md` dan susun laporan.

## Setelah 29 Nov (perkiraan, diperbarui setelah demo)
Uji Kom dan Tryout, RUMATH, Paket dan Coin, Leaderboard (reset dan hadiah), Laporan, Landing Page management, Task, Notifikasi.
Tanggal ditetapkan setelah umpan balik klien dan jawaban BD terkait.

## Hal yang perlu ditanyakan ke klien lebih awal
- BD-025 dan BD-026 (cabang dan arti "hapus"): memengaruhi desain DB dan auth.
- BD-016 dan BD-033 (rumus poin dan skala nilai Tryout): memengaruhi Daily Quiz.
- BD-019 dan BD-031 (jenis materi dan Subbab/Topik): memengaruhi struktur materi.

## Risiko
- Integrasi dengan FE biasanya menambah 20–30% waktu fitur: kunci kontrak OpenAPI sejak awal tiap slice.
- Revisi dari klien setelah demo pertama hampir pasti ada; sisakan ruang di jadwal.
