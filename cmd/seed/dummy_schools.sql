-- ============================================================================
-- DATA DUMMY sekolah (master data type 'school'). JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" (setelah akun dummy). Aman dijalankan berkali-kali.
-- Isinya = asal sekolah & target sekolah yang dipakai akun dummy + beberapa sekolah tambahan,
-- supaya tab Sekolah di menu Master Data dan dropdown "school" langsung berisi.
-- ============================================================================

INSERT INTO master_data_items (type, name, sort_order)
SELECT 'school', s.name, ROW_NUMBER() OVER (ORDER BY s.name)
FROM (
    SELECT DISTINCT btrim(name) AS name
    FROM (
        SELECT school_origin AS name FROM users    WHERE school_origin IS NOT NULL AND deleted_at IS NULL
        UNION
        SELECT target_school         FROM students WHERE target_school IS NOT NULL
        UNION
        SELECT v FROM (VALUES ('SDN Kertajaya 1'), ('SMPN 6 Surabaya'), ('SMAN 1 Sidoarjo'), ('SMP Al Hikmah Surabaya')) AS extra (v)
    ) all_names
    WHERE btrim(name) <> ''
) s
WHERE NOT EXISTS (
    SELECT 1 FROM master_data_items m
    WHERE m.type = 'school' AND LOWER(m.name) = LOWER(s.name) AND m.deleted_at IS NULL
);
