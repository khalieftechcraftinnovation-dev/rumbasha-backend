# BUSINESS REQUIREMENTS DOCUMENT (BRD)

**RUMBASHA E-LEARNING PLATFORM**

**Project:** Rumbasha E-Learning Platform  
**Client:** PT Rumbasha Menggapai Asa  
**Document Version:** 1.3  
**Date:** 24 November 2026  
**Backend:** Go (Gin Framework)  
**Frontend:** React  
**Database:** PostgreSQL  
**Architecture:** Modular Monolith (Microservices Ready)  
**Deployment:** Linux VPS (Hostinger)

**Riwayat Revisi**

| **Versi** | **Tanggal**     | **Perubahan**                                                                                                                                                                                                                                                         |
|-----------|-----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1.0       | 1 November 2026 | Draf awal BRD.                                                                                                                                                                                                                                                        |
| 1.1       | 9 November 2026 | Penyesuaian dengan desain UI: login, beranda siswa, modul, Daily Quiz, isi quiz, Uji Kom & Tryout, RUMATH, peringkat, paket & coin, notifikasi, profil, dan beranda admin. Fitur admin lainnya menyusul.                                                              |
| 1.2       | 9 November 2026 | Penyesuaian dengan desain panel admin: Materi, Bank Soal, Image, Master Data, Quiz, RUMATH, Leaderboard, Report Quiz, Report RUMATH, User, Coin & Akses, Paket, Landing Page, dan Task. Ditambahkan Bagian 3.16-3.20, BR-024 sampai BR-028, dan BD-025 sampai BD-033. |
| 1.3       | 24 November 2026 | Keputusan pemilik produk: (1) hierarki mapel: mapel sama untuk semua jenjang, Bidang Materi ke bawah terhubung ke jenjang dan kelas, Subbab dan Topik tetap dipakai; (2) bank soal: 4 tipe soal (single choice, multiple choice, true/false, free text), letak soal wajib sampai Bab, isi soal dapat memuat rumus, gambar, dan tabel; banyak soal satu tipe dapat dibuat sekaligus; ubah soal satu per satu; hapus soal (soft delete) bisa banyak sekaligus; upload Excel hanya mengecek data; gambar boleh link luar; isian teks memakai input_type "text"; Image Library: upload banyak gambar dengan kategori & hierarki mapel, hapus permanen banyak gambar (FR-IMG-005 sampai FR-IMG-008). Daftar Bank Soal: cari kode/teks soal, filter tipe, level, dan setiap tingkat hierarki, kolom pembuat & pengubah terakhir. Master Data: Tipe soal, Tipe quiz, dan Level (disimpan sebagai kode) serta data dropdown (FR-MST-009, FR-MST-010). Buat quiz Uji Kom/Tryout: info quiz, total soal harus sama dengan soal dipilih, syarat soal, penanda soal yang sudah dipakai quiz (FR-TRY-024, FR-TRY-025, FR-TRY-029); waktu mulai quiz (WIB), Uji Kom hanya Kelas 6/9/12, bundle soal (FR-TRY-030), kode akses acak 6 karakter yang diganti setiap jam (FR-TRY-026, FR-TRY-031); jendela event mulai-selesai (bisa lebih dari 1 hari), larangan jadwal beririsan untuk mapel & kelas sama (FR-TRY-032), list quiz dengan status jadwal, ubah, detail, dan hapus quiz (FR-TRY-022, FR-TRY-027); tombol Simpan draf / Publish dan pengumuman quiz di dashboard student (FR-TRY-024, FR-TRY-033); Daily Quiz di-generate student dari soal acak, level = tingkat kesulitan soal, kelas sesuai jenjang (SD: 1-3 & 4-6), total 5/10/15, tanpa kode akses & timer (FR-QUIZ-012 sampai FR-QUIZ-014); buka quiz: Daily Quiz dengan kuota gratis 2x/hari lalu potong coin (FR-QUIZ-021), Uji Kom/Tryout dengan kode akses (FR-TRY-034), kode akses untuk tutor/admin (FR-TRY-035), soal dibuka beserta kunci jawaban tanpa pembahasan; submit jawaban dinilai FE dengan nilai total 100 dan review berisi pembahasan (FR-QUIZ-041, FR-QUIZ-042); waktu pengerjaan per soal & total, quiz yang sudah dikerjakan langsung ke review (FR-QUIZ-043). Diperbarui: 3.4.1, FR-LMS-006, FR-LMS-022, FR-MST-008, FR-QUIZ-022, FR-QUIZ-023, FR-QUIZ-025, FR-QUIZ-026, FR-QUIZ-030, FR-QUIZ-031, FR-QUIZ-033 sampai FR-QUIZ-040, BR-029, 8.12, dan BD-031. |

Catatan v1.2: perubahan v1.2 ditandai sebagai tracked changes (perubahan v1.1 sudah diterima). Hal yang diambil dari desain tetapi belum pasti dicatat di Bagian 15 (Business Decisions Pending).

Catatan v1.3: BRD ini adalah dasar awal dan detailnya belum lengkap. Jika ada perbedaan, keputusan langsung dari pemilik produk lebih berlaku daripada isi BRD atau desain, dan BRD diperbarui mengikuti keputusan tersebut.

## 1. PROJECT OVERVIEW

### 1.1 Background

Rumbasha E-Learning merupakan platform pembelajaran digital yang dikembangkan untuk mendukung kegiatan belajar siswa secara online.

Platform ini menyediakan fasilitas pembelajaran, pengelolaan materi, latihan soal, tryout, serta monitoring perkembangan belajar siswa.

Berdasarkan desain, platform ditujukan untuk siswa jenjang SD dan SMP Rumbasha, termasuk siswa kelas 9 yang menyiapkan diri masuk SMA. Akun siswa dibuat oleh admin Rumbasha saat siswa mendaftar paket.

Sistem akan digunakan oleh beberapa jenis pengguna, yaitu:

- Student (Siswa).

- Tutor (Pengajar).

- Admin.

- Super Admin.

Platform dirancang untuk mendukung sekitar 800 pengguna pada tahap awal pengembangan dan dapat dikembangkan untuk menangani pertumbuhan jumlah pengguna di masa mendatang.

### 1.2 Project Objectives

Tujuan utama pengembangan:

1.  Menyediakan platform pembelajaran online yang mudah digunakan.

2.  Memudahkan pengelolaan materi pembelajaran berdasarkan struktur kurikulum.

3.  Menyediakan latihan soal dan evaluasi hasil belajar.

4.  Menyediakan fitur tryout untuk simulasi ujian.

5.  Menyediakan sistem subscription untuk mengatur akses siswa.

6.  Menyediakan dashboard monitoring bagi admin.

7.  Menyediakan sistem yang scalable dan mudah dikembangkan.

8.  Mempersiapkan arsitektur backend agar dapat dikembangkan menjadi microservices.

### 1.3 Project Scope

Pengembangan dibagi menjadi dua bagian utama:

**A. Public Website**

- Landing page.

- Informasi layanan pembelajaran.

- Informasi program dan paket berlangganan.

- Informasi profil Rumbasha.

- Informasi siswa berprestasi Rumbasha.

- Trial quiz berdasarkan jenjang (tombol "Coba quiz gratis" di halaman login).

- Contact dan informasi pendaftaran.

<!-- -->

- Isi landing page dikelola dari panel admin (lihat 3.19).

**B. E-Learning Application**

- Authentication dan authorization.

- User management.

- Learning management.

- Learning materials.

- Exercise dan quiz.

- Tryout management.

- RUMATH (sebelumnya Math Fun / Interactive Math).

- Subscription and coin management.

- Coin dan kuota gratis harian.

- Learning progress.

- Dashboard dan reporting.

<!-- -->

- Uji Kom (Uji Kompetensi) dan kode akses ujian.

- Peringkat (leaderboard).

- Notifikasi dan pengumuman in-app.

- Pencarian global (materi, PDF, quiz) dan akses bantuan tutor lewat WhatsApp.

- Panel admin (konten, quiz dan nilai, pengguna, sistem).

- Bank soal dengan Import Excel, notasi rumus, dan pratinjau tampilan siswa.

- Master data (mapel, bidang materi, kelas, sekolah, cabang) dan pengelolaan cabang.

- Image library untuk soal, materi, dan landing page.

- Laporan Quiz dan RUMATH dengan ekspor Excel.

- Pengelolaan konten Landing Page (draft dan publish).

- Penugasan (task) dari Super Admin ke Admin.

## 2. USER ROLES & ACCESS MANAGEMENT

### 2.1 Super Admin

Super Admin memiliki akses penuh terhadap keseluruhan sistem.

Hak akses:

- Mengelola seluruh admin.

- Mengelola tutor.

- Mengelola student.

- Mengelola mata pelajaran.

- Mengelola materi pembelajaran.

- Mengelola soal dan tryout.

- Mengelola subscription.

- Mengelola konfigurasi sistem.

- Melihat seluruh laporan dan aktivitas.

<!-- -->

- Membuat dan mengubah task untuk Admin (menu Task, lihat 3.20).

- Melihat data semua cabang dan mengelola akun Admin dan Super Admin.

### 2.2 Admin

Admin bertanggung jawab terhadap operasional platform.

Hak akses:

- Mengelola data student.

- Mengelola data tutor.

- Mengelola mata pelajaran.

- Mengelola materi pembelajaran.

- Mengelola soal.

- Mengelola tryout.

- Memantau aktivitas pembelajaran.

- Melihat laporan sesuai kewenangannya.

<!-- -->

- Membuat kode akses Uji Kom dan Tryout.

- Mengelola paket, coin, dan akses student.

- Melihat dan menyelesaikan task dari Super Admin.

- Mengelola bank soal (termasuk Import Excel), Image, dan Master Data.

- Mengelola Quiz, konten RUMATH, pengaturan Leaderboard, serta melihat Report Quiz dan Report RUMATH.

- Mengelola akun user (termasuk Import Excel), paket, serta coin dan akses student.

- Mengelola Landing Page (hak publish dibanding Super Admin dikonfirmasi, lihat BD-026).

- Bekerja dalam lingkup cabang yang tercatat pada akunnya (lihat BD-025).

### 2.3 Tutor

Tutor bertanggung jawab terhadap kegiatan pembelajaran.

Hak akses:

- Melihat daftar student yang ditugaskan.

- Melihat materi pembelajaran.

- Memantau progres belajar student.

- Melihat hasil latihan dan tryout.

- Mengelola aktivitas pembelajaran sesuai permission yang diberikan.

Catatan: Kewenangan tutor untuk membuat dan mengubah materi atau soal perlu dikonfirmasi sebelum implementasi.

Catatan desain: antarmuka siswa menyebut tutor sebagai sumber materi PDF, jadwal Uji Kom/Tryout, dan kode akses, serta sebagai tujuan pertanyaan siswa lewat WhatsApp. Siapa yang mengunggah dan menjadwalkan di dalam sistem (tutor atau admin) dikonfirmasi saat fitur admin dibahas (lihat BD-002).

### 2.4 Student

Student merupakan pengguna utama yang menggunakan fasilitas pembelajaran.

Hak akses:

- Melihat dashboard pembelajaran.

- Mengakses materi sesuai subscription.

- Mengerjakan latihan soal.

- Mengikuti tryout.

- Menggunakan fitur RUMATH.

- Melihat hasil latihan.

- Melihat progres pembelajaran.

- Melihat informasi subscription.

<!-- -->

- Mengerjakan Daily Quiz, Uji Kom, dan Tryout.

- Melihat peringkat dan notifikasi.

- Mengunduh materi PDF dan mengekspor riwayat belajar.

- Memilih atau memperpanjang paket dan mengisi coin.

- Mengelola profil, kontak orang tua/wali, dan kata sandi.

## 3. FUNCTIONAL REQUIREMENTS

### 3.1 Authentication & Authorization

Sistem harus menyediakan mekanisme autentikasi dan otorisasi.

Functional requirements:

- FR-AUTH-001: User dapat melakukan login menggunakan email atau nomor HP beserta kata sandi yang terdaftar.

- FR-AUTH-002: User dapat melakukan logout.

- FR-AUTH-003: Sistem membedakan akses berdasarkan role.

- FR-AUTH-004: Sistem menyediakan proteksi endpoint berdasarkan permission.

- FR-AUTH-005: Sistem menyediakan mekanisme pengelolaan password.

- FR-AUTH-006: Sistem menyediakan mekanisme reset password.

- FR-AUTH-007: Session atau token autentikasi harus memiliki masa berlaku.

- FR-AUTH-008: User tidak dapat mengakses fitur di luar kewenangannya.

- FR-AUTH-009: Akun student dibuat oleh admin saat student mendaftar paket; student tidak mendaftar sendiri.

- FR-AUTH-010: Halaman login menyediakan opsi "Ingat saya" agar sesi login bertahan lebih lama.

- FR-AUTH-011: Halaman login menyediakan tautan "Lupa kata sandi?" dan "Chat admin" untuk bantuan.

- FR-AUTH-012: Halaman login menyediakan tombol "Coba quiz gratis" untuk trial quiz (lihat 1.3 Public Website).

- FR-AUTH-013: Student dapat mengubah kata sandi dari halaman Profil.

- FR-AUTH-014: Setelah login, user diarahkan ke aplikasi student atau panel admin sesuai role.

Catatan: desain halaman login memuat kotak "Akun Demo (Preview)" berisi email dan kata sandi contoh. Kotak ini hanya untuk preview dan tidak boleh ada di production.

Rekomendasi implementasi:

- JWT Access Token.

- Refresh Token.

- Password hashing menggunakan bcrypt atau Argon2.

- Middleware authentication.

- Middleware role/permission authorization.

### 3.2 Student Management

Sistem menyediakan pengelolaan data siswa.

Functional requirements:

- FR-STU-001: Admin dapat membuat student baru.

- FR-STU-002: Admin dapat mengubah informasi student.

- FR-STU-003: Admin dapat menonaktifkan student.

- FR-STU-004: Admin dapat melihat daftar student.

- FR-STU-005: Admin dapat mencari student berdasarkan nama atau informasi lainnya.

- FR-STU-006: Sistem dapat menghubungkan student dengan subscription.

- FR-STU-007: Sistem dapat melihat riwayat aktivitas pembelajaran student.

<!-- -->

- FR-STU-008: Admin membuat akun student bersamaan dengan pendaftaran paket.

- FR-STU-009: Student dapat melihat dan mengubah data diri dari halaman Profil (field yang boleh diubah sendiri dikonfirmasi, lihat BD-014).

- FR-STU-010: Student dapat menyimpan kontak orang tua/wali; nomor ditampilkan tersamar (contoh 0812-xxxx-2210).

- FR-STU-011: Student memiliki target sekolah tujuan (contoh "Target SMAN 5 Surabaya") yang tampil di Profil dan Beranda.

- FR-STU-012: Menu User pada panel admin menampilkan akun siswa, tutor, admin, dan super admin dengan tab per role beserta jumlahnya (contoh desain: Semua 55, Siswa 47, Tutor 4, Admin 3, Super Admin 1).

- FR-STU-013: Daftar user dapat dicari berdasarkan nama atau email dan difilter berdasarkan kelas, cabang, dan status.

- FR-STU-014: Daftar user menampilkan nama, email, role, kelas atau keterangan (cabang untuk admin dan tutor; "Semua cabang" untuk super admin), dan status (Aktif).

- FR-STU-015: Admin dapat menambah user (tombol "Tambah user"), mengubah (Edit), dan menghapus user. Akun yang sedang login tidak dapat menghapus dirinya sendiri (tombol hapus tidak tersedia pada baris sendiri di desain). Arti "hapus" (nonaktif atau hapus permanen) dikonfirmasi (lihat BD-026).

- FR-STU-016: Admin dapat menambah banyak akun sekaligus lewat Import Excel (contoh task "Import 120 akun siswa baru"). Sistem memvalidasi setiap baris dan menampilkan baris yang gagal beserta alasannya.

- FR-STU-017: Setiap akun siswa, tutor, dan admin terhubung dengan satu cabang (lihat 3.16). Pembatasan data per cabang dikonfirmasi (lihat BD-025).

- FR-STU-018: Form Tambah user dan Edit user belum tersedia pada desain. Field akun dan tutor mengikuti informasi minimal di bawah dan 3.3.

Informasi student minimal:

- Student ID.

- Full Name.

- Email.

- Phone Number.

- Date of Birth.

- School.

