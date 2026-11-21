.PHONY: run build test vet fmt tidy check

# Menjalankan API (membaca file .env kalau ada)
run:
	go run ./cmd/api

# Build aplikasi ke folder bin/
build:
	go build -o bin/ ./cmd/...

# Menjalankan semua test
test:
	go test ./...

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
