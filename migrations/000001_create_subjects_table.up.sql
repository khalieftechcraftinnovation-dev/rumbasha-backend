-- Membuat tabel mata pelajaran.
-- gen_random_uuid() adalah fungsi bawaan PostgreSQL 13+ untuk membuat UUID otomatis.
CREATE TABLE subjects (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(20)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