- Grade / Class.

<!-- -->

- Education Level (SD / SMP).

- Target School.

- Branch (Cabang).

- Parent / Guardian Contact (hubungan dan nomor HP).

- Active Package.

<!-- -->

- Account Status.

- Created At.

- Updated At.

### 3.3 Tutor Management

Sistem menyediakan pengelolaan data tutor.

Functional requirements:

- FR-TUT-001: Admin dapat membuat akun tutor.

- FR-TUT-002: Admin dapat mengubah informasi tutor.

- FR-TUT-003: Admin dapat menonaktifkan tutor.

- FR-TUT-004: Admin dapat melihat daftar tutor.

- FR-TUT-005: Admin dapat menghubungkan tutor dengan student.

- FR-TUT-006: Tutor dapat melihat student yang menjadi tanggung jawabnya.

<!-- -->

- FR-TUT-007: Akun tutor dikelola dari menu User (tab Tutor) dan memiliki cabang.

### 3.4 Learning Management System (LMS)

Modul ini merupakan pusat pengelolaan materi pembelajaran.

#### 3.4.1 Learning Hierarchy

Struktur materi pembelajaran harus mengikuti hierarki berikut:

**Mata Pelajaran → Bidang Materi → Bab → Subbab → Topik**

Contoh:

- Mata Pelajaran: Matematika

  - Bidang Materi: Aritmatika

    - Bab: Bilangan

      - Subbab: Bilangan Bulat

        - Topik: Operasi Penjumlahan

        - Topik: Operasi Pengurangan

        - Topik: Operasi Perkalian

        - Topik: Operasi Pembagian

Contoh pada desain: Mata Pelajaran Matematika, Bidang Materi Bilangan, Bab Pecahan. Bab Pecahan berisi PDF Penjumlahan Pecahan, Pengurangan Pecahan, Perkalian Pecahan, Pembagian Pecahan, dan Pecahan Campuran. Mata pelajaran pada desain: Matematika, IPA, B. Indonesia, B. Inggris, IPS. Bidang materi Matematika pada desain: Bilangan, Aljabar, Geometri, Statistika. Subbab dan Topik bersifat opsional pada pemilihan Daily Quiz. Nama bidang materi final mengikuti Master Data.

Aturan hierarki (keputusan v1.3):

- Mata Pelajaran sama untuk semua jenjang. Contoh: satu mapel Matematika dipakai SD, SMP, dan SMA. Mapel tidak terikat jenjang maupun kelas.
- Mulai Bidang Materi ke bawah (Bidang Materi, Bab, Subbab, Topik), setiap item terhubung ke satu jenjang dan satu kelas. Contoh: Matematika → Bilangan (SD, Kelas 4) → KPK dan FPB → Menentukan KPK dan FPB → Menentukan KPK.
- Bidang Materi dengan nama yang sama boleh ada di kelas berbeda (contoh: Bilangan Kelas 1 dan Bilangan Kelas 4 adalah dua item terpisah dengan isi bab yang berbeda), tetapi tidak boleh kembar dalam mapel dan kelas yang sama.
- Jenjang dan kelas pada Bab, Subbab, dan Topik selalu sama dengan Bidang Materi induknya. Kelas harus milik jenjang yang dipilih.
- Subbab dan Topik tetap dipakai (BD-031 dijawab).

Catatan desain admin: form Tambah materi dan Tambah soal hanya memakai Jenjang, Kelas, Mapel, Bidang materi, dan Bab. Pilihan Bidang materi pada form disaring berdasarkan Mapel dan Kelas yang dipilih. Untuk soal, letak wajib diisi sampai Bab; Subbab dan Topik opsional (FR-QUIZ-026). Untuk materi masih perlu dikonfirmasi.

#### 3.4.2 Functional Requirements

- FR-LMS-001: Admin dapat membuat mata pelajaran.

- FR-LMS-002: Admin dapat membuat bidang materi.

- FR-LMS-003: Admin dapat membuat bab.

- FR-LMS-004: Admin dapat membuat subbab.

- FR-LMS-005: Admin dapat membuat topik.

- FR-LMS-006: Setiap entitas memiliki relasi hierarkis dengan parent-nya. Mapel tidak memiliki parent; Bidang Materi berinduk ke Mapel dan wajib memiliki jenjang dan kelas; Bab, Subbab, dan Topik mewarisi mapel, jenjang, dan kelas dari induknya.

- FR-LMS-022: Daftar Bidang Materi, Bab, Subbab, dan Topik dapat difilter berdasarkan Mapel, Jenjang, dan Kelas.

- FR-LMS-007: Admin dapat mengubah dan menghapus struktur materi.

- FR-LMS-008: Student dapat melihat materi berdasarkan mata pelajaran.

- FR-LMS-009: Student dapat melihat detail materi berdasarkan topik.

- FR-LMS-010: Sistem dapat menyimpan progres pembelajaran student.

<!-- -->

- FR-LMS-011: Materi dan soal ditandai dengan jenjang (SD / SMP) dan kelas.

- FR-LMS-012: Halaman Modul menampilkan mata pelajaran sebagai filter (Semua, Matematika, IPA, B. Indonesia, B. Inggris, IPS) dan bidang materi sebagai filter turunan.

- FR-LMS-013: Student melihat daftar bab (urut sesuai kurikulum) pada mata pelajaran terpilih, lengkap dengan jumlah PDF, jumlah PDF yang sudah dibuka, dan progress bar.

- FR-LMS-014: Student melihat daftar PDF dalam satu bab: judul, jumlah halaman, kelas, dan status (Dibuka, Selesai, Diunduh).

- FR-LMS-015: Student dapat mencari PDF berdasarkan judul dan memakai filter.

- FR-LMS-016: Halaman Modul menampilkan "Terakhir kamu buka" dan "Progres modul" per mata pelajaran (jumlah PDF dari total PDF).

- FR-LMS-017: Student dapat membaca PDF di dalam aplikasi; sistem menyimpan halaman terakhir dibaca dan student dapat melanjutkan lewat tombol "Lanjut baca".

- FR-LMS-018: Student dapat mengunduh PDF lewat tombol "Unduh"; sistem mencatat status Diunduh.

- FR-LMS-019: Menu Materi pada panel admin ("PDF yang tampil di menu Modul siswa") menampilkan daftar PDF dengan kolom Judul PDF, Jenjang dan kelas (contoh "SD 5", "SMP 8"), Mapel, Bidang \> Bab, jumlah halaman, tanggal diubah, Status (Publish atau Draft), dan Aksi (Edit, hapus).

- FR-LMS-020: Daftar materi dapat dicari berdasarkan judul PDF dan difilter berdasarkan Jenjang, Mapel, dan Status.

- FR-LMS-021: Admin menambah materi lewat halaman "Tambah materi" dengan bagian Letak materi (Jenjang, Kelas, Mapel, Bidang materi, Bab) dan Info PDF (Judul PDF, Deskripsi opsional, Jumlah halaman), serta mengunggah file PDF. Bagian bawah form tidak terlihat pada desain.

- FR-LMS-022: Letak materi menentukan posisi PDF pada filter level siswa. Bab dipilih dari daftar atau diketik sebagai bab baru; bab baru dibuat otomatis.

- FR-LMS-023: Judul PDF dipakai juga untuk pencarian siswa; Deskripsi tampil di bawah judul pada halaman Modul.

- FR-LMS-024: Materi dapat disimpan sebagai draf ("Simpan sebagai draf") atau langsung di-publish. Hanya materi berstatus Publish yang tampil di Modul siswa dan memicu notifikasi (FR-MAT-009).

- FR-LMS-025: Admin dapat mengubah dan menghapus materi. Arti "hapus" dan perlakuan progres siswa pada materi yang dihapus dikonfirmasi (lihat BD-026).

#### 3.4.3 Learning Materials

Setiap topik dapat memiliki beberapa materi pembelajaran.

Jenis materi:

- Text.

- PDF.

- Video.

- Image.

- External Learning Link.

Functional requirements:

- FR-MAT-001: Admin dapat menambahkan materi ke dalam topik.

- FR-MAT-002: Admin dapat mengunggah file PDF.

- FR-MAT-003: Admin dapat mengunggah video.

- FR-MAT-004: Admin dapat mengunggah gambar.

- FR-MAT-005: Admin dapat menentukan urutan materi.

- FR-MAT-006: Student dapat mengakses materi sesuai subscription.

- FR-MAT-007: Sistem menyimpan aktivitas akses materi.

- FR-MAT-008: Sistem menyimpan status penyelesaian materi.

<!-- -->

- FR-MAT-009: Sistem membuat notifikasi ketika PDF baru dipublish pada suatu bab (lihat 3.13).

File storage menggunakan object/file storage yang terpisah dari database. Database hanya menyimpan metadata dan lokasi file.

Catatan desain: halaman Modul siswa saat ini menampilkan materi dalam bentuk PDF ("PDF materi dari tutor Rumbasha"). Video muncul di RUMATH (3.7). Jenis materi lain (Text, Image, External Learning Link) tidak tampil pada desain dan perlu dikonfirmasi apakah masuk tahap awal (lihat BD-019).

Catatan desain admin: menu Materi hanya mengelola PDF. Video dikelola di menu RUMATH (3.7) dan gambar di menu Image (3.17). Jenis Text dan External Learning Link tidak ada di panel admin (lihat BD-019).

### 3.5 Exercise & Quiz Management (termasuk Daily Quiz)

Sistem menyediakan latihan soal untuk mengukur pemahaman siswa.

Pada desain, latihan soal yang dikerjakan student disebut Daily Quiz: student memilih materi dan level, lalu sistem menyusun soal secara otomatis dari bank soal.

Functional requirements:

- FR-QUIZ-001: Admin dapat membuat bank soal.

- FR-QUIZ-002: Soal dapat dihubungkan dengan mata pelajaran.

- FR-QUIZ-003: Soal dapat dihubungkan dengan topik.

- FR-QUIZ-004: Sistem mendukung pilihan ganda.

- FR-QUIZ-005: Sistem menyimpan jawaban yang benar.

- FR-QUIZ-006: Admin dapat menentukan tingkat kesulitan soal.

- FR-QUIZ-007: Admin dapat membuat latihan dari kumpulan soal.

- FR-QUIZ-008: Student dapat mengerjakan latihan.

- FR-QUIZ-009: Sistem dapat melakukan penilaian otomatis.

- FR-QUIZ-010: Student dapat melihat hasil latihan.

- FR-QUIZ-011: Sistem menyimpan riwayat pengerjaan latihan.

<!-- -->

- FR-QUIZ-012 (diubah v1.3): Student dapat membuat (generate) Daily Quiz dengan memilih Kelas, Mapel dan Bidang materi (wajib), Bab, Subbab, dan Topik (opsional, berurutan), Level, dan Total soal (5, 10, atau 15). Kelas default adalah kelas student dan hanya bisa diganti dengan kelas di jenjangnya; khusus SD dibagi dua kelompok, Kelas 1-3 dan Kelas 4-6 (contoh student Kelas 5 hanya dapat memilih Kelas 4, 5, atau 6). Daily Quiz tidak memiliki kode akses dan tidak memiliki timer.

- FR-QUIZ-013 (diubah v1.3): Level Daily Quiz adalah Mudah, Sedang, dan Sulit, dipetakan langsung ke tingkat kesulitan soal (easy, medium, hard; FR-QUIZ-034). Level tidak menentukan tipe soal: soal dalam satu Daily Quiz boleh bercampur tipenya (pilihan ganda, PG kompleks, benar/salah, isian).

- FR-QUIZ-014 (diubah v1.3): Sistem menyusun soal Daily Quiz secara acak dari bank soal: hanya soal aktif yang ditandai "Dipakai untuk" Daily Quiz, dengan level dan letak (kelas, bidang materi, serta bab/subbab/topik jika dipilih) sesuai pilihan student. Jika soal yang tersedia kurang dari total soal yang dipilih, sistem menolak dan memberi tahu jumlah soal yang tersedia. Soal yang dikirim ke student tidak berisi kunci jawaban dan pembahasan.

- FR-QUIZ-015: Layar quiz menampilkan satu soal per layar, nomor "Soal X dari N", tipe soal, timer hitung mundur, bar progres, dan tombol Kumpulkan.

- FR-QUIZ-016: Soal pilihan ganda pada desain memiliki opsi A sampai E. Konten soal dan opsi harus mendukung notasi matematika (pecahan, pangkat, akar, dan sejenisnya).

- FR-QUIZ-017: Student dapat berpindah soal lewat tombol Sebelumnya dan Selanjutnya, atau lewat panel Daftar soal.

- FR-QUIZ-018: Student dapat menandai soal "Ragu-ragu". Panel Daftar soal menampilkan status Dijawab, Sekarang, Ragu-ragu, dan Belum beserta jumlah masing-masing.

- FR-QUIZ-019: Halaman Daily Quiz menampilkan Riwayat (materi, level, waktu, nilai, dan poin) serta statistik: rata-rata nilai dan jumlah quiz, poin Daily Quiz, dan peringkat.

- FR-QUIZ-020: Student memperoleh poin dari Daily Quiz yang dikerjakan. Poin dipakai pada Peringkat (3.12). Contoh desain: nilai 20 menjadi +10 poin, nilai 40 menjadi +20 poin. Leaderboard admin menyebut rumus Daily Quiz "Nilai ÷ 2 per quiz" dan dapat diatur admin (lihat 3.12 dan BD-016).

- FR-QUIZ-021 (diubah v1.3): Daily Quiz memiliki kuota gratis 2 kali per hari (WIB), dihitung dari Daily Quiz yang dibuka student hari itu (membuka ulang Daily Quiz yang sama tidak dihitung lagi). Setelah kuota habis, setiap Daily Quiz yang dibuka memotong 10 coin (lihat 3.9) dan tercatat di riwayat coin; jika coin tidak cukup, Daily Quiz tidak dapat dibuka. Besar kuota dan biaya coin sementara diatur di konfigurasi server dan nanti dipindah ke master parameter quiz. Saat dibuka, student menerima seluruh soal beserta kunci jawabannya (tanpa pembahasan).

- FR-QUIZ-022 (diubah v1.3): Menu Bank Soal pada panel admin (Admin dan Super Admin) menampilkan daftar soal, urut dari yang terbaru dibuat, dengan kolom Kode (contoh Q-1256), Pertanyaan, Tipe beserta penanda isi tambahan soal (gambar, tabel, rumus; contoh "+1 media"), Hirarki mapel (jenjang, kelas, mapel, bidang materi, bab, subbab, topik), Level, Dipakai di, Dibuat oleh, Dibuat pada, Terakhir diubah oleh, Terakhir diubah pada, dan Aksi (Edit, hapus). Daftar memakai halaman (maksimal 100 soal per halaman) dan tidak menampilkan kunci jawaban maupun soal yang sudah dihapus. Tab per tipe soal beserta jumlahnya: Semua, Pilihan ganda (single choice), PG kompleks (multiple choice), Benar / Salah (true/false), dan Isian (free text), contoh desain: 264, 128, 2, 2, 132. Jumlah di tab mengikuti pencarian dan filter yang sedang dipakai (kecuali filter tipe).

- FR-QUIZ-023 (diubah v1.3): Daftar soal dapat dicari berdasarkan kode soal dan teks soal (sebagian kata, tidak membedakan huruf besar/kecil), serta difilter berdasarkan Tipe soal, Level, dan setiap tingkat hierarki mapel (Jenjang, Kelas, Mapel, Bidang materi, Bab, Subbab, Topik). Pencarian dan filter dapat digabung.

- FR-QUIZ-024: Admin menambah soal satu per satu lewat halaman "Tambah soal" (tombol Simpan soal, Simpan & tambah lagi, dan Batal), dan banyak soal sekaligus lewat Import Excel.

- FR-QUIZ-025 (diubah v1.3): Tipe soal ada 4, masing-masing dengan form jawaban sendiri:
  - Single choice: opsi A sampai E (minimal 2), satu kunci jawaban.
  - Multiple choice: opsi A sampai E (minimal 2), satu atau lebih kunci jawaban, dengan cara penilaian "all or nothing" (default) atau "partial".
  - True/false: 1 sampai 10 pernyataan (A, B, C, ...), setiap pernyataan diberi kunci benar atau salah.
  - Free text: isian angka atau teks. Isian angka menerima beberapa bentuk jawaban benar (contoh "0.75", "0,75", "3/4") dengan toleransi selisih. Isian teks memakai kata kunci: jawaban siswa benar jika mengandung salah satu kata kunci (huruf besar/kecil diabaikan kecuali diatur lain).
  
  Setiap opsi berisi teks, gambar, atau keduanya. Hal ini menggantikan versi sebelumnya (Pilihan ganda atau Isian angka).

