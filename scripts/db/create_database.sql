-- ============================================================================
-- Membuat database project Rumbasha.
--
-- Cara pakai (SEKALI saja):
--   1. Di DBeaver, buka SQL Editor dari koneksi "postgres" (database postgres).
--   2. Jalankan sebagai script: Alt+X (bukan Ctrl+Enter), karena CREATE DATABASE
--      tidak bisa dijalankan di dalam transaction.
--   3. Setelah itu, tabel dibuat lewat migration: make migrate-up
--
-- Kalau database sudah ada, baris CREATE DATABASE akan error "already exists";
-- itu aman diabaikan.
-- ============================================================================

-- Database untuk development
CREATE DATABASE rumbasha
    WITH ENCODING 'UTF8'
    TEMPLATE template0;

-- Database khusus test (isinya dihapus setiap kali test berjalan)
CREATE DATABASE rumbasha_test
    WITH ENCODING 'UTF8'
    TEMPLATE template0;

-- Zona waktu tampilan mengikuti aturan bisnis BR-014 (Asia/Jakarta).
-- Kolom TIMESTAMPTZ tetap menyimpan waktu yang sama persis; setting ini hanya
-- mengubah cara waktu ditampilkan, misalnya saat membuka tabel di DBeaver.
ALTER DATABASE rumbasha      SET timezone TO 'Asia/Jakarta';
ALTER DATABASE rumbasha_test SET timezone TO 'Asia/Jakarta';
