# Status Pengembangan Backend Rumbasha

Terakhir diperbarui: 18 November 2026
Lingkungan demo: _belum ada_ (isi URL staging setelah deploy)
Dokumentasi API: _belum ada_ (isi URL setelah slice 0)

Arti status: **Belum** (belum dimulai), **Berjalan** (sedang dikerjakan), **Bisa didemo** (jalan di staging, FE tersambung), **Selesai** (memenuhi Definition of Done BRD Bagian 16).

## Fitur (target laporan 29 November)

| Slice | Fitur | FR di BRD | Status | Catatan |
|---|---|---|---|---|
| 0 | Fondasi project, health check, dokumentasi API | BRD 5, 6, 7, 9 | Belum | |
| 1 | Login, logout, refresh, role, ganti kata sandi | AUTH-001–008, 010, 013, 014 | Belum | Reset kata sandi menyusul |
| 2 | Master Data dan Cabang | MST-001–007 | Belum | Menunggu BD-025 |
| 2 | User admin (siswa, tutor, admin) | STU-001–006, 012–015, 017; TUT-007 | Belum | Import Excel user: stretch |
| 3 | Materi: upload PDF, draft/publish, Modul siswa, progres baca | LMS-001–003, 006–025; MAT-002, 005–009 | Belum | Menunggu BD-019, BD-031 |
| 4 | Bank Soal dan Import Excel | QUIZ-001–006, 022–032 | Belum | Rumus LaTeX |
| 4 | Daily Quiz siswa (timer, penilaian, riwayat, poin) | QUIZ-009–021; TOK-008 | Belum | Rumus poin menunggu BD-016 |

## Fase berikutnya (belum dijadwalkan)

| Fitur | FR di BRD | Status |
|---|---|---|
| Quiz admin, Uji Kom, Tryout | TRY-001–028 | Belum |
| RUMATH | MATH-001–019 | Belum |
| Paket (langganan) | SUB-001–016 | Belum |
| Coin dan Akses | TOK-001–017 | Belum |
| Leaderboard | RNK-001–013 | Belum |
| Notifikasi dan Pengumuman | NTF-001–006 | Belum |
| Progres belajar dan Dashboard | PRG-001–011, BRD 3.11 | Belum |
| Profil, Pencarian, Bantuan | PRO, SRC, HLP | Belum |
| Image Library | IMG-001–004 | Belum |
| Laporan Quiz dan RUMATH | RPT-001–007 | Belum |
| Landing Page management | LDP-001–007 | Belum |
| Task Management | TSK-001–005 | Belum |

## Keputusan yang menunggu klien

| BD | Pertanyaan singkat | Dampak | Status |
|---|---|---|---|
| BD-025 | Admin dan Tutor dibatasi per cabang? | Auth dan semua filter data | Menunggu |
| BD-026 | "Hapus" = nonaktif atau permanen? | Seluruh modul admin | Menunggu |
| BD-016 | Rumus poin Daily Quiz dan lainnya | Daily Quiz, Leaderboard | Menunggu |
| BD-033 | Skala nilai Tryout (0–100 atau 0–1.000) | Tryout, Leaderboard | Menunggu |
| BD-019 | Jenis materi selain PDF | Materi | Menunggu |
| BD-031 | Subbab dan Topik masih dipakai? | Struktur materi dan soal | Menunggu |

Daftar lengkap ada di `docs/BRD.md` Bagian 15.

## Template laporan ke klien

Gunakan format berikut setiap pelaporan (ringkas, satu halaman):

```
Laporan Progres Backend Rumbasha: <tanggal>

Selesai dan bisa didemo
- <fitur>: <apa yang bisa dilakukan pengguna>. Demo: <link/video>

Sedang dikerjakan
- <fitur>: <perkiraan selesai>

Rencana berikutnya
- <fitur>

Perlu keputusan dari klien
- <BD-nnn>: <pertanyaan>

Catatan
- <kendala atau perubahan jadwal>
```