- FR-QUIZ-026 (diubah v1.3): Letak soal diisi dengan Jenjang, Kelas, Mapel, Bidang materi, dan Bab (wajib), serta Subbab dan Topik (opsional; Topik hanya bisa dipilih jika Subbab dipilih). Semua dipilih dari Master Data dan harus saling cocok (contoh: Bab harus milik Bidang materi dan Kelas yang dipilih). Letak soal dipakai untuk menyusun Daily Quiz otomatis.

- FR-QUIZ-027: Setiap soal ditandai "Dipakai untuk": Daily Quiz, Uji Kom, Tryout, dan/atau RUMATH (boleh lebih dari satu). Kolom "Dipakai di" pada daftar menampilkan peruntukan tersebut.

- FR-QUIZ-033 (baru v1.3): Isi soal terdiri dari teks pertanyaan (boleh berisi rumus LaTeX) dan pembahasan, keduanya wajib untuk semua tipe soal. Setiap tipe soal juga dapat memiliki rumus LaTeX terpisah, satu gambar (dari Image library atau link gambar dari luar http/https, salah satu saja), dan satu tabel (judul kolom opsional, maksimal 10 kolom dan 50 baris); ketiganya opsional dan boleh dipakai sendiri-sendiri atau sekaligus. Urutan tampil bagian soal (pertanyaan, rumus, gambar, tabel) dapat diatur admin.

- FR-QUIZ-034 (baru v1.3): Tingkat kesulitan soal: easy (mudah), medium (sedang), hard (sulit).

- FR-QUIZ-035 (baru v1.3): Status soal: active (dapat dipakai quiz) atau inactive. Default active.

- FR-QUIZ-036 (baru v1.3): Setiap soal mendapat kode otomatis berurutan (contoh Q-1256) dan mencatat admin pembuatnya. Hanya Admin dan Super Admin yang dapat membuat soal.

- FR-QUIZ-037 (baru v1.3): Admin dapat membuat banyak soal sekaligus dalam satu kali simpan (maksimal 100 soal), dengan syarat semua soal bertipe sama. Penyimpanan bersifat semua-atau-tidak-sama-sekali: jika ada satu soal yang salah, tidak ada soal yang tersimpan, dan setiap kesalahan ditandai dengan nomor soalnya.

- FR-QUIZ-038 (baru v1.3): Admin dan Super Admin dapat melihat detail satu soal, termasuk kunci jawaban, pembahasan, nama letak soal (jenjang, kelas, mapel, bidang materi, bab, subbab, topik), dan nama admin pembuatnya. Detail soal berisi kunci jawaban sehingga tidak boleh diakses siswa maupun tutor.

- FR-QUIZ-039 (baru v1.3): Admin dan Super Admin dapat mengubah satu soal per sekali simpan (seluruh isi soal, termasuk tipe, letak, kunci jawaban, dan pembahasan; aturan sama dengan saat membuat soal). Sistem mencatat admin yang terakhir mengubah soal.

- FR-QUIZ-040 (baru v1.3): Admin dan Super Admin dapat menghapus satu atau banyak soal sekaligus (maksimal 100). Penghapusan bersifat soft delete: soal tidak tampil lagi, tetapi datanya tetap tersimpan beserta waktu dan admin yang menghapus. Jika ada satu soal yang tidak ditemukan atau sudah dihapus, tidak ada soal yang terhapus. Perlakuan soal yang sudah dipakai pada quiz masih mengikuti BD-026.

- FR-QUIZ-041 (baru v1.3): Student mengumpulkan (submit) jawaban Daily Quiz, Uji Kom, atau Tryout yang sudah dibuka. Benar/salah setiap soal dinilai di aplikasi (FE) memakai kunci jawaban yang diterima saat quiz dibuka; sistem menyimpan jawaban dan hasilnya. Nilai total semua quiz adalah 100: nilai per soal = 100 dibagi total soal, nilai akhir = jumlah jawaban benar x nilai per soal; soal yang tidak dijawab dihitung salah. Jawaban hanya dapat dikumpulkan satu kali. Uji Kom dan Tryout tidak dapat dikumpulkan setelah batas waktu mengerjakan (toleransi 2 menit); Daily Quiz tidak memiliki batas waktu.

- FR-QUIZ-042 (baru v1.3): Setelah submit, student mendapatkan review: nilai, jumlah benar, dan setiap soal dengan urutan yang sama seperti saat dikerjakan, beserta jawaban student, jawaban yang benar, benar/salah, nilai per soal, dan pembahasan. Review dapat dibuka kembali setelahnya.

- FR-QUIZ-043 (baru v1.3): Sistem mencatat waktu pengerjaan: per soal (jam soal terakhir dijawab dan lama mengerjakan soal tersebut, dicatat aplikasi) dan total lama mengerjakan (waktu submit dikurangi waktu mulai mengerjakan, dihitung server). Waktu pengerjaan ditampilkan di review. Jika student membuka quiz (Daily Quiz, Uji Kom, atau Tryout) yang sudah pernah ia kerjakan dan kumpulkan, student langsung diarahkan ke halaman review quiz tersebut tanpa kode akses dan tanpa memotong kuota/coin, termasuk setelah event Uji Kom/Tryout selesai.

- FR-QUIZ-028: Pertanyaan dan opsi jawaban mendukung rumus yang ditulis di antara tanda \$ ... \$ (LaTeX). Form menyediakan tombol rumus (pecahan, pangkat, akar, pi, kali, bagi) dan panel Pratinjau yang menampilkan soal seperti pada HP siswa.

- FR-QUIZ-029: Admin dapat mengubah dan menghapus soal. Perlakuan soal yang sudah dipakai pada quiz atau sudah dikerjakan siswa dikonfirmasi (lihat BD-026).

- FR-QUIZ-030 (diubah v1.3): Gambar untuk soal dan opsi jawaban diambil dari Image library (3.17) atau berupa link gambar dari luar (http/https); satu gambar hanya boleh memakai salah satu sumber. Opsi jawaban boleh hanya berisi gambar tanpa teks. Cara menyisipkan gambar pada form soal tidak terlihat pada desain.

- FR-QUIZ-031 (diubah v1.3): Upload soal lewat Excel memakai 4 template (single choice, multiple choice, true/false, free text; satu file satu tipe soal, maksimal 100 soal). Upload hanya MENGECEK isi file tanpa menyimpan: letak soal ditulis dengan nama (jenjang, kelas, mapel, bidang materi, bab, subbab, topik) lalu dicocokkan ke Master Data, dan setiap baris dicek dengan aturan yang sama dengan Tambah soal ditambah aturan template (urutan tampil wajib; opsi wajib: single choice A-D, multiple choice A-C, true/false A-B). Jika ada kesalahan, semua kesalahan ditampilkan sekaligus lengkap dengan nomor baris dan kolom Excel, dan seluruh isi file ditolak. Jika valid, sistem mengembalikan data soal (letak soal sudah berisi ID dan nama) untuk ditinjau admin lalu disimpan lewat Tambah soal.

- FR-QUIZ-032: Soal isian bertipe angka dan dipakai antara lain oleh RUMATH (lihat 3.7 dan BD-029).

Rekomendasi: simpan notasi matematika dalam format LaTeX (admin menulis rumus di antara tanda \$ ... \$) dan render di frontend, termasuk Pratinjau admin, dengan KaTeX atau MathJax.

Informasi soal:

- Question ID.

- Subject.

- Topic.

- Question Type.

- Question Content.

- Answer Options.

- Correct Answer.

- Explanation.

- Difficulty Level.

<!-- -->

- Class (Kelas).

- Chapter (Bab).

- Used For (Daily Quiz, Uji Kom, Tryout, RUMATH).

- Education Level (SD / SMP).

<!-- -->

- Status.

### 3.6 Uji Kom & Tryout Management

Tryout merupakan fitur simulasi ujian yang digunakan untuk mengukur kesiapan siswa.

Pada desain, menu ini bernama "Uji Kom & Tryout" dan memuat dua jenis ujian. Uji Kom (Uji Kompetensi) adalah ujian per mata pelajaran dan bab, contoh "Uji Kompetensi Matematika — Pecahan". Tryout adalah ujian simulasi lintas mata pelajaran, contoh "Tryout Bersama Rumbasha" (semua mapel, 40 soal, 120 menit). Jadwal berasal dari tutor dan student masuk menggunakan kode akses.

#### 3.6.1 Tryout Categories

Sistem mendukung kategori tryout:

- Tryout Kelas 6.

- Tryout Kelas 9.

- Tryout Kelas 12 (perlu konfirmasi: desain menyebut platform untuk siswa SD dan SMP, lihat BD-020).

<!-- -->

