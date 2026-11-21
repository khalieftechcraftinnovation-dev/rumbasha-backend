.PHONY: run build test vet fmt tidy check migrate-up migrate-down migrate-version seed

# Menjalankan API (membaca file .env kalau ada)
run:
	go run ./cmd/api

# Build aplikasi ke folder bin/
build:
	go build -o bin/ ./cmd/...

# Menjalankan semua test (memakai database test, lihat DB_NAME_TEST di .env).
# -p 1 = package dites satu per satu, supaya test tidak berebut database yang sama.
test:
	go test -p 1 ./...

# Mengecek kode yang mencurigakan
vet:
	go vet ./...

# Merapikan format semua file Go
fmt:
	gofmt -s -w .

# Menyinkronkan go.mod dan go.sum dengan import yang dipakai
tidy:
	go mod tidy

# Jalankan sebelum push: vet + test
check: vet test

# Menjalankan semua migration yang belum dijalankan
migrate-up:
	go run ./cmd/migrate up

# Membatalkan 1 migration terakhir
migrate-down:
	go run ./cmd/migrate down

# Melihat versi migration saat ini
migrate-version:
	go run ./cmd/migrate version

# Mengisi data dummy (akun tiap role, paket, coin). Tidak bisa dijalankan di production.
seed:
	go run ./cmd/seed
