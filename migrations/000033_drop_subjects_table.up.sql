-- Hapus tabel contoh "subjects" (migration 000001) beserta API /subjects lamanya.
-- Mapel sekarang disimpan di master_data_items (type = 'subject') bersama hierarkinya
-- (bidang materi → bab → sub bab → topik) dan dipakai bank soal, quiz, dropdown.
-- Tabel ini tidak direferensikan tabel lain dan tidak dipakai fitur mana pun.
DROP TABLE IF EXISTS subjects;