- Tryout Masuk SMA (contoh desain: Tryout Masuk SMA \#2).

- Tryout Bersama Rumbasha (semua mata pelajaran, contoh desain).

#### 3.6.2 Functional Requirements

- FR-TRY-001: Admin dapat membuat tryout.

- FR-TRY-002: Admin dapat menentukan kategori kelas.

- FR-TRY-003: Admin dapat menentukan mata pelajaran.

- FR-TRY-004: Admin dapat menentukan kumpulan soal.

- FR-TRY-005: Admin dapat menentukan durasi pengerjaan.

- FR-TRY-006: Admin dapat menentukan jadwal tryout.

- FR-TRY-007: Student dapat mengikuti tryout sesuai ketentuan.

- FR-TRY-008: Sistem menyimpan jawaban student.

- FR-TRY-009: Sistem dapat melakukan penilaian otomatis.

- FR-TRY-010: Sistem menyimpan hasil tryout.

- FR-TRY-011: Student dapat melihat hasil tryout.

- FR-TRY-012: Sistem menyimpan riwayat tryout.

<!-- -->

- FR-TRY-013: Halaman Uji Kom & Tryout menyediakan tab Semua, Uji Kom, dan Tryout, kolom pencarian, dan filter mata pelajaran.

- FR-TRY-014: Student masuk ke Uji Kom atau Tryout menggunakan kode akses.

- FR-TRY-015: Admin dapat membuat kode akses untuk sesi Uji Kom atau Tryout (tombol "Buat kode akses" di beranda admin).

- FR-TRY-016: Ujian terjadwal tampil di bagian "Akan datang" (jenis, nama, mata pelajaran, jumlah soal, durasi, waktu dibuka, hitung mundur) dan terkunci sampai waktu dibuka.

- FR-TRY-017: Ujian yang sudah dikerjakan tampil di bagian "Sudah dikerjakan" (jenis, tanggal, nama, nilai, dan peringkat).

- FR-TRY-018: Student dapat melihat pembahasan untuk soal yang salah setelah hasil dirilis.

- FR-TRY-019: Sistem mengirim notifikasi ketika Uji Kom dibuka dan ketika hasil sudah keluar. Cara rilis hasil (otomatis atau manual) dikonfirmasi (lihat BD-021).

- FR-TRY-020: Sistem menghitung peringkat per sesi (contoh "peringkat \#1 dari 1") dan poin Uji Kom + Tryout.

- FR-TRY-021: Halaman menampilkan statistik: rata-rata nilai dan jumlah sesi, poin Uji Kom + Tryout, dan peringkat pada tryout terakhir.

- FR-TRY-022 (diubah v1.3): Menu Quiz pada panel admin ("Rakit paket soal dan atur jadwal") menampilkan semua quiz (yang belum dihapus), terbaru dibuat dulu, dengan tab Semua, Daily Quiz, Uji Kom, Tryout, dan RUMATH. Kolom: Nama quiz, Tipe, Mapel, Jenjang, Kelas, Jadwal (mulai dan selesai event, WIB), Status jadwal, Isi (jumlah soal dan durasi), Kode akses, Status, dan Aksi (Edit, hapus). Status jadwal dihitung otomatis dari waktu sekarang: Akan dimulai (belum mulai), Sedang berjalan (sudah mulai, belum selesai), dan Sudah lewat (event selesai), beserta jumlah quiz per status jadwal. Ada pencarian nama quiz serta filter Tipe, Jenjang, Kelas, Mapel, Status, dan Status jadwal.

- FR-TRY-023: Admin membuat quiz lewat tombol "Buat quiz" dalam tiga langkah: (1) Info quiz, (2) Pilih soal, (3) Jadwal & kode. Tersedia tombol Batal, Simpan draf, dan Lanjut.

- FR-TRY-024 (diubah v1.3): Langkah 1 (Info quiz) berisi Nama quiz, Tipe (hanya Uji Kom atau Tryout; Daily Quiz dan RUMATH dibuat otomatis), Jenjang, Kelas, hierarki mapel (Mapel wajib; Bidang materi, Bab, Subbab, dan Topik opsional, diisi berurutan), Hari, tanggal, dan jam mulai event (wajib, dalam WIB, harus setelah waktu sekarang), Hari, tanggal, dan jam selesai event (wajib, WIB; event boleh berlangsung lebih dari 1 hari), Durasi pengerjaan (menit, 1-600; contoh 60; lama mengerjakan satu kali di dalam jendela event, sehingga jendela event minimal sepanjang durasi), Biaya coin (opsional, 0 = gratis untuk siswa berpaket), dan Total soal. Quiz Uji Kom hanya dapat dibuat untuk Kelas 6, Kelas 9, dan Kelas 12; Tryout untuk semua kelas. Form buat dan ubah quiz memiliki dua tombol simpan: Simpan sebagai draf (status Draft) dan Simpan & publish (status Publish).

- FR-TRY-025 (diubah v1.3): Langkah 2 memilih soal dari Bank Soal dan/atau dari bundle (FR-TRY-030), maksimal 200 soal per quiz. Urutan pilihan menjadi nomor soal, dan satu soal tidak boleh dipilih dua kali dalam quiz yang sama. Jumlah soal yang dipilih harus sama dengan Total soal. Soal yang boleh dipilih hanya soal aktif yang ditandai untuk tipe quiz tersebut ("Dipakai untuk", FR-QUIZ-027), dengan Mapel dan Kelas yang sama dengan quiz; jika quiz mengisi Bidang materi, Bab, Subbab, atau Topik, soal juga harus berada di tingkat yang sama. Semua kesalahan ditampilkan sekaligus beserta kode soalnya, dan quiz tidak tersimpan. Langkah 3 mengatur jadwal dan kode akses; layar langkah 2 dan 3 belum tersedia pada desain (lihat BD-032).

- FR-TRY-029 (baru v1.3): Daftar Bank Soal menandai setiap soal yang sudah pernah dipilih ke quiz, beserta nama dan tipe quiz yang memakainya (quiz yang sudah dihapus tidak dihitung), supaya admin tahu soal mana yang sudah dipakai saat merakit quiz. Daftar Bank Soal juga dapat difilter berdasarkan peruntukan ("Dipakai untuk") dan status soal.

- FR-TRY-030 (baru v1.3): Saat membuat quiz, admin dapat memakai ulang soal dari quiz yang sudah dibuat sebelumnya (bundle): (1) mengambil semua soal satu atau beberapa bundle sekaligus (maksimal 20 bundle), atau (2) membuka isi bundle lalu memilih sebagian soalnya. Soal dari bundle tetap harus memenuhi syarat FR-TRY-025 (soal yang sudah dinonaktifkan atau dihapus sejak bundle dibuat ditolak, dengan kode soalnya). Soal yang ada di lebih dari satu bundle hanya diambil sekali. Daftar bundle dapat dicari berdasarkan nama dan difilter berdasarkan Tipe, Jenjang, Kelas, dan Mapel; quiz yang sudah dihapus tidak tampil sebagai bundle.

- FR-TRY-026 (diubah v1.3): Uji Kom dan Tryout membutuhkan kode akses. Sistem membuat kode akses acak saat quiz dibuat: 6 karakter berupa huruf besar (A-Z) dan/atau angka (0-9), unik di antara quiz yang belum dihapus. Pada daftar, kolom Kode akses bernilai "Otomatis" untuk quiz RUMATH.

- FR-TRY-031 (baru v1.3): Sistem mengganti kode akses secara otomatis setiap jam (tepat di menit 00 WIB) untuk semua quiz Uji Kom dan Tryout yang event-nya belum selesai (belum mulai maupun sedang berjalan). Quiz yang event-nya sudah selesai atau dihapus tidak diubah kodenya. Jadwal pergantian dapat diatur lewat konfigurasi server.

- FR-TRY-034 (baru v1.3): Student membuka quiz Uji Kom atau Tryout yang sudah publish, untuk kelasnya, dan sedang berjalan dengan memasukkan kode akses yang berlaku saat itu. Kode yang salah atau sudah berganti ditolak. Setelah berhasil dibuka sekali, student dapat membuka ulang quiz tersebut tanpa kode akses (contoh setelah memuat ulang halaman). Sistem mencatat waktu mulai mengerjakan; batas mengerjakan adalah waktu mulai + durasi, tetapi tidak melewati akhir event. Saat dibuka, student menerima seluruh info quiz dan soal beserta kunci jawabannya (tanpa pembahasan); kode akses tidak dikirim. Quiz yang belum dimulai atau sudah selesai tidak dapat dibuka.

- FR-TRY-035 (baru v1.3): Tutor, Admin, dan Super Admin dapat melihat kode akses quiz yang berlaku saat ini (beserta waktu terakhir kode diganti) untuk dibagikan kepada student.

- FR-TRY-033 (baru v1.3): Quiz Uji Kom dan Tryout yang sudah di-publish dan event-nya belum selesai tampil sebagai pengumuman di dashboard student yang kelasnya sama dengan kelas quiz, berisi nama quiz, tipe, mapel, hari, tanggal, dan jam mulai serta selesai (WIB), status jadwal (akan dimulai / sedang berjalan), durasi, jumlah soal, dan biaya coin; urut dari yang paling dekat dimulai. Kode akses tidak ditampilkan di pengumuman. Quiz berstatus Draft tidak tampil.

- FR-TRY-032 (baru v1.3): Tidak boleh ada dua quiz (yang belum dihapus, tipe dan status apa pun) dengan Mapel dan Kelas yang sama yang jendela event-nya beririsan. Quiz yang mulai tepat saat quiz lain selesai diperbolehkan. Jika bentrok, sistem menolak dan menampilkan nama serta jadwal quiz yang bentrok. Aturan berlaku saat membuat maupun mengubah quiz, dan dijaga juga oleh database.

- FR-TRY-027 (diubah v1.3): Quiz memiliki status Draft atau Publish; admin dapat mengubah dan menghapus quiz. Ubah quiz bersifat ganti penuh (isi form sama dengan buat quiz, aturan sama) dan hanya boleh untuk quiz yang event-nya belum dimulai; kode akses dan status tidak berubah, dan sistem mencatat admin yang terakhir mengubah. Admin dapat melihat detail quiz (info quiz, jadwal, status jadwal, kode akses, dan daftar soal berurutan tanpa kunci jawaban). Hapus quiz dapat untuk satu atau banyak quiz sekaligus (maksimal 100) secara soft delete (data tetap tersimpan beserta waktu dan admin yang menghapus); quiz yang event-nya sedang berjalan tidak dapat dihapus, dan jika ada satu quiz yang tidak dapat dihapus maka tidak ada yang terhapus. Quiz yang dihapus tidak tampil di daftar quiz dan daftar bundle, serta jadwal dan kode aksesnya tidak lagi menghalangi quiz lain. Quiz Uji Kom dan Tryout tidak dapat di-publish tanpa soal yang valid (BR-008) dan kode akses.

- FR-TRY-028: Biaya coin dapat ditetapkan per quiz oleh admin. Hal ini melengkapi FR-TOK-009 dan konfigurasi biaya coin pada 3.9.

#### 3.6.3 Tryout Result

Informasi hasil tryout minimal:

- Student.

- Tryout.

- Total Questions.

- Correct Answers.

- Incorrect Answers.

- Unanswered Questions.

- Score.

- Duration.

- Started At.

- Submitted At.

<!-- -->

- Rank (peringkat pada sesi).

- Points (poin).

Catatan: Desain sudah menampilkan ranking dan pembahasan soal yang salah. Passing grade serta rumus ranking dan poin masih memerlukan keputusan bisnis.

### 3.7 RUMATH (sebelumnya Math Fun / Interactive Math)

RUMATH merupakan fitur hitung cepat untuk siswa: siswa menonton video penjelasan bersama Rumbi, lalu mengerjakan latihan soal isian.

Fitur ini dirancang untuk memberikan pengalaman belajar melalui latihan soal yang lebih sederhana dan interaktif.

#### 3.7.1 Initial Scope

- Interactive Math.

- Generate soal otomatis.

- Penilaian jawaban otomatis.

- Tingkat kesulitan.

- Penyimpanan hasil latihan.

<!-- -->

- Video pembelajaran RUMATH dan progres tonton.

- Jenis soal: penjumlahan, pengurangan, perkalian, pembagian, dan soal cerita.

#### 3.7.2 Initial Configuration

Konfigurasi awal:

| **Configuration**     | **Value**                                                   |
|-----------------------|-------------------------------------------------------------|
| Question Type         | Soal isian (fill-in) berdasarkan jenis operasi              |
| Questions per Session | 10 soal isian                                               |
| Difficulty            | Level 1 (1-10)                                              |
| Difficulty            | Level 2 (10-100)                                            |
| Difficulty            | Level 3 (100-1.000)                                         |
| Duration              | 15 menit per sesi latihan                                   |
| Operation Type        | Penjumlahan, Pengurangan, Perkalian, Pembagian, Soal cerita |

Tingkat kesulitan:

- Level 1: bilangan 1-10 (sebelumnya Unit / satuan).

- Level 2: bilangan 10-100 (sebelumnya Tens / puluhan).

- Level 3: bilangan 100-1.000 (sebelumnya Hundreds / ratusan).

#### 3.7.3 Functional Requirements

- FR-MATH-001: Student dapat membuka fitur RUMATH.

- FR-MATH-002: Sistem dapat menghasilkan soal secara otomatis.

- FR-MATH-003: Sistem mendukung tiga level: Level 1 (1-10), Level 2 (10-100), dan Level 3 (100-1.000).

- FR-MATH-004: Sistem menghasilkan 10 soal isian per sesi latihan dengan batas waktu 15 menit.

- FR-MATH-005: Student dapat mengisi jawaban.

- FR-MATH-006: Sistem memeriksa jawaban secara otomatis.

- FR-MATH-007: Sistem menampilkan hasil pengerjaan.

- FR-MATH-008: Sistem menyimpan riwayat pengerjaan.

- FR-MATH-009: Admin dapat mengelola konfigurasi RUMATH.

<!-- -->

- FR-MATH-010: Student dapat memilih jenis soal (penjumlahan, pengurangan, perkalian, pembagian, soal cerita) dan level sebelum memulai latihan.

- FR-MATH-011: Student dapat menonton video RUMATH. Sistem menyimpan progres tonton dan menampilkannya di "Lanjutkan nonton" (judul, durasi, urutan seperti "Video 2 dari 4").

- FR-MATH-012: Halaman RUMATH menampilkan statistik (rata-rata nilai dan jumlah latihan, jumlah soal dijawab, sisa coin) dan daftar "Latihan terakhir" (jenis, level, waktu, nilai atau poin).

- FR-MATH-013: Latihan RUMATH tertulis "gratis di paketmu" pada desain. Penggunaan coin untuk RUMATH dikonfirmasi (lihat BD-006).

- FR-MATH-014: Paket latihan RUMATH tampil di menu Quiz admin sebagai quiz bertipe RUMATH per jenis soal dan level (contoh "RUMATH · Pembagian Lv 2", 10 soal · 15 mnt, kode akses "Otomatis").

- FR-MATH-015: Menu RUMATH pada panel admin memiliki tab Video materi (contoh desain: 17) dan Soal isian (contoh: 130), serta tombol Tambah video dan Tambah soal.

- FR-MATH-016: Daftar video menampilkan Judul, Jenis (Pembagian, Pengurangan, Penjumlahan, dan sejenisnya), Jenjang, Urutan (Video 1, Video 2, dan seterusnya), Durasi, jumlah Ditonton (contoh 821x), Status, dan Aksi (Edit, hapus). Ada pencarian judul dan filter Jenis dan Jenjang.

- FR-MATH-017: Admin dapat menambah, mengubah, dan menghapus video RUMATH, mengatur urutan dalam satu jenis, dan menetapkan Draft atau Publish. Form video belum tersedia pada desain; sumber video dikonfirmasi (lihat BD-030).

- FR-MATH-018: Sistem menghitung jumlah tonton setiap video dan total durasi tonton untuk laporan (3.18).

- FR-MATH-019: Tab Soal isian menampilkan soal bertipe Isian yang dipakai untuk RUMATH. Jumlahnya sama dengan tab Isian di Bank Soal (130), sehingga tampak memakai data yang sama. Apakah soal RUMATH dibuat admin atau dihasilkan otomatis (FR-MATH-002) dikonfirmasi (lihat BD-029).

### 3.8 Subscription Management

Subscription digunakan untuk mengatur hak akses siswa terhadap fasilitas pembelajaran.

Pada desain, subscription tampil sebagai "Paket", contoh Reguler SMP dan Intensif Masuk SMA.

Functional requirements:

- FR-SUB-001: Admin dapat membuat paket subscription.

- FR-SUB-002: Admin dapat mengubah paket subscription.

- FR-SUB-003: Admin dapat menghubungkan student dengan subscription.

- FR-SUB-004: Sistem menyimpan tanggal mulai subscription.

- FR-SUB-005: Sistem menyimpan tanggal berakhir subscription.

- FR-SUB-006: Sistem dapat menentukan status subscription.

- FR-SUB-007: Sistem memvalidasi subscription sebelum memberikan akses fitur.

- FR-SUB-008: Admin dapat melihat riwayat subscription student.

<!-- -->

- FR-SUB-009: Student dapat melihat paket aktif (nama paket, kelas, tanggal berakhir, sisa hari) di Beranda, Paket & Coin, dan Profil.

- FR-SUB-010: Halaman Paket & Coin menampilkan daftar paket (nama, deskripsi, harga, harga coret, fitur, label "Paling banyak dipilih") dengan pilihan periode Bulanan, Semester, dan Tahunan (Tahunan -20%).

- FR-SUB-011: Student dapat memilih paket dan memperpanjang paket aktif. Dari notifikasi pada desain, perpanjangan menambah masa aktif (3 Desember 2026 menjadi 2 Januari 2027).

- FR-SUB-012: Promo paket (contoh "Diskon 30% Paket Intensif Masuk SMA") tampil sebagai banner di Beranda dan mengarah ke halaman paket. Mekanisme promo dikonfirmasi (lihat BD-024).

- FR-SUB-013: Sistem membuat notifikasi ketika paket aktif atau diperpanjang.

- FR-SUB-014: Menu Paket pada panel admin menampilkan paket langganan dan paket coin dengan tab Semua, Langganan, dan Coin. Kolom: Nama paket (dengan label "Unggulan" untuk paket unggulan), Jenis, Harga (contoh "Rp450.000 / bln"), Isi (ringkasan fitur), jumlah Pengguna, Status (Aktif), dan Aksi (Edit, hapus).

- FR-SUB-015: Admin dapat menambah paket (tombol "Tambah paket"), mengubah, dan menghapus paket. Paket yang masih dipakai siswa (kolom Pengguna lebih dari 0) tidak boleh hilang dari riwayat siswa; aturan hapus dikonfirmasi (lihat BD-026).

- FR-SUB-016: Label "Unggulan" pada admin dianggap sama dengan label "Paling banyak dipilih" pada sisi siswa (FR-SUB-010). Form Tambah paket belum tersedia pada desain sehingga field harga coret, periode, dan fitur belum terkonfirmasi.

**Contoh Paket pada Desain**

| **Paket**                                  | **Sasaran**                                               | **Harga / bulan** | **Harga coret** | **Fitur**                                                                    |
|--------------------------------------------|-----------------------------------------------------------|-------------------|-----------------|------------------------------------------------------------------------------|
| Reguler SD                                 | Belajar rutin siswa SD (kelas tidak terlihat pada desain) | Rp350.000         | \-              | 2 pertemuan per minggu; Modul dan ... (teks terpotong pada desain admin).    |
| Reguler SMP                                | Belajar rutin kelas 7-9                                   | Rp450.000         | Rp600.000       | 2 pertemuan per minggu; Modul, Daily Quiz, Uji Kom; 100 coin tiap bulan.     |
| Intensif Masuk SMA (Paling banyak dipilih) | Persiapan TKA dan seleksi SMA                             | Rp750.000         | Rp950.000       | 4 pertemuan per minggu; Tryout min... (teks terpotong pada desain admin).    |
| Privat 1-on-1                              | Belajar privat dengan jadwal yang diatur siswa            | Rp1.500.000       | \-              | Jadwal kamu yang atur; Materi sesuai ... (teks terpotong pada desain admin). |

Nama paket, harga, periode, dan fitur harus berasal dari data paket yang dikelola admin (FR-SUB-001 dan FR-SUB-002), bukan nilai tetap di kode. Arti "pertemuan per minggu" pada paket dikonfirmasi (lihat BD-017).

Catatan: baris Reguler SD dan Privat 1-on-1 berasal dari daftar Paket pada panel admin. Pada desain admin, jumlah pengguna per paket adalah Reguler SD 10, Reguler SMP 24, Intensif Masuk SMA 8, dan Privat 1-on-1 0. Sasaran dan harga coret kedua paket tersebut tidak terlihat.

**Subscription Status**

- PENDING.

- ACTIVE.

- EXPIRED.

- CANCELLED.

**Important Business Rule**

Apabila subscription_end sudah melewati tanggal saat ini:

1.  Subscription berubah menjadi EXPIRED.

2.  Student tidak dapat mengakses fitur pembelajaran yang membutuhkan subscription aktif.

3.  Student masih dapat melakukan login.

4.  Student tetap dapat melihat informasi akun dan subscription.

5.  Sistem tidak menghapus data pembelajaran sebelumnya.

6.  Admin tetap dapat mengakses data student.

Validasi subscription harus dilakukan di backend, bukan hanya frontend.

### 3.9 Coin & Kuota Gratis Harian (sebelumnya Token Management)

Coin adalah saldo yang dipakai student untuk fitur tertentu di luar jatah gratis. Istilah "token" pada v1.0 kini terbagi dua: kuota gratis harian yang direset setiap hari, dan saldo coin yang bertambah dari paket atau pembelian dan berkurang saat dipakai.

Catatan penamaan: ID requirement FR-TOK dipertahankan agar traceability tetap terjaga, meskipun istilah di desain adalah "coin".

**Business Requirements**

- Setiap student memiliki saldo coin.

- Coin dapat digunakan untuk aktivitas yang ditentukan sistem (pada desain: Daily Quiz tambahan, tryout tambahan, dan RUMATH).

- Sistem mencatat penggunaan coin.

- Sistem mereset kuota gratis harian setiap hari. Saldo coin tidak direset pada desain (dikonfirmasi, lihat BD-005).

- Sistem memiliki konfigurasi jumlah kuota gratis harian dan biaya coin per aktivitas.

- Student tidak dapat menggunakan coin melebihi saldo yang tersedia.

<!-- -->

- Coin bertambah dari paket (contoh Reguler SMP: 100 coin tiap bulan) dan dari pembelian "Isi coin".

- Selain coin, student memiliki kuota gratis harian yang direset setiap hari (contoh Daily Quiz: 2 kali per hari).

**Functional Requirements**

- FR-TOK-001: Sistem menyediakan saldo coin student.

- FR-TOK-002: Sistem dapat mengurangi coin ketika student menggunakan fitur berbayar coin.

- FR-TOK-003: Sistem mencatat setiap transaksi coin.

- FR-TOK-004: Sistem melakukan reset kuota gratis harian.

- FR-TOK-005: Sistem mencegah saldo coin menjadi negatif.

- FR-TOK-006: Student dapat melihat sisa coin dan sisa kuota gratis hari ini.

- FR-TOK-007: Admin dapat melihat riwayat coin student.

<!-- -->

- FR-TOK-008: Sistem menerapkan kuota gratis Daily Quiz 2 kali per hari dan menampilkan pemakaiannya ("Jatah gratis hari ini 0 dari 2").

- FR-TOK-009: Setelah kuota gratis habis, sistem memotong 10 coin per Daily Quiz. Jika coin tidak cukup, quiz tidak dapat dimulai.

- FR-TOK-010: Student dapat membeli coin pada tab "Isi coin" dengan memilih salah satu paket coin; tombol beli menampilkan jumlah coin dan harga.

- FR-TOK-011: Sisa coin tampil di Beranda, Daily Quiz, RUMATH, Peringkat, Paket & Coin, dan Profil.

- FR-TOK-012: Admin dapat mengelola coin dan akses student lewat menu "Coin & Akses" (lihat FR-TOK-013 sampai FR-TOK-017).

- FR-TOK-013: Menu Coin & Akses menampilkan daftar siswa dengan kolom Siswa, Kelas, Paket, Akses, Coin, Berlaku s/d, dan Aksi (Edit). Ada pencarian siswa dan filter Paket, Kelas, dan Berakhir.

- FR-TOK-014: Kolom Akses merangkum fitur yang terbuka bagi siswa menurut paketnya (contoh desain: Reguler SD "Modul, Daily"; Reguler SMP "Modul, Daily, Uji Kom"; Intensif Masuk SMA "Semua + RUMATH"). Kolom Berlaku s/d menampilkan tanggal berakhir paket (FR-SUB-005).

- FR-TOK-015: Admin dapat mengubah coin dan akses siswa satu per satu (tombol Edit) atau sekaligus (memilih beberapa baris dengan kotak centang). Form edit belum tersedia pada desain.

- FR-TOK-016: Setiap perubahan coin dan akses oleh admin harus tercatat dalam transaction history dengan referensi, sesuai BR-010 dan BR-012; saldo tidak boleh menjadi negatif.

- FR-TOK-017: Admin dapat mengekspor daftar Coin & Akses (tombol Export; format dikonfirmasi).

**Paket Isi Coin pada Desain**

| **Paket coin** | **Keterangan** | **Harga** |
|----------------|----------------|-----------|
| 100 coin       | Untuk 2 tryout | Rp25.000  |
| 300 coin       | Hemat 10%      | Rp68.000  |
| 600 coin       | Hemat 15%      | Rp128.000 |
| 1.200 coin     | Hemat 20%      | Rp240.000 |

Daftar paket coin, harga, dan biaya coin per aktivitas harus dapat dikonfigurasi, bukan nilai tetap di kode.

**Important Technical Rules**

- Coin harus dikelola menggunakan database transaction.

- Setiap pengurangan coin harus memiliki transaction reference.

- Hindari double deduction akibat concurrent requests.

- Proses reset harus idempotent.

- Simpan histori transaksi coin.

- Reset harus menggunakan zona waktu Asia/Jakarta (UTC+7).

Detail yang masih perlu ditentukan:

- Jumlah kuota gratis harian untuk fitur selain Daily Quiz, dan biaya coin per aktivitas selain Daily Quiz (desain baru menunjukkan Daily Quiz: 2 kali gratis, lalu 10 coin).

- Fitur apa saja yang membutuhkan coin (desain menyebut Daily Quiz tambahan, tryout tambahan, dan RUMATH).

- Apakah coin atau kuota harian yang tidak digunakan dapat diakumulasi atau hangus.

- Apakah jumlah coin berbeda berdasarkan paket (desain: Reguler SMP memberi 100 coin tiap bulan).

### 3.10 Learning Progress

Sistem menyediakan monitoring perkembangan belajar siswa.

Functional requirements:

- FR-PRG-001: Sistem menyimpan progres setiap student.

- FR-PRG-002: Sistem menyimpan materi yang telah diselesaikan.

- FR-PRG-003: Sistem menyimpan riwayat latihan.

- FR-PRG-004: Sistem menyimpan riwayat tryout.

- FR-PRG-005: Student dapat melihat perkembangan belajar.

- FR-PRG-006: Tutor dapat melihat perkembangan student yang ditugaskan.

- FR-PRG-007: Admin dapat melihat progres seluruh student.

<!-- -->

- FR-PRG-008: Sistem mencatat streak belajar harian (jumlah hari berturut-turut) dan target belajar harian 15 menit ("Belajar 15 menit hari ini"), ditampilkan di sidebar student.

- FR-PRG-009: Sistem merekomendasikan materi berdasarkan nilai Daily Quiz terakhir ("Rekomendasi buatmu"). Aturan rekomendasi dikonfirmasi.

- FR-PRG-010: Student dapat mengekspor Riwayat belajar ke PDF untuk orang tua (menu Profil).

- FR-PRG-011: Aktivitas yang dihitung untuk streak dan target 15 menit dikonfirmasi (lihat BD-023).

### 3.11 Dashboard & Reporting

**Student Dashboard**

Menampilkan:

- Informasi student.

- Subscription status.

- Sisa coin dan kuota gratis hari ini.

- Daftar mata pelajaran.

- Learning progress.

- Materi terakhir diakses.

- Riwayat latihan.

- Riwayat tryout.

<!-- -->

- Banner pengumuman (contoh: jadwal Tryout Bersama).

- Lanjutkan belajar (PDF terakhir dibuka, progres halaman, tombol Lanjut baca dan Unduh).

- Rekomendasi materi berdasarkan nilai Daily Quiz terakhir.

- Menu cepat: Modul, Daily Quiz, Uji Kom & Tryout, RUMATH, Paket, Top-up Coin.

- Banner promo paket.

- Peringkat ringkas (tab Daily Quiz, Uji Kom, Tryout) dan posisi student.

- Paket aktif, sisa hari, dan tombol Perpanjang.

- Streak belajar dan target belajar 15 menit.

- Tombol "Tanya tutor" lewat WhatsApp (Senin-Sabtu 08.00-20.00).

- Pencarian global (materi, PDF, quiz) dan lonceng notifikasi.

**Tutor Dashboard**

Menampilkan:

- Total student yang ditugaskan.

- Daftar student.

- Aktivitas belajar student.

- Learning progress.

- Hasil latihan dan tryout.

**Admin Dashboard**

Menampilkan:

- Total student.

- Total tutor.

- Total subscription aktif.

- Total subscription expired.

- Aktivitas pembelajaran.

- Statistik latihan.

- Statistik tryout.

<!-- -->

- Kartu ringkasan Total user (rincian Siswa, Tutor, Admin; pertumbuhan akun bulan ini).

- Kartu ringkasan Bank soal (total, soal baru minggu ini, rincian SD dan SMP).

- Kartu ringkasan Materi PDF (total, PDF dipublish minggu ini, rincian SD dan SMP).

- Kartu ringkasan Quiz dikerjakan (persentase, jumlah quiz dikerjakan, dan yang belum dikerjakan dalam 30 hari).

- Grafik rata-rata nilai Daily Quiz per jenjang (SD dan SMP), skala 0-100, dengan pilihan rentang 12 minggu, 30 hari, dan 7 hari.

- Ringkasan task aktif dari Super Admin dan jumlah sesi Uji Kom yang berlangsung hari ini.

- Pintasan "Buat kode akses" dan "Tambah soal", serta pencarian global (user, soal, materi, quiz).

### 3.12 Peringkat (Leaderboard)

Peringkat menampilkan urutan student berdasarkan poin.

Functional requirements:

- FR-RNK-001: Student dapat melihat peringkat global (contoh "#12 dari 47 siswa").

- FR-RNK-002: Peringkat tersedia dalam tiga tab: Daily Quiz, Uji Kom, dan Tryout.

- FR-RNK-003: Daftar dapat difilter berdasarkan rentang waktu (Semua waktu), kelas, dan sekolah.

- FR-RNK-004: Halaman menampilkan podium tiga besar dan daftar peringkat berikutnya dengan nama, sekolah, poin, dan indikator naik/turun.

- FR-RNK-005: Posisi student sendiri selalu terlihat (baris "Kamu") beserta perubahan peringkat (contoh "naik 3").

- FR-RNK-006: Beranda menampilkan ringkasan peringkat (tiga teratas dan posisi student).

- FR-RNK-007: Admin dapat melihat dan mengatur Leaderboard dari menu Leaderboard (lihat FR-RNK-008 sampai FR-RNK-013).

- FR-RNK-008: Menu Leaderboard pada panel admin ("Atur poin, periode, dan hadiah tiap papan peringkat") menampilkan empat kartu konfigurasi: Daily Quiz, Uji Kom, Tryout, dan RUMATH. Setiap kartu berisi Periode, Reset, Poin, dan Hadiah, serta tombol Edit.

- FR-RNK-009: Admin dapat mengubah periode, waktu reset, rumus poin, dan hadiah setiap papan. Nilai pada tabel di bawah hanyalah contoh desain dan harus dapat dikonfigurasi, bukan nilai tetap di kode. Form Edit belum tersedia pada desain.

- FR-RNK-010: Sistem mereset papan peringkat sesuai periode dan waktu reset (zona Asia/Jakarta) dan memproses hadiah sesuai konfigurasi. Cara pemberian hadiah (coin otomatis, sertifikat, konsultasi) dikonfirmasi (lihat BD-028).

- FR-RNK-011: Tabel peringkat admin memiliki tab Daily Quiz, Uji Kom, dan Tryout, pencarian siswa, filter Periode, Kelas, dan Sekolah, serta kolom Peringkat, Siswa, Asal sekolah, Poin, dan Aksi. Judul tabel (contoh "Daily Quiz · semua waktu", "46 siswa · 10 tampil di aplikasi") menunjukkan admin melihat semua siswa, sedangkan aplikasi siswa hanya menampilkan 10 teratas.

- FR-RNK-012: Setiap baris peringkat memiliki tombol Edit (diasumsikan untuk koreksi poin manual; lihat BD-028).

- FR-RNK-013: Admin dapat mengekspor peringkat (tombol Export; format dikonfirmasi).

**Konfigurasi Leaderboard pada Desain Admin**

| **Papan**  | **Periode** | **Reset**           | **Poin**             | **Hadiah**              |
|------------|-------------|---------------------|----------------------|-------------------------|
| Daily Quiz | Mingguan    | Senin 00.00         | Nilai ÷ 2 per quiz   | 50 coin untuk 3 teratas |
| Uji Kom    | Per sesi    | Saat sesi ditutup   | Nilai × 1,4          | Sertifikat digital      |
| Tryout     | Per tryout  | Saat tryout ditutup | Skor 0-1.000         | Konsultasi gratis       |
| RUMATH     | Bulanan     | Tgl 1, 00.00        | 10 poin / soal benar | 100 coin untuk juara 1  |

Papan RUMATH memiliki kartu konfigurasi pada panel admin tetapi tidak memiliki tab pada tabel peringkat admin maupun pada aplikasi siswa (lihat BD-028).

Catatan: peringkat menampilkan nama dan sekolah siswa lain. Karena pengguna adalah siswa SD dan SMP, kebijakan tampilan data (nama lengkap atau inisial, opsi sembunyikan) dikonfirmasi (lihat BD-015).

### 3.13 Notifikasi & Pengumuman

Notifikasi memberi tahu student tentang kejadian penting di akunnya. Pengumuman dan promo tampil sebagai banner di Beranda dan sebagai notifikasi.

Functional requirements:

- FR-NTF-001: Student dapat melihat daftar notifikasi yang dikelompokkan "Minggu ini" dan "Lebih lama", berisi judul, isi singkat, dan waktu.

- FR-NTF-002: Notifikasi yang belum dibaca ditandai titik, dan jumlahnya tampil sebagai badge pada menu Notifikasi dan ikon lonceng.

- FR-NTF-003: Student dapat menandai notifikasi sebagai dibaca ("Tandai dibaca").

- FR-NTF-004: Jenis notifikasi pada desain: hasil Uji Kom atau Tryout keluar, Uji Kom dibuka, paket aktif atau diperpanjang, perubahan peringkat, materi baru, dan promo.

- FR-NTF-005: Sistem membuat notifikasi otomatis untuk kejadian terkait (hasil ujian, paket, peringkat, materi baru).

- FR-NTF-006: Pengumuman dan promo dikelola dari panel admin. Pada desain admin belum ada menu Pengumuman; banner promo berupa gambar kategori Banner (3.17) dan section Promo di Landing Page (3.19). Tempat membuat pengumuman dikonfirmasi (lihat BD-027).

Notifikasi lewat WhatsApp atau email belum termasuk dan masih menunggu keputusan (lihat BD-012).

### 3.14 Profil, Pencarian, dan Bantuan Student

Functional requirements:

- FR-PRO-001: Halaman Profil menampilkan nama, kelas, asal sekolah, paket aktif, target sekolah, ringkasan paket (coin, tanggal aktif sampai, bar masa aktif), dan statistik (peringkat, total poin, quiz selesai bulan ini, coin).

- FR-PRO-002: Menu Profil mencakup Data diri, Kontak orang tua / wali, Riwayat belajar (export PDF), Kata sandi, Bantuan & masukan, dan Keluar.

- FR-PRO-003: Student dapat mengirim bantuan atau masukan lewat menu "Bantuan & masukan" (tujuan dan mekanisme dikonfirmasi).

- FR-SRC-001: Kolom pencarian global di header mencari materi, PDF, dan quiz.

- FR-SRC-002: Halaman Modul dan Uji Kom & Tryout memiliki pencarian lokal (judul PDF, nama ujian).

- FR-HLP-001: Tombol "Tanya tutor" membuka WhatsApp dengan jam layanan Senin-Sabtu 08.00-20.00. Nomor tujuan dan jam layanan dapat dikonfigurasi.

### 3.15 Panel Admin (Ringkasan dari Desain)

Panel admin terdiri dari menu berikut. Setiap menu dirinci pada bagian yang dirujuk. Sidebar admin dapat di-scroll, sehingga daftar ini mungkin belum lengkap; menu Pengumuman tidak terlihat (lihat BD-027).

| **Grup**     | **Menu**           | **Keterangan / Referensi**                                       |
|--------------|--------------------|------------------------------------------------------------------|
| Utama        | Beranda            | Dashboard admin (3.11).                                          |
| Konten       | Materi             | Pengelolaan materi PDF (3.4).                                    |
| Konten       | Bank Soal          | Pengelolaan soal (3.5).                                          |
| Konten       | Image              | Image library untuk soal, materi, dan landing page (3.17).       |
| Konten       | Master Data        | Data acuan: mapel, bidang materi, kelas, sekolah, cabang (3.16). |
| Quiz & Nilai | Quiz               | Pengelolaan Daily Quiz, Uji Kom, dan Tryout (3.5, 3.6).          |
| Quiz & Nilai | RUMATH             | Konfigurasi dan konten RUMATH (3.7).                             |
| Quiz & Nilai | Leaderboard        | Peringkat (3.12).                                                |
| Quiz & Nilai | Report Quiz        | Laporan quiz (3.18).                                             |
| Quiz & Nilai | Report RUMATH      | Laporan RUMATH (3.18).                                           |
| Pengguna     | User               | Pengelolaan student, tutor, admin (3.2, 3.3).                    |
| Pengguna     | Coin & Akses       | Pengelolaan coin dan akses student (3.9).                        |
| Pengguna     | Paket              | Pengelolaan paket langganan (3.8).                               |
| Sistem       | Landing Page       | Konten situs publik (1.3, 3.19).                                 |
| Sistem       | Task (label Super) | Penugasan dari Super Admin ke Admin (2.1, 3.20).                 |

### 3.16 Master Data & Cabang

Master Data adalah data acuan untuk filter materi, soal, dan profil siswa. Menu ini juga memuat daftar cabang Rumbasha.

Functional requirements:

- FR-MST-001: Menu Master Data menampilkan data acuan dengan tab Semua, Mapel, Bidang materi, Kelas, Sekolah, dan Cabang beserta jumlahnya (contoh desain: 44, 5, 14, 9, 13, 3).

- FR-MST-002: Daftar menampilkan Nama, Tipe, Induk (contoh bidang materi Bilangan berinduk Matematika), Isi (jumlah PDF yang memakai data, contoh "14 PDF"), Status (Aktif), dan Aksi (Edit, hapus). Ada pencarian nama data dan filter Status.

- FR-MST-003: Admin dapat menambah (tombol "Tambah data"), mengubah, dan menghapus data. Data yang masih dipakai materi, soal, atau siswa tidak boleh dihapus begitu saja (lihat BD-026).

- FR-MST-004: Data acuan menjadi pilihan pada form Materi, Bank Soal, Quiz, dan User, serta pada filter panel admin dan aplikasi student.

- FR-MST-005: Mapel pada desain: Matematika, IPA, B. Indonesia, B. Inggris, dan IPS. Contoh Bidang materi: Bilangan, Aljabar, Geometri, dan Statistika (Matematika); Fisika, Biologi, dan Kimia (IPA); Teks dan Kebahasaan (B. Indonesia).

- FR-MST-006: Data Sekolah dipakai pada profil siswa (asal sekolah) dan filter Peringkat.

- FR-MST-007: Cabang pada desain: Surabaya Timur, Surabaya Barat, dan Sidoarjo. Cabang menjadi atribut akun admin, tutor, dan siswa; Super Admin tercatat "Semua cabang". Pembatasan data per cabang dikonfirmasi (lihat BD-025).

- FR-MST-008: Master Data menyimpan hierarki Mapel → Bidang Materi → Bab → Subbab → Topik sesuai aturan 3.4.1. Saat menambah Bidang Materi, admin memilih Mapel, Jenjang, dan Kelas. Saat menambah Bab, Subbab, atau Topik, admin cukup memilih induknya, dan jenjang serta kelas mengikuti induk tersebut.

- FR-MST-009 (baru v1.3): Master Data juga menyimpan pilihan Tipe soal (Pilihan ganda, PG kompleks, Benar / Salah, Isian), Tipe quiz (Daily Quiz, Uji Kom, Tryout, RUMATH), dan Level (Mudah, Sedang, Sulit). Data lain (soal, quiz) menyimpan KODE pilihan tersebut sebagai teks (contoh level "easy"), bukan relasi ke Master Data, sehingga kode tidak boleh diubah; yang boleh diubah hanya nama tampilan dan urutannya.

- FR-MST-010 (baru v1.3): Sistem menyediakan data dropdown dari Master Data untuk semua pengguna yang login (admin maupun siswa): Jenjang, Kelas (dapat disaring per jenjang), Mapel, Bidang materi (wajib memilih Mapel, dapat disaring per Kelas), Bab (wajib memilih Bidang materi), Subbab (wajib memilih Bab), Topik (wajib memilih Subbab), Tipe soal, Tipe quiz, dan Level. Hanya data aktif yang tampil, urut sesuai urutan yang diatur admin.

### 3.17 Image Library

Image library menyimpan gambar untuk soal, materi, dan landing page.

Functional requirements:

- FR-IMG-001: Menu Image menampilkan galeri gambar (tampilan grid atau daftar) dengan nama file, Kategori (contoh Banner dan Soal), dan ukuran file. Ada pencarian nama file, filter Kategori, dan pagination (contoh 15 baris per halaman).

- FR-IMG-002: Admin dapat mengunggah gambar (tombol "Upload gambar"). Contoh pada desain berformat SVG; format lain, batas ukuran, dan validasi tipe file mengikuti 10.2.

- FR-IMG-003: Gambar dipakai pada soal (contoh rangkaian listrik, garis bilangan, diagram batang, lingkaran, segitiga siku-siku), materi, dan landing page (contoh banner promo "Diskon 30% Paket Intensif Masuk SMA").

- FR-IMG-004: Gambar dapat dipilih lebih dari satu (indikator "0 dari 6 baris dipilih"). Aksi massal yang tersedia tidak terlihat pada desain.

- FR-IMG-005 (baru v1.3): Setiap gambar memiliki kategori: question (gambar soal), banner, atau landing (landing page), serta nama gambar. Gambar kategori question boleh dihubungkan ke hierarki mapel (Mapel, Bidang Materi, Bab, Subbab, Topik) yang diisi berurutan dari atas; kategori lain tidak memakai hierarki.

- FR-IMG-006 (baru v1.3): Admin dapat mengunggah banyak gambar sekaligus (maksimal 10 file, masing-masing maksimal 5 MB; format PNG, JPG, GIF, atau WebP yang dicek dari isi file). Semua gambar dalam satu unggahan memakai kategori dan hierarki yang sama; nama gambar diberi nomor urut di belakangnya (contoh "Segitiga 1", "Segitiga 2"). Jika ada satu file yang tidak valid, tidak ada gambar yang tersimpan. SVG tidak diterima karena dapat berisi script (keputusan v1.3, menggantikan contoh SVG pada desain).

- FR-IMG-007 (baru v1.3): Link gambar disimpan dan dikirim ke aplikasi untuk ditampilkan; gambar dapat dibuka tanpa login.

- FR-IMG-008 (baru v1.3): Admin dapat menghapus satu atau banyak gambar sekaligus (maksimal 100) secara permanen (data dan file). Penghapusan ditolak seluruhnya jika ada gambar yang tidak ditemukan atau masih dipakai soal aktif (sebagai gambar soal atau gambar opsi jawaban); soal yang sudah dihapus otomatis dilepas dari gambarnya.

File gambar disimpan di file storage server (folder STORAGE_DIR/images/<kategori>/<tahun>/<bulan>/); database (tabel media_assets) hanya menyimpan metadata, lokasi file, dan link gambar (lihat 3.4.3).

### 3.18 Laporan Quiz dan RUMATH

Laporan Quiz

- FR-RPT-001: Report Quiz ("Nilai Daily Quiz, Uji Kom, dan Tryout") memiliki tab per jenis, pemilih rentang tanggal (contoh 10 Sep 2026 - 9 Nov 2026), dan tombol Export Excel.

- FR-RPT-002: Kartu ringkasan (contoh tab Uji Kom): Sesi dikerjakan (jumlah sesi dan jumlah quiz berbeda), Rata-rata nilai (per jenjang SD dan SMP), Selesai tepat waktu (persentase dan jumlah kehabisan waktu), dan Soal sering salah (jumlah soal yang salah oleh lebih dari 15% siswa, beserta bab terbanyak, contoh "Bangun Datar").

- FR-RPT-003: Grafik Tren nilai rata-rata per minggu (12 minggu terakhir) untuk SD dan SMP pada skala 0-100, dan grafik Sebaran nilai (jumlah siswa per rentang 0-20, 21-40, 41-60, 61-80, dan 81-100) dengan rentang terbanyak disorot.

- FR-RPT-004: Tabel rincian per sesi: Tanggal, Quiz, Kelas, Peserta, Rata-rata, Tertinggi, Terendah, dan Aksi. Layar rincian per siswa tidak terlihat pada desain.

- FR-RPT-005: Ambang "soal sering salah" (15% pada desain) dan rentang waktu laporan harus dapat dikonfigurasi.

Laporan RUMATH

- FR-RPT-006: Report RUMATH menampilkan periode (contoh "Oktober 2026 - sekarang") dan empat kartu: Sesi latihan (jumlah sesi dan jumlah siswa berlatih), Rata-rata benar (contoh 6,7/10 beserta rincian Level 1 sampai 3), Video ditonton (total durasi tonton dan jumlah video diputar), dan Coin terpakai (total dan rata-rata per siswa, dari biaya latihan RUMATH).

- FR-RPT-007: Grafik Latihan per jenis soal (Penjumlahan, Pengurangan, Perkalian, Pembagian, Soal cerita) menampilkan jumlah sesi latihan pada periode laporan. Bagian bawah halaman tidak terlihat pada desain.

### 3.19 Landing Page Management

Landing Page mengatur isi halaman publik (lihat 1.3 Public Website). Section dapat digeser untuk mengubah urutan.

Functional requirements:

- FR-LDP-001: Menu Landing Page memiliki tab Beranda, Paket & Biaya, Siswa Berprestasi, dan Quiz Trial.

- FR-LDP-002: Tab Beranda berisi section berurutan: Hero (judul, tombol, foto alumni), Video profil (video company profile 2 menit), Bukti prestasi (alumni yang masuk SMA favorit), Program (Reguler, Intensif, Privat), Fasilitas (foto ruang kelas dan cabang), Testimoni (cerita orang tua dan siswa), Promo (banner promo bulanan), dan FAQ (12 pertanyaan).

- FR-LDP-003: Setiap section dapat ditampilkan atau disembunyikan (saklar "Tampil") dan dipindah urutannya dengan geser.

- FR-LDP-004: Admin dapat mengubah isi section lewat tombol Edit. Form edit belum tersedia pada desain; gambar diambil dari Image library (3.17).

- FR-LDP-005: Perubahan tidak langsung tampil di situs publik. Panel "Belum dipublish" mencantumkan perubahan yang menunggu (contoh "Hero · judul diubah", "Promo · disembunyikan", "Urutan section diubah") serta waktu dan pelaku publish terakhir. Perubahan tayang setelah tombol "Publish" ditekan.

- FR-LDP-006: Tombol "Lihat halaman" membuka halaman publik.

- FR-LDP-007: Isi tab Paket & Biaya, Siswa Berprestasi, dan Quiz Trial belum terlihat pada desain. Sumber data tab Paket & Biaya (manual atau dari menu Paket) dikonfirmasi (lihat BD-027).

### 3.20 Task Management

Task adalah penugasan dari Super Admin kepada Admin. Menu ini diberi label "Super" pada sidebar admin.

Functional requirements:

- FR-TSK-001: Super Admin dapat membuat dan mengubah task untuk Admin dengan judul, Kategori (contoh User, Bank soal, Pengumuman, Quiz), penerima (Untuk), Tenggat, dan Prioritas (contoh Tinggi dan Sedang). Form pembuatan task belum tersedia pada desain.

- FR-TSK-002: Halaman Task memiliki tab "Task saya" dan "Semua task". Kolom: Selesai (kotak centang), Task, Kategori, Untuk, Tenggat, Prioritas, dan pengirim (contoh "dari Hendrawan").

- FR-TSK-003: Penerima task menandai task selesai dengan kotak centang. Task selesai tampil dicoret, dan judul halaman menampilkan jumlah task yang belum selesai (contoh "2 belum selesai").

- FR-TSK-004: Hanya Super Admin yang dapat membuat dan mengubah task (BR-026). Ringkasan task aktif tampil di Beranda admin (3.11).

- FR-TSK-005: Contoh task pada desain: "Import 120 akun siswa baru", "Upload 40 soal Daily Quiz SMP - Pecahan", "Buat pengumuman Tryout Bersama", dan "Kode akses Uji Kom IPA 30 Okt".

## 4. BUSINESS RULES

Bagian ini berisi aturan bisnis yang wajib menjadi acuan seluruh implementasi.

| **ID** | **Business Rule**                                                                                                |
|--------|------------------------------------------------------------------------------------------------------------------|
| BR-001 | Setiap user memiliki role dan hak akses tertentu.                                                                |
| BR-002 | Student harus memiliki akun aktif untuk menggunakan platform.                                                    |
| BR-003 | Struktur materi mengikuti hierarchy yang telah ditentukan.                                                       |
| BR-004 | Setiap materi harus terhubung dengan topik.                                                                      |
| BR-005 | Student hanya dapat mengakses materi sesuai hak aksesnya.                                                        |
| BR-006 | Subscription yang sudah expired tidak dapat digunakan untuk mengakses fitur berlangganan.                        |
| BR-007 | Riwayat pembelajaran tidak dihapus ketika subscription expired.                                                  |
| BR-008 | Tryout harus memiliki kumpulan soal yang valid.                                                                  |
| BR-009 | Hasil latihan dan tryout harus tersimpan secara permanen.                                                        |
| BR-010 | Coin tidak boleh memiliki saldo negatif.                                                                         |
| BR-011 | Reset kuota gratis harian dilakukan setiap hari.                                                                 |
| BR-012 | Penggunaan coin harus tercatat dalam transaction history.                                                        |
| BR-013 | RUMATH menghasilkan 10 soal isian per sesi latihan.                                                              |
| BR-014 | Seluruh waktu operasional bisnis menggunakan Asia/Jakarta.                                                       |
| BR-015 | Semua validasi keamanan dan hak akses wajib dilakukan di backend.                                                |
| BR-016 | Akun student dibuat oleh admin; student tidak dapat mendaftar sendiri.                                           |
| BR-017 | Login menggunakan email atau nomor HP.                                                                           |
| BR-018 | Nilai Daily Quiz, Uji Kom, Tryout, dan RUMATH berskala 0-100 (skala Tryout dikonfirmasi, lihat BD-033).          |
| BR-019 | Daily Quiz memiliki kuota gratis 2 kali per hari; setelah itu 10 coin per quiz (nilai dapat dikonfigurasi).      |
| BR-020 | Uji Kom dan Tryout hanya dapat dimasuki dengan kode akses dan pada waktu yang dijadwalkan.                       |
| BR-021 | Peringkat dihitung dari poin pada tiga kategori terpisah: Daily Quiz, Uji Kom, dan Tryout.                       |
| BR-022 | Materi, soal, dan statistik dikelompokkan berdasarkan jenjang (SD / SMP).                                        |
| BR-023 | Student hanya melihat data miliknya sendiri, kecuali data peringkat (nama, sekolah, poin).                       |
| BR-024 | Admin dan Tutor bekerja dalam lingkup cabang pada akunnya; Super Admin melihat semua cabang (menunggu BD-025).   |
| BR-025 | Materi, quiz, dan konten Landing Page berstatus Draft tidak tampil kepada student atau publik sampai di-publish. |
| BR-026 | Hanya Super Admin yang dapat membuat dan mengubah task; penerima task dapat menandai task selesai.               |
| BR-027 | Soal hanya dapat dipilih untuk jenis quiz yang ditandai pada kolom "Dipakai untuk" soal tersebut.                |
| BR-028 | Perubahan Landing Page baru tampil di situs publik setelah tombol Publish ditekan.                               |
| BR-029 | Mapel berlaku untuk semua jenjang. Bidang Materi, Bab, Subbab, dan Topik terikat pada satu jenjang dan satu kelas yang sama dengan induknya. |

## 5. SYSTEM ARCHITECTURE

### 5.1 Architectural Approach

Sistem menggunakan pendekatan **Modular Monolith**.

Artinya, backend masih berjalan sebagai satu aplikasi Go Gin, tetapi kode dibagi berdasarkan domain bisnis.

Tujuan:

- Memisahkan tanggung jawab setiap modul.

- Mengurangi coupling antar-modul.

- Memudahkan testing.

- Memudahkan maintenance.

- Mempermudah migrasi menuju microservices.

### 5.2 High-Level Architecture

CLIENT APPLICATIONS

\|

+----------+----------+

\| \|

Public Website E-Learning

\| \|

+----------+----------+

\|

API Gateway

\|

GO GIN BACKEND

\|

+--------------+--------------+

\| \| \|

AUTH STUDENT TUTOR

\| \| \|

+--------------+--------------+

\|

MODULAR MONOLITH

\|

+-----------------+------------------+

\| \| \|

LEARNING QUIZ TRYOUT

\| \| \|

+-----------------+------------------+

\|

+-----------------+------------------+

\| \| \|

RUMATH SUBSCRIPTION COIN

\| \| \|

+-----------------+------------------+

\|

POSTGRESQL

\|

+----------+----------+

\| \|

FILE STORAGE BACKGROUND JOBS

PDF / VIDEO QUOTA RESET

### 5.3 Future Microservices Architecture

Pada tahap awal seluruh modul berada dalam satu aplikasi.

Namun, setiap modul harus memiliki batas domain yang jelas.

Potensi pemisahan menjadi service:

| **Module**              | **Future Service**           |
|-------------------------|------------------------------|
| Authentication          | Auth Service                 |
| Student                 | Student Service              |
| Tutor                   | Tutor Service                |
| Learning                | Learning Service             |
| Quiz                    | Assessment Service           |
| Tryout                  | Assessment Service           |
| RUMATH                  | Interactive Learning Service |
| Subscription            | Subscription Service         |
| Coin                    | Coin Service                 |
| Notification            | Notification Service         |
| Leaderboard (Peringkat) | Ranking Service (opsional)   |

Tidak semua modul wajib menjadi microservice. Pemisahan dilakukan berdasarkan kebutuhan skalabilitas dan kompleksitas bisnis.

### 5.4 Architecture Rules

Wajib mengikuti ketentuan:

1.  Setiap modul memiliki tanggung jawab bisnis yang jelas.

2.  Modul tidak boleh mengakses repository milik modul lain secara langsung.

3.  Komunikasi antar-modul melalui public service interface atau kontrak yang didefinisikan.

4.  Hindari circular dependency.

5.  Business logic tidak boleh ditempatkan di HTTP handler.

6.  Database access harus melalui repository.

7.  Gunakan dependency injection.

8.  Setiap modul memiliki unit test.

9.  Hindari shared mutable state antar-modul.

10. Setiap proses asynchronous harus memiliki mekanisme retry dan idempotency jika diperlukan.

Untuk tahap awal, database boleh menggunakan satu PostgreSQL dengan schema yang terorganisasi. Namun, batas kepemilikan data setiap modul harus jelas.

## 6. BACKEND TECHNOLOGY STACK

| **Component**        | **Technology**                                  |
|----------------------|-------------------------------------------------|
| Programming Language | Go                                              |
| Web Framework        | Gin                                             |
| Database             | PostgreSQL                                      |
| ORM / Query          | GORM atau SQLC                                  |
| Authentication       | JWT                                             |
| Password Hashing     | bcrypt / Argon2                                 |
| Validation           | go-playground/validator                         |
| Logging              | Structured Logging                              |
| API Documentation    | Swagger / OpenAPI                               |
| File Storage         | Local Storage (initial), S3-compatible (future) |
| Background Jobs      | Go Worker / Scheduler                           |
| Containerization     | Docker                                          |
| Deployment           | Linux VPS                                       |
| Version Control      | Git                                             |

**Database Access Recommendation**

Untuk project ini, pemilihan GORM atau SQLC perlu ditentukan saat technical setup.

Rekomendasi:

- GORM untuk mempercepat pengembangan CRUD.

- SQLC untuk query yang membutuhkan kontrol SQL lebih ketat.

- Hindari mencampurkan akses database secara sembarangan di seluruh layer aplikasi.

## 7. RECOMMENDED BACKEND PROJECT STRUCTURE

Struktur direktori yang direkomendasikan:

```text
backend/
│
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   │
│   ├── platform/
│   │   ├── database/
│   │   ├── logger/
│   │   ├── config/
│   │   ├── middleware/
│   │   ├── storage/
│   │   └── response/
│   │
│   ├── modules/
│   │   │
│   │   ├── auth/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   ├── model/
│   │   │   └── dto/
│   │   │
│   │   ├── student/
│   │   ├── tutor/
│   │   ├── learning/
│   │   ├── material/
│   │   ├── quiz/
│   │   ├── tryout/
│   │   ├── rumath/
│   │   ├── subscription/
│   │   ├── coin/
│   │   ├── progress/
│   │   ├── leaderboard/
│   │   ├── notification/
│   │   ├── masterdata/
│   │   ├── media/
│   │   ├── landingpage/
│   │   ├── task/
│   │   ├── report/
│   │   └── dashboard/
│   │
│   └── shared/
│       ├── errors/
│       ├── constants/
│       └── utils/
│
├── migrations/
│
├── docs/
│   ├── architecture/
│   ├── api/
│   └── database/
│
├── tests/
│
├── deployments/
│   └── docker/
│
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

Catatan: Struktur di atas merupakan rekomendasi awal. Claude Code dapat menyesuaikan detail implementasi tanpa menghilangkan pemisahan domain.

## 8. DATABASE DESIGN REQUIREMENTS

Database menggunakan PostgreSQL.

Berikut rancangan awal domain database.

### 8.1 User & Access

- users

- roles

- permissions

- role_permissions

- user_roles

- refresh_tokens

<!-- -->

- branches

### 8.2 Student & Tutor

- students

- tutors

- tutor_students

<!-- -->

- student_parent_contacts

### 8.3 Learning Management

- subjects

- subject_fields

- chapters

- subchapters

- topics

- learning_materials

- learning_material_attachments

- student_material_progress

<!-- -->

- student_material_downloads

### 8.4 Quiz & Assessment

- questions

- question_options

<!-- -->

- question_usages

<!-- -->

- exercises

- exercise_questions

- exercise_attempts

- exercise_answers

<!-- -->

- daily_quiz_sessions

### 8.5 Tryout

- tryouts

- tryout_questions

- tryout_attempts

- tryout_answers

- tryout_results

<!-- -->

- tryout_access_codes

### 8.6 RUMATH (sebelumnya Math Fun)

- rumath_configs

- rumath_sessions

- rumath_questions

- rumath_answers

<!-- -->

- rumath_videos

- rumath_video_progress

### 8.7 Subscription

- subscription_plans

- student_subscriptions

- subscription_histories

<!-- -->

- subscription_plan_prices

### 8.8 Coin & Kuota Harian

- student_coin_balances

- coin_transactions

- daily_quota_reset_logs

<!-- -->

- daily_quota_usages

- coin_packages

- coin_purchases

### 8.9 Progress & Reporting

- student_learning_progress

- student_activity_logs

<!-- -->

- student_learning_streaks

### 8.10 Peringkat & Poin

- point_transactions

- leaderboard_snapshots (opsional)

### 8.11 Notifikasi & Pengumuman

- notifications

- user_notifications

- announcements

### 8.12 Admin: Master Data, Media, dan Landing Page

- master_data_items (tipe: jenjang, kelas, mapel, bidang materi, bab, subbab, topik, sekolah; parent; status). Bidang materi s/d topik juga menyimpan mapel, jenjang, dan kelas (disalin dari induk) supaya filter cepat tanpa JOIN berlapis.

- media_assets (Image library)

- landing_pages

- landing_sections

- landing_publish_logs

- admin_tasks

- leaderboard_configs

- leaderboard_rewards

- student_access_overrides (perubahan coin dan akses oleh admin)

- import_batches (Import Excel user dan soal)

- audit_logs

**Database Standards**

- Gunakan UUID sebagai primary key untuk entitas utama.

- Gunakan foreign key untuk menjaga integritas relasi.

- Gunakan created_at dan updated_at.

- Gunakan deleted_at hanya pada entitas yang membutuhkan soft delete.

- Gunakan index pada kolom yang sering digunakan dalam filtering.

- Gunakan transaction untuk proses yang melibatkan beberapa perubahan data.

- Simpan timestamp dalam UTC dan konversikan ke Asia/Jakarta pada kebutuhan tampilan serta aturan bisnis.

- Gunakan migration versioning.

- Jangan mengandalkan auto-migration pada production.

Rancangan di atas adalah daftar kandidat tabel, bukan keputusan final seluruh kolom dan relasinya. Database ERD wajib dibuat dan direview sebelum implementasi modul.

## 9. API DEVELOPMENT STANDARDS

API menggunakan RESTful architecture.

Base URL:

/api/v1

Contoh endpoint:

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout

GET    /api/v1/students
POST   /api/v1/students
GET    /api/v1/students/:id
PUT    /api/v1/students/:id

GET    /api/v1/subjects
POST   /api/v1/subjects

GET    /api/v1/subjects/:id/fields
GET    /api/v1/topics/:id/materials

GET    /api/v1/exercises
POST   /api/v1/exercises/:id/attempts

GET    /api/v1/tryouts
POST   /api/v1/tryouts/:id/attempts

GET    /api/v1/rumath/config
POST   /api/v1/rumath/sessions

GET    /api/v1/subscriptions/me
GET    /api/v1/coins/me

GET    /api/v1/progress/me
GET    /api/v1/dashboard/student
POST   /api/v1/daily-quizzes
GET    /api/v1/daily-quizzes
POST   /api/v1/exams/access-codes/verify
GET    /api/v1/leaderboards
GET    /api/v1/notifications
POST   /api/v1/notifications/read-all
GET    /api/v1/subscription-plans
GET    /api/v1/coin-packages
POST   /api/v1/coin-purchases
GET    /api/v1/search
GET    /api/v1/progress/me/export
Contoh endpoint panel admin (prefiks /api/v1/admin):
GET    /api/v1/admin/users
POST   /api/v1/admin/users
POST   /api/v1/admin/users/import
GET    /api/v1/admin/materials
POST   /api/v1/admin/materials
PATCH  /api/v1/admin/materials/:id/publish
GET    /api/v1/admin/questions
POST   /api/v1/admin/questions
POST   /api/v1/admin/questions/import
GET    /api/v1/admin/quizzes
POST   /api/v1/admin/quizzes
POST   /api/v1/admin/quizzes/:id/access-codes
GET    /api/v1/admin/rumath/videos
POST   /api/v1/admin/rumath/videos
GET    /api/v1/admin/leaderboards/config
PUT    /api/v1/admin/leaderboards/config/:type
GET    /api/v1/admin/leaderboards/:type/export
GET    /api/v1/admin/reports/quiz
GET    /api/v1/admin/reports/quiz/export
GET    /api/v1/admin/reports/rumath
GET    /api/v1/admin/master-data
POST   /api/v1/admin/master-data
GET    /api/v1/admin/images
POST   /api/v1/admin/images
GET    /api/v1/admin/plans
POST   /api/v1/admin/plans
GET    /api/v1/admin/students/coin-access
PUT    /api/v1/admin/students/:id/coin-access
POST   /api/v1/admin/students/coin-access/bulk
GET    /api/v1/admin/landing-pages/:page/sections
PUT    /api/v1/admin/landing-pages/:page/sections
POST   /api/v1/admin/landing-pages/:page/publish
GET    /api/v1/admin/tasks
POST   /api/v1/admin/tasks
PATCH  /api/v1/admin/tasks/:id/complete
```

Standar response:

Success:

```json
{
  "success": true,
  "message": "Data retrieved successfully",
  "data": {},
  "meta": null
}
```

Error:

```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    { "field": "email", "message": "Email is required" }
  ]
}
```

Standar API:

- Gunakan HTTP status code yang sesuai.

- Validasi seluruh request.

- Gunakan pagination pada endpoint list.

- Gunakan filtering dan sorting bila diperlukan.

- Jangan mengembalikan password atau informasi sensitif.

- Dokumentasikan seluruh endpoint menggunakan Swagger.

- Gunakan konsistensi penamaan request dan response DTO.

## 10. NON-FUNCTIONAL REQUIREMENTS

### 10.1 Performance

- Sistem harus mendukung sekitar 400 pengguna pada tahap awal.

- API harus dirancang agar tidak melakukan query yang tidak diperlukan.

- Gunakan pagination untuk data berjumlah besar.

<!-- -->

- Daftar pada panel admin (user, soal, materi, peringkat, laporan) memakai pagination, pencarian, dan filter di sisi server.

<!-- -->

- Optimalkan query database.

- Gunakan database connection pooling.

- Proses berat dapat dipindahkan ke background worker.

Target awal pengujian:

- API CRUD umum: p95 maksimal 500 ms pada kondisi pengujian yang ditetapkan.

- API pembelajaran: p95 maksimal 1.000 ms pada kondisi pengujian yang ditetapkan.

- Proses penilaian dan tryout harus diuji menggunakan skenario concurrent requests.

Angka tersebut merupakan target engineering awal, bukan jaminan performa tanpa load testing.

### 10.2 Security

- Password wajib di-hash.

- Endpoint privat wajib menggunakan authentication middleware.

- Implementasikan authorization berdasarkan role dan permission.

- Validasi ownership data student.

- Batasi akses file berdasarkan permission.

- Lindungi endpoint dari SQL injection.

- Validasi ukuran dan tipe file upload.

- Jangan menyimpan credential di source code.

- Gunakan environment variables untuk konfigurasi rahasia.

<!-- -->

- Kode akses Uji Kom dan Tryout harus divalidasi di backend, memiliki masa berlaku, dan tidak dapat ditebak.

- Data pribadi siswa (anak) dan kontak orang tua/wali dilindungi sesuai ketentuan perlindungan data pribadi yang berlaku; nomor kontak ditampilkan tersamar.

- Data yang tampil pada peringkat dibatasi sesuai kebijakan privasi yang ditetapkan (BD-015).

- Akun demo dan data contoh tidak boleh ada di production.

- Import Excel (user dan soal) divalidasi di backend: tipe dan ukuran file, struktur kolom, dan duplikasi; hasil per baris ditampilkan.

- Disarankan: perubahan sensitif oleh admin (hapus data, ubah coin dan akses, ubah poin, publish Landing Page) dicatat dalam audit log.

### 10.3 Scalability

- Modul harus loosely coupled.

- Hindari ketergantungan langsung antarmodul.

- Gunakan interface untuk dependency injection.

- Background job harus terpisah dari HTTP request lifecycle.

- Gunakan object storage ketika kebutuhan file meningkat.

- Rancang kontrak API yang stabil.

### 10.4 Reliability

- Gunakan database transactions.

- Implementasikan structured logging.

- Implementasikan health check.

- Implementasikan graceful shutdown.

- Sediakan backup database.

- Proses scheduler harus idempotent.

- Implementasikan error handling secara konsisten.

### 10.5 Monitoring

Sistem minimal menyediakan:

- Application logs.

- Error logs.

- API response time.

- Database connection monitoring.

- Background job status.

- Scheduler execution logs.

- Basic CPU dan memory monitoring.

## 11. BACKGROUND JOBS & SCHEDULER

Sistem membutuhkan background process untuk tugas yang tidak harus dijalankan secara langsung oleh request pengguna.

Initial jobs:

| **Job**                      | **Description**                                                                                                                             |
|------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| Daily Quota Reset            | Reset kuota gratis harian student (contoh Daily Quiz 2 kali per hari) setiap hari.                                                          |
| Subscription Expiration      | Memperbarui status subscription yang expired.                                                                                               |
| Learning Progress Processing | Memproses progres jika diperlukan secara asynchronous.                                                                                      |
| Cleanup                      | Membersihkan data sementara atau expired sesuai kebijakan.                                                                                  |
| Ranking & Streak Processing  | Memperbarui peringkat, poin, dan streak belajar jika tidak dihitung real-time.                                                              |
| Notification Generation      | Membuat notifikasi otomatis dan terjadwal (hasil ujian, paket aktif, perubahan peringkat, materi baru).                                     |
| Leaderboard Reset & Reward   | Mereset papan peringkat sesuai periode dan waktu reset (mingguan, bulanan, atau saat sesi ditutup) dan memproses hadiah sesuai konfigurasi. |
| Report Aggregation           | Menyiapkan data laporan (tren, sebaran nilai, soal sering salah) jika tidak dihitung real-time.                                             |

Aturan scheduler:

- Menggunakan timezone Asia/Jakarta untuk jadwal bisnis.

- Mencatat setiap eksekusi.

- Mendukung retry.

- Tidak menjalankan transaksi bisnis secara berulang tanpa idempotency.

- Mencegah eksekusi ganda pada deployment multi-instance.

## 12. DEVELOPMENT PHASES

Pengembangan dilakukan secara bertahap agar setiap bagian dapat diuji sebelum melanjutkan ke modul berikutnya.

**Phase 1 — Project Foundation**

- Setup repository.

- Setup Go Gin.

- Setup PostgreSQL.

- Setup database migration.

- Setup project architecture.

- Setup configuration.

- Setup logging.

- Setup error handling.

- Setup API response standard.

- Setup Swagger.

- Setup Docker.

**Deliverable:** Backend foundation siap digunakan.

**Phase 2 — Authentication & User Management**

- Login.

- Logout.

- Refresh token.

- Role management.

- Permission management.

- Student management.

- Tutor management.

<!-- -->

- Cabang dan User management panel admin (termasuk Import Excel).

**Deliverable:** Sistem autentikasi dan pengelolaan user.

**Phase 3 — Learning Management**

- Subject management.

- Subject field management.

- Chapter management.

- Subchapter management.

- Topic management.

- Learning material management.

- File upload.

- Student learning access.

<!-- -->

- Student module browsing (mapel, bab, PDF), progres baca, dan unduh.

- Panel admin Materi (draft dan publish), Master Data, dan Image library.

**Deliverable:** Sistem pengelolaan dan akses materi.

**Phase 4 — Exercise & Assessment**

- Question bank.

- Exercise management.

- Exercise attempt.

- Answer validation.

- Automatic scoring.

- Result history.

<!-- -->

- Daily Quiz (soal otomatis, timer, ragu-ragu, poin, kuota gratis).

- Panel admin Bank Soal (Import Excel, rumus, pratinjau).

**Deliverable:** Sistem latihan soal.

**Phase 5 — Tryout**

- Tryout management.

- Question assignment.

- Scheduling.

- Student attempt.

- Answer persistence.

- Automatic scoring.

- Result reporting.

<!-- -->

- Uji Kom, kode akses, ranking, dan pembahasan.

- Panel admin Quiz (Buat quiz tiga langkah, biaya coin, kode akses).

**Deliverable:** Sistem simulasi ujian.

**Phase 6 — RUMATH (sebelumnya Math Fun)**

- RUMATH configuration.

- Automatic question generation.

- Interactive Math.

- Difficulty configuration.

- Scoring.

- Session history.

<!-- -->

- Video RUMATH dan progres tonton.

- Jenis soal dan level.

- Panel admin RUMATH (video materi dan soal isian).

**Deliverable:** Modul pembelajaran matematika interaktif.

**Phase 7 — Subscription & Coin**

- Subscription plans.

- Student subscription.

- Subscription validation.

- Coin balance.

- Coin transactions.

- Daily quota reset.

<!-- -->

- Katalog paket dan Isi coin.

- Perpanjangan paket dan riwayat pembelian coin.

- Panel admin Paket serta Coin & Akses (edit satuan dan massal).

Deliverable: Sistem pembatasan akses dan penggunaan coin.

**Phase 8 — Dashboard & Reporting**

- Student dashboard.

- Tutor dashboard.

- Admin dashboard.

- Learning progress.

- Activity reports.

<!-- -->

- Peringkat (leaderboard).

- Notifikasi dan pengumuman.

- Profil, export riwayat belajar, dan pencarian global.

- Panel admin Leaderboard (konfigurasi, hadiah, reset), Report Quiz, dan Report RUMATH (export Excel).

- Landing Page management (draft dan publish).

- Task management Super Admin.

**Deliverable:** Sistem monitoring dan reporting.

**Phase 9 — Testing & Deployment**

- Unit testing.

- Integration testing.

- API testing.

- Security testing.

- Load testing.

- Production deployment.

- Database backup.

- Monitoring.

**Deliverable:** Platform siap digunakan.

## 13. TESTING & ACCEPTANCE CRITERIA

Setiap modul harus memiliki acceptance criteria.

Contoh:

**Authentication**

- User dapat login menggunakan credential valid.

- User tidak dapat login menggunakan password salah.

- Token expired tidak dapat digunakan.

- User tidak dapat mengakses endpoint tanpa permission.

**Learning**

- Admin dapat membuat struktur materi sesuai hierarchy.

- Student dapat melihat materi yang diizinkan.

- Student tidak dapat mengakses materi yang tidak memiliki hak akses.

- Progres materi tersimpan dengan benar.

**Tryout**

- Student dapat mengikuti tryout yang tersedia.

- Student tidak dapat mengikuti tryout yang tidak memenuhi ketentuan akses.

- Jawaban student tersimpan.

- Hasil penilaian sesuai dengan kunci jawaban.

- Sistem dapat menangani pengiriman jawaban secara concurrent tanpa kehilangan data.

**Subscription**

- Student dengan subscription aktif dapat mengakses fitur berlangganan.

- Student dengan subscription expired tidak dapat mengakses fitur berlangganan.

- Data pembelajaran tetap tersimpan.

**Coin & Kuota Harian**

- Coin berkurang sesuai konfigurasi.

- Saldo tidak boleh negatif.

- Transaksi coin tersimpan.

- Daily reset berjalan sesuai jadwal.

- Reset tidak menyebabkan kuota gratis bertambah dua kali akibat job yang berjalan ulang.

**Daily Quiz**

- Student dapat membuat quiz dengan memilih mata pelajaran, bidang materi, bab, dan level; subbab dan topik bersifat opsional.

- Sistem menghasilkan 5 soal sesuai pilihan dan menampilkan timer 10 menit.

- Student dapat menandai soal ragu-ragu dan panel Daftar soal menampilkan status yang benar.

- Quiz yang dikumpulkan dinilai otomatis dan masuk ke Riwayat.

- Quiz setelah 2 kali gratis dalam sehari memotong 10 coin; jika coin kurang, quiz tidak dimulai.

**Uji Kom & Tryout**

- Student tidak dapat masuk sebelum waktu dibuka atau dengan kode akses yang salah.

- Nilai, peringkat, dan pembahasan tampil setelah hasil dirilis.

**RUMATH**

- Latihan menghasilkan 10 soal isian sesuai jenis soal dan level yang dipilih dengan batas waktu 15 menit.

- Progres tonton video tersimpan dan tampil di "Lanjutkan nonton".

**Peringkat**

- Urutan sesuai poin dan filter yang dipilih; posisi student sendiri selalu tampil.

**Notifikasi**

- Notifikasi baru muncul dengan badge; "Tandai dibaca" menghapus badge.

**Panel Admin**

- Materi berstatus Draft tidak tampil di Modul siswa sampai di-publish.

- Import Excel (user dan soal) menampilkan baris yang gagal beserta alasannya.

- Rumus di antara tanda \$ ... \$ tampil benar pada Pratinjau admin dan aplikasi siswa.

- Soal hanya dapat dipilih untuk jenis quiz sesuai "Dipakai untuk".

- Uji Kom dan Tryout tidak dapat di-publish tanpa soal yang valid dan kode akses.

- Perubahan coin dan akses oleh admin (satuan dan massal) tercatat dan saldo tidak menjadi negatif.

- Reset dan hadiah Leaderboard berjalan sesuai konfigurasi tanpa pemberian ganda.

- Perubahan Landing Page baru tampil di situs publik setelah Publish.

- Task hanya dapat dibuat dan diubah Super Admin; Admin dapat menandai task selesai.

- Report Quiz dan Report RUMATH sesuai filter tanggal dan dapat diekspor ke Excel.

## 14. OUT OF SCOPE / FUTURE ENHANCEMENTS

Fitur berikut tidak termasuk prioritas initial development dan dapat dipertimbangkan untuk pengembangan selanjutnya:

- Payment gateway integration.

- Live streaming classes.

- Real-time chat.

- Video conference.

- AI-powered learning assistant.

- AI-generated questions.

- Advanced analytics.

- Mobile application.

- Microservices migration.

- Advanced notification system.

Catatan: Leaderboard (Peringkat) dan notifikasi in-app sudah masuk scope berdasarkan desain (lihat 3.12 dan 3.13). "Advanced notification system" tetap di luar scope awal.

## 15. BUSINESS DECISIONS PENDING

Hal-hal berikut perlu dikonfirmasi kepada pihak bisnis sebelum implementasi final.

| **ID** | **Question**                                                                                                                                                                                                                                                                                                                                                                                    | **Status**                       |
|--------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------|
| BD-001 | Apakah student dapat memilih sendiri mata pelajaran?                                                                                                                                                                                                                                                                                                                                            | TBD                              |
| BD-002 | Apakah tutor dapat membuat materi dan soal? (Desain: materi, jadwal ujian, dan kode akses disebut berasal dari tutor.)                                                                                                                                                                                                                                                                          | TBD                              |
| BD-003 | Apakah subscription berbasis durasi atau paket materi?                                                                                                                                                                                                                                                                                                                                          | TBD                              |
| BD-004 | Berapa jumlah kuota gratis harian setiap student? (Desain: Daily Quiz 2 kali per hari; fitur lain belum jelas.)                                                                                                                                                                                                                                                                                 | Sebagian dijawab desain          |
| BD-005 | Apakah coin atau kuota harian yang tidak digunakan akan hangus? (Desain: saldo coin tampil bertahan, contoh 740.)                                                                                                                                                                                                                                                                               | TBD                              |
| BD-006 | Fitur apa saja yang menggunakan coin? (Desain: Daily Quiz tambahan 10 coin, tryout tambahan, RUMATH. Catatan: RUMATH tertulis "gratis di paketmu" tetapi Isi coin menyebut "untuk RUMATH". Desain admin: setiap quiz punya "Biaya coin" (0 = gratis untuk siswa berpaket); Report RUMATH mencatat "Coin terpakai"; akses RUMATH di Coin & Akses hanya tercantum pada paket Intensif Masuk SMA.) | Sebagian dijawab desain          |
| BD-007 | Apakah tryout memiliki passing grade?                                                                                                                                                                                                                                                                                                                                                           | TBD                              |
| BD-008 | Apakah tryout memiliki sistem ranking? (Desain: ranking sudah tampil untuk Daily Quiz, Uji Kom, dan Tryout.)                                                                                                                                                                                                                                                                                    | Dijawab desain, perlu konfirmasi |
| BD-009 | Apakah student dapat mengulang tryout?                                                                                                                                                                                                                                                                                                                                                          | TBD                              |
| BD-010 | Apakah terdapat batas maksimal percobaan latihan? (Desain: Daily Quiz gratis 2 kali per hari, lalu memakai coin.)                                                                                                                                                                                                                                                                               | Sebagian dijawab desain          |
| BD-011 | Apakah pembayaran dilakukan melalui admin atau payment gateway? (Desain: tombol "Pilih paket", "Perpanjang", dan "Beli coin" tersedia, alur pembayaran belum tergambar. Desain admin: Coin & Akses mengubah coin dan akses siswa secara manual, dan Paket tidak memuat pengaturan pembayaran, sehingga alur lewat admin tampak didukung.)                                                       | Sebagian dijawab desain          |
| BD-012 | Apakah terdapat notifikasi WhatsApp atau email? (Desain: notifikasi in-app sudah ada; "Tanya tutor" membuka WhatsApp.)                                                                                                                                                                                                                                                                          | TBD                              |
| BD-013 | Apakah satu student dapat memiliki lebih dari satu tutor?                                                                                                                                                                                                                                                                                                                                       | TBD                              |
| BD-014 | Field data diri apa yang boleh diubah student sendiri (nama, kelas, sekolah)?                                                                                                                                                                                                                                                                                                                   | TBD                              |
| BD-015 | Apakah nama lengkap dan sekolah siswa lain boleh tampil di Peringkat (pengguna SD/SMP)? Perlu opsi inisial atau sembunyikan?                                                                                                                                                                                                                                                                    | TBD                              |
| BD-016 | Bagaimana rumus poin Daily Quiz, Uji Kom, dan Tryout (desain: nilai 20 menjadi +10 poin) dan aturan peringkat saat poin sama? (Desain admin: Daily Quiz nilai ÷ 2 per quiz; Uji Kom nilai × 1,4; Tryout skor 0-1.000; RUMATH 10 poin per soal benar.)                                                                                                                                           | Sebagian dijawab desain          |
| BD-017 | Apakah "pertemuan per minggu" pada paket dikelola di aplikasi (jadwal, kehadiran) atau hanya informasi paket? (Desain admin: "pertemuan per minggu" hanya teks pada kolom Isi paket; tidak ada menu jadwal atau kehadiran.)                                                                                                                                                                     | Sebagian dijawab desain          |
| BD-018 | Apakah student boleh mengunduh semua PDF? Apakah ada batas unduhan atau watermark?                                                                                                                                                                                                                                                                                                              | TBD                              |
| BD-019 | Apakah jenis materi selain PDF (Text, Image, External Link) masuk tahap awal? Desain Modul dan menu Materi admin hanya mengelola PDF; video ada di RUMATH dan gambar di Image.                                                                                                                                                                                                                  | Dijawab desain, perlu konfirmasi |
| BD-020 | Apakah Tryout Kelas 12 / jenjang SMA masuk scope? Desain menyebut platform untuk siswa SD dan SMP; Master Data Kelas berjumlah 9 (diasumsikan Kelas 1-9) sehingga tidak ada kelas SMA.                                                                                                                                                                                                          | Sebagian dijawab desain          |
| BD-021 | Kapan hasil Uji Kom dan Tryout dirilis: otomatis saat selesai atau manual oleh tutor/admin?                                                                                                                                                                                                                                                                                                     | TBD                              |
| BD-022 | Apakah quiz otomatis dikumpulkan saat waktu habis, dan bagaimana level Mudah/Sedang/Sulit dipetakan ke tingkat kesulitan soal?                                                                                                                                                                                                                                                                  | TBD                              |
| BD-023 | Aktivitas apa yang dihitung untuk target belajar 15 menit dan streak harian?                                                                                                                                                                                                                                                                                                                    | TBD                              |
| BD-024 | Bagaimana mekanisme promo (banner "Diskon 30%" dan harga coret paket) dan siapa yang mengelolanya? (Desain admin: banner promo berupa gambar kategori Banner di Image dan section Promo di Landing Page; field harga coret belum terlihat karena form Tambah paket belum ada.)                                                                                                                  | TBD                              |
| BD-025 | Apakah Admin dan Tutor hanya melihat dan mengelola data cabangnya (siswa, soal, laporan, peringkat, paket), dan apakah Super Admin selalu melihat semua cabang? (Desain: akun memiliki cabang; Super Admin tercatat "Semua cabang".)                                                                                                                                                            | TBD                              |
| BD-026 | Apa beda kewenangan Admin dan Super Admin (kelola akun Admin dan Super Admin, publish Landing Page, hapus data)? Apakah "hapus" berarti nonaktif atau hapus permanen, dan bagaimana untuk soal, materi, paket, atau master data yang sudah dipakai?                                                                                                                                             | TBD                              |
| BD-027 | Di mana admin membuat pengumuman, banner promo, dan notifikasi manual? Sidebar admin tidak memiliki menu Pengumuman, padahal ada task "Buat pengumuman Tryout Bersama". Apa sumber data tab Paket & Biaya di Landing Page?                                                                                                                                                                      | TBD                              |
| BD-028 | Leaderboard: apakah papan RUMATH tampil di aplikasi siswa (kini hanya 3 tab)? Bagaimana hadiah (coin, sertifikat digital, konsultasi gratis) diberikan dan siapa yang memproses? Apa yang diedit pada tombol Edit tiap baris peringkat?                                                                                                                                                         | TBD                              |
| BD-029 | Apakah soal RUMATH dibuat admin dan disimpan di Bank Soal (desain: 130 soal isian bertanda RUMATH), atau dihasilkan otomatis oleh sistem (FR-MATH-002)?                                                                                                                                                                                                                                         | TBD                              |
| BD-030 | Dari mana video RUMATH berasal (unggah ke storage atau tautan platform video)? Apakah durasi diisi otomatis atau manual?                                                                                                                                                                                                                                                                        | TBD                              |
| BD-031 | Apakah Subbab dan Topik masih dipakai? Form admin Materi dan Soal hanya memakai Jenjang, Kelas, Mapel, Bidang materi, dan Bab. **Jawaban v1.3:** Subbab dan Topik tetap dipakai; mapel sama untuk semua jenjang dan Bidang Materi ke bawah terhubung ke jenjang dan kelas (lihat 3.4.1). Form soal wajib sampai Bab, Subbab dan Topik opsional; untuk form materi masih perlu dikonfirmasi.                                  | Dijawab pemilik produk (v1.3)    |
| BD-032 | Buat quiz: bagaimana langkah 2 (pilih soal manual atau acak) dan langkah 3 (jadwal, kode akses otomatis atau manual, masa berlaku)? Apa arti entri Daily Quiz di menu Quiz admin, padahal Daily Quiz dibuat otomatis oleh siswa?                                                                                                                                                                | TBD                              |
| BD-033 | Skala nilai Tryout: BR-018 menyebut 0-100, sedangkan Leaderboard admin menyebut skor Tryout 0-1.000. Mana yang berlaku, dan bagaimana konversinya ke poin?                                                                                                                                                                                                                                      | TBD                              |

Keputusan yang belum dikonfirmasi tidak boleh dianggap sebagai kebutuhan bisnis final.

## 16. DEFINITION OF DONE

Suatu modul dinyatakan selesai apabila:

1.  Business logic sudah diimplementasikan.

2.  API sudah tersedia.

3.  Request validation sudah diterapkan.

4.  Authorization sudah diterapkan.

5.  Database migration sudah tersedia.

6.  Unit test sudah tersedia.

7.  Integration test untuk alur penting sudah tersedia.

8.  Swagger sudah diperbarui.

9.  Logging sudah tersedia.

10. Error handling sudah konsisten.

11. Tidak terdapat hardcoded credentials.

12. Dokumentasi teknis sudah diperbarui.

13. Acceptance criteria sudah terpenuhi.

## 17. FINAL ARCHITECTURAL DECISION

Rumbasha E-Learning akan menggunakan:

**Modular Monolith with Domain-Oriented Architecture.**

Prinsip utama:

- Go Gin sebagai backend.

- React sebagai frontend.

- PostgreSQL sebagai primary database.

- REST API sebagai komunikasi frontend-backend.

- Domain-based modular architecture.

- Repository pattern.

- Service layer.

- Dependency injection.

- Database migration.

- JWT authentication.

- Background worker.

- Structured logging.

- Docker deployment.

Tujuan akhirnya adalah membangun sistem yang mudah dikembangkan, mudah dipelihara, dan dapat dipisahkan menjadi microservices ketika kebutuhan bisnis dan skalabilitas memang mengharuskannya.

**END OF DOCUMENT**
