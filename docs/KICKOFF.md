# Memulai dengan Claude Code

## Persiapan satu kali
1. Salin `CLAUDE.md`, `README.md`, `.gitignore`, `.env.example`, dan folder `docs/` ke root repo, lalu commit.
2. Buka repo di VSCode dan mulai sesi Claude Code dari root repo.
3. Aktifkan plan mode untuk langkah pertama, supaya rencana ditinjau sebelum kode dibuat.

## Prompt slice 0 (fondasi)
```
Baca CLAUDE.md, docs/ROADMAP.md bagian Slice 0, dan docs/BRD.md Bagian 5, 6, 7, dan 9.
Buat rencana (jangan menulis kode dulu) untuk fondasi project:
struktur folder sesuai BRD Bagian 7, docker-compose (PostgreSQL), Makefile,
golang-migrate, sqlc, config dari env, logger, middleware (request id, recovery, CORS),
helper response/error sesuai BRD Bagian 9, endpoint GET /health,
dan docs/openapi.yaml awal beserta halaman dokumentasinya.
Tampilkan rencana untuk saya setujui.
```

## Prompt slice 1 (auth), setelah slice 0 selesai
```
Baca CLAUDE.md, docs/ROADMAP.md bagian Slice 1, dan docs/BRD.md Bagian 3.1, 2, dan 8.1.
Rencanakan auth dan RBAC: tabel users, roles, refresh_tokens, branches; login dengan email
atau nomor HP; refresh; logout; ganti kata sandi; middleware role.
Mulai dari perubahan docs/openapi.yaml. Tunjukkan rencana dulu.
```

## Pola untuk slice berikutnya
```
Kerjakan Slice <n> di docs/ROADMAP.md. Baca bagian BRD yang dirujuk dan item BD yang terkait.
Rencanakan dulu: kontrak OpenAPI, migrasi, query, service, handler, test, seed.
Setelah selesai, perbarui docs/STATUS.md.
```

## Kebiasaan yang membantu
- Satu fitur per sesi. Mulai sesi baru ketika pindah slice.
- Review diff setiap fitur sebelum commit, terutama auth, kuota, dan poin.
- Kirim `docs/openapi.yaml` ke temanmu (FE) segera setelah kontrak slice disetujui, sebelum implementasi selesai.
- Commit kecil dan sering, satu commit per langkah (migrasi, repository, handler, test).
